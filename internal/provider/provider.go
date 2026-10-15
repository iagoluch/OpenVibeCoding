// Package provider implements the OpenAI Chat Completions streaming protocol.
// Other protocols implement Client; compatible endpoints are not native adapters.
package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
)

type Call struct {
	Signature string   `json:"thought_signature,omitempty"`
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Function  Function `json:"function"`
}
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
type Message struct {
	Images  []string `json:"images,omitempty"`
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Calls   []Call   `json:"tool_calls,omitempty"`
	CallID  string   `json:"tool_call_id,omitempty"`
}
type Tool struct {
	Type     string     `json:"type"`
	Function Definition `json:"function"`
}
type Definition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}
type Usage struct {
	Input  int `json:"prompt_tokens"`
	Output int `json:"completion_tokens"`
}
type Reply struct {
	Message Message
	Usage   Usage
}
type Client interface {
	Stream(context.Context, []Message, []Tool, func(string)) (Reply, error)
}
type Compatible struct {
	Settings config.Provider
	Model    config.Model
	HTTP     *http.Client
}

func (p Compatible) Stream(ctx context.Context, messages []Message, tools []Tool, emit func(string)) (Reply, error) {
	var reply Reply
	reply.Message.Role = "assistant"
	key := p.Settings.Secret
	if key == "" {
		key = os.Getenv(p.Settings.KeyEnv)
	}
	if p.Settings.KeyEnv != "" && key == "" {
		return reply, fmt.Errorf("no credential configured: set %s or store one with ovc --credential", p.Settings.KeyEnv)
	}
	// Native protocol metadata is stored for resumption but is not an OpenAI field.
	clean := make([]Message, len(messages))
	copy(clean, messages)
	for i := range clean {
		clean[i].Calls = append([]Call(nil), clean[i].Calls...)
		for j := range clean[i].Calls {
			clean[i].Calls[j].Signature = ""
		}
	}
	messages = clean
	body := map[string]any{"model": p.Model.ID, "messages": openAIMessages(messages), "stream": true, "max_tokens": 2048}
	if p.Settings.MaxOutput > 0 {
		body["max_tokens"] = p.Settings.MaxOutput
	}
	if p.Settings.Temperature != nil {
		body["temperature"] = *p.Settings.Temperature
	}
	if p.Settings.Usage {
		body["stream_options"] = map[string]bool{"include_usage": true}
	}
	if p.Model.Tools && len(tools) > 0 {
		body["tools"] = tools
	}
	data, e := json.Marshal(body)
	if e != nil {
		return reply, e
	}
	client := p.HTTP
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	// Credentials must never follow redirects to a different endpoint.
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		req, e := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(p.Settings.BaseURL, "/")+"/chat/completions", bytes.NewReader(data))
		if e != nil {
			return reply, errors.New("invalid provider request")
		}
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		resp, e = copyClient.Do(req)
		if e != nil {
			if ctx.Err() != nil {
				return reply, ctx.Err()
			}
			return reply, fmt.Errorf("could not reach %s: check the endpoint, network, or that a local server is running", endpointHost(p.Settings.BaseURL))
		}
		if resp.StatusCode != 429 && resp.StatusCode < 500 {
			break
		}
		if attempt == 2 {
			break
		}
		resp.Body.Close()
		select {
		case <-ctx.Done():
			return reply, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return reply, httpStatusError(resp.StatusCode)
	}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 8<<20))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	calls := map[int]*Call{}
	finished := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if raw == "[DONE]" {
			finished = true
			break
		}
		var event struct {
			Error   json.RawMessage `json:"error"`
			Usage   Usage           `json:"usage"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
					Calls   []struct {
						Index    int      `json:"index"`
						ID       string   `json:"id"`
						Function Function `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if e = json.Unmarshal([]byte(raw), &event); e != nil {
			return reply, errors.New("invalid provider stream")
		}
		if len(event.Error) > 0 && string(event.Error) != "null" {
			return reply, errors.New("provider reported a streaming error")
		}
		if event.Usage.Input > 0 || event.Usage.Output > 0 {
			reply.Usage = event.Usage
		}
		if len(event.Choices) == 0 {
			continue
		}
		d := event.Choices[0].Delta
		reply.Message.Content += d.Content
		if len(reply.Message.Content) > 1<<20 {
			return reply, errors.New("provider output exceeds limit")
		}
		if d.Content != "" {
			emit(d.Content)
		}
		for _, tc := range d.Calls {
			if tc.Index < 0 || tc.Index > 31 {
				return reply, errors.New("too many tool calls")
			}
			c := calls[tc.Index]
			if c == nil {
				c = &Call{Type: "function"}
				calls[tc.Index] = c
			}
			c.ID += tc.ID
			c.Function.Name += tc.Function.Name
			c.Function.Arguments += tc.Function.Arguments
			if len(c.Function.Arguments) > 256<<10 {
				return reply, errors.New("tool arguments exceed limit")
			}
		}
	}
	if e = scanner.Err(); e != nil {
		return reply, errors.New("provider stream interrupted")
	}
	if !finished {
		return reply, errors.New("provider stream ended without completion")
	}
	indices := make([]int, 0, len(calls))
	for i := range calls {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	for _, i := range indices {
		if calls[i].ID == "" || calls[i].Function.Name == "" {
			return reply, errors.New("incomplete tool call")
		}
		reply.Message.Calls = append(reply.Message.Calls, *calls[i])
	}
	return reply, nil
}

func endpointHost(base string) string {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return "the provider endpoint"
	}
	return u.Host
}

// httpStatusError converts a non-200 provider response into an actionable
// message. Response bodies stay hidden because they can echo credentials.
func httpStatusError(status int) error {
	switch {
	case status == 401:
		return errors.New("authentication failed (HTTP 401): the credential is missing or invalid; store one with ovc --credential or set the provider key")
	case status == 403:
		return errors.New("access denied (HTTP 403): the key cannot use this model or endpoint; check your provider plan and access")
	case status == 404:
		return errors.New("not found (HTTP 404): verify the model ID and the provider base URL")
	case status == 429:
		return errors.New("rate limited (HTTP 429): wait a moment, use a smaller request, or switch models")
	case status >= 300 && status < 400:
		return errors.New("provider redirected the request: redirects are refused to protect credentials; verify the base URL")
	case status >= 500:
		return fmt.Errorf("provider unavailable (HTTP %d): retry shortly or switch models", status)
	default:
		return fmt.Errorf("unexpected provider response (HTTP %d); response body kept hidden to protect credentials", status)
	}
}
