package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/iagoluch/OpenVibeCoding/internal/automation"
	"github.com/iagoluch/OpenVibeCoding/internal/credentials"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Model struct {
	ID          string  `json:"id"`
	Context     int     `json:"context"`
	Tools       bool    `json:"tools"`
	InputPrice  float64 `json:"input_per_million,omitempty"`
	OutputPrice float64 `json:"output_per_million,omitempty"`
}
type Provider struct {
	Protocol    string   `json:"protocol,omitempty"`
	Secret      string   `json:"-"`
	MaxOutput   int      `json:"max_output,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	BaseURL     string   `json:"base_url"`
	KeyEnv      string   `json:"key_env"`
	Models      []Model  `json:"models"`
	Usage       bool     `json:"stream_usage"`
}
type Permissions struct {
	Network bool `json:"network"`
	Install bool `json:"install"`
}
type Config struct {
	Debug       bool                `json:"debug,omitempty"`
	Permissions Permissions         `json:"permissions"`
	Automation  automation.Settings `json:"automation"`
	Provider    string              `json:"provider"`
	Model       string              `json:"model"`
	Harness     string              `json:"harness"`
	Providers   map[string]Provider `json:"providers"`
}

func Default() Config {
	return Config{Provider: "openai", Model: "gpt-4.1-mini", Harness: "review", Providers: map[string]Provider{
		"anthropic": {Protocol: "anthropic", BaseURL: "https://api.anthropic.com/v1", KeyEnv: "ANTHROPIC_API_KEY", Models: []Model{{ID: "claude-sonnet-4-5", Context: 32768, Tools: true}}},
		"gemini":    {Protocol: "gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta", KeyEnv: "GEMINI_API_KEY", Models: []Model{{ID: "gemini-2.5-flash", Context: 32768, Tools: true}}},
		"openai":    {BaseURL: "https://api.openai.com/v1", KeyEnv: "OPENAI_API_KEY", Usage: true, Models: []Model{{ID: "gpt-4.1-mini", Context: 32768, Tools: true}}},
		"deepseek":  {BaseURL: "https://api.deepseek.com", KeyEnv: "DEEPSEEK_API_KEY", Models: []Model{{ID: "deepseek-chat", Context: 32768, Tools: true}}},
		"kimi":      {BaseURL: "https://api.moonshot.ai/v1", KeyEnv: "MOONSHOT_API_KEY", Models: []Model{{ID: "kimi-k2", Context: 32768, Tools: true}}},
		"qwen":      {BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", KeyEnv: "DASHSCOPE_API_KEY", Models: []Model{{ID: "qwen-plus", Context: 32768, Tools: true}}},
		"local":     {BaseURL: "http://127.0.0.1:11434/v1", Models: []Model{{ID: "qwen3:8b", Context: 8192, Tools: true}}},
	}}
}
func Dir() (string, error) {
	if p := os.Getenv("OPENVIBECODING_HOME"); p != "" {
		return filepath.Abs(p)
	}
	p, e := os.UserConfigDir()
	return filepath.Join(p, "openvibecoding"), e
}
func Load(dir string) (Config, error) {
	c := Default()
	if e := os.MkdirAll(dir, 0700); e != nil {
		return c, e
	}
	b, e := os.ReadFile(filepath.Join(dir, "config.json"))
	if errors.Is(e, os.ErrNotExist) {
		b, _ = json.MarshalIndent(c, "", "  ")
		e = os.WriteFile(filepath.Join(dir, "config.json"), b, 0600)
	} else if e == nil {
		e = json.Unmarshal(b, &c)
	}
	if e != nil {
		return c, e
	}
	if os.Getenv("OPENVIBECODING_DEBUG") == "1" {
		c.Debug = true
	}
	if p := os.Getenv("OPENVIBECODING_PROVIDER"); p != "" {
		c.Provider = p
	}
	if m := os.Getenv("OPENVIBECODING_MODEL"); m != "" {
		c.Model = m
	}
	if c.Automation.BrowserModule == "" {
		c.Automation.BrowserModule = filepath.Join(dir, "qa", "node_modules", "playwright")
	}
	for name, p := range c.Providers {
		p.Secret = os.Getenv(p.KeyEnv)
		if p.Secret == "" && p.KeyEnv != "" {
			p.Secret = credentials.Get(dir, name)
		}
		c.Providers[name] = p
	}
	return c, c.Validate()
}
func (c Config) Active() (Provider, Model, error) {
	p, ok := c.Providers[c.Provider]
	if !ok {
		return p, Model{}, fmt.Errorf("unknown provider %q", c.Provider)
	}
	for _, m := range p.Models {
		if m.ID == c.Model {
			return p, m, nil
		}
	}
	return p, Model{}, fmt.Errorf("unknown model %q for %s", c.Model, c.Provider)
}
func (c Config) Validate() error {
	if c.Harness != "review" && c.Harness != "inspect" {
		return errors.New("harness must be review or inspect")
	}
	for name, p := range c.Providers {
		if p.Protocol != "" && p.Protocol != "openai" && p.Protocol != "anthropic" && p.Protocol != "gemini" {
			return fmt.Errorf("unsupported protocol for %s", name)
		}
		if p.MaxOutput > 16384 {
			return fmt.Errorf("max_output exceeds 16384 for %s", name)
		}
		if p.MaxOutput < 0 || (p.Temperature != nil && (*p.Temperature < 0 || *p.Temperature > 2)) {
			return fmt.Errorf("invalid request settings for %s", name)
		}
		u, e := url.Parse(p.BaseURL)
		if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("invalid endpoint for %s", name)
		}
		if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")) {
			return fmt.Errorf("%s requires HTTPS (HTTP only allowed on loopback)", name)
		}
		if len(p.Models) == 0 {
			return fmt.Errorf("%s has no models", name)
		}
		for _, m := range p.Models {
			if p.MaxOutput > m.Context-2048 {
				return fmt.Errorf("output reservation exceeds context budget in %s", name)
			}
			if strings.TrimSpace(m.ID) == "" || m.Context < 4096 || m.InputPrice < 0 || m.OutputPrice < 0 {
				return fmt.Errorf("invalid model configuration in %s", name)
			}
		}
	}
	_, _, e := c.Active()
	return e
}

// Redact removes configured credentials and terminal controls before persistence or display.
func (c Config) Redact(s string) string {
	for _, p := range c.Providers {
		k := p.Secret
		if k == "" {
			k = os.Getenv(p.KeyEnv)
		}
		if k != "" {
			s = strings.ReplaceAll(s, k, "[REDACTED]")
			// Streaming may end halfway through a credential. Hide matching
			// suffixes until more data arrives, including short prefixes.
			for n := min(len(k)-1, len(s)); n > 0; n-- {
				if strings.HasSuffix(s, k[:n]) {
					s = s[:len(s)-n] + "[REDACTED]"
					break
				}
			}
		}
	}
	return strings.Map(func(r rune) rune {
		if (r < 32 && r != '\n' && r != '\t') || r == 127 || (r >= 128 && r < 160) {
			return -1
		}
		return r
	}, s)
}
