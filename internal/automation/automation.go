package automation

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

//go:embed browser.mjs
var browserWorker string

type Settings struct {
	Backend       []string `json:"desktop_backend,omitempty"`
	Browser       bool     `json:"browser"`
	Desktop       bool     `json:"desktop"`
	Clipboard     bool     `json:"clipboard"`
	BrowserModule string   `json:"browser_module,omitempty"`
	Applications  []string `json:"applications,omitempty"`
}
type Action struct {
	Action      string `json:"action"`
	Text        string `json:"text,omitempty"`
	Key         string `json:"key,omitempty"`
	Application string `json:"application,omitempty"`
	Window      string `json:"window,omitempty"`
	X           int    `json:"x,omitempty"`
	Y           int    `json:"y,omitempty"`
	Button      int    `json:"button,omitempty"`
	Amount      int    `json:"amount,omitempty"`
}
type Capability struct {
	Name      string
	Available bool
	Reason    string
}

func Capabilities(s Settings) []Capability {
	module := s.BrowserModule
	_, node := exec.LookPath("node")
	_, pkg := os.Stat(filepath.Join(module, "index.mjs"))
	browser := s.Browser && node == nil && pkg == nil
	desktop := false
	reason := "disabled by configuration"
	if s.Desktop {
		switch runtime.GOOS {
		case "linux":
			_, e := exec.LookPath("xdotool")
			desktop = e == nil && os.Getenv("DISPLAY") != "" && os.Getenv("XDG_SESSION_TYPE") != "wayland"
			reason = "requires X11 DISPLAY and xdotool (Wayland unsupported)"
		case "darwin":
			_, e := exec.LookPath("osascript")
			desktop = e == nil
			reason = "requires Accessibility and Screen Recording grants; actions verify OS permission"
		case "windows":
			_, e := exec.LookPath("powershell.exe")
			desktop = e == nil
			reason = "requires interactive desktop; secure/UAC desktop is inaccessible"
		}
	}
	if s.Desktop && len(s.Backend) > 0 {
		_, err := exec.LookPath(s.Backend[0])
		desktop = err == nil
		reason = "configured external JSON desktop adapter; actions require approval"
	}
	return []Capability{{"browser", browser, "requires enabled browser, Node and installed Playwright Chromium"}, {"desktop", desktop, reason}, {"clipboard", desktop && s.Clipboard, "requires separate clipboard grant and desktop backend"}}
}
func Browser(ctx context.Context, s Settings, workflow string) (string, error) {
	if !s.Browser {
		return "", errors.New("browser permission disabled")
	}
	var req map[string]any
	if json.Unmarshal([]byte(workflow), &req) != nil {
		return "", errors.New("invalid workflow JSON")
	}
	req["module"] = s.BrowserModule
	b, _ := json.Marshal(req)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := command(ctx, "node", "--input-type=module", "-e", browserWorker)
	cmd.Stdin = bytes.NewReader(b)
	cmd.WaitDelay = time.Second
	var out bounded
	cmd.Stdout = &out
	cmd.Stderr = &bounded{}
	err := cmd.Run()
	if err != nil {
		return out.String(), errors.New("browser workflow failed; check result or run --setup-browser")
	}
	return out.String(), nil
}

type bounded struct{ bytes.Buffer }

