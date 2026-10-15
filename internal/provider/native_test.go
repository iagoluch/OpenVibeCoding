package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNativeProtocols(t *testing.T) {
	for _, protocol := range []string{"anthropic", "gemini"} {
		t.Run(protocol, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if json.NewDecoder(r.Body).Decode(&body) != nil {
					t.Error("bad request")
				}
				if strings.Contains(r.URL.RawQuery, "secret") {
					t.Error("key in URL")
				}
				if protocol == "anthropic" {
					if r.Header.Get("x-api-key") != "test-secret" || body["messages"] == nil {
						t.Error("wrong Anthropic protocol")
					}
					fmt.Fprint(w, "data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":9}}}\n\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"call1\",\"name\":\"read_file\"}}\n\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"partial_json\":\"{\\\"path\\\":\\\"README.md\\\"}\"}}\n\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":4}}\n\ndata: {\"type\":\"message_stop\"}\n\n")
				} else {
					if r.Header.Get("x-goog-api-key") != "test-secret" || body["contents"] == nil {
						t.Error("wrong Gemini protocol")
					}
					fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"functionCall\":{\"name\":\"read_file\",\"args\":{\"path\":\"README.md\"}},\"thoughtSignature\":\"sig\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":9,\"candidatesTokenCount\":4}}\n\n")
				}
			}))
			defer server.Close()
			p := Native{Settings: config.Provider{Protocol: protocol, BaseURL: server.URL, Secret: "test-secret"}, Model: config.Model{ID: "test", Tools: true}}
			reply, e := p.Stream(context.Background(), []Message{{Role: "user", Content: "read project"}}, nil, func(string) {})
			if e != nil {
				t.Fatal(e)
			}
			if len(reply.Message.Calls) != 1 || reply.Usage.Input != 9 || reply.Usage.Output != 4 {
				t.Fatalf("%+v", reply)
			}
			if protocol == "gemini" && reply.Message.Calls[0].Signature != "sig" {
				t.Fatal("signature lost")
			}
		})
	}
}
func TestNativeTruncated(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, `data: {"type":"ping"}`) }))
	defer s.Close()
	p := Native{Settings: config.Provider{Protocol: "anthropic", BaseURL: s.URL, Secret: "test"}}
	if _, e := p.Stream(context.Background(), nil, nil, func(string) {}); e == nil {
		t.Fatal("truncated stream accepted")
	}
}
