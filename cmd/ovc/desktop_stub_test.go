//go:build !desktop

package main

import (
	"flag"
	"strings"
	"testing"
)

// TestDesktopStubRefusesWithoutTag runs in the normal (untagged) suite: a
// plain build must accept the flag, refuse the shell with the build
// instruction, and never link the desktop code.
func TestDesktopStubRefusesWithoutTag(t *testing.T) {
	if flag.Lookup("desktop") == nil {
		t.Fatal("--desktop flag missing in a plain build")
	}
	err := runDesktop(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "--desktop requires a desktop build") {
		t.Fatalf("stub must refuse the desktop shell, got: %v", err)
	}
}
