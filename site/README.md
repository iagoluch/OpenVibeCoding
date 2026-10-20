# Presentation website

Two static pages, one shared stylesheet and one shared script:

| Path | Language | Served at |
| :--- | :--- | :--- |
| `index.html` | Brazilian Portuguese (primary) | `/` |
| `en/index.html` | English | `/en/` |
| `site.css` | shared styles | both |
| `site.js` | shared behavior | both |
| `icon.svg` | shared favicon | both |

There is no build step, no framework, no analytics and no API keys. Archivo is loaded from one Google Fonts link; both pages fall back to the system sans stack if it is unavailable.

The English page's terminal panel is an original HTML/CSS reconstruction of the real TUI, labeled as such. The Portuguese page's panel is an illustrative localized reconstruction in Portuguese, labeled as illustrative: the real TUI strings are English. Neither panel executes agent operations or makes a model request.

The language switch is a pair of real links, not a JavaScript-only toggle. Desktop links sit in the nav; at `max-width: 960px` the switch moves inside the mobile menu. `aria-current="page"` marks the current language, and every link carries `hreflang` and `lang`. There is no canonical link and no `<link rel="alternate" hreflang>` in the head: GitHub Pages is not enabled for this repository (`has_pages=false`), so no public URL exists yet. The language links are the only place `hreflang` appears.

Preview from the repository root:

```sh
python -m http.server 4173 --directory site
```

Open `http://localhost:4173/` for Portuguese and `http://localhost:4173/en/` for English.

The `.github/workflows/pages.yml` workflow uploads this directory when manually dispatched. GitHub Pages must be enabled with GitHub Actions in repository settings; the workflow does not publish on its own.

Copy changes belong in both HTML files (translated per page). CSS and JavaScript are single-source: edit `site.css` or `site.js` once. Visible strings in `site.js` are not hardcoded; the copy button and menu toggle read `data-label-*` and `data-status-*` attributes supplied by each page.

The pages support keyboard tabs and menu, a visible focus ring, clipboard feedback with a manual-copy fallback, a `file://` fallback path, and reduced-motion preferences. Installation commands and capability claims should stay aligned with `docs/guide.md` and `docs/security.md`; the technical documentation is English on both pages.
