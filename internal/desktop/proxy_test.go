//go:build desktop

package desktop

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
	daemon "github.com/iagoluch/OpenVibeCoding/internal/runtime"
)

// TestDesktopShellLifecycleHappyPath drives the proxy in front of a real
// daemon: the page loads with the wails tags injected, no token material
// ever appears in a response, /pair and /stop never pass through, and one
// unauthenticated API round-trip succeeds because the proxy injects the
// credentials server-side.
func TestDesktopShellLifecycleHappyPath(t *testing.T) {
	_, client, _ := startDaemon(t, config.Default())
	p := newProxy(nil, "")
	if e := p.retarget(client.Meta); e != nil {
		t.Fatal(e)
	}
	front := httptest.NewServer(p)
	defer front.Close()

	// The page loads, with the wails runtime and the bridge alias injected
	// before </head> (inline scripts are blocked by the daemon CSP).
	resp, err := http.Get(front.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page: HTTP %d", resp.StatusCode)
	}
	if !strings.Contains(string(page), wailsRuntimeTag) || !strings.Contains(string(page), aliasTag) {
		t.Fatal("wails runtime / alias tags not injected into the page")
	}
	if resp.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("daemon CSP must reach the webview untouched")
	}
	if strings.Contains(string(page), client.Meta.Token) {
		t.Fatal("token leaked into the page")
	}

	// Assets stay token-free too.
	resp, err = http.Get(front.URL + "/assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	asset, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("asset: HTTP %d", resp.StatusCode)
	}
	if strings.Contains(string(asset), client.Meta.Token) {
		t.Fatal("token leaked into an asset")
	}

	// /pair is blocked in both methods: a compromised page must not mint a
	// pairing code through the proxy.
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req, _ := http.NewRequest(method, front.URL+"/pair", strings.NewReader("{}"))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s /pair must be a proxy 404, got %d", method, resp.StatusCode)
		}
	}
	// /stop is a tray action, never a page action.
	req, _ := http.NewRequest(http.MethodPost, front.URL+"/stop", strings.NewReader("{}"))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("POST /stop must be a proxy 404, got %d", resp.StatusCode)
	}

	// One API round-trip with NO credentials on the request: the proxy
	// attaches the bearer token and its own client identity.
	session := storeNew(t, front.URL, t.TempDir())
	if session == "" {
		t.Fatal("no session created")
	}
	// The onboarding endpoint is allowlisted too.
	info, err := http.Post(front.URL+"/ui/info", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(info.Body)
	info.Body.Close()
	if info.StatusCode != http.StatusOK {
		t.Fatalf("/ui/info: HTTP %d %s", info.StatusCode, body)
	}
}

// storeNew posts an unauthenticated "new" through the proxy, the way the page
// would, and returns the session id.
func storeNew(t *testing.T, baseURL, workspace string) string {
	t.Helper()
	body := `{"Op":"new","Workspace":` + jsonString(workspace) + `}`
	resp, err := http.Post(baseURL+"/store", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("store new: HTTP %d %s", resp.StatusCode, raw)
	}
	var out struct {
		Value string
		Error string
	}
	if json.NewDecoder(resp.Body).Decode(&out) != nil || out.Error != "" {
		t.Fatalf("store new failed: %+v", out)
	}
	return out.Value
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// TestDesktopAliasModule: the shell serves exactly one local file, the
// bridge alias. The wails v3 beta.30 runtime is a pure ES module that never
// assigns window.wails, so the alias must IMPORT its Call interface from the
// same-origin runtime URL, INSTALL it on window.wails (merge, never clobber)
// and only then map the page's contract name to the real FQN. The Go suite
// pins the module's structure; the behavioral proof (fake runtime module in
// Node) and the real-window end-to-end live in the desktop smoke harness.
func TestDesktopAliasModule(t *testing.T) {
	p := newProxy(nil, "")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, aliasPath, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("alias module: %d", rec.Code)
	}
	body := rec.Body.String()
	// Install: a module import of the runtime's Call, then window.wails.
	if !strings.Contains(body, `import { Call as runtimeCall } from "/wails/runtime.js"`) {
		t.Fatal("alias must import the runtime Call interface from the same-origin runtime URL")
	}
	if !strings.Contains(body, "window.wails = wrapped") {
		t.Fatal("alias must install the Call interface on window.wails")
	}
	// The install must be a merge: pre-existing members are kept.
	if !strings.Contains(body, "wails.Call[had] = wails.Call[had]") && !strings.Contains(body, "call[had] = wails.Call[had]") {
		t.Fatal("alias must copy pre-existing Call members instead of clobbering them")
	}
	// Rename: exactly the page's contract name maps to the bound FQN.
	if !strings.Contains(body, `"desktop.UI.PickFolder"`) || !strings.Contains(body, bindingFqn()) {
		t.Fatal("alias module does not map the contract name to the bound FQN")
	}
	// The old no-op bail-out (which required window.wails to pre-exist) must
	// be gone: that guard is the shipped defect, not a fallback.
	if strings.Contains(body, "return;") {
		t.Fatal("alias must not bail out when window.wails is absent")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
		t.Fatalf("alias module content type: %q", ct)
	}
	// The alias path is GET-only.
	rec = httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, aliasPath, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST %s must be 404, got %d", aliasPath, rec.Code)
	}
}

// capturedRequest is what the synthetic daemon saw.
type capturedRequest struct {
	Method, Host, Path string
	Headers            http.Header
	Body               []byte
}

