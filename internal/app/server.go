// Package app is an offline HTTP server. It has no HTTP client or download path.
package app

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"ucprint/internal/printing"
	"ucprint/internal/snapshot"
)

type Server struct {
	Data         *snapshot.Dataset
	Web          fs.FS
	Reworks      fs.FS
	reworkImages map[int][]byte
	mu           sync.Mutex
	pdfs         map[string][]byte
	serial       int
}

func New(data *snapshot.Dataset, web fs.FS) *Server {
	return &Server{Data: data, Web: web, pdfs: map[string][]byte{}}
}
func jsonOut(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, e error) {
	jsonOut(w, status, map[string]string{"error": e.Error()})
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data: blob:; font-src 'self' data:; script-src 'self'; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	host := strings.Split(r.Host, ":")[0]
	if host != "127.0.0.1" && host != "localhost" {
		fail(w, 403, fmt.Errorf("local connections only"))
		return
	}
	if r.Method == http.MethodPost {
		if r.Header.Get("X-UCPrint") != "1" || r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+r.Host {
			fail(w, 403, fmt.Errorf("use the local app"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 40<<20)
		switch r.URL.Path {
		case "/api/deck":
			var req struct {
				Code string `json:"code"`
			}
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				fail(w, 400, e)
				return
			}
			d, e := DecodeDeck(req.Code, s.Data.Cards)
			if e != nil {
				fail(w, 400, e)
				return
			}
			jsonOut(w, 200, d)
		case "/api/export":
			var req struct {
				Cards    []printing.Entry `json:"cards"`
				CutMarks *bool            `json:"cutMarks"`
			}
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				fail(w, 400, e)
				return
			}
			known := map[int]bool{}
			for _, c := range s.Data.Cards {
				known[c.ID] = true
			}
			s.mu.Lock()
			for id := range s.reworkImages {
				known[id] = true
			}
			s.mu.Unlock()
			count := 0
			for _, c := range req.Cards {
				if !known[c.ID] {
					fail(w, 400, fmt.Errorf("unknown card %d", c.ID))
					return
				}
				count += c.Quantity
			}
			cuts := req.CutMarks == nil || *req.CutMarks
			b, e := printing.Export(req.Cards, cuts)
			if e != nil {
				fail(w, 400, e)
				return
			}
			s.mu.Lock()
			s.serial++
			name := fmt.Sprintf("uc-print-%d.pdf", s.serial)
			if len(s.pdfs) >= 5 {
				oldest := fmt.Sprintf("uc-print-%d.pdf", s.serial-5)
				delete(s.pdfs, oldest)
			}
			s.pdfs[name] = b
			s.mu.Unlock()
			jsonOut(w, 200, map[string]any{"url": "/output/" + name, "filename": name, "cards": count, "pages": (count + 8) / 9})
		default:
			fail(w, 404, fmt.Errorf("no runtime sync/import endpoint; use tools/ucsync explicitly"))
		}
		return
	}
	if r.Method != http.MethodGet {
		fail(w, 405, fmt.Errorf("method not allowed"))
		return
	}
	send := func(b []byte, kind string) { w.Header().Set("Content-Type", kind); w.Write(b) }
	switch {
	case r.URL.Path == "/api/reworked-cards":
		cards, err := s.loadReworks()
		if err != nil {
			fail(w, 400, err)
			return
		}
		jsonOut(w, 200, map[string]any{"cards": cards})
	case strings.HasPrefix(r.URL.Path, "/api/reworked-image/"):
		id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/reworked-image/"))
		s.mu.Lock()
		b, ok := s.reworkImages[id]
		s.mu.Unlock()
		if err != nil || !ok {
			fail(w, 404, fmt.Errorf("reworked image is not imported"))
			return
		}
		send(b, "image/png")
	case r.URL.Path == "/api/cards":
		jsonOut(w, 200, map[string]any{"cards": s.Data.Cards, "metadata": map[string]any{"source": "snapshot", "updated": s.Data.Manifest.DownloadedAt}, "manifest": s.Data.Manifest, "coverage": map[string]int{"rulesReady": len(s.Data.Cards), "artworkCached": len(s.Data.Cards)}, "message": "Validated offline snapshot. No internet connection is used."})
	case r.URL.Path == "/api/calibration":
		w.Header().Set("Content-Disposition", "attachment; filename=uc-print-calibration.pdf")
		send(printing.Calibration(), "application/pdf")
	case strings.HasPrefix(r.URL.Path, "/api/art/"):
		id, e := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/art/"))
		if e != nil {
			fail(w, 404, e)
			return
		}
		for _, c := range s.Data.Cards {
			if c.ID == id {
				artImage := c.Image
				if variantID := r.URL.Query().Get("variant"); variantID != "" {
					found := false
					for _, variant := range c.ArtVariants {
						if variant.ID == variantID {
							artImage = variant.Image
							found = true
							break
						}
					}
					if !found {
						fail(w, 404, fmt.Errorf("unknown artwork variant"))
						return
					}
				}
				b, ok := s.Data.Files["assets/cards/"+artImage]
				if !ok {
					fail(w, 500, fmt.Errorf("missing local artwork: %s; rebuild snapshot", c.Image))
					return
				}
				send(b, http.DetectContentType(b))
				return
			}
		}
		fail(w, 404, fmt.Errorf("unknown card"))
	case strings.HasPrefix(r.URL.Path, "/api/power/"):
		name := strings.TrimPrefix(r.URL.Path, "/api/power/")
		b, ok := s.Data.Files["assets/powers/"+name]
		if !ok {
			fail(w, 404, fmt.Errorf("missing local power image"))
			return
		}
		send(b, http.DetectContentType(b))
	case strings.HasPrefix(r.URL.Path, "/output/"):
		name := strings.TrimPrefix(r.URL.Path, "/output/")
		s.mu.Lock()
		b, ok := s.pdfs[name]
		s.mu.Unlock()
		if !ok {
			fail(w, 404, fmt.Errorf("export expired; export again"))
			return
		}
		send(b, "application/pdf")
	default:
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if !fs.ValidPath(name) {
			fail(w, 404, fmt.Errorf("file not found"))
			return
		}
		b, e := fs.ReadFile(s.Web, name)
		if e != nil {
			fail(w, 404, fmt.Errorf("file not found"))
			return
		}
		kind := mime.TypeByExtension(path.Ext(name))
		if strings.HasSuffix(name, ".js") {
			kind = "text/javascript"
		}
		send(b, kind)
	}
}
