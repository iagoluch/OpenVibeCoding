// Package webui serves the browser client of the authenticated loopback
// runtime over the port the daemon already owns. The shipped page is a
// self-contained offline interface: it holds the security contract (own-origin
// only, no inline code, no remote code) while the runtime keeps the real
// authority over sessions, models and approvals.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

//go:embed assets
var embedded embed.FS

// policy is the single header policy of the UI: everything is same-origin, no
// inline or remote code, no framing, no form posts to other origins.
const policy = "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// contentTypes pins the types we ship. Relying on the operating system's table
// could make the browser refuse to execute scripts under nosniff.
var contentTypes = map[string]string{
	".html":        "text/html; charset=utf-8",
	".js":          "text/javascript; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".svg":         "image/svg+xml",
	".png":         "image/png",
	".webmanifest": "application/manifest+json; charset=utf-8",
}

// Secure applies the header policy to every UI response, including the pairing
// pages served by the runtime.
func Secure(h http.Header) {
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Content-Security-Policy", policy)
}

// Handler serves the page and its assets: GET / is the document, GET
// /assets/<name> is one file. Unknown paths and directories are 404 and never
// listed; paths outside the embedded tree are refused by fs.ValidPath.
func Handler() (http.Handler, error) {
	assets, e := fs.Sub(embedded, "assets")
	if e != nil {
		return nil, e
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Secure(w.Header())
		if r.Method != http.MethodGet {
			http.Error(w, "GET required", http.StatusMethodNotAllowed)
			return
		}
		name := "index.html"
		if r.URL.Path != "/" {
			name = strings.TrimPrefix(r.URL.Path, "/assets/")
			if !strings.HasPrefix(r.URL.Path, "/assets/") || !fs.ValidPath(name) {
				http.NotFound(w, r)
				return
			}
		}
		b, e := fs.ReadFile(assets, name)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		if t, ok := contentTypes[path.Ext(name)]; ok {
			w.Header().Set("Content-Type", t)
		}
		w.Write(b)
	}), nil
}

// PairInvalid returns the page shown when a pairing link is expired, already
// used or forged.
func PairInvalid() []byte {
	b, _ := fs.ReadFile(embedded, "assets/pair-invalid.html")
	return b
}

// OpenBrowser asks the operating system to open one URL. It is best effort and
// never blocks: the caller keeps working when no browser can be started, and
// the child is reaped in the background.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if e := cmd.Start(); e != nil {
		return e
	}
	go cmd.Wait()
	return nil
}