type captureBackend struct {
	mu   sync.Mutex
	last capturedRequest
}

func (c *captureBackend) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.last = capturedRequest{Method: r.Method, Host: r.Host, Path: r.URL.Path, Headers: r.Header.Clone(), Body: body}
		c.mu.Unlock()
		w.Header().Set("Set-Cookie", "ovc_ui=leaked")
		w.Write([]byte("ok"))
	})
}

func (c *captureBackend) get() capturedRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.last
}

// TestDesktopProxyHeaderPolicy proves the trust boundary with a capturing
// daemon: page-spoofed identity and credentials are dropped and replaced, the
// outbound Host is the daemon's, Origin and cookies never cross, responses
// carry no cookies and no CORS headers, the 2 MiB cap is enforced before the
// daemon, and foreign hosts and fetches are refused.
func TestDesktopProxyHeaderPolicy(t *testing.T) {
	backend := &captureBackend{}
	target := httptest.NewServer(backend.handler())
	defer target.Close()
	p := newProxy(nil, "")
	if e := p.retarget(daemon.Metadata{URL: target.URL, Token: "policy-test-token", Version: 1, PID: 1}); e != nil {
		t.Fatal(e)
	}
	front := httptest.NewServer(p)
	defer front.Close()
	// INV5 origin discipline is active: only this host may address the proxy.
	p.ownHost = strings.TrimPrefix(front.URL, "http://")

	// A page-spoofed request: its identity, credentials, cookie and Origin
	// must all be overridden or removed by the proxy.
	req, _ := http.NewRequest(http.MethodPost, front.URL+"/store", strings.NewReader(`{"Op":"list"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer page-leaked-token")
	req.Header.Set("X-OpenVibeCoding-Client", "page-spoofed-id")
	req.Header.Set("Cookie", "ovc_ui=page-cookie")
	req.Header.Set("Origin", "http://wails.localhost")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	headers := resp.Header.Clone()
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("round-trip failed: %d", resp.StatusCode)
	}
	seen := backend.get()
	if got := seen.Headers.Get("Authorization"); got != "Bearer policy-test-token" {
		t.Fatalf("authorization must be the shell's, got %q", got)
	}
	if got := seen.Headers.Get("X-OpenVibeCoding-Client"); got != p.clientID {
		t.Fatalf("client identity must be the shell's, got %q", got)
	}
	if seen.Headers.Get("Cookie") != "" {
		t.Fatal("page cookie crossed the proxy")
	}
	if seen.Headers.Get("Origin") != "" {
		t.Fatal("Origin crossed the proxy")
	}
	targetHost := strings.TrimPrefix(target.URL, "http://")
	if seen.Host != targetHost {
		t.Fatalf("outbound Host must be %q (daemon 403s other hosts), got %q", targetHost, seen.Host)
	}
	if len(headers.Values("Set-Cookie")) != 0 {
		t.Fatal("Set-Cookie crossed the proxy")
	}
	for name := range headers {
		if strings.HasPrefix(name, "Access-Control-") {
			t.Fatalf("proxy must never emit CORS headers, found %s", name)
		}
	}

	// The mirrored body cap is enforced before the daemon.
	oversized := bytes.Repeat([]byte("x"), maxBodyBytes+1)
	resp, err = http.Post(front.URL+"/store", "application/json", bytes.NewReader(oversized))
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body must be 413, got %d", resp.StatusCode)
	}

	// Foreign fetch metadata is refused before the daemon.
	for _, site := range []string{"cross-site", "same-site"} {
		req, _ := http.NewRequest(http.MethodPost, front.URL+"/store", strings.NewReader(`{}`))
		req.Header.Set("Sec-Fetch-Site", site)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Sec-Fetch-Site %s must be refused, got %d", site, resp.StatusCode)
		}
	}

	// A request addressing another host never reaches the daemon.
	req, _ = http.NewRequest(http.MethodPost, front.URL+"/store", strings.NewReader(`{}`))
	req.Host = "evil.example"
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("foreign Host must be refused, got %d", resp.StatusCode)
	}

	// Tunneling and absolute-form targets are refused outright.
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodConnect, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("CONNECT must be refused, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://evil.example/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("absolute-form must be refused, got %d", rec.Code)
	}
}

// TestDesktopBodyCanonicalization covers the wails beta.30 Windows quirk:
// POST bodies delivered with a Content-Length of 0 while the body bytes are
// intact. The proxy buffers the body and repairs the length before
// forwarding; body bytes are untouched.
func TestDesktopBodyCanonicalization(t *testing.T) {
	backend := &captureBackend{}
	target := httptest.NewServer(backend.handler())
	defer target.Close()
	p := newProxy(nil, "")
	if e := p.retarget(daemon.Metadata{URL: target.URL, Token: "canonical-test", Version: 1, PID: 1}); e != nil {
		t.Fatal(e)
	}

	body := `{"Op":"new","Workspace":"C:\\scratch"}`
	req := httptest.NewRequest(http.MethodPost, "/store", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// The quirk: header and field disagree with the real bytes.
	req.Header.Set("Content-Length", "0")
	req.ContentLength = 0
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("canonicalized request refused: %d %s", rec.Code, rec.Body.String())
	}
	seen := backend.get()
	if string(seen.Body) != body {
		t.Fatalf("body bytes changed: %q", seen.Body)
	}
	if got := seen.Headers.Get("Content-Length"); got != strconv.Itoa(len(body)) {
		t.Fatalf("repaired Content-Length must be %d, got %q", len(body), got)
	}
}
