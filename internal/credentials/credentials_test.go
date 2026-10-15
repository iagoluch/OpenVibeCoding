package credentials

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExplicitFileFallback(t *testing.T) {
	dir := t.TempDir()
	if e := Set(dir, "synthetic-provider", "synthetic-secret", true); e != nil {
		t.Fatal(e)
	}
	if s := Get(dir, "synthetic-provider"); s != "synthetic-secret" {
		t.Fatal("credential not recovered")
	}
	info, e := os.Stat(filepath.Join(dir, "credentials.json"))
	if e != nil {
		t.Fatal(e)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal("unsafe mode")
	}
	if e = Delete(dir, "synthetic-provider", true); e != nil {
		t.Fatal(e)
	}
	if Get(dir, "synthetic-provider") != "" {
		t.Fatal("credential not deleted")
	}
}

func TestRealOSKeychain(t *testing.T) {
	if os.Getenv("INTERPRETER_TEST_KEYCHAIN") != "1" {
		t.Skip("requires explicit isolated OS keychain fixture")
	}
	dir := t.TempDir()
	name := "interpreter-synthetic-keychain-test"
	if e := Set(dir, name, "synthetic-fixture-value", false); e != nil {
		t.Fatal(e)
	}
	defer Delete(dir, name, false)
	if Get(dir, name) != "synthetic-fixture-value" {
		t.Fatal("OS keychain did not recover credential")
	}
	if _, e := os.Stat(filepath.Join(dir, "credentials.json")); !os.IsNotExist(e) {
		t.Fatal("OS keychain operation wrote fallback file")
	}
}
