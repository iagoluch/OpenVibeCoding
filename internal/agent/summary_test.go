package agent

import (
	"context"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
	"github.com/iagoluch/OpenVibeCoding/internal/store"
	"strings"
	"testing"
)

type summarizer struct {
	calls   int
	invalid bool
}

func (s *summarizer) Stream(_ context.Context, m []provider.Message, _ []provider.Tool, _ func(string)) (provider.Reply, error) {
	s.calls++
	text := `{"requirements":["preserve originals"],"pending":["verify"],"approval_history":[],"source_ids":[1]}`
	if s.invalid {
		text = "not JSON"
	}
	return provider.Reply{Message: provider.Message{Role: "assistant", Content: text}}, nil
}
func TestIncrementalSummaryPersistsOriginals(t *testing.T) {
	db, e := store.Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	id, _ := db.New("project")
	for i := 0; i < 40; i++ {
		db.Append(id, provider.Message{Role: "user", Content: strings.Repeat("x", 200)})
		db.Append(id, provider.Message{Role: "assistant", Content: "Observed test passed"})
	}
	a := Engine{Store: db, Config: config.Default(), Session: id}
	entries, _ := db.History(id)
	client := &summarizer{}
	messages, omitted, e := a.prepare(context.Background(), client, entries, 8192, func(Event) {})
	if e != nil || omitted == 0 || len(messages) == 0 {
		t.Fatal(omitted, e)
	}
	summary, _ := db.Summary(id)
	if summary.Through == 0 || !strings.Contains(summary.Content, "preserve originals") {
		t.Fatal("summary missing")
	}
	calls := client.calls
	if _, _, e = a.prepare(context.Background(), client, entries, 8192, func(Event) {}); e != nil || calls != client.calls {
		t.Fatal("repeated summary", e)
	}
	after, _ := db.History(id)
	if len(after) != len(entries) {
		t.Fatal("originals removed")
	}
}
