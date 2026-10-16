# OpenVibeCoding visual identity

This guide defines the visual system for OVC's repository assets and product-facing surfaces.

## Core palette

| Role | Color | Use |
| --- | --- | --- |
| Ink black | `#050A11` | Main background; avoid pure-black glare |
| Deep blue | `#0D2B46` | Byte silhouette, panels and secondary surfaces |
| Clear blue | `#69CFFF` | Primary accent, active states, links and brand marks |
| Ice blue | `#DFF7FF` | Strong text on dark surfaces and small mascot details |
| Blueprint line | `#193A52` | Dividers, technical construction lines and borders |
| Muted blue | `#8FAEC1` | Secondary text and inactive UI labels |

Use the bright blue sparingly. Black should carry the canvas; deep blue should create structure; clear blue should point to the one thing that matters. Do not add a second accent color just to make a composition more colorful.

## Byte

Byte is represented as a small, angular, seed-like creature with a folded silhouette and a few detached pixel marks. The shape hints at a byte becoming a living companion without depicting a robot, circuit board, humanoid, or generic assistant head.

- Keep Byte compact, around 32–48 px in interface contexts.
- Preserve the asymmetrical silhouette, two tiny ice-blue eyes and three loose pixels.
- Use flat fills and crisp edges; no metallic parts, limbs, antennae or glowing aura.
- Keep the mascot subordinate to the OVC wordmark and the task UI.
- Reuse `assets/byte.svg` instead of drawing a different mascot for each screen.

## Composition and tone

- Prefer editorial layouts, asymmetric alignment, measured spacing and precise technical rules.
- Use the restrained language of engineering drawings and code review: diffs, boundaries, annotations and visible states.
- Keep UI mockups schematic and label them as concepts, never as screenshots of an unreleased app.
- The current public release is CLI/TUI. Marketing copy must not imply that the integrated desktop application is already available.
- Provide localized banners where a README is localized: `banner-pt-br.svg`, `banner.svg` (English), `banner-es.svg` and `banner-zh-CN.svg`.

## Avoid

Generic AI gradients, decorative glowing orbs, excessive glassmorphism, robot mascots, random neon wireframes, stock icon grids, and fake product screenshots presented as real. Originality should come from the Byte silhouette, strong typography, negative space and a consistent blue-on-black system.
