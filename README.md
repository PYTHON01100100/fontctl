# fontctl — Linux Font Manager

> Cross-distro font discovery, language-specific preferences, and terminal/desktop font management from a Go CLI and Bubble Tea TUI.

**Project stage: v0.1 alpha source release.** No binaries or font files are bundled. This is an early project with real font discovery, staged settings, local/HTTPS font imports, a TUI, and backup/restore. It has not yet been built against downloaded Charm dependencies in the packaging environment; see **Build and verification**.

**Documentation:** [English (this file)](README.md) · [العربية](README_AR.md) · [Development agenda](AGENDA.md) · [Architecture](docs/ARCHITECTURE.md)

## Goals

- Discover installed font families with **Fontconfig** (`fc-list`, `fc-match`), independent of a particular Linux distribution's package manager.
- Stage **language-aware font preferences** for English, Arabic, Japanese, simplified/traditional Chinese, and Korean, using user-scoped Fontconfig rules.
- Change the normal font in **Kitty and Alacritty**, and the **GNOME** interface/monospace settings.
- Browse fonts in a **Bubble Tea v2** terminal UI; rotate an Arabic test phrase without disturbing user preferences.
- **Manual font installation, two ways**: import a downloaded `.ttf`, `.otf`, `.ttc` or `.zip`, or paste a direct **HTTPS** font/ZIP URL to download on demand. No silent downloads; no `sudo`.
- Plan changes, apply with a snapshot, and restore the previous settings.

## Requirements

- Linux (x86_64, ARM64, or other Go-supported Linux architectures).
- **Go 1.25+** to build the project from source (Go may download its toolchain on older systems).
- Working network when initially fetching Go modules.
- `fontconfig` tools (`fc-list`, `fc-match`; `fc-cache` recommended for importing files).
- Kitty / Alacritty / GNOME **only for their respective integrations**. You don't need all three.
- A terminal capable of showing Unicode; Arabic shaping and bidirectional rendering vary by terminal.

Install Fontconfig using your distro's package manager:

```bash
# Fedora / RHEL family
sudo dnf install fontconfig
# Ubuntu / Debian family
sudo apt install fontconfig
# Arch family
sudo pacman -S fontconfig
```

The fontctl program itself needs **no root privileges**. Those `sudo` commands are only to install the prerequisite system package if missing.

## Build and verification

```bash
git clone https://github.com/YOUR-USERNAME/fontctl.git
cd fontctl

go mod tidy
go test ./...
go build -o fontctl ./cmd/fontctl
./fontctl doctor
./fontctl
```

Or build from this ZIP by running the commands starting at `go mod tidy` in the extracted folder. **The repository URL above is a placeholder**, not a published repository.

> Packaging note: The source ZIP does not include `go.sum` or vendored third-party modules because dependency network access was unavailable in the build environment. `go mod tidy` will resolve and create `go.sum` on a connected developer machine. The Charm-based TUI was not compiled in that offline packaging environment. Standard-library core packages were independently tested; do not treat the alpha as production-verified.

## Commands

```bash
fontctl                           # Open TUI
fontctl fonts                     # List installed families
fontctl fonts --lang ar           # List Arabic-capable families
fontctl preview                   # All three English samples + next Arabic sample
fontctl preview --lang ar         # Cycle through three exact Arabic phrases
fontctl preview --lang en
fontctl match --lang ar --kind sans

# Stage changes; they are NOT applied until `fontctl apply`.
fontctl set lang ar --family 'Noto Sans Arabic' --kind sans
fontctl set lang ja --family 'Noto Sans CJK JP' --kind sans
fontctl set terminal kitty --font 'Cascadia Code NF' --size 12
fontctl set terminal alacritty --font 'Cascadia Code NF' --size 12
fontctl set desktop gnome --interface 'Noto Sans' --monospace 'DejaVu Sans Mono'
fontctl plan
fontctl apply --dry-run
fontctl apply
fontctl restore latest

# Manually downloaded fonts (including font ZIP releases):
fontctl import "$HOME/Downloads/CascadiaCode.zip"
fontctl import "$HOME/Downloads/ExampleFont-Regular.ttf"

# Direct downloadable font or ZIP link: opt-in, HTTPS only.
fontctl install --url 'https://example.org/fonts/ExampleFont.zip'
```

**A GitHub Releases page is not itself a downloadable `.zip` URL.** Copy the specific release asset's direct HTTPS link, or download the ZIP using a browser and run `fontctl import /path/file.zip`. The app does not scrape release pages, guess asset names, or bypass licenses.

