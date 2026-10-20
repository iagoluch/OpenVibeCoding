// Package desktop is the Windows desktop shell of OpenVibeCoding: a Wails v3
// window that loads the daemon's own web UI through a confined reverse proxy,
// plus a system tray and one native folder-picker binding.
//
// Every real file in this package is behind the `desktop` build tag so a plain
// build neither links nor names the Wails dependency. Invariant INV2: building
// without the tag must keep the module's dependency graph free of wails,
// which `go list -deps ./cmd/ovc` asserts in CI.
package desktop
