package store

import (
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
	"testing"
)

func TestPersistenceRecoveryAndDeletion(t *testing.T) {
	dir := t.TempDir()
	s, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	id, e := s.New("/project")
	if e != nil {
		t.Fatal(e)
	}
	m := provider.Message{Role: "assistant", Calls: []provider.Call{{ID: "interrupted", Type: "function", Function: provider.Function{Name: "write_file"}}}}
	if e = s.Append(id, m); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Recover(id); e != nil {
		t.Fatal(e)
	}
	if e = s.Recover(id); e != nil {
		t.Fatal(e)
	}
	h, e := s.History(id)
	if e != nil || len(h) != 2 || h[1].Message.CallID != "interrupted" {
		t.Fatalf("%+v %v", h, e)
	}
	if e = s.Rename(id, "Task"); e != nil {
		t.Fatal(e)
	}
	v, _ := s.Get(id)
	if v.Title != "Task" {
		t.Fatal(v)
	}
	if e = s.Delete(id); e != nil {
		t.Fatal(e)
	}
	h, e = s.History(id)
	if e != nil || len(h) != 0 {
		t.Fatal("cascade failed")
	}
}
