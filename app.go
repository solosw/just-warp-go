package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"just-warp-go/config"
	"just-warp-go/scanner"
	"just-warp-go/snapshot"
	"just-warp-go/terminal"
	"just-warp-go/watcher"
)

// App is the main application struct with bound methods.
type App struct {
	ctx              context.Context
	workspace        string
	startupWorkspace string // set via CLI --workspace flag

	snapEng  *snapshot.Engine
	termMgr  *terminal.Manager
	fsw      *watcher.Watcher
	cfgStore *config.Store

	scannedFiles []string
	mu           sync.Mutex
}

func NewApp() *App {
	store, err := config.NewStore()
	if err != nil {
		println("config store init failed:", err.Error())
		store = nil // Best-effort, workspace history won't persist
	}
	return &App{
		termMgr:  terminal.NewManager(),
		cfgStore: store,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetStartupWorkspace returns the workspace path from CLI, used by frontend to auto-open.
func (a *App) GetStartupWorkspace() string {
	return a.startupWorkspace
}

// OpenInNewWindow launches a new app instance for the given workspace.
func (a *App) OpenInNewWindow(path string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("找不到可执行文件: %w", err)
	}
	cmd := exec.Command(exe, "--workspace", path)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动新窗口失败: %w", err)
	}
	return nil
}

func (a *App) shutdown(ctx context.Context) {
	a.termMgr.CloseAll()
	if a.fsw != nil {
		a.fsw.Close()
	}
}

// ─── Workspace ───────────────────────────────────────

type WorkspaceInfo struct {
	Path         string               `json:"path"`
	Name         string               `json:"name"`
	FileCount    int                  `json:"fileCount"`
	Files        []string             `json:"files"`
	ChangedFiles []snapshot.FileChange `json:"changedFiles"`
}

// SelectWorkspace opens a folder dialog and initializes the workspace.
func (a *App) SelectWorkspace() (*WorkspaceInfo, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择工作区文件夹",
	})
	if err != nil {
		return nil, fmt.Errorf("选择文件夹失败: %w", err)
	}
	if path == "" {
		return nil, nil
	}
	return a.OpenWorkspace(path)
}

// OpenWorkspace initializes a workspace at the given path.
func (a *App) OpenWorkspace(path string) (*WorkspaceInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.fsw != nil {
		a.fsw.Close()
	}

	a.workspace = path
	a.snapEng = snapshot.NewEngine(path)

	result, err := scanner.Scan(path)
	if err != nil {
		return nil, fmt.Errorf("扫描失败: %w", err)
	}
	a.scannedFiles = result.Files

	if err := a.snapEng.LoadManifest(); err != nil {
		return nil, fmt.Errorf("加载快照失败: %w", err)
	}

	if !a.snapEng.HasSnapshot() {
		if err := a.snapEng.Init(result.Files); err != nil {
			return nil, fmt.Errorf("创建快照失败: %w", err)
		}
	}

	a.fsw, err = watcher.New(path, func(events []string) {
		a.onFileChanged()
	})
	if err != nil {
		return nil, fmt.Errorf("启动文件监听失败: %w", err)
	}

	// Save to workspace history
	if a.cfgStore != nil {
		a.cfgStore.SaveWorkspace(path)
	}

	info := a.makeWorkspaceInfo()
	a.emitChanges() // push initial changes to frontend event listener
	return info, nil
}

// GetWorkspaceHistory returns the list of previously opened workspaces.
func (a *App) GetWorkspaceHistory() []config.WorkspaceEntry {
	if a.cfgStore == nil {
		return nil
	}
	entries, err := a.cfgStore.LoadWorkspaces()
	if err != nil {
		return nil
	}
	return entries
}

// RemoveWorkspaceFromHistory removes a workspace from history.
func (a *App) RemoveWorkspaceFromHistory(path string) error {
	if a.cfgStore == nil {
		return nil
	}
	return a.cfgStore.RemoveWorkspace(path)
}

// GetWorkspaceInfo returns current workspace information.
func (a *App) GetWorkspaceInfo() *WorkspaceInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.workspace == "" || a.snapEng == nil {
		return nil
	}
	return a.makeWorkspaceInfo()
}

func (a *App) makeWorkspaceInfo() *WorkspaceInfo {
	changes := a.snapEng.ChangedFiles(a.scannedFiles)
	return &WorkspaceInfo{
		Path:         a.workspace,
		Name:         a.workspaceName(),
		FileCount:    len(a.scannedFiles),
		Files:        a.scannedFiles,
		ChangedFiles: changes,
	}
}

