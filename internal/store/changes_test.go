package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestChangesRoundTrip(t *testing.T) {
	s, e := Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	id, e := s.New("C:/ws")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.LastChange(id); !errors.Is(e, ErrNoChange) {
		t.Fatalf("empty log: %v", e)
	}
	if e = s.RecordChange(id, "a.txt", "digest", `{"action":"write"}`); e != nil {
		t.Fatal(e)
	}
	if e = s.RecordChange(id, "b.txt", "", `{"action":"delete"}`); e != nil {
		t.Fatal(e)
	}
	change, e := s.LastChange(id)
	if e != nil || change.Path != "b.txt" || change.Inverse != `{"action":"delete"}` || change.Expect != "" {
		t.Fatalf("%+v %v", change, e)
	}
	if e = s.MarkChangeUndone(id, change.ID); e != nil {
		t.Fatal(e)
	}
	change, e = s.LastChange(id)
	if e != nil || change.Path != "a.txt" || change.Expect != "digest" {
		t.Fatalf("undone change still listed: %+v %v", change, e)
	}
	if e = s.MarkChangeUndone(id, change.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.LastChange(id); !errors.Is(e, ErrNoChange) {
		t.Fatalf("log not drained: %v", e)
	}
}

func TestMigrationFromV2AddsChanges(t *testing.T) {
	dir := t.TempDir()
	db, e := sql.Open("sqlite", filepath.Join(dir, "sessions.db"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`CREATE TABLE sessions(id TEXT PRIMARY KEY,title TEXT NOT NULL,workspace TEXT NOT NULL,created TEXT NOT NULL);
CREATE TABLE messages(id INTEGER PRIMARY KEY AUTOINCREMENT,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,payload TEXT NOT NULL);
CREATE TABLE audit(id INTEGER PRIMARY KEY AUTOINCREMENT,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,created TEXT NOT NULL,event TEXT NOT NULL);
CREATE TABLE summaries(session TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,through_id INTEGER NOT NULL,content TEXT NOT NULL);
CREATE TABLE usage(id INTEGER PRIMARY KEY,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,provider TEXT,model TEXT,input_tokens INTEGER,output_tokens INTEGER,cost REAL);
PRAGMA user_version=2;`); e != nil {
		t.Fatal(e)
	}
	db.Close()
	s, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	id, e := s.New("ws")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.RecordChange(id, "x", "", "{}"); e != nil {
		t.Fatalf("changes table missing after upgrade: %v", e)
	}
	if _, e = s.LastChange(id); e != nil {
		t.Fatalf("change not readable after upgrade: %v", e)
	}
}
