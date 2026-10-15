# Presentation website

Static HTML/CSS/JavaScript; no build step, analytics, API keys or dependencies. The workflow panel is explicitly illustrative and does not execute agent operations.

Preview from the repository root:

```sh
python3 -m http.server 4173 --directory site --bind 0.0.0.0
```

The included `pages.yml` workflow uploads this directory for GitHub Pages when manually dispatched. Publishing requires enabling GitHub Pages with GitHub Actions in repository settings. No deployment is performed by merely creating these files.

Interactive controls support keyboard tabs/arrows, visible focus, clipboard feedback and reduced-motion preferences. Installation text and capability claims should stay aligned with docs/guide.md.
