# fontctl — Development Agenda

Project: **Cross-distro font discovery, language-aware preferences, terminal and desktop management**.

## North star

One CLI to rule them all: help Linux users choose, preview, install, and safely configure fonts, **including Arabic**, without caring which distribution package manager they use.

### Implemented in the v0.1 alpha source

- [x] Go project with Bubble Tea v2 and Lip Gloss v2 source code.
- [x] Discover installed font families using `fc-list`; optionally filter by language.
- [x] TUI with font selection, target cycling, language selection, and text filtering.
- [x] English samples exactly: `Hello World My friend`; `one cli to rule them all`; `I am the king of the world`.
- [x] Rotate the three supplied fully vocalized Arabic sample sentences persistently, each time the preview is requested.
- [x] Store staged preferences per language using XDG directories and a managed Fontconfig XML overlay.
- [x] Kitty and Alacritty user-level configuration overlays; GNOME GSettings integration.
- [x] Dry-run plans and snapshots / restore of affected config files and GNOME values.
- [x] **Manual alternative 1:** import an existing local `.ttf`, `.otf`, `.ttc`, or `.zip` file downloaded in a browser.
- [x] **Manual alternative 2:** paste a **direct HTTPS** URL to a font file or ZIP; fetch and import on explicit command. Support via CLI and TUI.
- [x] Download caps, ZIP entry caps, path traversal prevention, no system-wide privileged writes.
- [x] English-first README plus Arabic README; basic unit tests (core testable offline).

### Release readiness (before calling it stable)

- [ ] Resolve dependencies on a connected Linux machine (`go mod tidy`) and commit `go.sum`.
- [ ] Run `go test ./...`, `go vet ./...` and build against the actual Bubble Tea v2 module.
- [ ] Run end-to-end TUI tests on Fedora and Ubuntu, including Arabic input and clipboard pastes.
- [ ] Exercise imported fonts with real OpenType binaries and fonts carrying varied metadata.
- [ ] Test Kitty, modern TOML Alacritty, GNOME GSettings, and missing-component error handling.
- [ ] Harden symlinked parent directories, malformed Fontconfig, and interrupted backup/restore transactions.

## v0.2 — Internet catalog, previews & font downloads

- [ ] **Search fonts directly from the internet** by name, script/language, license, family, style, type, category, and monospaced/Nerd Font capability.
- [ ] Add source providers: **Google Fonts** (metadata/API or curated registry) and **GitHub releases with user review**. Avoid scraping unknown websites and respect providers' API terms and rate limits.
- [ ] **Remote font preview BEFORE installation**: download a small trusted specimen or approved preview metadata; render the selected font via an optional graphical helper (e.g., HarfBuzz/Pango/FreeType) for accurate Arabic shaping/RTL and CJK glyph comparisons.
- [ ] Show previews of the exact user-provided Arabic sentences, rotating once per preview request, and the three English samples together.
- [ ] Preview weights (Regular, Medium, SemiBold, Bold), slant, size, Arabic coverage, ligatures, and fallback behavior.
- [ ] Explicitly display font license, release URL, author, version, checksums (when supplied), and download size before install.
- [ ] Download progress, cancel, retry, download cache, resumable download with ETag where supported, and offline mode.
- [ ] Import via URLs ending in release assets; later optionally resolve GitHub release pages into user-selectable assets using provider APIs.
- [ ] Never auto-install from search results. Require confirmation before writing font files.
- [ ] Support importing a collection of archive formats safely, with image previews only when permitted by the upstream license.

## v0.3 — Desktop & terminal adapters

- [ ] KDE Plasma per-user settings adapter, tested for relevant Plasma versions.
- [ ] GNOME Terminal profile font adapter; Konsole profile adapter.
- [ ] WezTerm and Foot adapters; extra terminals on user demand.
- [ ] Explicit per-adapter capability matrix (font family, size, fallbacks, shaping/RTL, live reload).
- [ ] Detect conflicts in existing configs and support no-clobber merges.

## v0.4 — UX & quality

- [ ] Font render side-by-side (before/after) via an optional GUI preview helper.
- [ ] Saved profiles and profile export/import (e.g. `Coding`, `Arabic reading`, `Minimal`).
- [ ] Automatic installer detection for official distribution packages but do not invoke sudo automatically.
- [ ] Full locale selection: Arabic and English UI; accessibility and keyboard-only operation.
- [ ] CI across Fedora, Ubuntu and Arch containers plus system environment tests in VMs.

## v1.0 — Stable

- [ ] Public API/CLI contract and comprehensive tests.
- [ ] Signed release binaries for Linux amd64/arm64, packaging instructions, reproducible builds.
- [ ] Security audit of remote downloads and font parsers; document supported/distinct font licenses.
- [ ] Crash-safe backups and multi-snapshot history with restore selection.

## Acceptance scenarios

1. Fedora/Ubuntu/Arch user can list fonts, preview three English phrases and cycle the three exact Arabic test samples.
2. User selects `Cascadia Code NF` for Kitty, stages the preference, reviews `fontctl plan`, then applies it with a recoverable backup.
3. User downloads a released font ZIP using a browser, runs `fontctl import ~/Downloads/font.zip`, and finds the family via `fontctl fonts`.
4. User pastes an explicit HTTPS link into the TUI's URL import prompt; the font is installed locally without root.
5. A GNOME user changes interface/monospace font, while a non-GNOME user can still use core Fontconfig functions.
6. Preview never pretends to render an uninstalled font in-place; future graphical helper demonstrates actual typography.
