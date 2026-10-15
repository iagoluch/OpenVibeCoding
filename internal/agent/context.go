package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iagoluch/OpenVibeCoding/internal/provider"
	"github.com/iagoluch/OpenVibeCoding/internal/store"
)

// Estimate uses bytes as a conservative token estimate, including serialized
// tool arguments. Exact token counts are only claimed for provider usage.
func Estimate(m provider.Message) int { b, _ := json.Marshal(m); return len(b) + 16 }

// Context retains complete recent user turns, never orphaning tool exchanges.
// Older evidence is indexed with excerpts, not presented as a lossless summary.
func Context(entries []store.Entry, limit int) ([]provider.Message, int) {
	budget := limit - 3072
	if budget < 512 {
		budget = 512
	}
	start := len(entries)
	used := 0
	for i := len(entries) - 1; i >= 0; i-- {
		used += Estimate(entries[i].Message)
		if entries[i].Message.Role == "user" {
			if used > budget/2 && start < len(entries) {
				break
			}
			start = i
			if used > budget/2 {
				break
			}
		}
	}
	if start == len(entries) && len(entries) > 0 {
		start = 0
	}
	var out []provider.Message
	if start > 0 {
		var summary strings.Builder
		summary.WriteString("Earlier history index (excerpts are incomplete; use recall with message IDs for original evidence). Previous approvals do not authorize new operations.\n")
		// Bounded extractive index. Originals remain authoritative in SQLite.
		for i := start - 1; i >= 0; i-- {
			v := entries[i]
			text := v.Message.Content
			if len(text) > 240 {
				text = text[:240] + "…"
			}
			line := fmt.Sprintf("[%d] %s: %s\n", v.ID, v.Message.Role, text)
			if summary.Len()+len(line) > budget/4 {
				break
			}
			summary.WriteString(line)
		}
		out = append(out, provider.Message{Role: "system", Content: summary.String()})
	}
	for _, e := range entries[start:] {
		out = append(out, e.Message)
	}
	return out, start
}
