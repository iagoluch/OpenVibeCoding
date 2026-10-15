package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func New(p config.Provider, m config.Model) Client {
	if p.Protocol == "anthropic" || p.Protocol == "gemini" {
		return Native{Settings: p, Model: m}
	}
	return Compatible{Settings: p, Model: m}
}

type Native struct {
	Settings config.Provider
	Model    config.Model
	HTTP     *http.Client
}

func (p Native) Stream(ctx context.Context, messages []Message, tools []Tool, emit func(string)) (Reply, error) {
	var result Reply
	result.Message.Role = "assistant"
	key := p.Settings.Secret
	if key == "" {
		key = os.Getenv(p.Settings.KeyEnv)
	}
	if key == "" {
		return result, fmt.Errorf("no credential configured: set %s or store one with ovc --credential", p.Settings.KeyEnv)
	}
	var systems []string
	var converted []map[string]any
	names := map[string]string{}
	for _, m := range messages {
		if m.Role == "system" {
			systems = append(systems, m.Content)
			continue
		}
		if p.Settings.Protocol == "anthropic" {
			role := m.Role
			var blocks []any
			if role == "tool" {
				role = "user"
				blocks = append(blocks, map[string]any{"type": "tool_result", "tool_use_id": m.CallID, "content": m.Content})
			} else {
				if m.Content != "" {
					blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
				}
				for _, call := range m.Calls {
					var args any
					if json.Unmarshal([]byte(call.Function.Arguments), &args) != nil {
						return result, errors.New("invalid stored tool arguments")
					}
					blocks = append(blocks, map[string]any{"type": "tool_use", "id": call.ID, "name": call.Function.Name, "input": args})
				}
			}
			for _, img := range m.Images {
				blocks = append(blocks, map[string]any{"type": "image", "source": map[string]string{"type": "base64", "media_type": "image/png", "data": img}})
			}
			if len(blocks) > 0 {
				converted = append(converted, map[string]any{"role": role, "content": blocks})
			}
		} else {
			role := m.Role
			if role == "assistant" {
				role = "model"
			}
			var parts []any
			if role == "tool" {
				role = "user"
				parts = append(parts, map[string]any{"functionResponse": map[string]any{"name": names[m.CallID], "response": map[string]string{"result": m.Content}}})
			} else {
				if m.Content != "" {
					parts = append(parts, map[string]any{"text": m.Content})
				}
				for _, call := range m.Calls {
					names[call.ID] = call.Function.Name
					var args any
					if json.Unmarshal([]byte(call.Function.Arguments), &args) != nil {
						return result, errors.New("invalid stored tool arguments")
					}
					part := map[string]any{"functionCall": map[string]any{"name": call.Function.Name, "args": args}}
					if call.Signature != "" {
						part["thoughtSignature"] = call.Signature
					}
					parts = append(parts, part)
				}
			}
			for _, img := range m.Images {
				parts = append(parts, map[string]any{"inlineData": map[string]string{"mimeType": "image/png", "data": img}})
			}
			if len(parts) > 0 {
				converted = append(converted, map[string]any{"role": role, "parts": parts})
			}
		}
	}
	maxOutput := p.Settings.MaxOutput
	if maxOutput == 0 {
		maxOutput = 2048
	}
	var body map[string]any
	var endpoint string
	if p.Settings.Protocol == "anthropic" {
		body = map[string]any{"model": p.Model.ID, "max_tokens": maxOutput, "stream": true, "system": strings.Join(systems, "\n\n"), "messages": converted}
		endpoint = strings.TrimRight(p.Settings.BaseURL, "/") + "/messages"
		var defs []any
		if p.Model.Tools {
			for _, t := range tools {
				defs = append(defs, map[string]any{"name": t.Function.Name, "description": t.Function.Description, "input_schema": t.Function.Parameters})
			}
		}
		if len(defs) > 0 {
			body["tools"] = defs
		}
		if p.Settings.Temperature != nil {
			body["temperature"] = *p.Settings.Temperature
		}
	} else {
		gen := map[string]any{"maxOutputTokens": maxOutput}
		if p.Settings.Temperature != nil {
			gen["temperature"] = *p.Settings.Temperature
		}
		body = map[string]any{"contents": converted, "systemInstruction": map[string]any{"parts": []any{map[string]string{"text": strings.Join(systems, "\n\n")}}}, "generationConfig": gen}
		endpoint = strings.TrimRight(p.Settings.BaseURL, "/") + "/models/" + url.PathEscape(p.Model.ID) + ":streamGenerateContent?alt=sse"
		var defs []any
		if p.Model.Tools {
			for _, t := range tools {
				defs = append(defs, map[string]any{"name": t.Function.Name, "description": t.Function.Description, "parametersJsonSchema": t.Function.Parameters})
			}
		}
		if len(defs) > 0 {
			body["tools"] = []any{map[string]any{"functionDeclarations": defs}}
		}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return result, err
	}
	client := p.HTTP
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	hc := *client
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
		if err != nil {
			return result, errors.New("invalid native provider endpoint")
		}
		req.Header.Set("Content-Type", "application/json")
		if p.Settings.Protocol == "anthropic" {
			req.Header.Set("x-api-key", key)
			req.Header.Set("anthropic-version", "2023-06-01")
		} else {
			req.Header.Set("x-goog-api-key", key)
		}
		resp, err = hc.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			return result, fmt.Errorf("could not reach %s: check the endpoint, network, or that a local server is running", endpointHost(p.Settings.BaseURL))
		}
		if (resp.StatusCode != 429 && resp.StatusCode < 500) || attempt == 2 {
			break
		}
		resp.Body.Close()
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return result, httpStatusError(resp.StatusCode)
	}
	scan := bufio.NewScanner(io.LimitReader(resp.Body, 8<<20))
	scan.Buffer(make([]byte, 4096), 1<<20)
	finished := false
	indices := map[int]int{}
	for scan.Scan() {
		line := scan.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if p.Settings.Protocol == "anthropic" {
			var ev struct {
				Type    string
				Index   int
				Message struct {
					Usage struct {
						Input int `json:"input_tokens"`
					}
				}
				Block struct {
					Type, ID, Name string
					Input          json.RawMessage
				} `json:"content_block"`
				Delta struct {
					Text    string
					Partial string `json:"partial_json"`
				}
				Usage struct {
					Output int `json:"output_tokens"`
				}
			}
			if json.Unmarshal([]byte(data), &ev) != nil {
				return result, errors.New("invalid Anthropic stream")
			}
			switch ev.Type {
			case "error":
				return result, errors.New("Anthropic stream error")
			case "message_start":
				result.Usage.Input = ev.Message.Usage.Input
			case "message_delta":
				result.Usage.Output = ev.Usage.Output
			case "message_stop":
				finished = true
			case "content_block_start":
				if ev.Block.Type == "tool_use" {
					indices[ev.Index] = len(result.Message.Calls)
					result.Message.Calls = append(result.Message.Calls, Call{ID: ev.Block.ID, Type: "function", Function: Function{Name: ev.Block.Name}})
				}
			case "content_block_delta":
				if ev.Delta.Text != "" {
					result.Message.Content += ev.Delta.Text
					emit(ev.Delta.Text)
				}
				if ev.Delta.Partial != "" {
					i, ok := indices[ev.Index]
					if !ok {
						return result, errors.New("tool fragment without start")
					}
					result.Message.Calls[i].Function.Arguments += ev.Delta.Partial
				}
			}
		} else {
			var ev struct {
				Error      json.RawMessage
				Candidates []struct {
					Finish  string `json:"finishReason"`
					Content struct {
						Parts []struct {
							Text      string
							Thought   bool
							Signature string `json:"thoughtSignature"`
							Call      *struct {
								Name string
								Args json.RawMessage
							} `json:"functionCall"`
						}
					}
				}
				Usage struct {
					Input  int `json:"promptTokenCount"`
					Output int `json:"candidatesTokenCount"`
				} `json:"usageMetadata"`
			}
			if json.Unmarshal([]byte(data), &ev) != nil {
				return result, errors.New("invalid Gemini stream")
			}
			if len(ev.Error) > 0 {
				return result, errors.New("Gemini stream error")
			}
			if ev.Usage.Input > 0 {
				result.Usage = Usage{Input: ev.Usage.Input, Output: ev.Usage.Output}
			}
			if len(ev.Candidates) > 0 {
				candidate := ev.Candidates[0]
				if candidate.Finish != "" {
					if candidate.Finish != "STOP" {
						return result, fmt.Errorf("Gemini generation stopped: %s", candidate.Finish)
					}
					finished = true
				}
				for _, part := range candidate.Content.Parts {
					if part.Text != "" && !part.Thought {
						result.Message.Content += part.Text
						emit(part.Text)
					}
					if part.Call != nil {
						result.Message.Calls = append(result.Message.Calls, Call{ID: fmt.Sprintf("gemini-%d-%d", time.Now().UnixNano(), len(result.Message.Calls)), Type: "function", Function: Function{Name: part.Call.Name, Arguments: string(part.Call.Args)}, Signature: part.Signature})
					}
				}
			}
		}
		if len(result.Message.Content) > 1<<20 || len(result.Message.Calls) > 32 {
			return result, errors.New("native provider response too large")
		}
	}
	if scan.Err() != nil || !finished {
		return result, errors.New("native provider stream incomplete")
	}
	for i := range result.Message.Calls {
		c := &result.Message.Calls[i]
		if c.Function.Arguments == "" {
			c.Function.Arguments = "{}"
		}
		if !json.Valid([]byte(c.Function.Arguments)) || c.ID == "" {
			return result, errors.New("invalid native tool call")
		}
	}
	return result, nil
}
