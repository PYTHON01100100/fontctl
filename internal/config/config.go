// Package config handles user-only settings. It never writes system-wide settings.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TerminalSettings struct {
	Family string  `json:"family"`
	Size   float64 `json:"size"`
}

type DesktopSettings struct {
	Interface string `json:"interface,omitempty"`
	Monospace string `json:"monospace,omitempty"`
}

type Config struct {
	LanguageFonts map[string]map[string]string `json:"language_fonts"`
	Terminals     map[string]TerminalSettings  `json:"terminals"`
	GNOME         DesktopSettings              `json:"gnome"`
}

func Default() Config {
	return Config{LanguageFonts: map[string]map[string]string{}, Terminals: map[string]TerminalSettings{}}
}

func Dir() (string, error) {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		if !filepath.IsAbs(d) {
			return "", fmt.Errorf("XDG_CONFIG_HOME must be absolute")
		}
		return filepath.Join(d, "fontctl"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "fontctl"), nil
}

func Path() (string, error) { d, err := Dir(); return filepath.Join(d, "config.json"), err }

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Default(), err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("config: %w", err)
	}
	if cfg.LanguageFonts == nil {
		cfg.LanguageFonts = map[string]map[string]string{}
	}
	if cfg.Terminals == nil {
		cfg.Terminals = map[string]TerminalSettings{}
	}
	return cfg, nil
}

func Save(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, "config.json"))
}

var Languages = map[string]bool{"en": true, "ar": true, "ja": true, "zh-cn": true, "zh-tw": true, "ko": true}
var Kinds = map[string]bool{"sans": true, "serif": true, "mono": true}

func CleanFamily(f string) (string, error) {
	f = strings.TrimSpace(f)
	if f == "" || len(f) > 160 {
		return "", fmt.Errorf("font family must be 1–160 bytes")
	}
	if strings.ContainsAny(f, "\n\r\x00<>") {
		return "", fmt.Errorf("font family contains invalid characters")
	}
	return f, nil
}

func (c Config) Validate() error {
	for lang, kinds := range c.LanguageFonts {
		if !Languages[lang] {
			return fmt.Errorf("unsupported language %q", lang)
		}
		for kind, family := range kinds {
			if !Kinds[kind] {
				return fmt.Errorf("unsupported font category %q", kind)
			}
			if _, err := CleanFamily(family); err != nil {
				return err
			}
		}
	}
	for term, s := range c.Terminals {
		if term != "kitty" && term != "alacritty" {
			return fmt.Errorf("unsupported terminal %q", term)
		}
		if _, err := CleanFamily(s.Family); err != nil {
			return err
		}
		if s.Size < 6 || s.Size > 72 {
			return fmt.Errorf("font size must be between 6 and 72")
		}
	}
	for _, family := range []string{c.GNOME.Interface, c.GNOME.Monospace} {
		if family != "" {
			if _, err := CleanFamily(family); err != nil {
				return err
			}
		}
	}
	return nil
}
