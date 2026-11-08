// ucsync is the only project component that downloads Undercards resources.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ucprint/internal/snapshot"
)

const source = "https://undercards.net"

type options struct {
	ArtVariants                        string
	Data, Import, Assets, Translations string
	Offline                            bool
}
type fetchFunc func(string) ([]byte, error)

func download(target string) ([]byte, error) {
	client := http.Client{Timeout: 25 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Host != "undercards.net" || req.URL.Scheme != "https" {
			return fmt.Errorf("unexpected redirect")
		}
		if len(via) > 4 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}}
	req, e := http.NewRequest(http.MethodGet, target, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "UCPrint-Snapshot/1.0")
	req.Header.Set("Accept", "application/json,image/png,image/jpeg,*/*")
	r, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		if strings.Contains(target, "/images/") {
			return nil, fmt.Errorf("%s returned HTTP %d; JSON import cannot fix an artwork refusal. Use ucsync browser-assets to prepare a browser download helper, then import with --assets (see tools/ucsync/BROWSER.md). The current snapshot is unchanged", target, r.StatusCode)
		}
		return nil, fmt.Errorf("%s returned HTTP %d; use a manually saved JSON file with import. The current snapshot is unchanged", target, r.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if e != nil {
		return nil, e
	}
	if len(b) >= 32<<20 {
		return nil, fmt.Errorf("download too large: %s", target)
	}
	return b, nil
}
func write(root, name string, b []byte) error {
	p := filepath.Join(root, filepath.FromSlash(name))
	if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
		return e
	}
	return os.WriteFile(p, b, 0644)
}
func writeJSON(root, name string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return write(root, name, append(b, '\n'))
}
func removeWithin(parent, target string) error {
	p, e := filepath.Abs(parent)
	if e != nil {
		return e
	}
	t, e := filepath.Abs(target)
	if e != nil {
		return e
	}
	rel, e := filepath.Rel(p, t)
	if e != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("refusing cleanup outside %s", p)
	}
	return os.RemoveAll(t)
}
func exists(p string) bool { _, e := os.Stat(p); return e == nil }

