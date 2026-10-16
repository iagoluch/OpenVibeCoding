package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/workspace"
)

// browser mirrors what the paired page does: same host, explicit headers,
// redirects returned instead of followed and no implicit cookie jar.
var browser = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func ask(t *testing.T, c *Client, method, path string, headers map[string]string, body string) *http.Response {
	t.Helper()
	req, e := http.NewRequest(method, c.Meta.URL+path, strings.NewReader(body))
	if e != nil {
		t.Fatal(e)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, e := browser.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

// pairBrowser consumes a pairing link the way a browser navigation does.
func pairBrowser(t *testing.T, c *Client, link string) (*http.Response, string) {
	t.Helper()
	path := strings.TrimPrefix(link, c.Meta.URL)
	resp := ask(t, c, "GET", path, map[string]string{"Sec-Fetch-Site": "none"}, "")
	cookie := ""
	for _, c := range resp.Cookies() {
		if c.Name == "ovc_ui" {
			cookie = c.Value
		}
	}
	return resp, cookie
}

func TestRuntimeRejectsForeignHost(t *testing.T) {
	_, c := startFixture(t, config.Default())
	host := strings.TrimPrefix(c.Meta.URL, "http://")
	for _, foreign := range []string{"rebind.example", "localhost:" + strings.Split(host, ":")[1], "127.0.0.1"} {
		req, e := http.NewRequest("POST", c.Meta.URL+"/health", strings.NewReader("{}"))
		if e != nil {
			t.Fatal(e)
		}
		req.Host = foreign
		req.Header.Set("Authorization", "Bearer "+c.Meta.Token)
		req.Header.Set("X-OpenVibeCoding-Client", "test")
		resp, e := browser.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		resp.Body.Close()
		if resp.StatusCode != 403 {
			t.Fatalf("host %q accepted with %d", foreign, resp.StatusCode)
		}
	}
}

func TestRuntimePairingIsSingleUseAndExpires(t *testing.T) {
	_, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(link, c.Meta.URL+"/pair?token=") {
		t.Fatalf("unexpected link %q", link)
	}
	resp, cookie := pairBrowser(t, c, link)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" || cookie == "" {
		t.Fatalf("pairing rejected: %d %q %q", resp.StatusCode, resp.Header.Get("Location"), cookie)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("pairing answer is cacheable: Cache-Control = %q", got)
	}
	if again := ask(t, c, "GET", strings.TrimPrefix(link, c.Meta.URL), nil, ""); again.StatusCode != http.StatusGone {
		t.Fatalf("single-use code reused: %d", again.StatusCode)
	}
	unknown := ask(t, c, "GET", "/pair?token=deadbeef", nil, "")
	if unknown.StatusCode != http.StatusGone {
		t.Fatalf("unknown code accepted: %d", unknown.StatusCode)
	}
	if got := unknown.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("refused pairing answer is cacheable: Cache-Control = %q", got)
	}
	b, _ := io.ReadAll(unknown.Body)
	if !strings.Contains(string(b), "ovc --ui") {
		t.Fatalf("pairing failure page is not actionable: %q", string(b))
	}
	restore := pairTTL
	pairTTL = -time.Second
	defer func() { pairTTL = restore }()
	stale, e := c.Pair(context.Background(), "")
	if e != nil {
		t.Fatal(e)
	}
	if resp, _ := pairBrowser(t, c, stale); resp.StatusCode != http.StatusGone {
		t.Fatalf("expired code accepted: %d", resp.StatusCode)
	}
}

