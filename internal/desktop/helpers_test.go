//go:build desktop

package desktop

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
	daemon "github.com/iagoluch/OpenVibeCoding/internal/runtime"
	"github.com/iagoluch/OpenVibeCoding/internal/workspace"
)

// canonicalWorkspace returns the path the daemon will store for a workspace
// directory: workspace.Open applies Abs and EvalSymlinks, so the raw temp
// directory name can differ from the canonical form.
func canonicalWorkspace(t *testing.T, dir string) string {
	t.Helper()
	w, e := workspace.Open(dir, false)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	return w.Path
}

// streamClient is used for /run requests: no timeout, the NDJSON stream stays
// open while the turn is parked.
var streamClient = &http.Client{}

// connectDaemon waits for the daemon on home to answer a health poll.
func connectDaemon(home string) (*daemon.Client, error) {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c, e := daemon.Connect(home); e == nil {
			return c, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, errors.New("daemon not ready")
}

// startDaemon mirrors the runtime package's startFixture: a real Serve on a
// scratch home, connected with a real client. The returned stop shuts the
// daemon down and is idempotent; it also runs at cleanup.
func startDaemon(t *testing.T, cfg config.Config) (string, *daemon.Client, func()) {
	t.Helper()
	home := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- daemon.Serve(ctx, home, cfg) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			select {
			case e := <-done:
				if e != nil {
					t.Error(e)
				}
			case <-time.After(12 * time.Second):
				t.Error("daemon shutdown timed out")
			}
		})
	}
	t.Cleanup(stop)
	client, e := connectDaemon(home)
	if e != nil {
		t.Fatal(e)
	}
	return home, client, stop
}

// fakeProvider serves the OpenAI-compatible SSE the fixtures use: the first
// call asks for a write_file tool call (which parks the turn on approval),
// later calls answer with plain text.
func fakeProvider(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Messages []struct {
				Role string
			}
		}
		json.NewDecoder(r.Body).Decode(&b)
		hasTool := false
		for _, m := range b.Messages {
			if m.Role == "tool" {
				hasTool = true
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if !hasTool {
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"review1\",\"function\":{\"name\":\"write_file\",\"arguments\":\"{\\\"path\\\":\\\"approved.txt\\\",\\\"content\\\":\\\"approved\\\"}\"}}]}}]}\n\ndata: [DONE]\n\n"))
			return
		}
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Observed file write\"}}]}\n\ndata: [DONE]\n\n"))
	}))
}

// fixtureConfig points the local provider preset at the fake SSE server.
func fixtureConfig(providerURL string) config.Config {
	cfg := config.Default()
	p := cfg.Providers["local"]
	p.BaseURL = providerURL
	cfg.Providers["local"] = p
	cfg.Provider = "local"
	cfg.Model = "qwen3:8b"
	return cfg
}

// parkedTurn is a /run request whose turn is parked on an approval.
type parkedTurn struct {
	approval string
	resp     *http.Response
	cancel   context.CancelFunc
}

// release cancels the run request: the daemon drops the turn and the
// pending approval.
func (p *parkedTurn) release() {
	p.cancel()
	p.resp.Body.Close()
}

// parkTurn starts a /run and returns once the daemon hands out an approval.
// owner/token are set when the request goes directly to the daemon; through
// the proxy both are injected by the proxy itself.
func parkTurn(t *testing.T, url, owner, token, session, workspace string) *parkedTurn {
	t.Helper()
	body := fmt.Sprintf(`{"Session":%q,"Workspace":%q,"Prompt":"write","Provider":"local","Model":"qwen3:8b","Harness":"review"}`, session, workspace)
	ctx, cancel := context.WithCancel(context.Background())
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, url+"/run", strings.NewReader(body))
	if e != nil {
		cancel()
		t.Fatal(e)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if owner != "" {
		req.Header.Set("X-OpenVibeCoding-Client", owner)
	}
	resp, e := streamClient.Do(req)
	if e != nil {
		cancel()
		t.Fatal(e)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		t.Fatalf("run refused: HTTP %d", resp.StatusCode)
	}
	approval := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		var event struct {
			Kind     string
			Text     string
			Approval string
		}
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "{") {
				continue
			}
			if json.Unmarshal([]byte(line), &event) == nil && event.Approval != "" {
				approval <- event.Approval
				return // body stays open: the turn stays parked
			}
		}
	}()
	select {
	case id := <-approval:
		return &parkedTurn{approval: id, resp: resp, cancel: cancel}
	case <-time.After(30 * time.Second):
		cancel()
		resp.Body.Close()
		t.Fatal("turn never parked on an approval")
		return nil
	}
}