func syncSnapshot(o options, fetch fetchFunc) (err error) {
	root, e := filepath.Abs(o.Data)
	if e != nil {
		return e
	}
	parent := filepath.Dir(root)
	base := filepath.Base(root)
	if root == parent || base == "." {
		return fmt.Errorf("data must be a dedicated snapshot directory")
	}
	cwd, e := os.Getwd()
	if e != nil {
		return e
	}
	rel, e := filepath.Rel(root, cwd)
	if e == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return fmt.Errorf("data must not be the workspace or one of its parent directories")
	}
	if e = os.MkdirAll(parent, 0755); e != nil {
		return e
	}
	lock := filepath.Join(parent, "."+base+".ucsync.lock")
	f, e := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return fmt.Errorf("snapshot is locked (%s); close other sync/build processes before removing a stale lock: %w", lock, e)
	}
	f.Close()
	defer os.Remove(lock)
	backup := filepath.Join(parent, "."+base+".previous")
	// Recover an interrupted promotion before doing any new work.
	if exists(backup) {
		if !exists(root) {
			if e = os.Rename(backup, root); e != nil {
				return e
			}
		} else {
			if _, e = snapshot.Load(os.DirFS(root), true); e != nil {
				return fmt.Errorf("current snapshot invalid and recovery copy exists at %s; refusing overwrite", backup)
			}
			if e = removeWithin(parent, backup); e != nil {
				return e
			}
		}
	}
	if e = os.MkdirAll(root, 0755); e != nil {
		return e
	}
	stage := filepath.Join(root, ".staging")
	if e = removeWithin(root, stage); e != nil {
		return e
	}
	if e = os.MkdirAll(stage, 0755); e != nil {
		return e
	}
	defer func() {
		if err != nil {
			_ = removeWithin(root, stage)
		}
	}()
	get := func(path string) ([]byte, error) {
		if o.Offline {
			return nil, fmt.Errorf("offline import is missing %s; supply local translations/assets", path)
		}
		return fetch(source + path)
	}
	var raw []byte
	if o.Import != "" {
		raw, e = os.ReadFile(o.Import)
	} else {
		raw, e = get("/AllCards")
	}
	if e != nil {
		return e
	}
	if e = write(stage, "raw/AllCards.json", raw); e != nil {
		return e
	}
	cardsRaw, tr, e := snapshot.RawCards(raw)
	if e != nil {
		return fmt.Errorf("AllCards: %w", e)
	}
	if o.Translations != "" || snapshot.NeedsTranslations(cardsRaw) && len(tr) == 0 {
		var b []byte
		if o.Translations != "" {
			b, e = os.ReadFile(o.Translations)
		} else {
			b, e = get("/translation/en.json")
		}
		if e != nil {
			return e
		}
		if e = json.Unmarshal(b, &tr); e != nil {
			return fmt.Errorf("English translations: %w", e)
		}
		if e = write(stage, "raw/en.json", b); e != nil {
			return e
		}
	}
	cards, e := snapshot.Normalize(cardsRaw, tr)
	if e != nil {
		return e
	}
	if o.ArtVariants != "" {
		if o.Assets == "" {
			return fmt.Errorf("--art-variants requires --assets with local skin images; remote skin paths are not inferred")
		}
		b, err := os.ReadFile(o.ArtVariants)
		if err != nil {
			return err
		}
		if err = snapshot.ApplyArtVariants(cards, b); err != nil {
			return err
		}
		if err = write(stage, "raw/art-variants.json", b); err != nil {
			return err
		}
	}
	if e = writeJSON(stage, "cards.json", cards); e != nil {
		return e
	}
	paths, e := snapshot.AssetPaths(cards)
	if e != nil {
		return e
	}
	names := []string{}
	for p := range paths {
		names = append(names, p)
	}
	sort.Strings(names)
	for _, name := range names {
		var b []byte
		if o.Assets != "" {
			b, e = os.ReadFile(filepath.Join(o.Assets, filepath.FromSlash(strings.TrimPrefix(name, "assets/"))))
			if e != nil {
				return fmt.Errorf("required local asset %s: %w", name, e)
			}
		} else {
			b, e = os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
			if e != nil || snapshot.CheckImage(b) != nil {
				parts := strings.Split(name, "/")
				b, e = get("/images/" + parts[1] + "/" + url.PathEscape(parts[2]))
			}
			if e != nil {
				return e
			}
		}
		if e = snapshot.CheckImage(b); e != nil {
			return fmt.Errorf("%s: %w", name, e)
		}
		if e = write(stage, name, b); e != nil {
			return e
		}
		fmt.Println("Asset ready:", name)
	}
	m := snapshot.Manifest{SchemaVersion: snapshot.SchemaVersion, DownloadedAt: time.Now().Format(time.RFC3339), CardCount: len(cards), Source: source + "/AllCards", Checksums: map[string]string{}, Components: map[string]string{"cards": "included", "tokens": "none in this dataset", "artifacts": "not included; separate source required", "souls": "restrictions only; separate rules/artwork source required"}}
	m.Demo = true
	for _, c := range cards {
		if !c.Demo {
			m.Demo = false
		}
		if c.Rarity == "TOKEN" || c.Rarity == "GENERATED" {
			m.Components["tokens"] = "included in cards.json"
		}
	}
	for name := range paths {
		if strings.HasPrefix(name, "assets/cards/") {
			m.ImageCount++
		} else {
			m.PowerImageCount++
		}
	}
	e = filepath.WalkDir(stage, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(stage, p)
		if e != nil {
			return e
		}
		m.Checksums[filepath.ToSlash(rel)] = snapshot.Hash(b)
		return nil
	})
	if e != nil {
		return e
	}
	if e = writeJSON(stage, "manifest.json", m); e != nil {
		return e
	}
	if _, e = snapshot.Load(os.DirFS(stage), true); e != nil {
		return fmt.Errorf("staging validation failed: %w", e)
	}
	// Promote a complete directory, not individual card/image files. Old snapshot
	// stays recoverable until the second rename succeeds.
	if e = os.Rename(root, backup); e != nil {
		return e
	}
	if e = os.Rename(filepath.Join(backup, ".staging"), root); e != nil {
		rollback := os.Rename(backup, root)
		return fmt.Errorf("install failed: %v; rollback: %v", e, rollback)
	}
	if e = removeWithin(parent, backup); e != nil {
		fmt.Fprintln(os.Stderr, "Snapshot installed; old backup cleanup pending:", e)
	}
	fmt.Printf("Snapshot installed: %d cards, %d card images, %d power images.\n", m.CardCount, m.ImageCount, m.PowerImageCount)
	return nil
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ucsync download|import FILE|browser-assets FILE|browser-capture|validate|renderer-download|renderer-import ZIP|renderer-preview [--data data] [--translations en.json] [--assets assets] [--offline] [--output path]")
	}
	command := args[0]
	args = args[1:]
	o := options{Data: "data"}
	if command == "import" || command == "browser-assets" || command == "renderer-import" {
		if len(args) == 0 {
			return fmt.Errorf("%s requires an input filename", command)
		}
		o.Import = args[0]
		args = args[1:]
	}
	flags := flag.NewFlagSet("ucsync "+command, flag.ContinueOnError)
	flags.StringVar(&o.Data, "data", "data", "snapshot directory")
	flags.StringVar(&o.Assets, "assets", "", "local assets directory with cards/ and powers/; enables offline mode")
	flags.StringVar(&o.Translations, "translations", "", "local English translation JSON")
	flags.StringVar(&o.ArtVariants, "art-variants", "", "local artwork variant catalog (requires --assets)")
	flags.BoolVar(&o.Offline, "offline", false, "forbid all network requests")
	defaultOutput := "JSON/download-assets.js"
	if command == "browser-capture" {
		defaultOutput = "JSON/update/capture-snapshot.js"
	}
	if command == "renderer-download" || command == "renderer-import" {
		defaultOutput = "Sources/CardEditor"
	}
	if command == "renderer-preview" {
		defaultOutput = "CardTest/RendererComparison.html"
	}
	if command == "renderer-bundle" {
		defaultOutput = "web/renderer-assets.json"
	}
	output := flags.String("output", defaultOutput, "browser helper filename or renderer source folder")
	rendererSource := flags.String("renderer-source", "Sources/CardEditor", "downloaded renderer source folder for comparison")
	if e := flags.Parse(args); e != nil {
		return e
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if o.Assets != "" {
		o.Offline = true
	}
	switch command {
	case "browser-capture":
		return browserCapture(*output)
	case "renderer-bundle":
		return rendererBundle(*rendererSource, *output)
	case "renderer-preview":
		return rendererPreview(o.Data, *rendererSource, *output)
	case "renderer-download", "renderer-import":
		if command == "renderer-download" && o.Offline {
			return fmt.Errorf("renderer-download requires network access; use renderer-import ZIP for offline acquisition")
		}
		return rendererResources(o.Import, *output)
	case "browser-assets":
		return browserAssets(o, *output)
	case "validate":
		d, e := snapshot.Load(os.DirFS(o.Data), true)
		if e != nil {
			return e
		}
		fmt.Printf("PASS: %d cards, %d card images, %d power images.\n", len(d.Cards), d.Manifest.ImageCount, d.Manifest.PowerImageCount)
		return nil
	case "download", "import":
		return syncSnapshot(o, download)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}
