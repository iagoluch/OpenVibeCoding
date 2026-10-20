package setup

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
)

func TestParseOpenCodeAuth(t *testing.T) {
	data := []byte(`{"google":{"type":"api","key":"g-key"},"nvidia":{"type":"api","key":"n-key"},"openai":{"type":"oauth","key":""}}`)
	m := parseOpenCodeAuth(data)
	if m["gemini"] != "g-key" || m["nvidia"] != "n-key" || len(m) != 2 {
		t.Fatalf("%v", m)
	}
	if parseOpenCodeAuth([]byte("not json")) != nil {
		t.Fatal("malformed accepted")
	}
	if len(parseOpenCodeAuth([]byte(`{"other":{"key":"x"}}`))) != 0 {
		t.Fatal("unknown providers must be ignored")
	}
}

func TestPickLocalModel(t *testing.T) {
	if got := pickLocalModel([]string{"a", "qwen3:8b"}, "qwen3:8b"); got != "qwen3:8b" {
		t.Fatal(got)
	}
	if got := pickLocalModel([]string{"a", "b"}, "qwen3:8b"); got != "a" {
		t.Fatal(got)
	}
	if got := pickLocalModel(nil, "qwen3:8b"); got != "qwen3:8b" {
		t.Fatal(got)
	}
}

func TestEnsureModelAddsMissing(t *testing.T) {
	p := config.Provider{Models: []config.Model{{ID: "qwen3:8b", Context: 40960}}}
	p = EnsureModel(p, "mistral:7b", 32768)
	if len(p.Models) != 2 || p.Models[1].ID != "mistral:7b" || p.Models[1].InputPrice == nil {
		t.Fatalf("%+v", p.Models)
	}
	if again := EnsureModel(p, "qwen3:8b", 32768); len(again.Models) != 2 {
		t.Fatal("duplicate model added")
	}
}

func TestSelectRefusesUnknownProviderAndModel(t *testing.T) {
	dir := t.TempDir()
	c := config.Default()
	if _, e := Select(dir, c, "missing", "gpt-4.1-mini"); e == nil {
		t.Fatal("unknown provider accepted")
	}
	if _, e := Select(dir, c, "openai", "gpt-4.1-turbo"); e == nil {
		t.Fatal("unknown model accepted")
	}
	updated, e := Select(dir, c, "openai", "gpt-4.1")
	if e != nil || updated.Provider != "openai" || updated.Model != "gpt-4.1" {
		t.Fatalf("select did not report the new configuration: %v", e)
	}
	if c.Provider != "openrouter" || c.Model != "thinkingmachines/inkling:free" {
		t.Fatal("select must not mutate the caller's configuration")
	}
	back, e := config.Load(dir)
	if e != nil || back.Provider != "openai" || back.Model != "gpt-4.1" {
		t.Fatalf("%s/%s %v", back.Provider, back.Model, e)
	}
}

func TestStoreRejectsEmptyCredential(t *testing.T) {
	c := config.Default()
	if e := Store(t.TempDir(), &c, "openai", "  "); e == nil {
		t.Fatal("empty credential accepted")
	}
	if e := Store(t.TempDir(), &c, "missing", "key"); e == nil {
		t.Fatal("unknown provider accepted")
	}
	if c.Providers["openai"].Secret != "" {
		t.Fatal("rejected credential leaked into the configuration")
	}
}

func TestWizardOptionsKeepBeginnerOrder(t *testing.T) {
	options := wizardOptions(Local{Ollama: []string{"qwen3:8b"}})
	if len(options) != 1+len(Providers())+1 {
		t.Fatalf("%d options", len(options))
	}
	if options[0].provider != "local" || options[0].title != "Ollama (running, 1 model(s))" {
		t.Fatalf("%+v", options[0])
	}
	if options[1].provider != "openrouter" || options[len(options)-1].provider != "import" {
		t.Fatalf("catalog order changed: %+v", options)
	}
	if len(wizardOptions(Local{})) != len(Providers())+1 {
		t.Fatal("undetected local servers must not be offered")
	}
	if !LocalProvider("lmstudio") || LocalProvider("openai") {
		t.Fatal("local provider predicate drifted")
	}
}

func TestActivateFallsBackToNextModel(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "retired-model") {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer s.Close()
	dir := t.TempDir()
	c := config.Default()
	c.Providers["probe"] = config.Provider{BaseURL: s.URL, Models: []config.Model{
		{ID: "retired-model", Context: 32768, Tools: true},
		{ID: "healthy-model", Context: 32768, Tools: true},
	}}
	if !activate(context.Background(), dir, c, bufio.NewReader(strings.NewReader("n\n")), "probe", "") {
		t.Fatal("activate failed")
	}
	back, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if back.Provider != "probe" || back.Model != "healthy-model" {
		t.Fatalf("expected fallback to healthy-model, got %s/%s", back.Provider, back.Model)
	}
}

func TestActivateFallsBackAfterModelTimeout(t *testing.T) {
	restore := smokeTimeout
	smokeTimeout = 300 * time.Millisecond
	defer func() { smokeTimeout = restore }()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "hang-model") {
			<-r.Context().Done() // hold the connection until the client gives up
			return
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer s.Close()
	dir := t.TempDir()
	c := config.Default()
	c.Providers["probe"] = config.Provider{BaseURL: s.URL, Models: []config.Model{
		{ID: "hang-model", Context: 32768, Tools: true},
		{ID: "healthy-model", Context: 32768, Tools: true},
	}}
	if !activate(context.Background(), dir, c, bufio.NewReader(strings.NewReader("n\n")), "probe", "") {
		t.Fatal("activate failed")
	}
	back, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if back.Provider != "probe" || back.Model != "healthy-model" {
		t.Fatalf("expected healthy-model after timeout fallback, got %s/%s", back.Provider, back.Model)
	}
}

func TestDetectModels(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"models":[{"name":"qwen3:8b"},{"name":"llama3.2"}]}`)
	}))
	defer ollama.Close()
	if got := detectModels(ollama.URL, "models", "name"); len(got) != 2 || got[0] != "qwen3:8b" {
		t.Fatalf("%v", got)
	}
	if detectModels("http://127.0.0.1:1", "models", "name") != nil {
		t.Fatal("expected nil for unreachable server")
	}
	lmstudio := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"object":"list","data":[{"id":"qwen/qwen3-coder-30b"}]}`)
	}))
	defer lmstudio.Close()
	if got := detectModels(lmstudio.URL, "data", "id"); len(got) != 1 || got[0] != "qwen/qwen3-coder-30b" {
		t.Fatalf("%v", got)
	}
}
