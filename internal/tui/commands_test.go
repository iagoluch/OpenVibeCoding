package tui

import (
	"strings"
	"testing"
)

func TestCommands(t *testing.T) {
	name, arg, e := Parse(" /model local/qwen3:8b ")
	if e != nil || name != "/model" || arg != "local/qwen3:8b" {
		t.Fatal(name, arg, e)
	}
	if _, _, e = Parse("/unknown"); e == nil {
		t.Fatal("unknown accepted")
	}
	if _, _, e = Parse("/exit extra"); e == nil {
		t.Fatal("unexpected argument accepted")
	}
	if _, _, e = Parse("/init extra"); e == nil {
		t.Fatal("unexpected argument accepted")
	}
	if len(Complete("/mod")) != 1 {
		t.Fatal("completion failed")
	}
}

func TestParseLanguageLocalizesValidation(t *testing.T) {
	if _, _, err := ParseLanguage("/unknown", "pt-BR"); err == nil || !strings.Contains(err.Error(), "comando desconhecido") {
		t.Fatalf("expected localized unknown-command error, got %v", err)
	}
	if _, _, err := ParseLanguage("/help extra", "es"); err == nil || !strings.Contains(err.Error(), "uso") {
		t.Fatalf("expected localized usage error, got %v", err)
	}
	if name, arg, err := ParseLanguage("/language pt-BR", "en"); err != nil || name != "/language" || arg != "pt-BR" {
		t.Fatalf("language command parse = %q %q %v", name, arg, err)
	}
}
