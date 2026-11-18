# ADR-012: Tailwind CSS for Styling

## Status

Accepted

## Context and Problem Statement

The browser UI ([ADR-011](011-htmx-for-frontend.md)) is built from server-rendered `html/template` partials. It needs a consistent look for the dashboard, board, turn indicator and move history (`SPEC.md` UI-001..011), and it should support responsive desktop and mobile layouts (UI-013). Styles live next to markup in many small partials, and swapped fragments must look right without separate per-component stylesheets. `SPEC.md` §19 already recommends Tailwind CSS.

How should we style the UI?

## Decision Drivers

- **Co-location**: styles should be visible in the template that renders a fragment
- **Responsive by default**: mobile and desktop layouts without hand-written media queries everywhere (UI-013)
- **Small output**: ship only the CSS the templates use
- **No Node.js requirement**: fit the single-toolchain goal of [ADR-011](011-htmx-for-frontend.md)
- **Consistency**: a shared design scale (spacing, color, type) instead of ad hoc values

## Considered Options

- **Option A**: Tailwind CSS v4, built with the standalone CLI binary
- **Option B**: Hand-written CSS with custom properties (as in the throwaway `prototype/`)
- **Option C**: A component framework such as Bootstrap or Pico.css

## Pros and Cons of the Options

### Option A: Tailwind CSS

- Pro: utility classes live in the templates, so each fragment carries its own styling
- Pro: the build scans the templates and emits only the classes they use
- Pro: the standalone CLI is a single executable, so no Node.js or npm is needed
- Pro: responsive (`sm:`, `md:`) and dark-mode (`dark:`) variants cover UI-013 directly
- Con: long class lists in templates; repeated patterns should become partials, not copy-paste
- Con: adds a CSS build step to `make build`

### Option B: Hand-written CSS

- Pro: no tooling
- Con: grows without bound and drifts from the markup as fragments change

### Option C: Bootstrap or Pico.css

- Pro: ready-made components
- Con: generic look, harder to customise for a game board, and ships CSS we don't use

## Decision Outcome

Chosen option: **Option A, Tailwind CSS v4 with the standalone CLI**, because it keeps styling next to the server-rendered markup and produces minimal CSS with no Node.js toolchain. `SPEC.md` §19 also recommends it.

We will:

1. Pin an exact Tailwind CLI version and verify the downloaded binary's checksum in `make` and CI (`dependency-management.md` rules 4 and 9).
2. Configure Tailwind's sources to scan the `html/template` files, and keep game colors (red and yellow discs, board) as theme tokens.
3. Build the CSS as part of `make build`, embed the output with `go:embed`, and serve it from the Go binary. Generated CSS is a build artifact that `make clean` removes ([ADR-001](001-five-command-interface.md)).
4. Write board-specific effects (the disc drop animation, the winning-line highlight) as a small amount of custom CSS in the Tailwind entry file.

## Consequences

### Positive

- Every fragment is self-describing; a template diff shows both markup and style changes
- Small, predictable CSS bundle; responsive and dark-mode support with no extra framework
- No Node.js on developer machines or in CI for the application build

### Negative

- `make build` gains a CSS step and a pinned binary download, which the Dockerfile and CI must account for
- The team has to agree on when to extract a repeated class list into a partial

## Dependency Review

| Item | Finding (2026-11-18) |
|------|----------------------|
| Tool | Tailwind CSS standalone CLI (`tailwindlabs/tailwindcss`) |
| License | MIT |
| Maintenance | Active; latest release v4.3.5 (2026-11-06) |
| Footprint | Build-time only; nothing from Tailwind ships except the generated CSS |

## Compliance

| Spec | Enforcement |
|------|-------------|
| `SPEC.md` UI-013 | Playwright checks at phone and desktop viewport widths (no horizontal overflow) |
| `security/dependency-management.md` rules 4 and 9 | Pinned CLI version and checksum verification in `make` and CI |
| [ADR-001](001-five-command-interface.md) | CSS built by `make build` and removed by `make clean` |

## References

- **internal** [ADR-001](001-five-command-interface.md) — Five Command Project Interface
- **internal** [ADR-011](011-htmx-for-frontend.md) — htmx for the Browser UI
- **spec**: `SPEC.md` UI-001..013, §19
- **external**: https://tailwindcss.com/
- **external**: https://tailwindcss.com/blog/standalone-cli
