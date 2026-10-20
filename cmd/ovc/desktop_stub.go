//go:build !desktop

package main

import (
	"errors"
	"flag"
)

var desktopFlag = flag.Bool("desktop", false, "open the desktop window (third client of the shared runtime)")

// runDesktop refuses the desktop shell in a plain build: the desktop code is
// compiled only with the `desktop` build tag so ordinary binaries do not
// carry the Wails dependency.
func runDesktop(dir string) error {
	return errors.New("--desktop requires a desktop build (build with -tags desktop)")
}
