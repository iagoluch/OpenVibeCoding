package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidationAndRedaction(t *testing.T) {
	c := Default()
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	t.Setenv("OPENAI_API_KEY", "sensitive-test-key")
	s := c.Redact("sensitive-test-key\x1b]52;clipboard\x07")
	if strings.Contains(s, "sensitive-test-key") || strings.ContainsAny(s, "\x1b\x07") {
		t.Fatal(s)
	}
	p := c.Providers["openai"]
	p.BaseURL = "http://example.com/v1"
	c.Providers["openai"] = p
	if c.Validate() == nil {
		t.Fatal("plaintext remote accepted")
	}
}
func TestLoadPreservesConfiguration(t *testing.T) {
	dir := t.TempDir()
	c, e := Load(dir)
	if e != nil {
		t.Fatal(e)
	}
	again, e := Load(dir)
	if e != nil || again.Model != c.Model {
		t.Fatal(e)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := Default()
	c.Provider, c.Model = "openrouter", "thinkingmachines/inkling:free"
	if e := Save(dir, c); e != nil {
		t.Fatal(e)
	}
	back, e := Load(dir)
	if e != nil || back.Provider != c.Provider || back.Model != c.Model {
		t.Fatal(e)
	}
}

func TestExampleConfigValid(t *testing.T) {
	b, e := os.ReadFile(filepath.Join("..", "..", "config.example.json"))
	if e != nil {
		t.Skip("example config not present")
	}
	// The example is a diff on top of the shipped defaults, exactly like a
	// user's config.json: load defaults first, then apply the file.
	c := Default()
	if e = json.Unmarshal(b, &c); e != nil {
		t.Fatal(e)
	}
	if e = c.Validate(); e != nil {
		t.Fatal(e)
	}
}

func TestLoadAdoptsStockPresets(t *testing.T) {
	dir := t.TempDir()
	stock := `{
  "provider": "gemini",
  "model": "gemini-2.5-flash",
  "harness": "review",
  "providers": {
    "gemini": {"protocol": "gemini", "base_url": "https://generativelanguage.googleapis.com/v1beta", "key_env": "GEMINI_API_KEY", "models": [{"id": "gemini-2.5-flash", "context": 32768, "tools": true}]},
    "anthropic": {"protocol": "anthropic", "base_url": "https://custom.example/v1", "key_env": "ANTHROPIC_API_KEY", "models": [{"id": "claude-sonnet-4-5", "context": 32768, "tools": true}]},
    "custom": {"base_url": "https://example.com/v1", "key_env": "CUSTOM_KEY", "models": [{"id": "m", "context": 8192, "tools": true}]}
  }
}`
	if e := os.WriteFile(filepath.Join(dir, "config.json"), []byte(stock), 0600); e != nil {
		t.Fatal(e)
	}
	c, e := Load(dir)
	if e != nil {
		t.Fatal(e)
	}
	if got := c.Providers["gemini"].Models[0].ID; got != "gemini-3.8-flash" {
		t.Fatalf("stock preset not upgraded: %s", got)
	}
	if c.Model != "gemini-3.8-flash" {
		t.Fatalf("active model not repaired after upgrade: %s", c.Model)
	}
	if c.Providers["anthropic"].BaseURL != "https://custom.example/v1" {
		t.Fatal("customized stock entry replaced")
	}
	if c.Providers["custom"].BaseURL != "https://example.com/v1" {
		t.Fatal("custom provider lost")
	}
}

func TestSaveOmitsStockProviders(t *testing.T) {
	dir := t.TempDir()
	c := Default()
	c.Provider, c.Model = "openrouter", "thinkingmachines/inkling:free"
	c.Providers["custom"] = Provider{BaseURL: "https://example.com/v1", KeyEnv: "CUSTOM_KEY", Models: []Model{{ID: "m", Context: 8192, Tools: true}}}
	if e := Save(dir, c); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(dir, "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	var raw struct {
		Providers map[string]any `json:"providers"`
	}
	if e = json.Unmarshal(b, &raw); e != nil {
		t.Fatal(e)
	}
	if len(raw.Providers) != 1 {
		t.Fatalf("expected only the custom provider on disk, got %d entries", len(raw.Providers))
	}
	if _, ok := raw.Providers["custom"]; !ok {
		t.Fatal("custom provider missing from disk")
	}
	back, e := Load(dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(back.Providers) != len(Default().Providers)+1 {
		t.Fatalf("defaults not merged back: %d", len(back.Providers))
	}
	if back.Provider != "openrouter" || back.Model != "thinkingmachines/inkling:free" {
		t.Fatal("active selection lost")
	}
}

func TestCredentialAcrossStreamFrames(t *testing.T) {
	c := Default()
	p := c.Providers["openai"]
	p.Secret = "synthetic-super-secret-key"
	c.Providers["openai"] = p
	var output string
	redactor := c.Stream(func(s string) {
		output += s
		if strings.Contains(output, "super-secret") {
			t.Fatal("stream leaked credential suffix")
		}
	})
	for _, part := range []string{"Before synthetic-", "super-", "secret-key", " after response"} {
		redactor.Write(part)
	}
	redactor.Close()
	if strings.Contains(output, p.Secret) || !strings.Contains(output, "[REDACTED]") {
		t.Fatal(output)
	}
}

func TestPriceConfiguredDistinctFromUnknown(t *testing.T) {
	c := Default()
	free := c.Providers["openrouter"].Models[0]
	if free.InputPrice == nil || free.OutputPrice == nil || *free.InputPrice != 0 || *free.OutputPrice != 0 {
		t.Fatal("free preset must carry configured zero prices")
	}
	c.Providers["plain"] = Provider{BaseURL: "https://example.com/v1", KeyEnv: "PLAIN_KEY", Models: []Model{{ID: "custom-model", Context: 8192, Tools: true}}}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var back Config
	if err = json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if m := back.Providers["openrouter"].Models[0]; m.InputPrice == nil || *m.InputPrice != 0 {
		t.Fatal("configured zero price lost in round trip")
	}
	if m := back.Providers["openai"].Models[0]; m.InputPrice == nil || *m.InputPrice != 0.40 {
		t.Fatal("paid price lost in round trip")
	}
	if m := back.Providers["plain"].Models[0]; m.InputPrice != nil || m.OutputPrice != nil {
		t.Fatal("unset prices must remain unknown")
	}
}

func TestNegativePriceRejected(t *testing.T) {
	c := Default()
	p := c.Providers["openai"]
	negative := -1.0
	p.Models[0].InputPrice = &negative
	c.Providers["openai"] = p
	if c.Validate() == nil {
		t.Fatal("negative price accepted")
	}
}


func TestLanguagePreferenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := Default()
	c.Language = "pt-BR"
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Language != "pt-BR" {
		t.Fatalf("language preference = %q, want pt-BR", loaded.Language)
	}
}

func TestUnsupportedLanguageRejected(t *testing.T) {
	c := Default()
	c.Language = "fr-FR"
	if err := c.Validate(); err == nil {
		t.Fatal("unsupported interface language accepted")
	}
}
