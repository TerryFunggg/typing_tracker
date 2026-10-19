// Package store saves heartbeats in a local SQLite file.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Heartbeat is one editor activity point.
type Heartbeat struct {
	At       time.Time
	FilePath string
	Project  string
	Branch   string
	Language string
	Editor   string
	IsWrite  bool
}

// Store is the local database.
type Store struct {
	db   *sql.DB
	path string
}

// Open creates the database file and schema when needed.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	_ = os.Chmod(path, 0o600)
	return &Store{db: db, path: path}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Insert appends one heartbeat.
func (s *Store) Insert(h Heartbeat) error {
	write := 0
	if h.IsWrite {
		write = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO heartbeats (at_ms, file_path, project, branch, language, editor, is_write)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		h.At.UTC().UnixMilli(), h.FilePath, h.Project, h.Branch, h.Language, h.Editor, write,
	)
	if err != nil {
		return fmt.Errorf("insert heartbeat: %w", err)
	}
	return nil
}

// Between returns heartbeats in [start, end), oldest first.
func (s *Store) Between(start, end time.Time) ([]Heartbeat, error) {
	rows, err := s.db.Query(
		`SELECT at_ms, file_path, project, branch, language, editor, is_write
		 FROM heartbeats
		 WHERE at_ms >= ? AND at_ms < ?
		 ORDER BY at_ms ASC, id ASC`,
		start.UTC().UnixMilli(), end.UTC().UnixMilli(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Heartbeat
	for rows.Next() {
		var ms int64
		var write int
		var h Heartbeat
		if err := rows.Scan(&ms, &h.FilePath, &h.Project, &h.Branch, &h.Language, &h.Editor, &write); err != nil {
			return nil, err
		}
		h.At = time.UnixMilli(ms).UTC()
		h.IsWrite = write != 0
		out = append(out, h)
	}
	return out, rows.Err()
}

// Latest returns the newest heartbeat. ok is false when the table is empty.
func (s *Store) Latest() (Heartbeat, bool, error) {
	var ms int64
	var write int
	var h Heartbeat
	err := s.db.QueryRow(
		`SELECT at_ms, file_path, project, branch, language, editor, is_write
		 FROM heartbeats ORDER BY at_ms DESC, id DESC LIMIT 1`,
	).Scan(&ms, &h.FilePath, &h.Project, &h.Branch, &h.Language, &h.Editor, &write)
	if err == sql.ErrNoRows {
		return Heartbeat{}, false, nil
	}
	if err != nil {
		return Heartbeat{}, false, err
	}
	h.At = time.UnixMilli(ms).UTC()
	h.IsWrite = write != 0
	return h, true, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS heartbeats (
  id INTEGER PRIMARY KEY,
  at_ms INTEGER NOT NULL,
  file_path TEXT NOT NULL,
  project TEXT NOT NULL,
  branch TEXT NOT NULL,
  language TEXT NOT NULL,
  editor TEXT NOT NULL,
  is_write INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS heartbeats_at ON heartbeats(at_ms);
`
