package i18n

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"": "auto",
		"system": "auto",
		"en-US": "en",
		"pt": "pt-BR",
		"pt_br": "pt-BR",
		"es-MX": "es",
		"zh-Hans": "zh-CN",
	}
	for input, want := range cases {
		got, ok := Normalize(input)
		if !ok || got != want {
			t.Errorf("Normalize(%q) = %q, %t; want %q, true", input, got, ok, want)
		}
	}
	if _, ok := Normalize("fr-FR"); ok {
		t.Fatal("unsupported locale accepted")
	}
}

func TestResolveFromEnvironment(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "pt_BR.UTF-8")
	if got := Resolve(Auto); got != "pt-BR" {
		t.Fatalf("Resolve(auto) = %q, want pt-BR", got)
	}
}

func TestCatalogFallback(t *testing.T) {
	if got := T("pt-BR", "language.changed"); got != "Idioma da interface salvo: %s" {
		t.Fatalf("unexpected Portuguese translation: %q", got)
	}
	if got := T("pt-BR", "missing.catalog.key"); got != "missing.catalog.key" {
		t.Fatalf("missing key should remain visible, got %q", got)
	}
}

func TestSupportedReturnsCopy(t *testing.T) {
	first := Supported()
	first[0] = "corrupted"
	second := Supported()
	if second[0] != Auto {
		t.Fatal("Supported exposes its internal slice")
	}
}
