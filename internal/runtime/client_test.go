package runtime

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestConnectReportsStaleMetadataAsNotRunning covers the daemon that died
// before removing runtime.json: the metadata still validates, but nothing
// answers on its address, and callers must give the user the answer they get
// before the first start instead of a raw connection error.
func TestConnectReportsStaleMetadataAsNotRunning(t *testing.T) {
	dir := t.TempDir()
	listener, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	addr := listener.Addr().String()
	listener.Close()
	if conn, e := net.DialTimeout("tcp", addr, time.Second); e == nil {
		conn.Close()
		t.Skip("the address was taken between closing the listener and the probe")
	}
	b, e := json.Marshal(Metadata{URL: "http://" + addr, Token: strings.Repeat("a", 64), Version: Protocol, PID: 1})
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "runtime.json"), b, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Discover(dir); e != nil {
		t.Fatalf("stale metadata must still parse: %v", e)
	}
	if _, e = Connect(dir); !errors.Is(e, ErrNotRunning) {
		t.Fatalf("stale metadata must read as not running, got %v", e)
	}
}
