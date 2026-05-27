package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"

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
	workspaceName    string
	startupWorkspace string
	isRemote         bool

	// Remote connection (lifetime = workspace session)
	remoteClient  *ssh.Client
	remoteSFTP    *sftp.Client
	remotePath    string
	remoteSSHCfg  terminal.SSHConfig // saved for auto-creating SSH terminals

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
		store = nil
	}
	return &App{
		termMgr:  terminal.NewManager(),
		cfgStore: store,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetStartupWorkspace() string { return a.startupWorkspace }

func (a *App) OpenInNewWindow(path string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("找不到可执行文件: %w", err)
	}
	return exec.Command(exe, "--workspace", path).Start()
}

func (a *App) shutdown(ctx context.Context) {
	a.termMgr.CloseAll()
	if a.fsw != nil {
		a.fsw.Close()
	}
	a.closeRemote()
}

func (a *App) closeRemote() {
	if a.remoteSFTP != nil {
		a.remoteSFTP.Close()
		a.remoteSFTP = nil
	}
	if a.remoteClient != nil {
		a.remoteClient.Close()
		a.remoteClient = nil
	}
	a.isRemote = false
	a.remotePath = ""
}

// ─── Workspace ───────────────────────────────────────

type WorkspaceInfo struct {
	Path         string               `json:"path"`
	Name         string               `json:"name"`
	FileCount    int                  `json:"fileCount"`
	Files        []string             `json:"files"`
	IsRemote     bool                 `json:"isRemote"`
	ChangedFiles []snapshot.FileChange `json:"changedFiles"`
}

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

func (a *App) OpenWorkspace(path string) (*WorkspaceInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closeRemote()

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

	a.fsw, err = watcher.New(path, func(events []string) { a.onFileChanged() })
	if err != nil {
		return nil, fmt.Errorf("启动文件监听失败: %w", err)
	}

	if a.cfgStore != nil {
		a.cfgStore.SaveWorkspace(path)
	}

	info := a.makeWorkspaceInfo()
	a.emitChanges()
	return info, nil
}

// ─── Remote Workspace (SFTP Direct) ──────────────────

func (a *App) GetRemoteWorkspaces() ([]config.RemoteWorkspaceEntry, error) {
	if a.cfgStore == nil {
		return nil, nil
	}
	return a.cfgStore.LoadRemoteWorkspaces()
}

func (a *App) SaveRemoteWorkspace(entry config.RemoteWorkspaceEntry) error {
	if a.cfgStore == nil {
		return fmt.Errorf("配置存储不可用")
	}
	return a.cfgStore.SaveRemoteWorkspace(entry)
}

func (a *App) RemoveRemoteWorkspace(name string) error {
	if a.cfgStore == nil {
		return nil
	}
	return a.cfgStore.RemoveRemoteWorkspace(name)
}

type SSHConfig struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	KeyPath  string `json:"keyPath"`
}

func (a *App) OpenRemoteWorkspace(cfg SSHConfig, remotePath string) (*WorkspaceInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closeRemote()
	if a.fsw != nil {
		a.fsw.Close()
		a.fsw = nil
	}

	if cfg.Port == 0 {
		cfg.Port = 22
	}
	tCfg := terminal.SSHConfig{
		Name: cfg.Name, Host: cfg.Host, Port: cfg.Port,
		User: cfg.User, Password: cfg.Password, KeyPath: cfg.KeyPath,
	}
	auth, err := terminal.BuildSSHAuth(tCfg)
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %w", err)
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("SFTP初始化失败: %w", err)
	}

	// List remote files
	remoteFiles, err := a.listRemoteFiles(sftpClient, remotePath)
	if err != nil {
		sftpClient.Close()
		client.Close()
		return nil, fmt.Errorf("扫描远程目录失败: %w", err)
	}

	a.remoteClient = client
	a.remoteSFTP = sftpClient
	a.remotePath = remotePath
	a.isRemote = true
	a.workspace = cfg.Name + ":" + remotePath
	a.remoteSSHCfg = tCfg
	a.scannedFiles = remoteFiles

	// Manifest stored in config dir (hash-only, no file copies)
	// Use a workspace identifier derived from host+path
	wsID := sanitizeID(cfg.Host + "_" + remotePath)
	a.snapEng = snapshot.NewEngine(wsID)
	if err := a.snapEng.SetStorageDir(); err != nil {
		return nil, err
	}
	if err := a.snapEng.LoadManifest(); err != nil {
		return nil, err
	}
	if !a.snapEng.HasSnapshot() {
		// Init with in-memory hashes (read remote files)
		hashes := make(map[string]string)
		for _, f := range remoteFiles {
			h, err := a.readRemoteHash(f)
			if err != nil {
				continue
			}
			hashes[f] = h
		}
		if err := a.snapEng.InitHashOnly(hashes); err != nil {
			return nil, err
		}
	}

	// Save entry
	if a.cfgStore != nil {
		a.cfgStore.SaveRemoteWorkspace(config.RemoteWorkspaceEntry{
			Name:       cfg.Name,
			Host:       cfg.Host,
			Port:       cfg.Port,
			User:       cfg.User,
			RemotePath: remotePath,
		})
	}

	info := a.makeWorkspaceInfo()
	a.emitChanges()
	return info, nil
}

func (a *App) RefreshRemoteWorkspace() (*WorkspaceInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.isRemote || a.remoteSFTP == nil {
		return nil, fmt.Errorf("当前不是远程工作区")
	}
	files, err := a.listRemoteFiles(a.remoteSFTP, a.remotePath)
	if err != nil {
		return nil, err
	}
	a.scannedFiles = files
	info := a.makeWorkspaceInfo()
	a.emitChanges()
	return info, nil
}