func TestRuntimeCookieNeedsOwnOrigin(t *testing.T) {
	_, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	resp, cookie := pairBrowser(t, c, link)
	if resp.StatusCode != http.StatusSeeOther || cookie == "" {
		t.Fatalf("pairing failed: %d", resp.StatusCode)
	}
	own := map[string]string{"Cookie": "ovc_ui=" + cookie, "X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL, "Sec-Fetch-Site": "same-origin"}
	if got := ask(t, c, "POST", "/ui/info", own, "{}"); got.StatusCode != 200 {
		t.Fatalf("same-origin cookie rejected: %d", got.StatusCode)
	}
	cases := []struct {
		name    string
		headers map[string]string
	}{
		{"cross-origin page", map[string]string{"Cookie": "ovc_ui=" + cookie, "X-OpenVibeCoding-Client": "tab-1", "Origin": "http://evil.example"}},
		{"cross-site fetch metadata", map[string]string{"Cookie": "ovc_ui=" + cookie, "X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL, "Sec-Fetch-Site": "cross-site"}},
		{"same-site fetch metadata", map[string]string{"Cookie": "ovc_ui=" + cookie, "X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL, "Sec-Fetch-Site": "same-site"}},
		{"foreign cookie", map[string]string{"Cookie": "ovc_ui=forged", "X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL}},
		{"no credential at all", map[string]string{"X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL}},
		{"bearer from a foreign page", map[string]string{"Authorization": "Bearer " + c.Meta.Token, "X-OpenVibeCoding-Client": "tab-1", "Origin": "http://evil.example"}},
	}
	for _, tc := range cases {
		if got := ask(t, c, "POST", "/ui/info", tc.headers, "{}"); got.StatusCode != 401 {
			t.Fatalf("%s accepted: %d", tc.name, got.StatusCode)
		}
	}
	// A cookie is a browser credential: it may use the API, never mint links.
	mint := map[string]string{"Cookie": "ovc_ui=" + cookie, "X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL, "Sec-Fetch-Site": "same-origin"}
	if got := ask(t, c, "POST", "/pair", mint, "{}"); got.StatusCode != 401 {
		t.Fatalf("cookie minted a pairing link: %d", got.StatusCode)
	}
	// The bearer path keeps working for same-origin and headless clients.
	bearer := map[string]string{"Authorization": "Bearer " + c.Meta.Token, "X-OpenVibeCoding-Client": "tui", "Origin": c.Meta.URL}
	if got := ask(t, c, "POST", "/ui/info", bearer, "{}"); got.StatusCode != 200 {
		t.Fatalf("bearer with own origin rejected: %d", got.StatusCode)
	}
}

func TestRuntimeUIRoutesAndHeaders(t *testing.T) {
	_, c := startFixture(t, config.Default())
	for _, path := range []string{"/", "/assets/app.js", "/assets/style.css"} {
		resp := ask(t, c, "GET", path, nil, "")
		if resp.StatusCode != 200 {
			t.Fatalf("%s: %d", path, resp.StatusCode)
		}
		for header, want := range map[string]string{
			"Cache-Control":          "no-cache",
			"X-Content-Type-Options": "nosniff",
			"Referrer-Policy":        "no-referrer",
		} {
			if got := resp.Header.Get(header); got != want {
				t.Fatalf("%s: %s = %q", path, header, got)
			}
		}
		if got := resp.Header.Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") || !strings.Contains(got, "frame-ancestors 'none'") {
			t.Fatalf("%s: CSP = %q", path, got)
		}
		if resp.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("%s: CORS header served", path)
		}
	}
	for _, path := range []string{"/assets/missing.js", "/assets/", "/missing"} {
		if got := ask(t, c, "GET", path, nil, ""); got.StatusCode != 404 {
			t.Fatalf("%s: %d", path, got.StatusCode)
		}
	}
	if got := ask(t, c, "GET", "/health", nil, ""); got.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("health served on GET: %d", got.StatusCode)
	}
	bearer := map[string]string{"Authorization": "Bearer " + c.Meta.Token, "X-OpenVibeCoding-Client": "tui"}
	if got := ask(t, c, "POST", "/", bearer, "{}"); got.StatusCode != 404 {
		t.Fatalf("page served on POST: %d", got.StatusCode)
	}
}

func TestRuntimeUIInfoDescribesProvidersAndWorkspaces(t *testing.T) {
	root, e := workspace.Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	ws := root.Path
	_, c := startFixture(t, config.Default())
	if _, e := c.New(ws); e != nil {
		t.Fatal(e)
	}
	if _, e := c.New(ws); e != nil {
		t.Fatal(e)
	}
	info, e := c.UIInfo(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if len(info.Workspaces) != 1 || info.Workspaces[0] != ws {
		t.Fatalf("workspaces not deduplicated: %v", info.Workspaces)
	}
	if info.Configured {
		t.Fatal("a provider without credential or local endpoint must not read as configured")
	}
	if len(info.Providers) == 0 || info.Providers[0].ID != "openrouter" {
		t.Fatalf("provider order is not beginner first: %v", info.Providers)
	}
	active := 0
	for _, p := range info.Providers {
		if p.Active {
			active++
		}
		if p.ID == "local" && !p.Local {
			t.Fatal("local endpoint not marked local")
		}
	}
	if active != 1 {
		t.Fatalf("%d active providers", active)
	}
}

func TestRuntimeUISetupUsesWizardActivation(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "retired-model") {
			w.WriteHeader(503)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer fake.Close()
	cfg := config.Default()
	cfg.Providers["probe"] = config.Provider{BaseURL: fake.URL, Secret: "in-memory", Models: []config.Model{
		{ID: "retired-model", Context: 32768, Tools: true},
		{ID: "healthy-model", Context: 32768, Tools: true},
	}}
	dir, c := startFixture(t, cfg)
	action, e := c.UISetup(context.Background(), "probe", "retired-model", "", false)
	if e != nil || !action.OK || action.Model != "healthy-model" {
		t.Fatalf("setup did not fall back: %+v %v", action, e)
	}
	saved := readSavedConfig(t, dir)
	if saved.Provider != "probe" || saved.Model != "healthy-model" {
		t.Fatalf("not persisted: %s/%s", saved.Provider, saved.Model)
	}
	info, e := c.UIInfo(context.Background())
	if e != nil || info.Active.Provider != "probe" || info.Active.Model != "healthy-model" || !info.Configured {
		t.Fatalf("daemon did not adopt the new configuration: %+v %v", info.Active, e)
	}
}

func TestRuntimeUISetupFailureKeepsConfigurationAndForcesOnRequest(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer down.Close()
	cfg := config.Default()
	cfg.Providers["probe"] = config.Provider{BaseURL: down.URL, Models: []config.Model{{ID: "only-model", Context: 32768, Tools: true}}}
	dir, c := startFixture(t, cfg)
	before := readSavedConfig(t, dir)
	if _, e := c.UISetup(context.Background(), "probe", "", "", false); e == nil {
		t.Fatal("unreachable model accepted")
	}
	after := readSavedConfig(t, dir)
	if before.Provider != after.Provider || before.Model != after.Model {
		t.Fatalf("failed setup changed the active configuration: %s/%s -> %s/%s", before.Provider, before.Model, after.Provider, after.Model)
	}
	action, e := c.UISetup(context.Background(), "probe", "only-model", "", true)
	if e != nil || !action.OK || action.Model != "only-model" {
		t.Fatalf("forced activation failed: %+v %v", action, e)
	}
	if _, e := c.UISetup(context.Background(), "missing", "", "", true); e == nil {
		t.Fatal("unknown provider accepted")
	}
}

func TestRuntimeUISetupKeepsTheLiveRegistryClean(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer down.Close()
	cfg := config.Default()
	cfg.Providers["local"] = config.Provider{BaseURL: down.URL, Models: []config.Model{{ID: "preset-model", Context: 32768, Tools: true}}}
	_, c := startFixture(t, cfg)
	// A model the preset does not know is registered in a copy of the
	// configuration; a failed onboarding must not leave it behind for the
	// next reader of the live registry.
	if _, e := c.UISetup(context.Background(), "local", "ghost-model", "", false); e == nil {
		t.Fatal("unreachable local model accepted")
	}
	info, e := c.UIInfo(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range info.Providers {
		for _, m := range p.Models {
			if m.ID == "ghost-model" {
				t.Fatalf("failed setup leaked %q into the live registry of %s", m.ID, p.ID)
			}
		}
	}
}

func TestDiscoverReportsNotRunning(t *testing.T) {
	if _, e := Discover(t.TempDir()); !errors.Is(e, ErrNotRunning) {
		t.Fatalf("missing metadata must map to ErrNotRunning, got %v", e)
	}
	if _, e := Connect(t.TempDir()); !errors.Is(e, ErrNotRunning) {
		t.Fatalf("connect must report the sentinel, got %v", e)
	}
}

// TestRuntimeCookieAuthAcceptsAnyMatchingCookie covers a page served from
// another loopback port: it can send its own ovc_ui cookie, and whichever one
// the browser presents first must not turn the paired tab into a silent 401.
func TestRuntimeCookieAuthAcceptsAnyMatchingCookie(t *testing.T) {
	_, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	resp, cookie := pairBrowser(t, c, link)
	if resp.StatusCode != http.StatusSeeOther || cookie == "" {
		t.Fatalf("pairing failed: %d", resp.StatusCode)
	}
	headers := map[string]string{"X-OpenVibeCoding-Client": "tab-1", "Origin": c.Meta.URL, "Sec-Fetch-Site": "same-origin"}
	for _, raw := range []string{
		"ovc_ui=forged; ovc_ui=" + cookie,
		"ovc_ui=" + cookie + "; ovc_ui=forged",
		"ovc_ui=forged; ovc_ui=" + cookie + "; ovc_ui=also-forged",
	} {
		headers["Cookie"] = raw
		if got := ask(t, c, "POST", "/ui/info", headers, "{}"); got.StatusCode != 200 {
			t.Fatalf("cookie %q rejected: %d", raw, got.StatusCode)
		}
	}
	headers["Cookie"] = "ovc_ui=forged; ovc_ui=also-forged"
	if got := ask(t, c, "POST", "/ui/info", headers, "{}"); got.StatusCode != 401 {
		t.Fatalf("forged cookies accepted: %d", got.StatusCode)
	}
}

// TestRuntimePairingCookieIsHostOnlyAndStrict pins the attributes the browser
// enforces: a Domain or an expiry here would outlive the bond between one tab
// and this runtime.
func TestRuntimePairingCookieIsHostOnlyAndStrict(t *testing.T) {
	_, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	resp, cookie := pairBrowser(t, c, link)
	if resp.StatusCode != http.StatusSeeOther || cookie == "" {
		t.Fatalf("pairing failed: %d", resp.StatusCode)
	}
	set := ""
	for _, v := range resp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "ovc_ui=") {
			set = v
		}
	}
	if set == "" {
		t.Fatal("pairing did not set the ovc_ui cookie")
	}
	for _, want := range []string{"HttpOnly", "SameSite=Strict", "Path=/"} {
		if !strings.Contains(set, want) {
			t.Fatalf("Set-Cookie %q lacks %s", set, want)
		}
	}
	for _, unwanted := range []string{"Domain=", "Max-Age", "Expires"} {
		if strings.Contains(set, unwanted) {
			t.Fatalf("Set-Cookie %q carries %s", set, unwanted)
		}
	}
}

// TestRuntimeCrossSiteNavigationIsRejectedWithoutBurningTheCode checks the
// refusal happens before the pairing code is spent: a page that tries to steal
// the link must not be able to invalidate it for the user.
func TestRuntimeCrossSiteNavigationIsRejectedWithoutBurningTheCode(t *testing.T) {
	_, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	cross := map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "http://evil.example"}
	for _, target := range []string{"/", strings.TrimPrefix(link, c.Meta.URL)} {
		if got := ask(t, c, "GET", target, cross, ""); got.StatusCode != 401 {
			t.Fatalf("cross-site GET %s accepted: %d", target, got.StatusCode)
		}
	}
	if resp, cookie := pairBrowser(t, c, link); resp.StatusCode != http.StatusSeeOther || cookie == "" {
		t.Fatalf("a refused attempt spent the single-use code: %d", resp.StatusCode)
	}
}

// TestRuntimePairingCodeIsSingleUseUnderConcurrency races the exchange itself:
// two tabs opening the same link at once must yield exactly one cookie.
func TestRuntimePairingCodeIsSingleUseUnderConcurrency(t *testing.T) {
	_, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	path := strings.TrimPrefix(link, c.Meta.URL)
	const attempts = 8
	results := make(chan int, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest("GET", c.Meta.URL+path, nil)
			if err != nil {
				results <- 0
				return
			}
			req.Header.Set("Sec-Fetch-Site", "none")
			resp, err := browser.Do(req)
			if err != nil {
				results <- 0
				return
			}
			resp.Body.Close()
			results <- resp.StatusCode
		}()
	}
	wg.Wait()
	close(results)
	accepted, refused := 0, 0
	for code := range results {
		switch code {
		case http.StatusSeeOther:
			accepted++
		case http.StatusGone:
			refused++
		default:
			t.Fatalf("unexpected pairing status %d", code)
		}
	}
	if accepted != 1 || refused != attempts-1 {
		t.Fatalf("%d accepted and %d refused, want 1 and %d", accepted, refused, attempts-1)
	}
}

// TestRuntimeRejectsRequestsWithoutClientIdentity keeps every authenticated
// call bound to a client: without the identity header an approval cannot be
// attributed to the turn that asked for it.
func TestRuntimeRejectsRequestsWithoutClientIdentity(t *testing.T) {
	_, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	_, cookie := pairBrowser(t, c, link)
	cases := []struct {
		name    string
		headers map[string]string
	}{
		{"cookie", map[string]string{"Cookie": "ovc_ui=" + cookie, "Origin": c.Meta.URL, "Sec-Fetch-Site": "same-origin"}},
		{"bearer", map[string]string{"Authorization": "Bearer " + c.Meta.Token}},
	}
	for _, tc := range cases {
		if got := ask(t, c, "POST", "/ui/info", tc.headers, "{}"); got.StatusCode != 400 {
			t.Fatalf("%s without a client identity accepted: %d", tc.name, got.StatusCode)
		}
	}
}

// TestRuntimeAPICarriesDefenceHeaders checks the API answers with the browser
// defence headers on every path, errors included: no stored response, no
// sniffing, no referrer leaving the daemon.
func TestRuntimeAPICarriesDefenceHeaders(t *testing.T) {
	_, c := startFixture(t, config.Default())
	unauthenticated := map[string]string{"X-OpenVibeCoding-Client": "tui"}
	token := map[string]string{"Authorization": "Bearer " + c.Meta.Token}
	authenticated := map[string]string{"Authorization": "Bearer " + c.Meta.Token, "X-OpenVibeCoding-Client": "tui"}
	cases := []struct {
		name       string
		method     string
		path       string
		headers    map[string]string
		wantStatus int
	}{
		{"health", "POST", "/health", authenticated, 200},
		{"unauthenticated", "POST", "/health", unauthenticated, 401},
		{"missing identity", "POST", "/health", token, 400},
		{"unknown path", "POST", "/missing", authenticated, 404},
		{"page on POST", "POST", "/", authenticated, 404},
		{"POST-only path on GET", "GET", "/health", nil, 405},
		{"missing page", "GET", "/missing", nil, 404},
	}
	for _, tc := range cases {
		got := ask(t, c, tc.method, tc.path, tc.headers, "{}")
		if got.StatusCode != tc.wantStatus {
			t.Fatalf("%s: status %d, want %d", tc.name, got.StatusCode, tc.wantStatus)
		}
		for header, want := range map[string]string{
			"Cache-Control":          "no-store",
			"X-Content-Type-Options": "nosniff",
			"Referrer-Policy":        "no-referrer",
		} {
			if v := got.Header.Get(header); v != want {
				t.Fatalf("%s: %s = %q, want %q", tc.name, header, v, want)
			}
		}
	}
}

// TestRuntimeLogsOmitPairingSecrets is the regression guard for the runtime
// log: the pairing code and the runtime token are credentials and never belong
// to a file that outlives the process.
func TestRuntimeLogsOmitPairingSecrets(t *testing.T) {
	dir, c := startFixture(t, config.Default())
	link, e := c.Pair(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	code := strings.TrimPrefix(link, c.Meta.URL+"/pair?token=")
	resp, cookie := pairBrowser(t, c, link)
	if resp.StatusCode != http.StatusSeeOther || cookie == "" {
		t.Fatalf("pairing failed: %d", resp.StatusCode)
	}
	logs, e := os.ReadFile(filepath.Join(dir, "runtime.log"))
	if e != nil {
		t.Fatal(e)
	}
	for name, secret := range map[string]string{"pairing code": code, "runtime token": c.Meta.Token} {
		if strings.Contains(string(logs), secret) {
			t.Fatalf("runtime.log leaked the %s", name)
		}
	}
}

// TestRuntimeApprovalOwnership drives one turn that needs an approval over raw
// HTTP: someone else's identity, an unknown ID and a replayed ID are refused,
// and only the owner's first answer decides the turn.
func TestRuntimeApprovalOwnership(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Messages []struct{ Role string } }
		json.NewDecoder(r.Body).Decode(&b)
		hasTool := false
		for _, m := range b.Messages {
			if m.Role == "tool" {
				hasTool = true
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if !hasTool {
			io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"o1\",\"function\":{\"name\":\"write_file\",\"arguments\":\"{\\\"path\\\":\\\"owned.txt\\\",\\\"content\\\":\\\"owned\\\"}\"}}]}}]}\n\ndata: [DONE]\n\n")
			return
		}
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Observed write\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	cfg := config.Default()
	p := cfg.Providers["local"]
	p.BaseURL = provider.URL
	cfg.Providers["local"] = p
	cfg.Provider = "local"
	cfg.Model = "qwen3:8b"
	_, c := startFixture(t, cfg)
	ws, e := workspace.Open(t.TempDir(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer ws.Close()
	session, e := c.New(ws.Path)
	if e != nil {
		t.Fatal(e)
	}
	body, e := json.Marshal(turn{Session: session, Prompt: "write owned.txt", Provider: cfg.Provider, Model: cfg.Model, Harness: cfg.Harness, Workspace: ws.Path, Shell: ws.AllowShell, Permissions: cfg.Permissions, Automation: cfg.Automation})
	if e != nil {
		t.Fatal(e)
	}
	owner := "tab-owner"
	run := ask(t, c, "POST", "/run", map[string]string{"Authorization": "Bearer " + c.Meta.Token, "X-OpenVibeCoding-Client": owner}, string(body))
	if run.StatusCode != 200 {
		t.Fatalf("turn refused: %d", run.StatusCode)
	}
	for header, want := range map[string]string{
		"Cache-Control":          "no-store",
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "no-referrer",
	} {
		if got := run.Header.Get(header); got != want {
			t.Fatalf("stream %s = %q, want %q", header, got, want)
		}
	}
	// The approval event is read while the turn is blocked on it, which is also
	// what proves the stream reaches the client without buffering the turn.
	stream := json.NewDecoder(run.Body)
	approval := ""
	for approval == "" {
		var event wireEvent
		if e := stream.Decode(&event); e != nil {
			t.Fatal(e)
		}
		if event.Kind == "done" {
			t.Fatal("turn finished without asking for approval")
		}
		approval = event.Approval
	}
	approve := func(client, id string) int {
		resp := ask(t, c, "POST", "/approve", map[string]string{"Authorization": "Bearer " + c.Meta.Token, "X-OpenVibeCoding-Client": client}, fmt.Sprintf(`{"ID":%q,"Allow":true}`, id))
		return resp.StatusCode
	}
	if got := approve("another-tab", approval); got != 403 {
		t.Fatalf("another client approved: %d", got)
	}
	if got := approve(owner, "no-such-approval"); got != 403 {
		t.Fatalf("unknown approval accepted: %d", got)
	}
	if got := approve(owner, approval); got != 200 {
		t.Fatalf("owner refused: %d", got)
	}
	if got := approve(owner, approval); got != 403 {
		t.Fatalf("approval replayed: %d", got)
	}
	for {
		var event wireEvent
		if e := stream.Decode(&event); e != nil {
			t.Fatal(e)
		}
		if event.Kind == "done" {
			break
		}
	}
	if text, e := ws.Read("owned.txt"); e != nil || text != "owned" {
		t.Fatalf("approved write did not land: %q %v", text, e)
	}
}

func readSavedConfig(t *testing.T, dir string) config.Config {
	t.Helper()
	var c config.Config
	b, e := os.ReadFile(filepath.Join(dir, "config.json"))
	if errors.Is(e, os.ErrNotExist) {
		return config.Default()
	}
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &c); e != nil {
		t.Fatal(e)
	}
	return c
}
