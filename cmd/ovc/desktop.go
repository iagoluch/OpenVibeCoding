//go:build desktop

package main

import (
	"context"
	"flag"
	"os/signal"

	"github.com/iagoluch/OpenVibeCoding/internal/desktop"
)

var desktopFlag = flag.Bool("desktop", false, "open the desktop window (third client of the shared runtime)")

// runDesktop opens the desktop shell against the runtime of the configured
// home directory. The runtime is spawned from this executable when absent
// and keeps running after the window closes.
func runDesktop(dir string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), terminationSignals()...)
	defer cancel()
	return desktop.Run(ctx, dir)
}
