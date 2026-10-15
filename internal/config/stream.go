package config

import (
	"os"
	"strings"
	"unicode/utf8"
)

// StreamRedactor retains enough trailing bytes that a credential split across
// arbitrary provider frames is never partially emitted before recognition.
type StreamRedactor struct {
	config  Config
	pending string
	hold    int
	emit    func(string)
}

func (c Config) Stream(emit func(string)) *StreamRedactor {
	hold := 0
	for _, p := range c.Providers {
		k := p.Secret
		if k == "" {
			k = os.Getenv(p.KeyEnv)
		}
		hold = max(hold, len(k))
	}
	return &StreamRedactor{config: c, hold: hold, emit: emit}
}
func (s *StreamRedactor) Write(text string) {
	s.pending += text
	for _, p := range s.config.Providers {
		k := p.Secret
		if k == "" {
			k = os.Getenv(p.KeyEnv)
		}
		if k != "" {
			s.pending = strings.ReplaceAll(s.pending, k, "[REDACTED]")
		}
	}
	n := len(s.pending) - s.hold
	if n <= 0 {
		return
	}
	for n > 0 && n < len(s.pending) && !utf8.RuneStart(s.pending[n]) {
		n--
	}
	if n > 0 {
		s.emit(s.config.Redact(s.pending[:n]))
		s.pending = s.pending[n:]
	}
}
func (s *StreamRedactor) Close() {
	if s.pending != "" {
		s.emit(s.config.Redact(s.pending))
		s.pending = ""
	}
}
