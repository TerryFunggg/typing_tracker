// Package server accepts editor heartbeats on the loopback interface.
package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"typing_tracker/internal/config"
	"typing_tracker/internal/gitinfo"
	"typing_tracker/internal/language"
	"typing_tracker/internal/store"
)

const maxBody = 64 << 10

// Server handles heartbeat posts.
type Server struct {
	Token  string
	Store  *store.Store
	Ignore []string
	// Lookup finds the git root and branch. Tests replace it.
	Lookup func(path string) (root, branch string)
	// Now is the clock. Tests replace it.
	Now func() time.Time
}

type heartbeatRequest struct {
	Entity   string  `json:"entity"`
	Editor   string  `json:"editor"`
	Language string  `json:"language"`
	IsWrite  bool    `json:"is_write"`
	Time     float64 `json:"time"`
}

// Handler is the HTTP API. GET /health is open. POST /heartbeat needs the token.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("POST /heartbeat", s.heartbeat)
	return mux
}

func (s *Server) heartbeat(w http.ResponseWriter, r *http.Request) {
	if !authorized(r, s.Token) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body := http.MaxBytesReader(w, r.Body, maxBody)
	defer body.Close()
	var req heartbeatRequest
	dec := json.NewDecoder(body)
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	at, err := s.when(req.Time)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path, editor, err := cleanInput(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if config.Ignored(path, s.Ignore) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	lookup := s.Lookup
	if lookup == nil {
		lookup = gitinfo.Lookup
	}
	root, branch := lookup(path)
	h := store.Heartbeat{
		At:       at,
		FilePath: path,
		Project:  root,
		Branch:   branch,
		Language: language.Resolve(req.Language, path),
		Editor:   editor,
		IsWrite:  req.IsWrite,
	}
	if err := s.Store.Insert(h); err != nil {
		http.Error(w, "store failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) when(unix float64) (time.Time, error) {
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	if unix == 0 {
		return now, nil
	}
	sec := int64(unix)
	at := time.Unix(sec, 0)
	if at.Before(now.Add(-24*time.Hour)) || at.After(now.Add(10*time.Minute)) {
		return time.Time{}, errors.New("time is too far from now")
	}
	return at, nil
}

func cleanInput(req heartbeatRequest) (path, editor string, err error) {
	path = filepath.Clean(req.Entity)
	if path == "" || !filepath.IsAbs(path) || !utf8.ValidString(path) || strings.ContainsRune(path, 0) {
		return "", "", errors.New("entity must be an absolute file path")
	}
	if len(path) > 4096 {
		return "", "", errors.New("entity is too long")
	}
	editor = strings.TrimSpace(req.Editor)
	if editor == "" || len(editor) > 32 || !utf8.ValidString(editor) || strings.ContainsAny(editor, "\r\n") {
		return "", "", errors.New("editor is missing or invalid")
	}
	if len(req.Language) > 64 || !utf8.ValidString(req.Language) {
		return "", "", errors.New("language is invalid")
	}
	return path, editor, nil
}

func authorized(r *http.Request, token string) bool {
	got := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(got, prefix) {
		return false
	}
	got = got[len(prefix):]
	if len(got) != len(token) || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

// ValidateBind allows only loopback IP addresses.
func ValidateBind(bind string) error {
	ip := net.ParseIP(bind)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("bind must be a loopback address such as 127.0.0.1")
	}
	return nil
}
