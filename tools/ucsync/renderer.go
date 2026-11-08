package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"ucprint/internal/snapshot"
)

const rendererArchiveURL = "https://github.com/UCProjects/CardEditor/archive/refs/heads/master.zip"

// Acquires reference material only; never executes it or modifies the snapshot.
func fetchRendererArchive() ([]byte, error) {
	client := http.Client{Timeout: 90 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || (r.URL.Host != "github.com" && r.URL.Host != "codeload.github.com") || len(via) > 5 {
			return fmt.Errorf("unexpected archive redirect")
		}
		return nil
	}}
	r, e := client.Get(rendererArchiveURL)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub returned HTTP %d", r.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, (128<<20)+1))
	if e != nil {
		return nil, e
	}
	if len(b) > 128<<20 {
		return nil, fmt.Errorf("archive exceeds 128 MB")
	}
	return b, nil
}

func collectRendererArchive(archive []byte, output string) error {
	z, e := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if e != nil {
		return fmt.Errorf("expected a GitHub ZIP archive: %w", e)
	}
	files := map[string][]byte{}
	var total int64
	for _, f := range z.File {
		if f.FileInfo().IsDir() {
			continue
		}
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) != 2 || !strings.HasPrefix(parts[0], "CardEditor-") {
			continue
		}
		name := parts[1]
		if strings.Contains(name, "\\") || path.Clean(name) != name || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") {
			return fmt.Errorf("unsafe archive path: %s", name)
		}
		selected := strings.HasPrefix(name, "public/rarity/") || strings.HasPrefix(name, "public/resources/modules/") ||
			strings.HasPrefix(name, "public/resources/tribes/") ||
			strings.HasPrefix(name, "public/resources/data/") || name == "public/index.html" || name == "package.json" ||
			strings.HasPrefix(strings.ToLower(name), "license") || strings.HasPrefix(strings.ToLower(name), "readme") ||
			(path.Dir(name) == "public/resources" && (strings.HasSuffix(name, ".css") || strings.HasSuffix(name, ".otf") || strings.HasSuffix(name, ".js")))
		if !selected {
			continue
		}
		if _, ok := files[name]; ok {
			return fmt.Errorf("duplicate archive file: %s", name)
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unexpected archive symlink: %s", name)
		}
		r, e := f.Open()
		if e != nil {
			return e
		}
		b, e := io.ReadAll(io.LimitReader(r, (8<<20)+1))
		r.Close()
		if e != nil {
			return fmt.Errorf("%s: %w", name, e)
		}
		total += int64(len(b))
		if len(b) == 0 || len(b) > 8<<20 || total > 64<<20 {
			return fmt.Errorf("empty or oversized renderer resource: %s", name)
		}
		if strings.HasSuffix(strings.ToLower(name), ".png") {
			if e := snapshot.CheckImage(b); e != nil {
				return fmt.Errorf("%s: %w", name, e)
			}
		}
		files[name] = b
	}
	for _, required := range []string{"public/resources/undercards.css", "public/resources/DTM-Mono.otf", "public/resources/modules/card.js", "public/rarity/TOKEN.png"} {
		if len(files[required]) == 0 {
			return fmt.Errorf("archive is missing %s", required)
		}
	}
	root, e := filepath.Abs(output)
	if e != nil {
		return e
	}
	if exists(root) {
		return fmt.Errorf("%s already exists; use --output with a new folder to keep the previous source bundle", root)
	}
	parent := filepath.Dir(root)
	if e = os.MkdirAll(parent, 0755); e != nil {
		return e
	}
	stage, e := os.MkdirTemp(parent, ".renderer-")
	if e != nil {
		return e
	}
	defer removeWithin(parent, stage)
	checksums := map[string]string{}
	for name, b := range files {
		if e = write(stage, name, b); e != nil {
			return e
		}
		checksums[name] = snapshot.Hash(b)
	}
	if e = write(stage, "source.zip", archive); e != nil {
		return e
	}
	if e = writeJSON(stage, "source-manifest.json", map[string]any{
		"source": rendererArchiveURL, "collectedAt": time.Now().Format(time.RFC3339), "archiveSHA256": snapshot.Hash(archive),
		"fileCount": len(files), "checksums": checksums, "purpose": "Renderer source inspection; not installed into the application",
	}); e != nil {
		return e
	}
	if e = os.Rename(stage, root); e != nil {
		return e
	}
	fmt.Printf("Saved %d renderer source files to %s. The original ZIP and checksums are included.\n", len(files), output)
	fmt.Println("Next: compare the source renderer with CardTest/IntendedSave.png before connecting it to UC Print. Your working cards and executable were not changed.")
	return nil
}

func rendererResources(input, output string) error {
	var b []byte
	var e error
	if input != "" {
		b, e = os.ReadFile(input)
	} else {
		b, e = fetchRendererArchive()
	}
	if e != nil {
		return fmt.Errorf("renderer acquisition failed: %w\nBrowser fallback: save %s as Sources/CardEditor.zip, then run go run ./tools/ucsync renderer-import ./Sources/CardEditor.zip", e, rendererArchiveURL)
	}
	return collectRendererArchive(b, output)
}
