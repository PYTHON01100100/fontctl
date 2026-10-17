// Package engine plans, validates, applies, and rolls back font settings.
// The implementation is user-scoped, deliberately refusing to follow config symlinks.
package engine

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PYTHON01100100/fontctl/internal/config"
	"github.com/PYTHON01100100/fontctl/internal/fonts"
	"github.com/PYTHON01100100/fontctl/internal/preview"
)

type FileChange struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	Description string `json:"description"`
}
type SettingChange struct {
	Schema      string `json:"schema"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}
type Plan struct {
	Files    []FileChange    `json:"files"`
	Settings []SettingChange `json:"settings"`
}
type fileBackup struct {
	Path    string `json:"path"`
	Existed bool   `json:"existed"`
	Data    []byte `json:"data"`
	Mode    uint32 `json:"mode"`
}
type settingBackup struct {
	Schema string `json:"schema"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}
type snapshot struct {
	Files    []fileBackup    `json:"files"`
	Settings []settingBackup `json:"settings"`
}

func baseConfigDir() (string, error) {
	if root := os.Getenv("XDG_CONFIG_HOME"); root != "" {
		if !filepath.IsAbs(root) {
			return "", fmt.Errorf("XDG_CONFIG_HOME must be absolute")
		}
		return root, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

func xmlText(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func FontconfigXML(cfg config.Config) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"fonts.dtd\">\n<fontconfig>\n")
	languages := make([]string, 0, len(cfg.LanguageFonts))
	for lang := range cfg.LanguageFonts {
		languages = append(languages, lang)
	}
	sort.Strings(languages)
	for _, lang := range languages {
		kinds := make([]string, 0, len(cfg.LanguageFonts[lang]))
		for kind := range cfg.LanguageFonts[lang] {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
		for _, kind := range kinds {
			alias := map[string]string{"sans": "sans-serif", "serif": "serif", "mono": "monospace"}[kind]
			family := cfg.LanguageFonts[lang][kind]
			b.WriteString(fmt.Sprintf("  <!-- %s / %s (managed by fontctl) -->\n", lang, kind))
			b.WriteString("  <match target=\"pattern\">\n")
			b.WriteString("    <test name=\"lang\" compare=\"contains\"><string>" + xmlText(lang) + "</string></test>\n")
			b.WriteString("    <test name=\"family\" qual=\"any\" compare=\"eq\"><string>" + alias + "</string></test>\n")
			b.WriteString("    <edit name=\"family\" mode=\"prepend\" binding=\"strong\"><string>" + xmlText(family) + "</string></edit>\n")
			b.WriteString("  </match>\n")
		}
	}
	b.WriteString("</fontconfig>\n")
	return b.String()
}

func appendLine(original, line string) string {
	if strings.Contains(original, line) {
		return original
	}
	if original != "" && !strings.HasSuffix(original, "\n") {
		original += "\n"
	}
	return original + line + "\n"
}

// BuildPlan is read-only. It only produces managed overlays and minimal include edits.
func BuildPlan(cfg config.Config) (Plan, error) {
	var p Plan
	if err := cfg.Validate(); err != nil {
		return p, err
	}
	base, err := baseConfigDir()
	if err != nil {
		return p, err
	}
	if len(cfg.LanguageFonts) > 0 {
		p.Files = append(p.Files, FileChange{filepath.Join(base, "fontconfig", "conf.d", "60-fontctl.conf"), FontconfigXML(cfg), "Fontconfig language-aware preferences"})
	}
	if t, ok := cfg.Terminals["kitty"]; ok {
		path := filepath.Join(base, "kitty", "kitty.conf")
		managed := filepath.Join(base, "kitty", "fontctl.conf")
		original, err := readOptional(path)
		if err != nil {
			return p, err
		}
		include := "include " + managed
		p.Files = append(p.Files, FileChange{path, appendLine(original, include), "Kitty include"})
		content := fmt.Sprintf("# Managed by fontctl; edit via fontctl instead\nfont_family %s\nfont_size %.1f\n", t.Family, t.Size)
		p.Files = append(p.Files, FileChange{managed, content, "Kitty font overlay"})
	}
	if t, ok := cfg.Terminals["alacritty"]; ok {
		path := filepath.Join(base, "alacritty", "alacritty.toml")
		managed := filepath.Join(base, "alacritty", "fontctl.toml")
		original, err := readOptional(path)
		if err != nil {
			return p, err
		}
		// An imported file is overridden by duplicate keys in the main config;
		// refuse ambiguous existing font sections instead of claiming success.
		if strings.Contains(original, "[font]") || strings.Contains(original, "[font.normal]") {
			return p, fmt.Errorf("Alacritty has its own [font] settings; remove conflicting font settings or configure Alacritty manually before applying fontctl")
		}
		if strings.Contains(original, "import =") || strings.Contains(original, "import=") || strings.Contains(original, "import\t=") {
			if !strings.Contains(original, managed) {
				return p, fmt.Errorf("Alacritty already has imports: manually add %q to [general].import first", managed)
			}
		} else if strings.Contains(original, "[general]") {
			original = strings.Replace(original, "[general]", "[general]\nimport = ["+strconv.Quote(managed)+"]", 1)
		} else {
			// Append a [general] section to avoid changing any existing TOML root fields.
			if original != "" && !strings.HasSuffix(original, "\n") {
				original += "\n"
			}
			original += "\n[general]\nimport = [" + strconv.Quote(managed) + "]\n"
		}
		p.Files = append(p.Files, FileChange{path, original, "Alacritty import"})
		content := fmt.Sprintf("# Managed by fontctl\n[font]\nsize = %.1f\n[font.normal]\nfamily = %s\n", t.Size, strconv.Quote(t.Family))
		p.Files = append(p.Files, FileChange{managed, content, "Alacritty font overlay"})
	}
	if cfg.GNOME.Interface != "" {
		p.Settings = append(p.Settings, SettingChange{"org.gnome.desktop.interface", "font-name", cfg.GNOME.Interface + " 11", "GNOME interface font"})
	}
	if cfg.GNOME.Monospace != "" {
		p.Settings = append(p.Settings, SettingChange{"org.gnome.desktop.interface", "monospace-font-name", cfg.GNOME.Monospace + " 11", "GNOME monospace font"})
	}
	// Skip identical files; no reason to overwrite their metadata.
	filtered := make([]FileChange, 0, len(p.Files))
	for _, f := range p.Files {
		existing, err := readOptional(f.Path)
		if err != nil {
			return p, err
		}
		if existing != f.Content {
			filtered = append(filtered, f)
		}
	}
	p.Files = filtered
	return p, nil
}

func readOptional(path string) (string, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("refusing non-regular config path: %s", path)
	}
	b, err := os.ReadFile(path)
	return string(b), err
}

func validateFonts(cfg config.Config) error {
	if err := fonts.Available(); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, m := range cfg.LanguageFonts {
		for _, family := range m {
			seen[family] = true
		}
	}
	for _, t := range cfg.Terminals {
		seen[t.Family] = true
	}
	if cfg.GNOME.Interface != "" {
		seen[cfg.GNOME.Interface] = true
	}
	if cfg.GNOME.Monospace != "" {
		seen[cfg.GNOME.Monospace] = true
	}
	for family := range seen {
		ok, err := fonts.Exists(family)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("font %q is not installed; install it first or choose a font from `fontctl fonts`", family)
		}
	}
	return nil
}

