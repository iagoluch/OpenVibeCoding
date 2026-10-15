package tui

import (
	"fmt"
	"strings"
)

var commands = []string{"/changes", "/capabilities", "/help", "/model", "/provider", "/harness", "/sessions", "/resume", "/new", "/clear", "/context", "/cost", "/permissions", "/config", "/status", "/rename", "/delete", "/exit"}

func Parse(s string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(s), " ", 2)
	name := parts[0]
	arg := ""
	if len(parts) == 2 {
		arg = strings.TrimSpace(parts[1])
	}
	for _, c := range commands {
		if name == c {
			switch name {
			case "/help", "/sessions", "/new", "/clear", "/context", "/cost", "/config", "/status", "/exit":
				if arg != "" {
					return "", "", fmt.Errorf("usage: %s (no arguments)", name)
				}
			}
			return name, arg, nil
		}
	}
	return "", "", fmt.Errorf("unknown command %q; use /help", name)
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
