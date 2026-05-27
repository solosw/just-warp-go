package remote

import "testing"

type fileMeta struct {
	Path string
	Hash string
}

func TestBuildMirrorPlanCreatesWritesForNewFiles(t *testing.T) {
	local := map[string]string{}
	remote := []fileMeta{{Path: "src/app.go", Hash: "a1"}}

	plan := BuildMirrorPlan(local, remote)

	if len(plan.WritePaths) != 1 || plan.WritePaths[0] != "src/app.go" {
		t.Fatalf("expected write plan for src/app.go, got %#v", plan.WritePaths)
	}
	if len(plan.DeletePaths) != 0 {
		t.Fatalf("expected no deletes, got %#v", plan.DeletePaths)
	}
}

func TestBuildMirrorPlanDeletesMissingLocalFiles(t *testing.T) {
	local := map[string]string{"old.txt": "h1"}
	remote := []fileMeta{}

	plan := BuildMirrorPlan(local, remote)

	if len(plan.DeletePaths) != 1 || plan.DeletePaths[0] != "old.txt" {
		t.Fatalf("expected delete plan for old.txt, got %#v", plan.DeletePaths)
	}
	if len(plan.WritePaths) != 0 {
		t.Fatalf("expected no writes, got %#v", plan.WritePaths)
	}
}

func TestBuildMirrorPlanUpdatesChangedFilesOnly(t *testing.T) {
	local := map[string]string{
		"same.txt":    "same",
		"changed.txt": "old",
	}
	remote := []fileMeta{
		{Path: "same.txt", Hash: "same"},
		{Path: "changed.txt", Hash: "new"},
	}

	plan := BuildMirrorPlan(local, remote)

	if len(plan.WritePaths) != 1 || plan.WritePaths[0] != "changed.txt" {
		t.Fatalf("expected only changed.txt to be written, got %#v", plan.WritePaths)
	}
	if len(plan.DeletePaths) != 0 {
		t.Fatalf("expected no deletes, got %#v", plan.DeletePaths)
	}
}
