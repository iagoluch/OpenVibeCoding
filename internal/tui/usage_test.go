package tui

import (
	"strings"
	"testing"

	"github.com/iagoluch/OpenVibeCoding/internal/agent"
	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
)

func TestUsageLabelReflectsConfiguredPrices(t *testing.T) {
	m := &Model{Engine: &agent.Engine{Config: config.Default()}, status: "idle"}
	// Pin a paid pair explicitly: the shipped default is free (decision
	// 2026-10-19), and this test is about the price math, not the default.
	m.Engine.Config.Provider, m.Engine.Config.Model = "openai", "gpt-4.1-mini"
	if label := m.usageLabel(); label != "" {
		t.Fatal("expected no label without usage", label)
	}
	m.usage = provider.Usage{Input: 1500, Output: 200}
	label := m.usageLabel()
	if !strings.Contains(label, "↑1.5k") || !strings.Contains(label, "↓200") {
		t.Fatal(label)
	}
	if !strings.Contains(label, "~$0.0009") {
		t.Fatal("expected estimated cost for gpt-4.1-mini", label)
	}
	m.Engine.Config.Provider, m.Engine.Config.Model = "openrouter", "thinkingmachines/inkling:free"
	if label = m.usageLabel(); !strings.Contains(label, "free") {
		t.Fatal("configured zero price must read as free", label)
	}
	p := m.Engine.Config.Providers["lmstudio"]
	p.Models[0].InputPrice, p.Models[0].OutputPrice = nil, nil
	m.Engine.Config.Provider, m.Engine.Config.Model = "lmstudio", "qwen/qwen3-coder-30b"
	if label = m.usageLabel(); !strings.Contains(label, "price unset") {
		t.Fatal("unset prices must stay unknown", label)
	}
}
