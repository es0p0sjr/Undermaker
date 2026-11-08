package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"ucprint/internal/snapshot"
	"ucprint/internal/testfixture"
)

func TestOfflineImportAndRawPreservation(t *testing.T) {
	input := t.TempDir()
	raw, e := testfixture.Input(input)
	if e != nil {
		t.Fatal(e)
	}
	raw = append([]byte(" \n"), raw...)
	os.WriteFile(filepath.Join(input, "AllCards.json"), raw, 0644)
	root := filepath.Join(t.TempDir(), "data")
	o := options{Data: root, Import: filepath.Join(input, "AllCards.json"), Assets: filepath.Join(input, "assets"), Offline: true}
	if e = syncSnapshot(o, func(string) ([]byte, error) { t.Fatal("network used"); return nil, nil }); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(filepath.Join(root, "raw/AllCards.json"))
	if !bytes.Equal(raw, b) {
		t.Fatal("raw response changed")
	}
	if _, e = snapshot.Load(os.DirFS(root), true); e != nil {
		t.Fatal(e)
	}
}
func TestFailedUpdatesPreserveSnapshot(t *testing.T) {
	for _, mode := range []string{"403", "missing-image", "bad-json", "unresolved"} {
		t.Run(mode, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "data")
			if e := testfixture.Snapshot(root); e != nil {
				t.Fatal(e)
			}
			before, _ := os.ReadFile(filepath.Join(root, "manifest.json"))
			input := t.TempDir()
			testfixture.Input(input)
			o := options{Data: root}
			fetch := func(string) ([]byte, error) { return nil, fmt.Errorf("HTTP 403") }
			if mode != "403" {
				o.Import = filepath.Join(input, "AllCards.json")
				o.Assets = filepath.Join(input, "assets")
				o.Offline = true
			}
			switch mode {
			case "missing-image":
				os.Remove(filepath.Join(input, "assets/cards/test.png"))
			case "bad-json":
				os.WriteFile(o.Import, []byte("not json"), 0644)
			case "unresolved":
				os.WriteFile(o.Import, []byte(`[{"id":1,"name":"Bad","description":"{{UNKNOWN:1}}","image":"test","cost":1}]`), 0644)
			}
			if e := syncSnapshot(o, fetch); e == nil {
				t.Fatal("update unexpectedly succeeded")
			}
			after, _ := os.ReadFile(filepath.Join(root, "manifest.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("working manifest changed")
			}
			if _, e := snapshot.Load(os.DirFS(root), true); e != nil {
				t.Fatal(e)
			}
			if exists(filepath.Join(root, ".staging")) {
				t.Fatal("failed staging not cleaned")
			}
		})
	}
}
func TestReferencedPowersOnlyAndDownloads(t *testing.T) {
	input := t.TempDir()
	raw, _ := testfixture.Input(input)
	calls := map[string]int{}
	root := filepath.Join(t.TempDir(), "data")
	fetch := func(u string) ([]byte, error) {
		calls[u]++
		if u == source+"/AllCards" {
			return raw, nil
		}
		if u == source+"/images/cards/test.png" || u == source+"/images/powers/Taunt.png" {
			return testfixture.PNG(), nil
		}
		return nil, fmt.Errorf("unexpected URL %s", u)
	}
	if e := syncSnapshot(options{Data: root}, fetch); e != nil {
		t.Fatal(e)
	}
	if len(calls) != 3 {
		t.Fatal(calls)
	}
}
func TestRecoveryOfInterruptedPromotion(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "data")
	backup := filepath.Join(parent, ".data.previous")
	testfixture.Snapshot(backup)
	if e := syncSnapshot(options{Data: root}, func(string) ([]byte, error) { return nil, fmt.Errorf("403") }); e == nil {
		t.Fatal("expected download failure")
	}
	if _, e := snapshot.Load(os.DirFS(root), true); e != nil {
		t.Fatal(e)
	}
}

func TestRejectWorkspaceAsSnapshotDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{cwd, filepath.Dir(cwd)} {
		if e := syncSnapshot(options{Data: root}, func(string) ([]byte, error) {
			t.Fatal("must reject unsafe data target before any download")
			return nil, nil
		}); e == nil {
			t.Fatal("accepted workspace as snapshot directory")
		}
	}
}