func (a *App) listRemoteFiles(c *sftp.Client, root string) ([]string, error) {
	var files []string
	w := c.Walk(root)
	for w.Step() {
		if w.Err() != nil {
			continue
		}
		s := w.Stat()
		if s == nil || s.IsDir() {
			continue
		}
		rel := strings.TrimPrefix(path.Clean(w.Path()), path.Clean(root))
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			continue
		}
		files = append(files, filepath.ToSlash(rel))
	}
	return files, nil
}

func (a *App) readRemoteFile(relPath string) ([]byte, error) {
	rp := path.Join(a.remotePath, relPath)
	r, err := a.remoteSFTP.Open(rp)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func (a *App) readRemoteHash(relPath string) (string, error) {
	data, err := a.readRemoteFile(relPath)
	if err != nil {
		return "", err
	}
	return snapshot.HashBytes(data), nil
}

func (a *App) GetWorkspaceHistory() []config.WorkspaceEntry {
	if a.cfgStore == nil {
		return nil
	}
	entries, _ := a.cfgStore.LoadWorkspaces()
	return entries
}

func (a *App) RemoveWorkspaceFromHistory(path string) error {
	if a.cfgStore == nil {
		return nil
	}
	return a.cfgStore.RemoveWorkspace(path)
}

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
	name := a.workspace
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '\\' || name[i] == '/' {
			name = name[i+1:]
			break
		}
	}
	if a.isRemote {
		name = a.remotePath
		for i := len(name) - 1; i >= 0; i-- {
			if name[i] == '/' {
				name = name[i+1:]
				break
			}
		}
	}
	return &WorkspaceInfo{
		Path:         a.workspace,
		Name:         name,
		FileCount:    len(a.scannedFiles),
		Files:        a.scannedFiles,
		IsRemote:     a.isRemote,
		ChangedFiles: changes,
	}
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
	paths := make([]string, len(changes))
	for i, c := range changes {
		paths[i] = c.Path
	}
	if err := a.snapEng.AcceptAll(paths); err != nil {
		return err
	}
	a.refreshScanLocked()
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
	paths := make([]string, len(changes))
	for i, c := range changes {
		paths[i] = c.Path
	}
	if err := a.snapEng.RevertAll(paths); err != nil {
		return err
	}
	a.refreshScanLocked()
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
	a.refreshScanLocked()
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
	a.refreshScanLocked()
	a.emitChanges()
	return nil
}

func (a *App) GetFileDiff(path string) (map[string]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapEng == nil {
		return nil, fmt.Errorf("未选择工作区")
	}
	if a.isRemote {
		newData, err := a.readRemoteFile(path)
		if err != nil {
			return nil, err
		}
		oldData := a.snapEng.GetSnapshotContent(path)
		return map[string]string{
			"old": string(oldData),
			"new": string(newData),
		}, nil
	}
	oldC, newC, err := a.snapEng.Diff(path)
	if err != nil {
		return nil, err
	}
	return map[string]string{"old": oldC, "new": newC}, nil
}

func (a *App) GetFileContent(path string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.workspace == "" || a.snapEng == nil {
		return "", fmt.Errorf("未选择工作区")
	}
	if a.isRemote {
		data, err := a.readRemoteFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return snapshot.ReadFileContent(a.workspace, path)
}

// ─── Terminal ────────────────────────────────────────

func (a *App) CreateTerminal() (string, error) {
	var id string
	var err error
	if a.isRemote {
		id, err = a.termMgr.CreateSSH(a.remoteSSHCfg)
	} else {
		id, err = a.termMgr.Create()
	}
	if err != nil {
		return "", err
	}
	sess, _ := a.termMgr.Get(id)
	go a.readTerminalOutput(id, sess)
	return id, nil
}

func (a *App) WriteToTerminal(tabId, data string) error {
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

// ─── SSH ─────────────────────────────────────────────

func (a *App) CreateSSHTerminal(cfg SSHConfig) (string, error) {
	tCfg := terminal.SSHConfig{
		Name: cfg.Name, Host: cfg.Host, Port: cfg.Port,
		User: cfg.User, Password: cfg.Password,
	}
	id, err := a.termMgr.CreateSSH(tCfg)
	if err != nil {
		return "", err
	}
	sess, _ := a.termMgr.Get(id)
	go a.readTerminalOutput(id, sess)
	return id, nil
}

func (a *App) GetSSHConfigs() ([]config.SSHConfig, error) {
	if a.cfgStore == nil {
		return nil, nil
	}
	return a.cfgStore.LoadSSHConfigs()
}

func (a *App) SaveSSHConfig(cfg config.SSHConfig) error {
	if a.cfgStore == nil {
		return fmt.Errorf("配置存储不可用")
	}
	return a.cfgStore.SaveSSHConfig(cfg)
}

func (a *App) RemoveSSHConfig(name string) error {
	if a.cfgStore == nil {
		return nil
	}
	return a.cfgStore.RemoveSSHConfig(name)
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
	a.mu.Lock()
	defer a.mu.Unlock()
	a.refreshScanLocked()
}

func (a *App) refreshScanLocked() {
	if a.isRemote {
		files, err := a.listRemoteFiles(a.remoteSFTP, a.remotePath)
		if err != nil {
			return
		}
		a.scannedFiles = files
		return
	}
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

func sanitizeID(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
