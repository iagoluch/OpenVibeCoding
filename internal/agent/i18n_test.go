package agent

import (
	"strings"
	"testing"
)

func TestLanguageInstruction(t *testing.T) {
	cases := []struct {
		language string
		want     string
	}{
		{"pt-BR", "Brazilian Portuguese"},
		{"es", "Spanish"},
		{"zh-CN", "简体中文"},
		{"en", "English"},
	}
	for _, tc := range cases {
		if got := languageInstruction(tc.language); !strings.Contains(got, tc.want) {
			t.Errorf("languageInstruction(%q) does not mention %q", tc.language, tc.want)
		}
	}
}

func TestAutoLanguageInstructionResolvesEnvironment(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "pt_BR.UTF-8")
	if got := languageInstruction("auto"); !strings.Contains(got, "Brazilian Portuguese") {
		t.Fatalf("auto preference did not follow the system locale: %q", got)
	}
}
