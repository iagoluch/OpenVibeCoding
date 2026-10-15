package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/iagoluch/OpenVibeCoding/internal/agent"
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
	"github.com/iagoluch/OpenVibeCoding/internal/store"
	"io"
	"net/http"
	"time"
)

type Client struct {
	Meta Metadata
	ID   string
	HTTP *http.Client
}

func Connect(dir string) (*Client, error) {
	meta, e := Discover(dir)
	if e != nil {
		return nil, e
	}
	c := &Client{Meta: meta, ID: random(), HTTP: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r, e := c.request(ctx, "/health", nil)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	var v struct{ Protocol int }
	if json.NewDecoder(r.Body).Decode(&v) != nil || v.Protocol != Protocol {
		return nil, errors.New("runtime protocol mismatch")
	}
	return c, nil
}
func (c *Client) request(ctx context.Context, path string, v any) (*http.Response, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	r, e := http.NewRequestWithContext(ctx, "POST", c.Meta.URL+path, bytes.NewReader(b))
	if e != nil {
		return nil, e
	}
	r.Header.Set("Authorization", "Bearer "+c.Meta.Token)
	r.Header.Set("X-Interpreter-Client", c.ID)
	r.Header.Set("Content-Type", "application/json")
	resp, e := c.HTTP.Do(r)
	if e != nil {
		return nil, errors.New("runtime connection failed")
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("runtime HTTP %d; session may be busy or request invalid", resp.StatusCode)
	}
	return resp, nil
}
func (c *Client) Status(stop bool) (string, error) {
	path := "/health"
	if stop {
		path = "/stop"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, e := c.request(ctx, path, nil)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	b, e := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return string(b), e
}
func (c *Client) call(req request, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, e := c.request(ctx, "/store", req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	var v response
	if e = json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&v); e != nil {
		return e
	}
	if v.Error != "" {
		return errors.New(v.Error)
	}
	if out != nil {
		return json.Unmarshal(v.Value, out)
	}
	return nil
}
func (c *Client) New(w string) (v string, e error) {
	e = c.call(request{Op: "new", Workspace: w}, &v)
	return
}
func (c *Client) List() (v []store.Session, e error) { e = c.call(request{Op: "list"}, &v); return }
func (c *Client) Get(id string) (v store.Session, e error) {
	e = c.call(request{Op: "get", ID: id}, &v)
	return
}
func (c *Client) History(id string) (v []store.Entry, e error) {
	e = c.call(request{Op: "history", ID: id}, &v)
	return
}
func (c *Client) Audit(id, text string) error {
	return c.call(request{Op: "audit", ID: id, Text: text}, nil)
}
func (c *Client) Rename(id, text string) error {
	return c.call(request{Op: "rename", ID: id, Text: text}, nil)
}
func (c *Client) Delete(id string) error  { return c.call(request{Op: "delete", ID: id}, nil) }
func (c *Client) Recover(id string) error { return c.call(request{Op: "recover", ID: id}, nil) }
func (c *Client) Summary(id string) (v store.Summary, e error) {
	e = c.call(request{Op: "summary", ID: id}, &v)
	return
}
func (c *Client) Usage(id string) (v []store.Usage, e error) {
	e = c.call(request{Op: "usage", ID: id}, &v)
	return
}
func (c *Client) Export(id string) (v []byte, e error) {
	e = c.call(request{Op: "export", ID: id}, &v)
	return
}
func (c *Client) Append(string, provider.Message) error { return errors.New("runtime owns messages") }
func (c *Client) SaveSummary(string, store.Summary) error {
	return errors.New("runtime owns summaries")
}
func (c *Client) AddUsage(string, store.Usage) error { return errors.New("runtime owns usage") }
func (c *Client) Run(ctx context.Context, e *agent.Engine, prompt string, emit func(agent.Event)) error {
	resp, err := c.request(ctx, "/run", turn{Session: e.Session, Prompt: prompt, Provider: e.Config.Provider, Model: e.Config.Model, Harness: e.Config.Harness, Workspace: e.Workspace.Path, Shell: e.Workspace.AllowShell, Automation: e.Config.Automation, Permissions: e.Config.Permissions})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	for {
		var v wireEvent
		if err = decoder.Decode(&v); err != nil {
			return errors.New("runtime stream interrupted; inspect saved session before retrying")
		}
		if v.Kind == "done" {
			if v.Text != "" {
				return errors.New(v.Text)
			}
			return nil
		}
		event := agent.Event{Kind: v.Kind, Text: v.Text, Usage: v.Usage}
		if v.Approval != "" {
			event.Decision = make(chan bool, 1)
		}
		emit(event)
		if event.Decision != nil {
			var yes bool
			select {
			case yes = <-event.Decision:
			case <-ctx.Done():
				return ctx.Err()
			}
			response, err := c.request(ctx, "/approve", map[string]any{"ID": v.Approval, "Allow": yes})
			if err != nil {
				return err
			}
			response.Body.Close()
		}
	}
}
