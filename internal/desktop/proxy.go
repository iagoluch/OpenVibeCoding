//go:build desktop

package desktop

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	daemon "github.com/iagoluch/OpenVibeCoding/internal/runtime"
)

// maxBodyBytes mirrors the daemon's own request cap (http.MaxBytesReader with
// 2<<20 in the runtime dispatcher). The proxy enforces the same limit before
// forwarding so an oversized page request is refused at the trust boundary.
const maxBodyBytes = 2 << 20

// assetHost is the origin the webview uses to address this handler. INV5
// origin discipline: a request that addresses any other host never reaches
// the daemon, because nothing but this webview should be talking to the
// proxy.
const assetHost = "wails.localhost"

// aliasPath is served by the shell itself (never proxied): the only local
// content the handler answers.
const aliasPath = "/ovc-bridge.js"

// The wails runtime module is not injected by wails v3 beta.30 and the daemon
// CSP (script-src 'self') forbids inline scripts, so HTML responses get both
// tags below injected immediately before </head> (POC-proven under the CSP).
const (
	wailsRuntimeTag = `<script type="module" src="/wails/runtime.js"></script>`
	aliasTag        = `<script type="module" src="` + aliasPath + `"></script>`
)

// blocked is refused on every method, ahead of the allowlist, so a future
// widening of the allowlist cannot reopen them:
//
//   - /pair would let a compromised page mint a pairing code through the
//     proxy, materializing a cookie with the same authority as the bearer
//     token outside the proxy's trust position (token laundering).
//   - /stop is a tray action; the tray talks to the daemon directly with its
//     own client identity. The page cannot stop the runtime.
var blocked = map[string]bool{"/pair": true, "/stop": true}

// allowed is the default-deny allowlist: the page may read itself and call
// exactly the daemon API the desktop flow needs. Everything else is a proxy
// 404. API paths are POST-only; reads are GET-only.
func allowed(method, path string) bool {
	if blocked[path] {
		return false
	}
	switch method {
	case http.MethodGet:
		return path == "/" || strings.HasPrefix(path, "/assets/")
	case http.MethodPost:
		switch path {
		case "/health", "/run", "/approve", "/store":
			return true
		}
		return strings.HasPrefix(path, "/ui/")
	}
	return false
}

// errBodyTooLarge marks a request that exceeds the mirrored daemon cap.
var errBodyTooLarge = errors.New("request body exceeds the daemon cap")

// daemonRef is one validated attachment point of the daemon. It is swapped
// atomically when the shell respawns the daemon, which may come back on a
// different port with a different token.
type daemonRef struct {
	url   *url.URL
	host  string
	token string
}

// proxy is the trust boundary between the webview page and the loopback
// daemon. The bearer token lives in this process only: it is attached to
// outbound requests and never reaches the page, whose cookie jar is also
// stripped in both directions.
type proxy struct {
	ref      atomic.Pointer[daemonRef]
	clientID string
	ownHost  string
	log      *slog.Logger
	rp       *httputil.ReverseProxy
}

// newProxy builds the proxy with its per-window client identity. ownHost is
// the host inbound requests must carry (the webview asset origin); empty
// disables that check for tests that exercise the handler directly.
func newProxy(log *slog.Logger, ownHost string) *proxy {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	p := &proxy{clientID: "ovc-desktop-" + randomID(), ownHost: ownHost, log: log}
	p.rp = &httputil.ReverseProxy{
		FlushInterval:  -1, // flush every write: /run NDJSON must stream
		Rewrite:        p.rewrite,
		ModifyResponse: p.modifyResponse,
		ErrorHandler:   p.errorHandler,
	}
	return p
}

