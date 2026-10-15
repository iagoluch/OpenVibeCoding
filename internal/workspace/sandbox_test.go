package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSandboxBoundary(t *testing.T) {
	if os.Getenv("INTERPRETER_TEST_SANDBOX") != "1" {
		t.Skip("set INTERPRETER_TEST_SANDBOX=1 on bubblewrap-capable Linux")
	}
	w := setup(t)
	secret := filepath.Join(t.TempDir(), "outside")
	os.WriteFile(secret, []byte("private"), 0600)
	os.WriteFile(filepath.Join(w.Path, ".env"), []byte("local-secret"), 0600)
	result, e := w.Shell(context.Background(), "test ! -e '"+secret+"' && test ! -s .env && printf isolated > result.txt && cat result.txt")
	if e != nil || !strings.Contains(result, "isolated") {
		t.Fatalf("%v %s", e, result)
	}
	if b, e := w.Read("result.txt"); e != nil || b != "isolated" {
		t.Fatal("workspace binding unavailable", e)
	}
}
func TestIgnoreAndMoveDelete(t *testing.T) {
	w := setup(t)
	os.WriteFile(filepath.Join(w.Path, ".gitignore"), []byte("ignored/\n*.secret\n"), 0600)
	os.Mkdir(filepath.Join(w.Path, "ignored"), 0700)
	os.WriteFile(filepath.Join(w.Path, "ignored", "x"), []byte("hidden"), 0600)
	if _, e := w.Read("ignored/x"); e == nil {
		t.Fatal("ignored file readable")
	}
	c, _ := w.Propose("nested/new.txt", "hello")
	if e := w.Apply(c); e != nil {
		t.Fatal(e)
	}
	c, _ = w.Propose("nested/new.txt", "")
	if e := w.Move(c, "other/moved.txt"); e != nil {
		t.Fatal(e)
	}
	c, _ = w.Propose("other/moved.txt", "")
	if e := w.Delete(c); e != nil {
		t.Fatal(e)
	}
}
