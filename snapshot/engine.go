package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const snapDir = ".warp-snapshots"

// Manifest maps relative file paths to their content hashes.
type Manifest struct {
	Files map[string]string `json:"files"`
}

// Engine manages file snapshots for a workspace.
type Engine struct {
	workspace string
	snapPath  string
	manifest  *Manifest
}

// NewEngine creates a snapshot engine for the given workspace.
func NewEngine(workspace string) *Engine {
	return &Engine{
		workspace: workspace,
		snapPath:  filepath.Join(workspace, snapDir),
		manifest:  &Manifest{Files: make(map[string]string)},
	}
}

// Init creates the snapshot directory and saves initial snapshots for all files.
func (e *Engine) Init(files []string) error {
	if err := os.MkdirAll(e.snapPath, 0755); err != nil {
		return fmt.Errorf("create snapshot dir: %w", err)
	}
	for _, f := range files {
		if err := e.snapshotFile(f); err != nil {
			return fmt.Errorf("snapshot %s: %w", f, err)
		}
	}
	return e.saveManifest()
}

// AcceptAll re-snapshots all changed files.
func (e *Engine) AcceptAll(files []string) error {
	for _, f := range files {
		if err := e.snapshotFile(f); err != nil {
			// If file was deleted, remove from manifest
			if os.IsNotExist(err) {
				delete(e.manifest.Files, f)
				os.Remove(e.snapFilePath(f))
				continue
			}
			return err
		}
	}
	return e.saveManifest()
}

// AcceptFile re-snapshots a single file.
func (e *Engine) AcceptFile(path string) error {
	if err := e.snapshotFile(path); err != nil {
		if os.IsNotExist(err) {
			delete(e.manifest.Files, path)
			os.Remove(e.snapFilePath(path))
			return e.saveManifest()
		}
		return err
	}
	return e.saveManifest()
}

// RevertAll restores all files from their snapshots.
func (e *Engine) RevertAll(files []string) error {
	for _, f := range files {
		if err := e.RevertFile(f); err != nil {
			return err
		}
	}
	return nil
}

// RevertFile restores a single file from its snapshot.
func (e *Engine) RevertFile(path string) error {
	snapFile := e.snapFilePath(path)
	if _, ok := e.manifest.Files[path]; !ok {
		// File was newly created, delete it
		e.deleteFromManifest(path)
		return os.Remove(filepath.Join(e.workspace, path))
	}
	data, err := os.ReadFile(snapFile)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	targetPath := filepath.Join(e.workspace, path)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0644)
}

// Diff returns the diff between current file and snapshot.
func (e *Engine) Diff(path string) (oldContent, newContent string, err error) {
	oldData, err := os.ReadFile(e.snapFilePath(path))
	if err != nil {
		oldContent = "" // new file
	} else {
		oldContent = string(oldData)
	}
	newData, err := os.ReadFile(filepath.Join(e.workspace, path))
	if err != nil {
		newContent = "" // deleted file
	} else {
		newContent = string(newData)
	}
	return oldContent, newContent, nil
}

// HasSnapshot returns true if the engine has an existing manifest with files.
func (e *Engine) HasSnapshot() bool {
	return len(e.manifest.Files) > 0
}

// ChangedFiles returns files that differ from their snapshots, with line stats.
func (e *Engine) ChangedFiles(currentFiles []string) []FileChange {
	currentSet := make(map[string]bool, len(currentFiles))
	for _, f := range currentFiles {
		currentSet[f] = true
	}
	var changes []FileChange
	for _, f := range currentFiles {
		oldHash, existed := e.manifest.Files[f]
		if !existed {
			adds, _ := e.diffStats("", filepath.Join(e.workspace, f))
			changes = append(changes, FileChange{Path: f, Status: StatusAdded, Additions: adds})
		} else {
			newHash := hashFile(filepath.Join(e.workspace, f))
			if newHash != oldHash {
				adds, dels := e.diffStats(e.snapFilePath(f), filepath.Join(e.workspace, f))
				changes = append(changes, FileChange{Path: f, Status: StatusModified, Additions: adds, Deletions: dels})
			}
		}
	}
	for f := range e.manifest.Files {
		if !currentSet[f] {
			_, dels := e.diffStats(e.snapFilePath(f), "")
			changes = append(changes, FileChange{Path: f, Status: StatusDeleted, Deletions: dels})
		}
	}
	return changes
}

// diffStats reads oldPath and newPath, returns (additions, deletions) line counts.
// Pass empty string for a non-existent path (new or deleted file).
func (e *Engine) diffStats(oldPath, newPath string) (additions, deletions int) {
	oldLines := readLines(oldPath)
	newLines := readLines(newPath)

	oldCount := make(map[string]int, len(oldLines))
	for _, l := range oldLines {
		oldCount[l]++
	}
	newCount := make(map[string]int, len(newLines))
	for _, l := range newLines {
		newCount[l]++
	}

	// Lines in new but not (fully) in old = additions
	for l, n := range newCount {
		o := oldCount[l]
		if n > o {
			additions += n - o
		}
	}
	// Lines in old but not (fully) in new = deletions
	for l, o := range oldCount {
		n := newCount[l]
		if o > n {
			deletions += o - n
		}
	}
	return
}

