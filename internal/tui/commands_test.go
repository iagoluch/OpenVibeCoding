package tui

import "testing"

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
	if len(Complete("/mod")) != 1 {
		t.Fatal("completion failed")
	}
}
