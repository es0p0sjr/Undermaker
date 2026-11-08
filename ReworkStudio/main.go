// Rework Studio is a separate companion. Official data is only read.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"ucprint/internal/app"
	"ucprint/internal/snapshot"
)

type Rework struct {
	Card      snapshot.Card `json:"card"`
	Reference string        `json:"reference"`
	Reviewed  bool          `json:"reviewed"`
}
type Studio struct {
	root    string
	data    *snapshot.Dataset
	mu      sync.Mutex
	reworks []Rework
	base    http.Handler
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (s *Studio) validate(r Rework) error {
	var original *snapshot.Card
	for i := range s.data.Cards {
		if s.data.Cards[i].ID == r.Card.ID {
			original = &s.data.Cards[i]
			break
		}
	}
	if original == nil {
		return fmt.Errorf("unknown original card ID %d", r.Card.ID)
	}
	c := r.Card
	if r.Reference != original.Name+".png" {
		return errors.New("reference must match the original card name")
	}
	if c.Image != original.Image || c.Type != original.Type {
		return errors.New("artwork and card type must match the original")
	}
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 100 || len(c.Description) > 4000 {
		return errors.New("name or rules length is invalid")
	}
	for _, value := range []*int{c.Cost, c.Attack, c.Health} {
		if value == nil || *value < 0 || *value > 999 {
			return errors.New("stats must be whole numbers from 0 to 999")
		}
	}
	switch c.Rarity {
	case "COMMON", "RARE", "EPIC", "LEGENDARY", "DETERMINATION", "TOKEN", "GENERATED":
	default:
		return errors.New("unknown rarity")
	}
	switch c.Extension {
	case "BASE", "DELTARUNE", "DR", "UTY":
	default:
		return errors.New("unknown expansion")
	}
	if len(c.Statuses) > 9 {
		return errors.New("at most nine status icons fit on a card")
	}
	for _, status := range c.Statuses {
		if _, ok := s.data.Files["assets/powers/"+status.Image]; !ok {
			return fmt.Errorf("unknown status image %s", status.Image)
		}
		if status.Counter != nil && (*status.Counter < 0 || *status.Counter > 999) {
			return errors.New("status counter must be 0 to 999")
		}
	}
	return nil
}
func newStudio(root string) (*Studio, error) {
	d, err := snapshot.Load(os.DirFS(filepath.Join(root, "data")), false)
	if err != nil {
		return nil, err
	}
	s := &Studio{root: root, data: d, base: app.New(d, os.DirFS(filepath.Join(root, "web")))}
	b, err := os.ReadFile(filepath.Join(root, "ReworkedCards", "reworks.json"))
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &s.reworks); err != nil {
		return nil, err
	}
	seen := map[int]bool{}
	for _, r := range s.reworks {
		if err = s.validate(r); err != nil {
			return nil, fmt.Errorf("%s: %w", r.Card.Name, err)
		}
		if seen[r.Card.ID] {
			return nil, fmt.Errorf("duplicate rework %d", r.Card.ID)
		}
		seen[r.Card.ID] = true
	}
	return s, nil
}
func (s *Studio) save(r Rework) error {
	if err := s.validate(r); err != nil {
		return err
	}
	next := append([]Rework(nil), s.reworks...)
	found := false
	for i := range next {
		if next[i].Card.ID == r.Card.ID {
			next[i] = r
			found = true
			break
		}
	}
	if !found {
		return errors.New("card is not in the rework collection")
	}
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Join(s.root, "ReworkedCards"), ".reworks-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), filepath.Join(s.root, "ReworkedCards", "reworks.json")); err != nil {
		return err
	}
	s.reworks = next
	return nil
}
func (s *Studio) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data: blob:; font-src 'self' data:; script-src 'self'; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil || (host != "127.0.0.1" && host != "localhost") {
		respond(w, 403, map[string]string{"error": "local connections only"})
		return
	}
	if r.URL.Path == "/api/reworks" {
		s.mu.Lock()
		defer s.mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			respond(w, 200, s.reworks)
		case http.MethodPost:
			if r.Header.Get("X-UCPrint") != "1" || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+r.Host) {
				respond(w, 403, map[string]string{"error": "use the local studio"})
				return
			}
			var rework Rework
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
			err := decoder.Decode(&rework)
			if err == nil {
				var extra any
				if decoder.Decode(&extra) != io.EOF {
					err = errors.New("expected one JSON object")
				}
			}
			if err == nil {
				err = s.save(rework)
			}
			if err != nil {
				respond(w, 400, map[string]string{"error": err.Error()})
				return
			}
			respond(w, 200, rework)
		default:
			respond(w, 405, map[string]string{"error": "method not allowed"})
		}
		return
	}
	if strings.HasPrefix(r.URL.Path, "/reference/") {
		if r.Method != http.MethodGet {
			w.WriteHeader(405)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/reference/")
		s.mu.Lock()
		known := false
		for _, v := range s.reworks {
			if v.Reference == name {
				known = true
				break
			}
		}
		s.mu.Unlock()
		if !known {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(s.root, "ReworkedCards", name))
		return
	}
	if r.URL.Path == "/" || r.URL.Path == "/studio.js" || r.URL.Path == "/studio.css" {
		if r.Method != http.MethodGet {
			w.WriteHeader(405)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		http.ServeFile(w, r, filepath.Join(s.root, "ReworkStudio", "web", name))
		return
	}
	s.base.ServeHTTP(w, r)
}
func run() error {
	root := flag.String("root", ".", "project directory")
	port := flag.Int("port", 0, "local port; 0 selects an available port")
	open := flag.Bool("open", false, "open browser")
	flag.Parse()
	abs, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	s, err := newStudio(abs)
	if err != nil {
		return err
	}
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return err
	}
	url := "http://" + l.Addr().String()
	fmt.Println("Rework Studio:", url)
	if *open {
		go func() {
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "windows":
				cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
			case "darwin":
				cmd = exec.Command("open", url)
			default:
				cmd = exec.Command("xdg-open", url)
			}
			_ = cmd.Run()
		}()
	}
	return (&http.Server{Handler: s, ReadHeaderTimeout: 5 * time.Second}).Serve(l)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Rework Studio:", err)
		os.Exit(1)
	}
}