// retarget points the proxy at a daemon. The address comes from the runtime's
// own metadata and is revalidated here: loopback http only.
func (p *proxy) retarget(meta daemon.Metadata) error {
	u, e := url.Parse(meta.URL)
	if e != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Path != "" || u.User != nil {
		return errors.New("invalid daemon address: " + meta.URL)
	}
	p.ref.Store(&daemonRef{url: u, host: u.Host, token: meta.Token})
	return nil
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == aliasPath {
		if r.Method == http.MethodGet {
			serveAlias(w)
			return
		}
		http.NotFound(w, r)
		return
	}
	ref := p.ref.Load()
	if ref == nil {
		http.Error(w, "runtime not attached", http.StatusBadGateway)
		return
	}
	// Default deny: tunneling methods and absolute-form targets never reach
	// the daemon, and the allowlist decides everything else.
	if r.Method == http.MethodConnect || r.URL.IsAbs() || !allowed(r.Method, r.URL.Path) {
		p.log.Warn("desktop proxy: refused", "method", r.Method, "host", r.Host, "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	if p.ownHost != "" && r.Host != p.ownHost {
		p.log.Warn("desktop proxy: foreign host", "host", r.Host, "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	// A request that does not come from this window's origin is refused. The
	// webview sends same-origin (or none); a cross-site or same-site fetch
	// metadata value means someone else is driving the request.
	switch r.Header.Get("Sec-Fetch-Site") {
	case "cross-site", "same-site":
		p.log.Warn("desktop proxy: foreign fetch", "site", r.Header.Get("Sec-Fetch-Site"), "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	if e := canonicalizeBody(w, r); e != nil {
		return // response already written (413 or 502)
	}
	p.log.Debug("desktop proxy: forwarding", "method", r.Method, "path", r.URL.Path)
	p.rp.ServeHTTP(w, r)
}

// rewrite forwards to the current daemon with the shell's own credentials.
func (p *proxy) rewrite(pr *httputil.ProxyRequest) {
	ref := p.ref.Load()
	pr.SetURL(ref.url)
	pr.Out.Host = ref.host // the daemon answers only its own address (403 otherwise)
	h := pr.Out.Header
	h.Del("Cookie") // the page must not forward a cookie jar
	h.Del("Origin") // the daemon requires no foreign Origin (401 otherwise)
	// Override, never append: a page-spoofed identity or credential is
	// dropped and replaced by this shell's own.
	h.Del("Authorization")
	h.Del("X-OpenVibeCoding-Client")
	h.Set("Authorization", "Bearer "+ref.token)
	h.Set("X-OpenVibeCoding-Client", p.clientID)
	// Sec-Fetch-Site is forwarded untouched: the daemon already accepts only
	// same-origin/none, and the proxy refused cross-site/same-site above.
}

// modifyResponse keeps token material out of the webview and teaches the page
// about the wails runtime and the bridge alias.
func (p *proxy) modifyResponse(resp *http.Response) error {
	resp.Header.Del("Set-Cookie") // no token material in the webview cookie jar
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return nil
	}
	body, e := io.ReadAll(resp.Body)
	resp.Body.Close()
	if e != nil {
		return e
	}
	page := strings.Replace(string(body), "</head>", wailsRuntimeTag+aliasTag+"</head>", 1)
	resp.Body = io.NopCloser(strings.NewReader(page))
	resp.ContentLength = int64(len(page))
	resp.Header.Set("Content-Length", strconv.Itoa(len(page)))
	return nil
}

func (p *proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	p.log.Warn("desktop proxy: upstream unreachable", "path", r.URL.Path, "error", err)
	w.WriteHeader(http.StatusBadGateway)
}

// canonicalizeBody buffers the request body and re-sets Content-Length. Wails
// v3 beta.30 on Windows delivers POST bodies with a Content-Length of 0
// while the body bytes are intact; buffering here repairs the length before
// forwarding (body bytes untouched; POC-proven fix; upstream issue to file).
func canonicalizeBody(w http.ResponseWriter, r *http.Request) error {
	if r.Body == nil || r.Method != http.MethodPost {
		return nil
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	r.Body.Close()
	if e != nil {
		http.Error(w, "request body unreadable", http.StatusBadGateway)
		return e
	}
	if len(b) > maxBodyBytes {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return errBodyTooLarge
	}
	r.Body = io.NopCloser(bytes.NewReader(b))
	r.ContentLength = int64(len(b))
	r.Header.Set("Content-Length", strconv.Itoa(len(b)))
	return nil
}

// serveAlias serves the shell-owned bridge alias module (see
// assets/ovc-bridge.js) with the real binding FQN substituted in.
func serveAlias(w http.ResponseWriter) {
	js := strings.Replace(bridgeAlias, "__BINDING_FQN__", bindingFqn(), 1)
	h := w.Header()
	h.Set("Content-Type", "text/javascript; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(js))
}
