package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"typing_tracker/internal/store"
)

func TestHeartbeat(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "tracker.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	api := &Server{
		Token:  "secret-token",
		Store:  db,
		Ignore: []string{"/tmp/notes"},
		Lookup: func(path string) (string, string) {
			return "/repos/app", "main"
		},
		Now: func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(mustJSON(t, now)))
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("auth status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(mustJSON(t, now)))
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("store status %d body %s", rec.Code, rec.Body.String())
	}
	latest, ok, err := db.Latest()
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	if latest.Project != "/repos/app" || latest.Branch != "main" || latest.Language != "go" || latest.Editor != "vim" {
		t.Fatalf("stored %#v", latest)
	}

	ignored := []byte(`{"entity":"/tmp/notes/a.md","editor":"emacs","language":"markdown","is_write":true,"time":0}`)
	req = httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(ignored))
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("ignore status %d", rec.Code)
	}
	rows, err := db.Between(now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil || len(rows) != 1 {
		t.Fatalf("ignored path was stored: %#v %v", rows, err)
	}

	rel := []byte(`{"entity":"main.go","editor":"vim","language":"go"}`)
	req = httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(rel))
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("relative status %d", rec.Code)
	}

	health := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec = httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, health)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"ok":true}` {
		t.Fatalf("health %d %s", rec.Code, rec.Body.String())
	}
}

func mustJSON(t *testing.T, now time.Time) []byte {
	t.Helper()
	return []byte(`{"entity":"/repos/app/main.go","editor":"vim","language":"go","is_write":false,"time":` +
		strconvFormat(now.Unix()) + `}`)
}

func strconvFormat(n int64) string {
	return strconvItoa(n)
}

func strconvItoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func TestValidateBind(t *testing.T) {
	if err := ValidateBind("127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBind("::1"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"0.0.0.0", "192.168.1.10", "localhost", ""} {
		if err := ValidateBind(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}