func readLines(path string) []string {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	text := strings.TrimSuffix(string(data), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// LoadManifest loads existing manifest from disk.
func (e *Engine) LoadManifest() error {
	data, err := os.ReadFile(filepath.Join(e.snapPath, "manifest.json"))
	if err != nil {
		if os.IsNotExist(err) {
			e.manifest = &Manifest{Files: make(map[string]string)}
			return nil
		}
		return err
	}
	e.manifest = &Manifest{}
	return json.Unmarshal(data, e.manifest)
}

func (e *Engine) snapshotFile(relPath string) error {
	src := filepath.Join(e.workspace, relPath)
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if !IsTextFile(strings.ToLower(filepath.Ext(relPath)), FirstBytes(data)) {
		return fmt.Errorf("skip binary file")
	}
	dst := e.snapFilePath(relPath)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return err
	}
	e.manifest.Files[relPath] = hashBytes(data)
	return nil
}

// FirstBytes returns the first 512 bytes of data for content-type detection.
func FirstBytes(data []byte) []byte {
	if len(data) > 512 {
		return data[:512]
	}
	return data
}

func (e *Engine) saveManifest() error {
	data, err := json.MarshalIndent(e.manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.snapPath, "manifest.json"), data, 0644)
}

func (e *Engine) snapFilePath(relPath string) string {
	return filepath.Join(e.snapPath, relPath)
}

func (e *Engine) deleteFromManifest(path string) {
	delete(e.manifest.Files, path)
	e.saveManifest()
}

// FileChange types
const (
	StatusAdded    = "added"
	StatusModified = "modified"
	StatusDeleted  = "deleted"
)

type FileChange struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

func hashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return hashBytes(data)
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HashBytes returns the hex SHA-256 of data (public).
func HashBytes(data []byte) string {
	return hashBytes(data)
}

// Common binary file extensions.
var binaryExts = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".7z": true, ".rar": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".ico": true, ".webp": true, ".svg": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mov": true, ".mkv": true, ".wmv": true, ".flv": true,
	".woff": true, ".woff2": true, ".ttf": true, ".otf": true, ".eot": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".o": true, ".obj": true, ".a": true, ".lib": true,
	".class": true, ".pyc": true, ".pyo": true,
	".jar": true, ".war": true, ".ear": true,
	".db": true, ".sqlite": true, ".sqlite3": true,
	".wasm": true,
}

// IsTextFile checks whether a file is a text file based on its extension and content.
// ext should be the lowercase file extension (e.g. ".go").
// peek should be the first up-to-512 bytes of the file (or nil if unavailable).
func IsTextFile(ext string, peek []byte) bool {
	if binaryExts[ext] {
		return false
	}
	if len(peek) > 0 {
		for _, b := range peek {
			if b == 0 {
				return false
			}
		}
	}
	return true
}

// DiffStats computes line additions and deletions between old and new content.
func DiffStats(oldData, newData []byte) (additions, deletions int) {
	oldLines := readLinesFromBytes(oldData)
	newLines := readLinesFromBytes(newData)
	oldCount := make(map[string]int, len(oldLines))
	for _, l := range oldLines {
		oldCount[l]++
	}
	newCount := make(map[string]int, len(newLines))
	for _, l := range newLines {
		newCount[l]++
	}
	for l, n := range newCount {
		o := oldCount[l]
		if n > o {
			additions += n - o
		}
	}
	for l, o := range oldCount {
		n := newCount[l]
		if o > n {
			deletions += o - n
		}
	}
	return
}

func readLinesFromBytes(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	text := strings.TrimSuffix(string(data), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// ChangedFilesByHash compares current hashes against stored manifest without reading files.
func (e *Engine) ChangedFilesByHash(currentHashes map[string]string) []FileChange {
	currentSet := make(map[string]bool, len(currentHashes))
	for f := range currentHashes {
		currentSet[f] = true
	}
	var changes []FileChange
	for f, newHash := range currentHashes {
		oldHash, existed := e.manifest.Files[f]
		if !existed {
			changes = append(changes, FileChange{Path: f, Status: StatusAdded})
		} else if newHash != oldHash {
			changes = append(changes, FileChange{Path: f, Status: StatusModified})
		}
	}
	for f := range e.manifest.Files {
		if !currentSet[f] {
			changes = append(changes, FileChange{Path: f, Status: StatusDeleted})
		}
	}
	return changes
}

// SetFileHash sets a single file hash in the manifest in-memory (does not save).
func (e *Engine) SetFileHash(path, hash string) {
	e.manifest.Files[path] = hash
}

// LoadManifestFrom parses manifest JSON from bytes.
func (e *Engine) LoadManifestFrom(data []byte) error {
	e.manifest = &Manifest{}
	return json.Unmarshal(data, e.manifest)
}

// MarshalManifest returns the manifest as JSON bytes.
func (e *Engine) MarshalManifest() ([]byte, error) {
	return json.MarshalIndent(e.manifest, "", "  ")
}

// RemoveFromManifest deletes entries from manifest (for remote revert).
func (e *Engine) RemoveFromManifest(paths []string) error {
	for _, path := range paths {
		delete(e.manifest.Files, path)
	}
	return e.saveManifest()
}

// FilterManifest removes entries that don't match the keep predicate.
// Used to clean stale directory entries from older remote workspace manifests.
func (e *Engine) FilterManifest(keep func(path string) bool) {
	cleaned := make(map[string]string, len(e.manifest.Files))
	for p, h := range e.manifest.Files {
		if keep(p) {
			cleaned[p] = h
		}
	}
	e.manifest.Files = cleaned
	e.saveManifest()
}

// ReadFileContent reads a file from workspace.
func ReadFileContent(workspace, relPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(workspace, relPath))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