The online catalog + graphical pre-install preview are planned in [AGENDA.md](AGENDA.md), **not available in v0.1**. `fontctl install --url` already downloads direct files and archives; `fontctl import` is the manual alternative.

## TUI shortcuts

| Key | Action |
| --- | --- |
| `↑ / ↓`, `j / k` | Select an installed font |
| `Tab` | Cycle target (language sans/serif/mono, Kitty, Alacritty, GNOME) |
| `l` | Cycle current language (en, ar, ja, zh-cn, zh-tw, ko) |
| `/` | Filter font families by text |
| `Enter` | Stage selected family to the chosen target and save preferences |
| `p` | Rotate Arabic preview phrase |
| `i` | Type a local font/ZIP path to import |
| `u` | Type a direct HTTPS font/ZIP URL to install |
| `d` | Dry-run change count (use `fontctl plan` for file details) |
| `a` | Apply with explicit `y` confirmation and a backup |
| `r` | Restore last backup with confirmation |
| `q` | Quit |

**Preview limitation:** the TUI prints the chosen sample strings in the **currently active terminal font**. It cannot force the emulator to temporarily display an arbitrary uninstalled font. Real visual comparison / browser preview is a future GUI helper, not currently claimed.

## Exact test phrases

English (the same three lines on every preview):

```text
Hello World My friend
one cli to rule them all
I am the king of the world
```

Arabic (one phrase per preview; rotates persistently 1 → 2 → 3 → 1):

```text
صِفْ خَلْقَ خَوْدٍ كَمِثْلِ الشَّمْسِ إِذْ بَزَغَتْ يَحْظَى ضَجِيعٌ بِهَا نَجْلَاءَ مِعْطَارِ
بَلاغَةُ الحَرْفِ تُغْنِي عَنِ الكَلامِ
العَرَبِيَّةُ: أَصَالَةُ المَاضِي وَأُفُقُ المُسْتَقْبَلِ
```

The Arabic rotation index is stored in `$XDG_STATE_HOME/fontctl/preview.json` or `~/.local/state/fontctl/preview.json`.

## Configuration and platform compatibility

| Feature | v0.1 coverage |
| --- | --- |
| Font discovery / preview text / import | Distro-independent through Fontconfig and XDG user directories |
| Language preferences | Fontconfig user overlay: `~/.config/fontconfig/conf.d/60-fontctl.conf` |
| Kitty | `kitty.conf` + managed `fontctl.conf` include |
| Alacritty | Modern TOML `alacritty.toml` + managed import; conflicts are detected and refused |
| GNOME | Uses `gsettings`; on GNOME sessions only |
| KDE Plasma / Konsole / GNOME Terminal profiles / WezTerm / Foot | Future adapters |
| Actual on-screen language-aware fallback | Depends on the application and terminal; no universal guarantee |

Config preferences are stored as JSON in `$XDG_CONFIG_HOME/fontctl/config.json` (fallback `~/.config/fontctl/config.json`). Manual font files are stored in `$XDG_DATA_HOME/fonts/fontctl` (fallback `~/.local/share/fonts/fontctl`). Backups live under `$XDG_STATE_HOME/fontctl/snapshots` (fallback `~/.local/state/fontctl/snapshots`).

**Distribution-independent** does not mean **identical results in every desktop/terminal**. The core operates anywhere Fontconfig is installed, while desktop integrations depend on the desktop environment, not on Fedora vs Ubuntu vs Arch.

## Security and limits

- User-only writes; explicit opt-in for network download. HTTPS required, including redirects.
- Allowed imported formats: TTF, OTF, TTC, or ZIP containing them. WOFF2 website fonts are not directly imported into Fontconfig.
- Basic file signature checks; capped download, ZIP expansion, entry count and file sizes; ignores ZIP symlinks and directory paths.
- This is **not a full font parser or malware scanner**. Do not import files from untrusted sites; verify font licenses before use.
- No privileged system font installation, package-manager modifications, or font binaries packaged with this source.
- Safe overlays and latest-snapshot restoration are implemented, but audit & test on your distro before relying on it.

## Architecture & roadmap

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the component model and [AGENDA.md](AGENDA.md) for next milestones, including an Internet font catalog, font licensing, pre-install previews, KDE support, and Linux CI integration tests.

## License

[MIT](LICENSE) for fontctl source code. **Font licenses are independent**: importing a font does not grant rights to redistribute it.