func (b *bounded) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len()+n > 8<<20 {
		return 0, errors.New("automation output exceeded limit")
	}
	return b.Buffer.Write(p)
}
func run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := command(ctx, name, args...)
	cmd.WaitDelay = time.Second
	var out bounded
	cmd.Stdout = &out
	cmd.Stderr = &bounded{}
	e := cmd.Run()
	if e != nil {
		return "", fmt.Errorf("%s action failed (check OS permissions or capability prerequisites)", name)
	}
	return out.String(), nil
}
func Desktop(ctx context.Context, s Settings, a Action) (string, error) {
	if !s.Desktop {
		return "", errors.New("desktop permission disabled")
	}
	if strings.HasPrefix(a.Action, "clipboard_") && !s.Clipboard {
		return "", errors.New("clipboard permission disabled")
	}
	if len(a.Text) > 10000 || a.X < 0 || a.Y < 0 || a.X > 32767 || a.Y > 32767 || a.Amount > 100 || a.Amount < -100 {
		return "", errors.New("desktop action exceeds limits")
	}
	if a.Action == "launch" || a.Action == "focus" {
		ok := false
		for _, app := range s.Applications {
			if app == a.Application {
				ok = true
			}
		}
		if !ok {
			return "", errors.New("application is not in configured allowlist")
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if len(s.Backend) > 0 {
		b, _ := json.Marshal(a)
		return runInput(ctx, string(b), s.Backend[0], s.Backend[1:]...)
	}
	switch runtime.GOOS {
	case "linux":
		return linuxDesktop(ctx, a)
	case "darwin":
		return macDesktop(ctx, a)
	case "windows":
		return windowsDesktop(ctx, a)
	}
	return "", errors.New("desktop backend unavailable")
}
func linuxDesktop(ctx context.Context, a Action) (string, error) {
	if os.Getenv("DISPLAY") == "" || os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return "", errors.New("X11 desktop unavailable; Wayland input is unsupported")
	}
	switch a.Action {
	case "inspect":
		out, err := run(ctx, "xdotool", "getactivewindow", "getwindowname")
		if err != nil {
			return run(ctx, "xdotool", "getwindowfocus", "getwindowname")
		}
		return out, nil
	case "move":
		return run(ctx, "xdotool", "mousemove", "--sync", strconv.Itoa(a.X), strconv.Itoa(a.Y))
	case "click":
		button := a.Button
		if button == 0 {
			button = 1
		}
		if button < 1 || button > 3 {
			return "", errors.New("button must be 1-3")
		}
		return run(ctx, "xdotool", "mousemove", strconv.Itoa(a.X), strconv.Itoa(a.Y), "click", strconv.Itoa(button))
	case "scroll":
		button := "5"
		n := a.Amount
		if n < 0 {
			button = "4"
			n = -n
		}
		return run(ctx, "xdotool", "click", "--repeat", strconv.Itoa(n), "--delay", "50", button)
	case "type":
		return run(ctx, "xdotool", "type", "--clearmodifiers", "--", a.Text)
	case "key":
		if strings.HasPrefix(a.Key, "-") || strings.ContainsAny(a.Key, " \n") {
			return "", errors.New("invalid key chord")
		}
		return run(ctx, "xdotool", "key", "--clearmodifiers", a.Key)
	case "focus":
		ids, e := run(ctx, "xdotool", "search", "--onlyvisible", "--class", a.Application)
		if e != nil {
			return "", e
		}
		fields := strings.Fields(ids)
		if len(fields) == 0 {
			return "", errors.New("application window not found")
		}
		out, err := run(ctx, "xdotool", "windowactivate", "--sync", fields[0])
		if err != nil {
			return run(ctx, "xdotool", "windowfocus", "--sync", fields[0])
		}
		return out, nil
	case "close":
		if _, e := strconv.ParseUint(a.Window, 10, 64); e != nil {
			return "", errors.New("numeric window ID required")
		}
		return run(ctx, "xdotool", "windowclose", a.Window)
	case "launch":
		cmd := exec.Command(a.Application)
		cmd.Env = os.Environ()
		if e := cmd.Start(); e != nil {
			return "", e
		}
		go cmd.Wait()
		return "Application launch requested; verify with inspect/screenshot", nil
	case "screenshot":
		return screenshot(ctx, "import", "-window", "root")
	case "clipboard_read":
		return run(ctx, "xclip", "-selection", "clipboard", "-o")
	case "clipboard_write":
		cmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(a.Text)
		if e := cmd.Run(); e != nil {
			return "", errors.New("clipboard write failed")
		}
		return "Clipboard updated", nil
	}
	return "", errors.New("unsupported desktop action")
}
func macDesktop(ctx context.Context, a Action) (string, error) {
	script := ""
	var args []string
	switch a.Action {
	case "inspect":
		script = `tell application "System Events" to get name of first application process whose frontmost is true`
	case "launch", "focus":
		script = `on run argv
 tell application (item 1 of argv) to activate
 end run`
		args = []string{a.Application}
	case "type":
		script = `on run argv
 tell application "System Events" to keystroke (item 1 of argv)
 end run`
		args = []string{a.Text}
	case "key":
		n, e := strconv.Atoi(a.Key)
		if e != nil || n < 0 || n > 127 {
			return "", errors.New("macOS key expects virtual key code 0-127")
		}
		script = fmt.Sprintf(`tell application "System Events" to key code %d`, n)
	case "close":
		script = `tell application "System Events" to keystroke "w" using command down`
	case "clipboard_read":
		script = `get the clipboard as text`
	case "clipboard_write":
		script = `on run argv
 set the clipboard to (item 1 of argv)
 end run`
		args = []string{a.Text}
	case "screenshot":
		return screenshot(ctx, "screencapture", "-x")
	case "move", "click", "scroll":
		// JavaScript for Automation bridges fixed CoreGraphics functions. User data
		// is numeric and bounded; no generated script can contain model-supplied code.
		code := `ObjC.import('Cocoa'); ObjC.import('ApplicationServices'); if (!$.AXIsProcessTrusted()) { throw Error('Accessibility permission required'); } `
		if a.Action == "scroll" {
			code += fmt.Sprintf(`var e=$.CGEventCreate(null); $.CGEventSetType(e,22); $.CGEventSetIntegerValueField(e,11,%d); $.CGEventPost(0,e);`, a.Amount)
		} else {
			code += fmt.Sprintf(`var p=$.CGPointMake(%d,%d); var e=$.CGEventCreateMouseEvent(null,5,p,0); $.CGEventPost(0,e);`, a.X, a.Y)
			if a.Action == "click" {
				button := a.Button
				if button == 0 {
					button = 1
				}
				if button < 1 || button > 3 {
					return "", errors.New("button must be 1-3")
				}
				down, up, which := 1, 2, 0
				if button == 2 {
					down, up, which = 25, 26, 2
				}
				if button == 3 {
					down, up, which = 3, 4, 1
				}
				code += fmt.Sprintf(`$.CGEventPost(0,$.CGEventCreateMouseEvent(null,%d,p,%d)); $.CGEventPost(0,$.CGEventCreateMouseEvent(null,%d,p,%d));`, down, which, up, which)
			}
		}
		return run(ctx, "osascript", "-l", "JavaScript", "-e", code)

	default:
		return "", errors.New("unsupported desktop action")
	}
	return run(ctx, "osascript", append([]string{"-e", script}, args...)...)
}
