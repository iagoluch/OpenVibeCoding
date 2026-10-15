package config

import (
	"encoding/json"
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
