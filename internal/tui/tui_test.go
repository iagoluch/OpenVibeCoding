package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/iagoluch/OpenVibeCoding/internal/agent"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
	"github.com/iagoluch/OpenVibeCoding/internal/store"
	"github.com/iagoluch/OpenVibeCoding/internal/workspace"
)

func fixture(t *testing.T) *Model {
	t.Helper()
	db, e := store.Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	w, e := workspace.Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { w.Close() })
	id, e := db.New(w.Path)
	if e != nil {
		t.Fatal(e)
	}
	return New(&agent.Engine{Config: config.Default(), Store: db, Workspace: w, Session: id})
}
func TestSwitchAndClear(t *testing.T) {
	m := fixture(t)
	m.command("/model missing/nope")
	if m.Engine.Config.Provider != "openai" {
		t.Fatal("invalid switch applied")
	}
	m.command("/model local/qwen3:8b")
	if m.Engine.Config.Provider != "local" {
		t.Fatal("switch failed")
	}
	m.command("/harness inspect")
	if m.Engine.Config.Harness != "inspect" {
		t.Fatal("harness failed")
	}
	if e := m.Engine.Store.Append(m.Engine.Session, provider.Message{Role: "user", Content: "saved"}); e != nil {
		t.Fatal(e)
	}
	m.command("/clear")
	entries, e := m.Engine.Store.History(m.Engine.Session)
	if e != nil || len(entries) != 1 {
		t.Fatal("clear deleted history")
	}
	m.Update(tea.WindowSizeMsg{Width: 30, Height: 12})
	if !strings.Contains(m.View().Content, "OPENVIBECODING") {
		t.Fatal("missing view")
	}
}

type cancellationClient struct{}

func (cancellationClient) Stream(ctx context.Context, _ []provider.Message, _ []provider.Tool, _ func(string)) (provider.Reply, error) {
	<-ctx.Done()
	return provider.Reply{}, ctx.Err()
}
type echoClient struct{}

func (echoClient) Stream(_ context.Context, _ []provider.Message, _ []provider.Tool, _ func(string)) (provider.Reply, error) {
	return provider.Reply{Message: provider.Message{Role: "assistant", Content: "ok"}}, nil
}
func TestInitSubmitsPrompt(t *testing.T) {
	m := fixture(t)
	m.Engine.Client = echoClient{}
	if cmd := m.command("/init"); cmd == nil || !m.busy {
		t.Fatal("init did not start a turn")
	}
	found := false
	for _, l := range m.lines {
		if strings.Contains(l, "AGENTS.md") {
			found = true
		}
	}
	if !found {
		t.Fatal("init prompt missing")
	}
	m.Shutdown()
}
func TestDoctorCommand(t *testing.T) {
	m := fixture(t)
	m.Version = "0.2.0-test"
	m.command("/doctor")
	joined := strings.Join(m.lines, "\n")
	for _, want := range []string{"0.2.0-test", "OS sandbox", "Runtime", "Workspace"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("doctor output missing %q", want)
		}
	}
}
func TestShutdownCancelsAndDrains(t *testing.T) {
	m := fixture(t)
	m.Engine.Client = cancellationClient{}
	m.input.SetValue("inspect")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	closed := make(chan struct{})
	go func() { m.Shutdown(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not cancel worker")
	}
}


func TestUndoRequiresRuntimeCapability(t *testing.T) {
	m := fixture(t)
	m.command("/undo")
	if !strings.Contains(strings.Join(m.lines, "\n"), "shared runtime") {
		t.Fatal("missing capability message")
	}
}

