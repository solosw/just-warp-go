package remote

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// fileMeta describes a remote file entry for mirror planning.
type fileMeta struct {
	Path string
	Hash string
}

// hashBytes returns the hex-encoded SHA-256 hash of data.
func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// MirrorPlan describes which files should be written or deleted locally
// to make the mirror match the remote workspace.
type MirrorPlan struct {
	WritePaths  []string
	DeletePaths []string
}

// BuildMirrorPlan compares local file hashes and remote file hashes.
// Files missing locally or changed remotely are written.
// Files present locally but missing remotely are deleted.
func BuildMirrorPlan(local map[string]string, remote []fileMeta) MirrorPlan {
	plan := MirrorPlan{}
	remoteSet := make(map[string]string, len(remote))
	for _, f := range remote {
		remoteSet[f.Path] = f.Hash
		if localHash, ok := local[f.Path]; !ok || localHash != f.Hash {
			plan.WritePaths = append(plan.WritePaths, f.Path)
		}
	}
	for path := range local {
		if _, ok := remoteSet[path]; !ok {
			plan.DeletePaths = append(plan.DeletePaths, path)
		}
	}
	sort.Strings(plan.WritePaths)
	sort.Strings(plan.DeletePaths)
	return plan
}
