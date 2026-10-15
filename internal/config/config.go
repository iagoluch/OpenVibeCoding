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
	ID          string   `json:"id"`
	Context     int      `json:"context"`
	Tools       bool     `json:"tools"`
	InputPrice  *float64 `json:"input_per_million,omitempty"`
	OutputPrice *float64 `json:"output_per_million,omitempty"`
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

// price marks a model price as configured. nil means unknown; 0 means free.
func price(v float64) *float64 { return &v }

func Default() Config {
	return Config{Provider: "openai", Model: "gpt-4.1-mini", Harness: "review", Providers: map[string]Provider{
		"anthropic": {Protocol: "anthropic", BaseURL: "https://api.anthropic.com/v1", KeyEnv: "ANTHROPIC_API_KEY", Models: []Model{
			{ID: "claude-sonnet-4-5", Context: 200000, Tools: true, InputPrice: price(3), OutputPrice: price(15)},
		}},
		"gemini": {Protocol: "gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta", KeyEnv: "GEMINI_API_KEY", Models: []Model{
			{ID: "gemini-2.5-flash", Context: 1048576, Tools: true, InputPrice: price(0.30), OutputPrice: price(2.50)},
			{ID: "gemini-2.5-flash-lite", Context: 1048576, Tools: true, InputPrice: price(0.10), OutputPrice: price(0.40)},
		}},
		"openai": {BaseURL: "https://api.openai.com/v1", KeyEnv: "OPENAI_API_KEY", Usage: true, Models: []Model{
			{ID: "gpt-4.1-mini", Context: 1047576, Tools: true, InputPrice: price(0.40), OutputPrice: price(1.60)},
			{ID: "gpt-4.1", Context: 1047576, Tools: true, InputPrice: price(2), OutputPrice: price(8)},
		}},
		"deepseek": {BaseURL: "https://api.deepseek.com", KeyEnv: "DEEPSEEK_API_KEY", Usage: true, Models: []Model{
			{ID: "deepseek-flash", Context: 1000000, Tools: true, InputPrice: price(0.15), OutputPrice: price(0.60)},
			{ID: "deepseek-v4-pro", Context: 1000000, Tools: true, InputPrice: price(0.66), OutputPrice: price(1.98)},
		}},
		"kimi": {BaseURL: "https://api.moonshot.ai/v1", KeyEnv: "MOONSHOT_API_KEY", Usage: true, Models: []Model{
			{ID: "kimi-k3", Context: 1048576, Tools: true, InputPrice: price(3), OutputPrice: price(15)},
			{ID: "kimi-k2.6", Context: 262144, Tools: true, InputPrice: price(0.95), OutputPrice: price(4)},
		}},
		"qwen": {BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", KeyEnv: "DASHSCOPE_API_KEY", Usage: true, Models: []Model{
			{ID: "qwen-plus", Context: 1000000, Tools: true, InputPrice: price(0.40), OutputPrice: price(1.20)},
			{ID: "qwen3-coder-plus", Context: 1048576, Tools: true, InputPrice: price(1), OutputPrice: price(5)},
		}},
		"openrouter": {BaseURL: "https://openrouter.ai/api/v1", KeyEnv: "OPENROUTER_API_KEY", Usage: true, Models: []Model{
			{ID: "thinkingmachines/inkling:free", Context: 1048576, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
			{ID: "nvidia/nemotron-3.5-lightning:free", Context: 1000000, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
			{ID: "poolside/laguna-s-2.1:free", Context: 262144, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
			{ID: "cohere/north-mini-code:free", Context: 256000, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
			{ID: "liquid/lfm-2.5-2.6b:free", Context: 65536, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
		}},
		"groq": {BaseURL: "https://api.groq.com/openai/v1", KeyEnv: "GROQ_API_KEY", Usage: true, Models: []Model{
			{ID: "openai/gpt-oss-120b", Context: 131072, Tools: true, InputPrice: price(0.15), OutputPrice: price(0.60)},
			{ID: "llama-3.3-70b-versatile", Context: 131072, Tools: true, InputPrice: price(0.59), OutputPrice: price(0.79)},
			{ID: "qwen/qwen3.8-27b", Context: 131042, Tools: true, InputPrice: price(0.80), OutputPrice: price(4)},
			{ID: "llama-3.1-8b-instant", Context: 131072, Tools: true, InputPrice: price(0.05), OutputPrice: price(0.08)},
		}},
		"cerebras": {BaseURL: "https://api.cerebras.ai/v1", KeyEnv: "CEREBRAS_API_KEY", Usage: true, Models: []Model{
			{ID: "gpt-oss-120b", Context: 65536, Tools: true, InputPrice: price(0.35), OutputPrice: price(0.75)},
			{ID: "qwen-3.8-27b", Context: 65536, Tools: true, InputPrice: price(0.99), OutputPrice: price(1.49)},
		}},
		"nvidia": {BaseURL: "https://integrate.api.nvidia.com/v1", KeyEnv: "NVIDIA_API_KEY", Usage: true, Models: []Model{
			{ID: "z-ai/glm-5.3-flash", Context: 1000000, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
			{ID: "bytedance/seed-oss-36b-instruct", Context: 262000, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
			{ID: "poolside/laguna-xs-2.1", Context: 262144, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
		}},
		"zai": {BaseURL: "https://api.z.ai/api/paas/v4", KeyEnv: "ZAI_API_KEY", Usage: true, Models: []Model{
			{ID: "glm-4.7-flash", Context: 200000, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
			{ID: "glm-4.5-flash", Context: 131072, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
		}},
		"mistral": {BaseURL: "https://api.mistral.ai/v1", KeyEnv: "MISTRAL_API_KEY", Models: []Model{
			{ID: "mistral-small-latest", Context: 262144, Tools: true, InputPrice: price(0.15), OutputPrice: price(0.60)},
			{ID: "devstral-medium-latest", Context: 262144, Tools: true, InputPrice: price(0.40), OutputPrice: price(2)},
		}},
		"local": {BaseURL: "http://127.0.0.1:11434/v1", Models: []Model{
			{ID: "qwen3:8b", Context: 40960, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
		}},
		"lmstudio": {BaseURL: "http://127.0.0.1:1234/v1", Models: []Model{
			{ID: "qwen/qwen3-coder-30b", Context: 262144, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
			{ID: "openai/gpt-oss-20b", Context: 131072, Tools: true, InputPrice: price(0), OutputPrice: price(0)},
		}},
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
			if strings.TrimSpace(m.ID) == "" || m.Context < 4096 ||
				(m.InputPrice != nil && *m.InputPrice < 0) || (m.OutputPrice != nil && *m.OutputPrice < 0) {
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