func gsettings(args ...string) (string, error) {
	if _, err := exec.LookPath("gsettings"); err != nil {
		return "", fmt.Errorf("GNOME settings require the gsettings command")
	}
	out, err := exec.Command("gsettings", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gsettings %s: %s: %w", strings.Join(args[:min(2, len(args))], " "), strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func capture(p Plan) (snapshot, error) {
	var s snapshot
	for _, f := range p.Files {
		info, err := os.Lstat(f.Path)
		if os.IsNotExist(err) {
			s.Files = append(s.Files, fileBackup{Path: f.Path})
			continue
		}
		if err != nil {
			return s, err
		}
		if !info.Mode().IsRegular() {
			return s, fmt.Errorf("refusing symlink or special file: %s", f.Path)
		}
		b, err := os.ReadFile(f.Path)
		if err != nil {
			return s, err
		}
		s.Files = append(s.Files, fileBackup{Path: f.Path, Existed: true, Data: b, Mode: uint32(info.Mode().Perm())})
	}
	for _, setting := range p.Settings {
		before, err := gsettings("get", setting.Schema, setting.Key)
		if err != nil {
			return s, err
		}
		s.Settings = append(s.Settings, settingBackup{setting.Schema, setting.Key, before})
	}
	return s, nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular target: %s", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".fontctl-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func restoreSnapshot(s snapshot) error {
	var failures []string
	// Restore in reverse file order so managed overlays are restored before imports.
	for i := len(s.Files) - 1; i >= 0; i-- {
		f := s.Files[i]
		if !f.Existed {
			if err := os.Remove(f.Path); err != nil && !os.IsNotExist(err) {
				failures = append(failures, err.Error())
			}
		} else if err := atomicWrite(f.Path, f.Data, os.FileMode(f.Mode)); err != nil {
			failures = append(failures, err.Error())
		}
	}
	for _, v := range s.Settings {
		if _, err := gsettings("set", v.Schema, v.Key, v.Value); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("incomplete restore: %s", strings.Join(failures, "; "))
	}
	return nil
}

func saveSnapshot(s snapshot) (string, error) {
	dir, err := preview.StateDir()
	if err != nil {
		return "", err
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	dir = filepath.Join(dir, "snapshots", id)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	if err := atomicWrite(filepath.Join(dir, "manifest.json"), b, 0600); err != nil {
		return "", err
	}
	return id, nil
}

func latestPath() (string, error) { d, e := preview.StateDir(); return filepath.Join(d, "latest"), e }

// Apply makes a snapshot BEFORE writing anything, and automatically rolls back on failure.
func Apply(cfg config.Config) (string, error) {
	if err := validateFonts(cfg); err != nil {
		return "", err
	}
	p, err := BuildPlan(cfg)
	if err != nil {
		return "", err
	}
	if len(p.Files) == 0 && len(p.Settings) == 0 {
		return "No changes required", nil
	}
	s, err := capture(p)
	if err != nil {
		return "", err
	}
	id, err := saveSnapshot(s)
	if err != nil {
		return "", err
	}
	rollback := func(e error) (string, error) {
		if rerr := restoreSnapshot(s); rerr != nil {
			return "", fmt.Errorf("%v; ROLLBACK FAILED: %v (snapshot %s)", e, rerr, id)
		}
		return "", fmt.Errorf("%w (rolled back)", e)
	}
	for _, f := range p.Files {
		if err := atomicWrite(f.Path, []byte(f.Content), 0644); err != nil {
			return rollback(err)
		}
	}
	for _, setting := range p.Settings {
		if _, err := gsettings("set", setting.Schema, setting.Key, setting.Value); err != nil {
			return rollback(err)
		}
	}
	latest, err := latestPath()
	if err != nil {
		return rollback(err)
	}
	if err := atomicWrite(latest, []byte(id+"\n"), 0600); err != nil {
		return rollback(err)
	}
	return id, nil
}

func RestoreLatest() error {
	latest, err := latestPath()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(latest)
	if err != nil {
		return fmt.Errorf("no previous snapshot: %w", err)
	}
	id := strings.TrimSpace(string(b))
	// ID is numeric and user state folder only; never accept traversal.
	if id == "" || strings.Trim(id, "0123456789") != "" {
		return errors.New("invalid snapshot ID")
	}
	dir, err := preview.StateDir()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "snapshots", id, "manifest.json"))
	if err != nil {
		return err
	}
	var s snapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	if err := restoreSnapshot(s); err != nil {
		return err
	}
	return os.Remove(latest)
}
