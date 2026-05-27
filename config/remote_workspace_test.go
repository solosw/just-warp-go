package config

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadRemoteWorkspaces(t *testing.T) {
	tmp := t.TempDir()
	store := &Store{dir: tmp}

	item := RemoteWorkspaceEntry{
		Name:       "prod-api",
		Host:       "10.0.0.8",
		Port:       22,
		User:       "root",
		RemotePath: "/srv/app",
		CachePath:  filepath.Join(tmp, "cache", "prod-api"),
	}

	if err := store.SaveRemoteWorkspace(item); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	items, err := store.LoadRemoteWorkspaces()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].RemotePath != "/srv/app" {
		t.Fatalf("expected remote path /srv/app, got %q", items[0].RemotePath)
	}
}

func TestSaveRemoteWorkspaceReplacesSameName(t *testing.T) {
	tmp := t.TempDir()
	store := &Store{dir: tmp}

	first := RemoteWorkspaceEntry{Name: "prod", Host: "1.1.1.1", Port: 22, User: "root", RemotePath: "/a", CachePath: "c1"}
	second := RemoteWorkspaceEntry{Name: "prod", Host: "2.2.2.2", Port: 2222, User: "deploy", RemotePath: "/b", CachePath: "c2"}

	if err := store.SaveRemoteWorkspace(first); err != nil {
		t.Fatalf("save first failed: %v", err)
	}
	if err := store.SaveRemoteWorkspace(second); err != nil {
		t.Fatalf("save second failed: %v", err)
	}

	items, err := store.LoadRemoteWorkspaces()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after replace, got %d", len(items))
	}
	if items[0].Host != "2.2.2.2" || items[0].RemotePath != "/b" {
		t.Fatalf("expected replacement to win, got %#v", items[0])
	}
}
