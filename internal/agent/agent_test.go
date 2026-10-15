package agent

import (
	"context"
	"encoding/json"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
	"github.com/iagoluch/OpenVibeCoding/internal/store"
	"github.com/iagoluch/OpenVibeCoding/internal/workspace"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type scripted struct{ turn int }

func (s *scripted) Stream(_ context.Context, _ []provider.Message, _ []provider.Tool, emit func(string)) (provider.Reply, error) {
	s.turn++
	if s.turn == 1 {
		args, _ := json.Marshal(map[string]string{"path": "hello.txt", "content": "hello\n"})
		return provider.Reply{Message: provider.Message{Role: "assistant", Content: "Plan: create hello.txt", Calls: []provider.Call{{ID: "c1", Type: "function", Function: provider.Function{Name: "write_file", Arguments: string(args)}}}}}, nil
	}
	emit("done")
	return provider.Reply{Message: provider.Message{Role: "assistant", Content: "done"}}, nil
}
func TestAgentApprovalFlow(t *testing.T) {
	for _, approve := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject", true: "approve"}[approve], func(t *testing.T) {
			w, e := workspace.Open(t.TempDir(), false)
			if e != nil {
				t.Fatal(e)
			}
			defer w.Close()
			db, e := store.Open(t.TempDir())
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			id, _ := db.New(w.Path)
			a := Engine{Config: config.Default(), Store: db, Workspace: w, Session: id, Client: &scripted{}}
			reviews := 0
			e = a.Run(context.Background(), "create hello", func(ev Event) {
				if ev.Kind == "approval" {
					reviews++
					if !strings.Contains(ev.Text, "+hello") {
						t.Error("missing diff")
					}
					if _, err := os.Stat(filepath.Join(w.Path, "hello.txt")); !os.IsNotExist(err) {
						t.Error("wrote before approval")
					}
					ev.Decision <- approve
				}
			})
			if e != nil {
				t.Fatal(e)
			}
			_, e = os.Stat(filepath.Join(w.Path, "hello.txt"))
			if (e == nil) != approve || reviews != 1 {
				t.Fatalf("approval bypass: %v reviews %d", e, reviews)
			}
			h, _ := db.History(id)
			if len(h) != 4 {
				t.Fatalf("lost history: %d", len(h))
			}
		})
	}
}
type plainClient struct{}

func (plainClient) Stream(_ context.Context, _ []provider.Message, _ []provider.Tool, _ func(string)) (provider.Reply, error) {
	return provider.Reply{Message: provider.Message{Role: "assistant", Content: "ok"}}, nil
}

func TestProjectInstructionsAndAutoTitle(t *testing.T) {
	w, e := workspace.Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if e = os.WriteFile(filepath.Join(w.Path, "AGENTS.md"), []byte("Always run go test before proposing a change."), 0600); e != nil {
		t.Fatal(e)
	}
	db, e := store.Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	id, _ := db.New(w.Path)
	a := Engine{Config: config.Default(), Store: db, Workspace: w, Session: id, Client: plainClient{}}
	if p := a.projectInstructions(); !strings.Contains(p, "Always run go test") || !strings.Contains(p, "never authorize") {
		t.Fatalf("AGENTS.md not framed into the system message: %q", p)
	}
	if e = a.Run(context.Background(), "create a parser\nwith more detail", func(Event) {}); e != nil {
		t.Fatal(e)
	}
	session, e := db.Get(id)
	if e != nil {
		t.Fatal(e)
	}
	if session.Title != "create a parser" {
		t.Fatalf("auto title = %q", session.Title)
	}
	if e = db.Rename(id, "Mine"); e != nil {
		t.Fatal(e)
	}
	if e = a.Run(context.Background(), "another prompt", func(Event) {}); e != nil {
		t.Fatal(e)
	}
	if session, e = db.Get(id); e != nil {
		t.Fatal(e)
	}
	if session.Title != "Mine" {
		t.Fatalf("manual title overwritten: %q", session.Title)
	}
}

func TestProjectInstructionsTruncated(t *testing.T) {
	w, e := workspace.Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if e = os.WriteFile(filepath.Join(w.Path, "AGENTS.md"), []byte(strings.Repeat("a", 20<<10)), 0600); e != nil {
		t.Fatal(e)
	}
	a := Engine{Workspace: w}
	p := a.projectInstructions()
	if !strings.HasSuffix(p, "[truncated]") || len(p) > (16<<10)+512 {
		t.Fatalf("truncation failed: len %d", len(p))
	}
}

func TestContextKeepsToolExchange(t *testing.T) {
	var entries []store.Entry
	for i := 0; i < 20; i++ {
		entries = append(entries, store.Entry{ID: int64(i*3 + 1), Message: provider.Message{Role: "user", Content: strings.Repeat("x", 400)}}, store.Entry{ID: int64(i*3 + 2), Message: provider.Message{Role: "assistant", Calls: []provider.Call{{ID: "call"}}}}, store.Entry{ID: int64(i*3 + 3), Message: provider.Message{Role: "tool", CallID: "call", Content: "result"}})
	}
	m, n := Context(entries, 8192)
	if n == 0 || len(m) == 0 || m[0].Role != "system" || m[1].Role != "user" || m[len(m)-1].Role != "tool" {
		t.Fatalf("context boundary invalid: %d %+v", n, m)
	}
	if len(entries) != 60 {
		t.Fatal("history modified")
	}
}
