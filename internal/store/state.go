package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
)

type Summary struct {
	Through int64
	Content string
}
type Usage struct {
	Provider, Model string
	Input, Output   int
	Cost            *float64
}
type Repository interface {
	New(string) (string, error)
	List() ([]Session, error)
	Get(string) (Session, error)
	Append(string, provider.Message) error
	History(string) ([]Entry, error)
	Audit(string, string) error
	Rename(string, string) error
	Delete(string) error
	Recover(string) error
	Summary(string) (Summary, error)
	SaveSummary(string, Summary) error
	AddUsage(string, Usage) error
	Usage(string) ([]Usage, error)
	Export(string) ([]byte, error)
}

func (s *Store) Summary(id string) (Summary, error) {
	var v Summary
	e := s.db.QueryRow(`SELECT through_id,content FROM summaries WHERE session=?`, id).Scan(&v.Through, &v.Content)
	if errors.Is(e, sql.ErrNoRows) {
		e = nil
	}
	return v, e
}
func (s *Store) SaveSummary(id string, v Summary) error {
	_, e := s.db.Exec(`INSERT INTO summaries VALUES(?,?,?) ON CONFLICT(session) DO UPDATE SET through_id=excluded.through_id,content=excluded.content`, id, v.Through, v.Content)
	return e
}
func (s *Store) AddUsage(id string, v Usage) error {
	_, e := s.db.Exec(`INSERT INTO usage(session,provider,model,input_tokens,output_tokens,cost) VALUES(?,?,?,?,?,?)`, id, v.Provider, v.Model, v.Input, v.Output, v.Cost)
	return e
}
func (s *Store) Usage(id string) ([]Usage, error) {
	rows, e := s.db.Query(`SELECT provider,model,input_tokens,output_tokens,cost FROM usage WHERE session=? ORDER BY id`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []Usage
	for rows.Next() {
		var v Usage
		if e = rows.Scan(&v.Provider, &v.Model, &v.Input, &v.Output, &v.Cost); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) Export(id string) ([]byte, error) {
	session, e := s.Get(id)
	if e != nil {
		return nil, e
	}
	messages, e := s.History(id)
	if e != nil {
		return nil, e
	}
	summary, e := s.Summary(id)
	if e != nil {
		return nil, e
	}
	usage, e := s.Usage(id)
	if e != nil {
		return nil, e
	}
	rows, e := s.db.Query(`SELECT created,event FROM audit WHERE session=? ORDER BY id`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var audit []map[string]string
	for rows.Next() {
		var at, event string
		if e = rows.Scan(&at, &event); e != nil {
			return nil, e
		}
		audit = append(audit, map[string]string{"at": at, "event": event})
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	return json.MarshalIndent(map[string]any{"schema": 1, "session": session, "messages": messages, "summary": summary, "usage": usage, "audit": audit}, "", "  ")
}