func (a *App) workspaceName() string {
	if a.workspace == "" {
		return ""
	}
	for i := len(a.workspace) - 1; i >= 0; i-- {
		if a.workspace[i] == '\\' || a.workspace[i] == '/' {
			return a.workspace[i+1:]
		}
	}
	return a.workspace
}

func (a *App) onFileChanged() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapEng == nil {
		return
	}
	changes := a.snapEng.ChangedFiles(a.scannedFiles)
	runtime.EventsEmit(a.ctx, "file-changes", changes)
}

// ─── File Changes ────────────────────────────────────

func (a *App) GetChangedFiles() []snapshot.FileChange {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapEng == nil {
		return nil
	}
	return a.snapEng.ChangedFiles(a.scannedFiles)
}

func (a *App) AcceptAll() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapEng == nil {
		return fmt.Errorf("未选择工作区")
	}
	changes := a.snapEng.ChangedFiles(a.scannedFiles)
	var paths []string
	for _, c := range changes {
		paths = append(paths, c.Path)
	}
	if err := a.snapEng.AcceptAll(paths); err != nil {
		return err
	}
	a.refreshScan()
	a.emitChanges()
	return nil
}

func (a *App) RevertAll() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapEng == nil {
		return fmt.Errorf("未选择工作区")
	}
	changes := a.snapEng.ChangedFiles(a.scannedFiles)
	var paths []string
	for _, c := range changes {
		paths = append(paths, c.Path)
	}
	if err := a.snapEng.RevertAll(paths); err != nil {
		return err
	}
	a.refreshScan()
	a.emitChanges()
	return nil
}

func (a *App) AcceptFile(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapEng == nil {
		return fmt.Errorf("未选择工作区")
	}
	if err := a.snapEng.AcceptFile(path); err != nil {
		return err
	}
	a.refreshScan()
	a.emitChanges()
	return nil
}

func (a *App) RevertFile(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapEng == nil {
		return fmt.Errorf("未选择工作区")
	}
	if err := a.snapEng.RevertFile(path); err != nil {
		return err
	}
	a.refreshScan()
	a.emitChanges()
	return nil
}

func (a *App) GetFileDiff(path string) (map[string]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapEng == nil {
		return nil, fmt.Errorf("未选择工作区")
	}
	oldC, newC, err := a.snapEng.Diff(path)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"old": oldC,
		"new": newC,
	}, nil
}

// GetFileContent reads the current content of a file in the workspace.
func (a *App) GetFileContent(path string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.workspace == "" {
		return "", fmt.Errorf("未选择工作区")
	}
	return snapshot.ReadFileContent(a.workspace, path)
}

// ─── Terminal ────────────────────────────────────────

func (a *App) CreateTerminal() (string, error) {
	id, err := a.termMgr.Create()
	if err != nil {
		return "", err
	}
	sess, _ := a.termMgr.Get(id)
	go a.readTerminalOutput(id, sess)
	return id, nil
}

func (a *App) WriteToTerminal(tabId string, data string) error {
	sess, err := a.termMgr.Get(tabId)
	if err != nil {
		return err
	}
	_, err = sess.Write([]byte(data))
	return err
}

func (a *App) ResizeTerminal(tabId string, cols, rows int) error {
	sess, err := a.termMgr.Get(tabId)
	if err != nil {
		return err
	}
	return sess.Resize(uint16(rows), uint16(cols))
}

func (a *App) CloseTerminal(tabId string) error {
	return a.termMgr.Close(tabId)
}

func (a *App) readTerminalOutput(id string, sess *terminal.Session) {
	buf := make([]byte, 4096)
	for {
		n, err := sess.Read(buf)
		if err != nil {
			runtime.EventsEmit(a.ctx, "terminal-output:"+id, "\r\n[终端已关闭]")
			return
		}
		if n > 0 {
			runtime.EventsEmit(a.ctx, "terminal-output:"+id, string(buf[:n]))
		}
	}
}

func (a *App) refreshScan() {
	result, err := scanner.Scan(a.workspace)
	if err != nil {
		return
	}
	a.scannedFiles = result.Files
}

func (a *App) emitChanges() {
	changes := a.snapEng.ChangedFiles(a.scannedFiles)
	runtime.EventsEmit(a.ctx, "file-changes", changes)
}
