package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// WorkspaceEntry represents a saved workspace.
type WorkspaceEntry struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	LastOpened string `json:"lastOpened"`
}

// Store manages persistent app configuration.
type Store struct {
	dir string
}

// NewStore creates a config store in the user's config directory.
func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "just-warp-go")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// LoadWorkspaces reads the workspace history.
func (s *Store) LoadWorkspaces() ([]WorkspaceEntry, error) {
	path := filepath.Join(s.dir, "workspaces.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []WorkspaceEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// SaveWorkspace adds or updates a workspace entry.
func (s *Store) SaveWorkspace(wsPath string) error {
	entries, _ := s.LoadWorkspaces()

	// Remove existing entry with same path
	filtered := make([]WorkspaceEntry, 0, len(entries))
	for _, e := range entries {
		if e.Path != wsPath {
			filtered = append(filtered, e)
		}
	}

	entry := WorkspaceEntry{
		Path:       wsPath,
		Name:       filepath.Base(wsPath),
		LastOpened: time.Now().Format(time.RFC3339),
	}

	// Prepend (most recent first), keep max 20
	filtered = append([]WorkspaceEntry{entry}, filtered...)
	if len(filtered) > 20 {
		filtered = filtered[:20]
	}

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "workspaces.json"), data, 0644)
}

// RemoveWorkspace removes a workspace from history.
func (s *Store) RemoveWorkspace(wsPath string) error {
	entries, err := s.LoadWorkspaces()
	if err != nil {
		return err
	}
	filtered := make([]WorkspaceEntry, 0, len(entries))
	for _, e := range entries {
		if e.Path != wsPath {
			filtered = append(filtered, e)
		}
	}
	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "workspaces.json"), data, 0644)
}
