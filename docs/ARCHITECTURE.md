# Architecture

`fontctl` is a small, user-space Go application, not a daemon or root service. There is one executable with CLI and Bubble Tea UI entry points.

```text
cmd/fontctl/main.go
    ├── internal/tui          Bubble Tea v2 + Lip Gloss v2, interactive UX
    ├── internal/config       XDG JSON preference store + allowlists
    ├── internal/preview      English lines + persistent Arabic rotation
    ├── internal/fonts        fc-list, fc-match (Fontconfig)
    ├── internal/install      opt-in HTTPS/local import, TTF/OTF/TTC/ZIP safety limits
    └── internal/engine       build plan → capture snapshot → write overlays → restore
                                 ├── Fontconfig conf.d
                                 ├── Kitty include and overlay
                                 ├── Alacritty [general].import and overlay
                                 └── GNOME gsettings
```

## Cross-distribution boundary

The same commands and user-scoped directories work independent of package manager. System configuration varies by DE; for this reason adapters are explicit and opt-in. Fontconfig pattern language rules cannot guarantee application-wide fallback; terminal-emulator rendering, shaping and RTL are outside the scope of a pure TUI.

## App flows

- **Select:** UI or CLI changes a preference and persists in JSON. No system setting changes yet.
- **Plan:** `BuildPlan` reads existing config, detects conflicts and calculates managed writes. No writes.
- **Apply:** Validate chosen fonts exist → capture file/GSettings values → save snapshot → apply managed changes → save pointer to latest snapshot. Failures attempt rollback.
- **Restore:** Read last saved snapshot → restore prior file content and GSettings values.
- **Import:** User chooses local path or direct HTTPS URL → format and size constraints → ZIP in-memory walk (no path extraction) → user-local font directory → optional fc-cache refresh.

## Security model & extensions

Explicit HTTPS downloads; no shell expansion of URLs; no auto-sudo. The app avoids following config-file symlinks, but broader threat-model review, parent-directory race mitigation, cryptographic pinning, real font parsing and audited rollback are pre-stable agenda items. Archives are treated as hostile and subject to limits.

The built-in TUI cannot render a font that the host terminal is not currently using. Remote pre-install previews will require an optional local GUI/font-shaping helper, not simulated color previews.
