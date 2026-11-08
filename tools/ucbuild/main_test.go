package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidSnapshotBlocksBuild(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "release.exe")
	// An existing release must survive a rejected update.
	if e := os.WriteFile(out, []byte("previous release"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := build(filepath.Join(root, "missing"), out); e == nil || !strings.Contains(e.Error(), "build blocked") {
		t.Fatalf("expected validation failure, got %v", e)
	}
	b, e := os.ReadFile(out)
	if e != nil || string(b) != "previous release" {
		t.Fatal("invalid build changed release")
	}
}
