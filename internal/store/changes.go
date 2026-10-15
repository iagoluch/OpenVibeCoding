package store

import (
	"database/sql"
	"errors"
	"time"
)

// ErrNoChange reports that a session has nothing left to undo.
var ErrNoChange = errors.New("nothing to undo in this session")

// AppliedChange is one workspace change recorded after approval so /undo can
// reverse it later. Expect is the digest of the file state right after the
// change ("" when the file was removed afterwards) and Inverse is a JSON
// workspace.Inverse payload.
type AppliedChange struct {
	ID      int64
	Path    string
	Created string
	Expect  string
	Inverse string
}

// ChangeRecorder is implemented by stores that persist applied workspace
// changes for /undo. The runtime's SQLite store implements it; remote clients
// do not, because the runtime owns the mutation log.
type ChangeRecorder interface {
	RecordChange(session, path, expect, inverse string) error
}

func (s *Store) RecordChange(session, path, expect, inverse string) error {
	_, e := s.db.Exec(`INSERT INTO changes(session,created,path,expect,inverse) VALUES(?,?,?,?,?)`,
		session, time.Now().UTC().Format(time.RFC3339), path, expect, inverse)
	return e
}

// LastChange returns the most recent change of the session that was not undone
// yet, or ErrNoChange.
func (s *Store) LastChange(session string) (AppliedChange, error) {
	var c AppliedChange
	e := s.db.QueryRow(`SELECT id,created,path,expect,inverse FROM changes WHERE session=? AND undone=0 ORDER BY id DESC LIMIT 1`, session).
		Scan(&c.ID, &c.Created, &c.Path, &c.Expect, &c.Inverse)
	if errors.Is(e, sql.ErrNoRows) {
		return c, ErrNoChange
	}
	return c, e
}

func (s *Store) MarkChangeUndone(session string, id int64) error {
	_, e := s.db.Exec(`UPDATE changes SET undone=1 WHERE id=? AND session=?`, id, session)
	return e
}
