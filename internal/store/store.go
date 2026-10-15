package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/provider"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }
type Session struct{ ID, Title, Workspace, Created string }
type Entry struct {
	ID      int64
	Message provider.Message
}

func Open(dir string) (*Store, error) {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	path := filepath.Join(dir, "sessions.db")
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	f.Close()
	if e = os.Chmod(path, 0600); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	if _, e = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;`); e != nil {
		return fail(e)
	}
	var version int
	if e = db.QueryRow(`PRAGMA user_version`).Scan(&version); e != nil {
		return fail(e)
	}
	if version > 2 {
		return fail(errors.New("session database is newer than this application"))
	}
	if version == 0 {
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		_, e = tx.Exec(`CREATE TABLE IF NOT EXISTS sessions(id TEXT PRIMARY KEY,title TEXT NOT NULL,workspace TEXT NOT NULL,created TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS messages(id INTEGER PRIMARY KEY AUTOINCREMENT,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,payload TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS audit(id INTEGER PRIMARY KEY AUTOINCREMENT,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,created TEXT NOT NULL,event TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS messages_session ON messages(session,id);
PRAGMA user_version=1;`)
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}
	if version < 2 {
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		_, e = tx.Exec(`CREATE TABLE IF NOT EXISTS summaries(session TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,through_id INTEGER NOT NULL,content TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS usage(id INTEGER PRIMARY KEY,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,provider TEXT,model TEXT,input_tokens INTEGER,output_tokens INTEGER,cost REAL);
 PRAGMA user_version=2;`)
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}
	return &Store{db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) New(workspace string) (string, error) {
	var b [8]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	id := hex.EncodeToString(b[:])
	_, e := s.db.Exec(`INSERT INTO sessions VALUES(?,?,?,?)`, id, "Untitled", workspace, time.Now().UTC().Format(time.RFC3339))
	return id, e
}
func (s *Store) List() ([]Session, error) {
	rows, e := s.db.Query(`SELECT id,title,workspace,created FROM sessions ORDER BY created DESC,id DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var result []Session
	for rows.Next() {
		var v Session
		if e = rows.Scan(&v.ID, &v.Title, &v.Workspace, &v.Created); e != nil {
			return nil, e
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (s *Store) Get(id string) (Session, error) {
	var v Session
	e := s.db.QueryRow(`SELECT id,title,workspace,created FROM sessions WHERE id=?`, id).Scan(&v.ID, &v.Title, &v.Workspace, &v.Created)
	return v, e
}
func (s *Store) Append(id string, m provider.Message) error {
	b, e := json.Marshal(m)
	if e != nil {
		return e
	}
	_, e = s.db.Exec(`INSERT INTO messages(session,payload) VALUES(?,?)`, id, string(b))
	return e
}
func (s *Store) History(id string) ([]Entry, error) {
	rows, e := s.db.Query(`SELECT id,payload FROM messages WHERE session=? ORDER BY id`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var v Entry
		var b string
		if e = rows.Scan(&v.ID, &b); e != nil {
			return nil, e
		}
		if e = json.Unmarshal([]byte(b), &v.Message); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) Audit(id, event string) error {
	_, e := s.db.Exec(`INSERT INTO audit(session,created,event) VALUES(?,?,?)`, id, time.Now().UTC().Format(time.RFC3339Nano), event)
	return e
}
func (s *Store) Rename(id, title string) error {
	_, e := s.db.Exec(`UPDATE sessions SET title=? WHERE id=?`, title, id)
	return e
}
func (s *Store) Delete(id string) error {
	_, e := s.db.Exec(`DELETE FROM sessions WHERE id=?`, id)
	return e
}

// Recover closes interrupted tool exchanges without replaying operations.
func (s *Store) Recover(id string) error {
	entries, e := s.History(id)
	if e != nil {
		return e
	}
	pending := map[string]bool{}
	for _, v := range entries {
		for _, c := range v.Message.Calls {
			pending[c.ID] = true
		}
		if v.Message.Role == "tool" {
			delete(pending, v.Message.CallID)
		}
	}
	for call := range pending {
		if e = s.Append(id, provider.Message{Role: "tool", CallID: call, Content: "Interrupted before result was durably recorded. Inspect workspace before retrying; operation may have occurred."}); e != nil {
			return e
		}
	}
	return nil
}
