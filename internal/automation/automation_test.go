package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPermissionBoundaries(t *testing.T) {
	if _, e := Browser(context.Background(), Settings{}, `{}`); e == nil {
		t.Fatal("browser enabled by default")
	}
	if _, e := Desktop(context.Background(), Settings{}, Action{Action: "click"}); e == nil {
		t.Fatal("desktop enabled by default")
	}
	if _, e := Desktop(context.Background(), Settings{Desktop: true}, Action{Action: "clipboard_read"}); e == nil {
		t.Fatal("clipboard grant bypass")
	}
	if _, e := Desktop(context.Background(), Settings{Desktop: true}, Action{Action: "launch", Application: "arbitrary"}); e == nil {
		t.Fatal("application allowlist bypass")
	}
}
func TestRealBrowserWorkflow(t *testing.T) {
	module := os.Getenv("INTERPRETER_TEST_PLAYWRIGHT")
	if module == "" {
		t.Skip("set INTERPRETER_TEST_PLAYWRIGHT to installed Playwright module")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><title>QA fixture</title><input id="name"><button onclick="document.querySelector('#result').textContent='Hello '+document.querySelector('#name').value">Go</button><p id="result"></p></html>`)
	}))
	defer site.Close()
	workflow, _ := json.Marshal(map[string]any{"origins": []string{site.URL}, "steps": []any{map[string]string{"action": "open", "url": site.URL}, map[string]string{"action": "fill", "selector": "#name", "text": "Synthetic"}, map[string]string{"action": "click", "selector": "button"}, map[string]string{"action": "expect_text", "selector": "#result", "text": "Hello Synthetic"}, map[string]string{"action": "screenshot"}}})
	result, e := Browser(context.Background(), Settings{Browser: true, BrowserModule: module}, string(workflow))
	if e != nil {
		t.Fatalf("%v %s", e, result)
	}
	var v struct {
		OK      bool
		Text    string
		Results []map[string]any
	}
	if json.Unmarshal([]byte(result), &v) != nil || !v.OK || !strings.Contains(v.Text, "Hello Synthetic") || v.Results[4]["image"] == nil {
		t.Fatal("browser evidence missing")
	}
	bad := fmt.Sprintf(`{"origins":[%q],"steps":[{"action":"open","url":"https://example.com"}]}`, site.URL)
	if _, e = Browser(context.Background(), Settings{Browser: true, BrowserModule: module}, bad); e == nil {
		t.Fatal("origin policy bypass")
	}
}
func TestRealDesktop(t *testing.T) {
	if os.Getenv("INTERPRETER_TEST_DESKTOP") != "1" || runtime.GOOS != "linux" {
		t.Skip("requires explicit isolated X11 test desktop")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s := Settings{Desktop: true}
	for _, a := range []Action{{Action: "move", X: 40, Y: 60}, {Action: "click", X: 40, Y: 60}, {Action: "scroll", Amount: 1}} {
		if _, e := Desktop(ctx, s, a); e != nil {
			t.Fatal(e)
		}
	}
	out, e := exec.Command("xdotool", "getmouselocation", "--shell").Output()
	if e != nil || !strings.Contains(string(out), "X=40") || !strings.Contains(string(out), "Y=60") {
		t.Fatal("mouse position not observed", e)
	}
	screen, e := Desktop(ctx, s, Action{Action: "screenshot"})
	if e != nil || !strings.Contains(screen, `"image"`) {
		t.Fatal("screenshot failed", e)
	}
	target := filepath.Join(t.TempDir(), "typed.txt")
	terminal := exec.CommandContext(ctx, "xterm", "-title", "Interpreter synthetic keyboard fixture", "-e", "sh", "-c", `read -r text; printf '%s' "$text" > "$1"; sleep 1`, "fixture", target)
	if err := terminal.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { terminal.Process.Kill(); terminal.Wait() }()
	var id string
	for i := 0; i < 40; i++ {
		b, _ := exec.Command("xdotool", "search", "--name", "Interpreter synthetic keyboard fixture").Output()
		if fields := strings.Fields(string(b)); len(fields) > 0 {
			id = fields[0]
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if id == "" {
		t.Fatal("fixture window unavailable")
	}
	if _, e := run(ctx, "xdotool", "windowfocus", "--sync", id); e != nil {
		t.Fatal(e)
	}
	if _, e := Desktop(ctx, s, Action{Action: "type", Text: "synthetic keyboard evidence"}); e != nil {
		t.Fatal(e)
	}
	if _, e := Desktop(ctx, s, Action{Action: "key", Key: "Return"}); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 40; i++ {
		b, _ := os.ReadFile(target)
		if string(b) == "synthetic keyboard evidence" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("keyboard input not observed in application")

}
