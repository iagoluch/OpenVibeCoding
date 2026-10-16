package webui

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesPageAndAssets(t *testing.T) {
	h, e := Handler()
	if e != nil {
		t.Fatal(e)
	}
	cases := []struct {
		path, ctype, contains string
	}{
		{"/", "text/html; charset=utf-8", "/assets/app.js"},
		{"/assets/app.js", "text/javascript; charset=utf-8", "crypto.randomUUID"},
		{"/assets/style.css", "text/css; charset=utf-8", "color-scheme"},
		{"/assets/icon.svg", "image/svg+xml", "<svg"},
		{"/assets/manifest.webmanifest", "application/manifest+json; charset=utf-8", `"name"`},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", c.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s: %d", c.path, w.Code)
		}
		if got := w.Header().Get("Content-Type"); got != c.ctype {
			t.Fatalf("%s: content type %q", c.path, got)
		}
		if !strings.Contains(w.Body.String(), c.contains) {
			t.Fatalf("%s: body missing %q", c.path, c.contains)
		}
		for header, want := range map[string]string{
			"Cache-Control":          "no-cache",
			"X-Content-Type-Options": "nosniff",
			"Referrer-Policy":        "no-referrer",
		} {
			if got := w.Header().Get(header); got != want {
				t.Fatalf("%s: %s = %q", c.path, header, got)
			}
		}
		if !strings.Contains(w.Header().Get("Content-Security-Policy"), "default-src 'self'") {
			t.Fatalf("%s: missing CSP", c.path)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("%s: CORS header served", c.path)
		}
	}
}

// TestHandlerServesBinaryIcons checks the PNG route: the handler must pin the
// type even though the bytes carry no readable text.
func TestHandlerServesBinaryIcons(t *testing.T) {
	h, e := Handler()
	if e != nil {
		t.Fatal(e)
	}
	magic := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	for _, path := range []string{"/assets/icon-192.png", "/assets/icon-512.png", "/assets/icon-maskable-512.png"} {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s: %d", path, w.Code)
		}
		if got := w.Header().Get("Content-Type"); got != "image/png" {
			t.Fatalf("%s: content type %q", path, got)
		}
		if !bytes.HasPrefix(w.Body.Bytes(), magic) {
			t.Fatalf("%s: not a PNG", path)
		}
	}
}

// TestContentTypeFallsBackToOpaqueBytes keeps the pinned type honest for a file
// the table does not know: serving it unset would let the runtime sniff a type
// the browser then trusts under nosniff.
func TestContentTypeFallsBackToOpaqueBytes(t *testing.T) {
	for name, want := range map[string]string{
		"index.html":           "text/html; charset=utf-8",
		"app.js":               "text/javascript; charset=utf-8",
		"icon.svg":             "image/svg+xml",
		"manifest.webmanifest": "application/manifest+json; charset=utf-8",
		"notes.bin":            "application/octet-stream",
		"archive.tar.gz":       "application/octet-stream",
		"LICENSE":              "application/octet-stream",
	} {
		if got := contentType(name); got != want {
			t.Fatalf("%s: %q, want %q", name, got, want)
		}
	}
}

// TestHandlerRefusesEscapesAndUnknownPaths keeps unknown paths out of the tree.
func TestHandlerRefusesEscapesAndUnknownPaths(t *testing.T) {
	h, e := Handler()
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"/missing.html", "/assets/missing.js", "/assets/", "/assets/../assets/app.js", "/assets/%2e%2e/webui.go", "/other"} {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 404 {
			t.Fatalf("%s: expected 404, got %d", path, w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST accepted: %d", w.Code)
	}
}

// TestAssetsCarryNoInlineCode keeps the CSP enforceable: every script and style
// must come from its own file.
func TestAssetsCarryNoInlineCode(t *testing.T) {
	for _, name := range []string{"assets/index.html", "assets/pair-invalid.html"} {
		b, e := embedded.ReadFile(name)
		if e != nil {
			t.Fatal(e)
		}
		page := string(b)
		for _, forbidden := range []string{"<script>", "<style", " onclick=", "http://", "https://"} {
			if strings.Contains(page, forbidden) {
				t.Fatalf("%s: contains %q", name, forbidden)
			}
		}
	}
	js, e := embedded.ReadFile("assets/app.js")
	if e != nil || len(js) == 0 {
		t.Fatal("app.js missing")
	}
}

func TestPairInvalidPage(t *testing.T) {
	if !strings.Contains(string(PairInvalid()), "ovc --ui") {
		t.Fatal("pairing page must tell the user how to get a new link")
	}
}

func TestSecureReplacesAnyEarlierPolicy(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Security-Policy", "default-src *")
	Secure(h)
	if got := h.Get("Content-Security-Policy"); got != policy {
		t.Fatalf("policy not replaced: %q", got)
	}
}
