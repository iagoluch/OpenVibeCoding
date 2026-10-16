package tui

import (
	"fmt"
	"strings"

	"github.com/iagoluch/OpenVibeCoding/internal/i18n"
)

var commands = []string{"/changes", "/capabilities", "/help", "/model", "/provider", "/harness", "/sessions", "/resume", "/new", "/clear", "/context", "/cost", "/permissions", "/config", "/status", "/rename", "/delete", "/init", "/undo", "/doctor", "/exit"}

func Parse(s string) (string, string, error) {
	return ParseLanguage(s, "en")
}

// ParseLanguage parses stable command identifiers while localizing user-facing
// validation messages. Slash commands and their arguments stay language-neutral.
func ParseLanguage(s, language string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(s), " ", 2)
	name := parts[0]
	arg := ""
	if len(parts) == 2 {
		arg = strings.TrimSpace(parts[1])
	}
	for _, c := range commands {
		if name == c {
			switch name {
			case "/help", "/sessions", "/new", "/clear", "/context", "/cost", "/config", "/status", "/init", "/undo", "/doctor", "/exit":
				if arg != "" {
					return "", "", fmt.Errorf("%s", fmt.Sprintf(i18n.T(language, "command.usage"), name))
				}
			}
			return name, arg, nil
		}
	}
	return "", "", fmt.Errorf("%s", fmt.Sprintf(i18n.T(language, "command.unknown"), name))
}
func Complete(s string) []string {
	var out []string
	for _, c := range commands {
		if strings.HasPrefix(c, s) {
			out = append(out, c)
		}
	}
	return out
}
