package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamToolsAndUsage(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-only" {
			t.Error("bad request")
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\",\"tool_calls\":[{\"index\":0,\"id\":\"c1\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\"}}]}}]}\n\ndata: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"README.md\\\"}\"}}]}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":6}}\n\ndata: [DONE]\n\n")
	}))
	defer s.Close()
	t.Setenv("TEST_PROVIDER_KEY", "test-only")
	p := Compatible{Settings: config.Provider{BaseURL: s.URL, KeyEnv: "TEST_PROVIDER_KEY", Usage: true}, Model: config.Model{ID: "test", Tools: true}}
	var stream string
	r, e := p.Stream(context.Background(), nil, nil, func(s string) { stream += s })
	if e != nil {
		t.Fatal(e)
	}
	if stream != "Hello" || r.Usage.Input != 12 || len(r.Message.Calls) != 1 || r.Message.Calls[0].Function.Arguments != `{"path":"README.md"}` {
		t.Fatalf("%+v", r)
	}
}
func TestErrorsAndRedirects(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
	}{{"auth", "sensitive provider body", 401}, {"malformed", "data: broken\n", 200}, {"truncated", "data: {\"choices\":[]}\n", 200}, {"redirect", "", 302}} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "http://example.invalid")
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.body)
			}))
			defer s.Close()
			p := Compatible{Settings: config.Provider{BaseURL: s.URL}}
			_, e := p.Stream(context.Background(), nil, nil, func(string) {})
			if e == nil || strings.Contains(e.Error(), "sensitive") {
				t.Fatalf("unsafe/missing error: %v", e)
			}
		})
	}
}
func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := Compatible{Settings: config.Provider{BaseURL: "http://127.0.0.1:1"}}
	if _, e := p.Stream(ctx, nil, nil, func(string) {}); e == nil {
		t.Fatal("expected cancellation")
	}
}

func TestStatusErrorsAreActionable(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   string
	}{{401, "authentication failed"}, {403, "access denied"}, {404, "not found"}, {429, "rate limited"}, {503, "unavailable"}, {302, "redirect"}} {
		if e := httpStatusError(tc.status); !strings.Contains(e.Error(), tc.want) {
			t.Fatalf("status %d: %v", tc.status, e)
		}
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, "sensitive body with key sk-123")
	}))
	defer s.Close()
	p := Compatible{Settings: config.Provider{BaseURL: s.URL}}
	_, e := p.Stream(context.Background(), nil, nil, func(string) {})
	if e == nil || !strings.Contains(e.Error(), "401") || strings.Contains(e.Error(), "sensitive") {
		t.Fatalf("unsafe/missing error: %v", e)
	}
}

func TestModelUnavailableClassification(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   bool
	}{{404, true}, {429, true}, {500, true}, {503, true}, {401, false}, {403, false}, {400, false}, {302, false}} {
		if got := ModelUnavailable(httpStatusError(tc.status)); got != tc.want {
			t.Fatalf("status %d: got %v, want %v", tc.status, got, tc.want)
		}
	}
	if ModelUnavailable(errors.New("network trouble")) {
		t.Fatal("non-status errors must not be classified as model-level")
	}
	if ModelUnavailable(fmt.Errorf("wrapped: %w", httpStatusError(503))) != true {
		t.Fatal("wrapped status errors must keep their classification")
	}
}

func TestUnreachableEndpointMessage(t *testing.T) {
	p := Compatible{Settings: config.Provider{BaseURL: "http://127.0.0.1:1"}}
	_, e := p.Stream(context.Background(), nil, nil, func(string) {})
	if e == nil || !strings.Contains(e.Error(), "could not reach 127.0.0.1:1") {
		t.Fatalf("%v", e)
	}
}
