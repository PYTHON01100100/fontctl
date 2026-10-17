// fontctl: a user-local, cross-distro Linux font manager.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PYTHON01100100/fontctl/internal/config"
	"github.com/PYTHON01100100/fontctl/internal/engine"
	"github.com/PYTHON01100100/fontctl/internal/fonts"
	"github.com/PYTHON01100100/fontctl/internal/install"
	"github.com/PYTHON01100100/fontctl/internal/preview"
	"github.com/PYTHON01100100/fontctl/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "fontctl:", err)
		os.Exit(1)
	}
}
func usage() {
	fmt.Print(`fontctl - Linux font manager (Bubble Tea TUI + CLI)

Usage:
  fontctl                              Open the interactive TUI
  fontctl fonts [--lang ar]            List locally installed font families
  fontctl preview [--lang ar|en]       Print samples; rotate Arabic on each call
  fontctl set lang ar --family NAME [--kind sans|serif|mono]
  fontctl set terminal kitty --font NAME [--size 12]
  fontctl set terminal alacritty --font NAME [--size 12]
  fontctl set desktop gnome [--interface NAME] [--monospace NAME]
  fontctl plan                         Show file/settings edits before applying
  fontctl apply [--dry-run]            Write user-only settings; create backup
  fontctl restore latest               Restore last applied snapshot
  fontctl import /path/Font.zip        Import a manually downloaded font/ZIP
  fontctl install --url https://...    Download user-provided direct font URL
  fontctl match --lang ar [--kind sans]
  fontctl doctor                       Print dependency and environment status
  fontctl help

No sudo is required. No font binaries are bundled. URL installation is opt-in.
`)
}
func run(args []string) error {
	if len(args) == 0 {
		return tui.Run()
	}
	switch args[0] {
	case "help", "-h", "--help":
		usage()
		return nil
	case "fonts", "list":
		fs := flag.NewFlagSet("fonts", flag.ContinueOnError)
		lang := fs.String("lang", "", "Language tag (e.g. ar, ja)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		list, err := fonts.List(*lang)
		if err != nil {
			return err
		}
		for _, family := range list {
			fmt.Println(family)
		}
		fmt.Fprintf(os.Stderr, "%d font families\n", len(list))
		return nil
	case "preview":
		fs := flag.NewFlagSet("preview", flag.ContinueOnError)
		lang := fs.String("lang", "both", "en, ar or both")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *lang != "en" && *lang != "ar" && *lang != "both" {
			return fmt.Errorf("invalid preview language: %s", *lang)
		}
		if *lang == "en" || *lang == "both" {
			for _, line := range preview.English {
				fmt.Println(line)
			}
		}
		if *lang == "ar" || *lang == "both" {
			s, err := preview.NextArabic()
			if err != nil {
				return err
			}
			fmt.Println(s)
		}
		return nil
	case "set":
		return set(args[1:])
	case "plan":
		if len(args) > 1 {
			return fmt.Errorf("plan takes no arguments")
		}
		return plan()
	case "apply":
		fs := flag.NewFlagSet("apply", flag.ContinueOnError)
		dry := fs.Bool("dry-run", false, "Show changes without writing")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *dry {
			return plan()
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		id, err := engine.Apply(cfg)
		if err != nil {
			return err
		}
		fmt.Println("Applied:", id)
		fmt.Println("Restart affected applications if fonts do not refresh immediately.")
		return nil
	case "restore":
		if len(args) != 2 || args[1] != "latest" {
			return fmt.Errorf("usage: fontctl restore latest")
		}
		if err := engine.RestoreLatest(); err != nil {
			return err
		}
		fmt.Println("Restored previous settings")
		return nil
	case "import":
		if len(args) != 2 {
			return fmt.Errorf("usage: fontctl import /path/font.ttf|otf|ttc|zip")
		}
		results, err := install.Import(args[1:][0])
		for _, p := range results {
			fmt.Println("Installed:", p)
		}
		return err
	case "install":
		fs := flag.NewFlagSet("install", flag.ContinueOnError)
		url := fs.String("url", "", "Explicit direct HTTPS download URL")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *url == "" {
			return fmt.Errorf("usage: fontctl install --url https://example.com/font.zip")
		}
		results, err := install.Download(*url)
		for _, p := range results {
			fmt.Println("Installed:", p)
		}
		return err
	case "match":
		fs := flag.NewFlagSet("match", flag.ContinueOnError)
		lang := fs.String("lang", "ar", "Language tag")
		kind := fs.String("kind", "sans", "sans, serif or mono")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if !config.Languages[*lang] || !config.Kinds[*kind] {
			return fmt.Errorf("unsupported language or category")
		}
		family := map[string]string{"sans": "sans-serif", "serif": "serif", "mono": "monospace"}[*kind]
		result, err := fonts.Match(family + ":lang=" + *lang)
		if err != nil {
			return err
		}
		fmt.Println(result)
		return nil
	case "doctor":
		return doctor()
	default:
		return fmt.Errorf("unknown command %q (try fontctl help)", args[0])
	}
}
func set(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: fontctl set lang|terminal|desktop TARGET --flags")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	category, target := args[0], args[1]
	switch category {
	case "lang":
		if !config.Languages[target] {
			return fmt.Errorf("unsupported language: %s", target)
		}
		fs := flag.NewFlagSet("set lang", flag.ContinueOnError)
		family := fs.String("family", "", "Installed font family")
		kind := fs.String("kind", "sans", "sans, serif or mono")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if !config.Kinds[*kind] || *family == "" {
			return fmt.Errorf("set lang requires --family and valid --kind")
		}
		if cfg.LanguageFonts[target] == nil {
			cfg.LanguageFonts[target] = map[string]string{}
		}
		cfg.LanguageFonts[target][*kind] = *family
	case "terminal":
		if target != "kitty" && target != "alacritty" {
			return fmt.Errorf("terminal %q is not supported in v0.1", target)
		}
		fs := flag.NewFlagSet("set terminal", flag.ContinueOnError)
		family := fs.String("font", "", "Installed font family")
		size := fs.Float64("size", 12, "Font point size")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *family == "" {
			return fmt.Errorf("--font is required")
		}
		cfg.Terminals[target] = config.TerminalSettings{Family: *family, Size: *size}
	case "desktop":
		if target != "gnome" {
			return fmt.Errorf("desktop %q is not supported in v0.1", target)
		}
		fs := flag.NewFlagSet("set desktop", flag.ContinueOnError)
		iface := fs.String("interface", "", "Interface font family")
		mono := fs.String("monospace", "", "Monospace family")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *iface == "" && *mono == "" {
			return fmt.Errorf("specify --interface and/or --monospace")
		}
		if *iface != "" {
			cfg.GNOME.Interface = *iface
		}
		if *mono != "" {
			cfg.GNOME.Monospace = *mono
		}
	default:
		return fmt.Errorf("invalid set category: %s", category)
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Println("Preference saved (not applied). Inspect with `fontctl plan`, then `fontctl apply`.")
	return nil
}
func plan() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	p, err := engine.BuildPlan(cfg)
	if err != nil {
		return err
	}
	if len(p.Files) == 0 && len(p.Settings) == 0 {
		fmt.Println("No pending file changes. GNOME values (if configured) are reapplied on apply.")
		return nil
	}
	for _, f := range p.Files {
		fmt.Printf("FILE: %s\n  %s\n", f.Path, f.Description)
	}
	for _, s := range p.Settings {
		fmt.Printf("GNOME: %s %s -> %s\n", s.Schema, s.Key, s.Value)
	}
	fmt.Printf("Plan: %d file writes, %d GNOME setting updates\n", len(p.Files), len(p.Settings))
	return nil
}
func doctor() error {
	for _, bin := range []string{"fc-list", "fc-match", "fc-cache", "gsettings", "kitty", "alacritty"} {
		if path, err := exec.LookPath(bin); err == nil {
			fmt.Printf("[OK] %-13s %s\n", bin, path)
		} else {
			fmt.Printf("[--] %-13s not available\n", bin)
		}
	}
	fmt.Println("XDG_CURRENT_DESKTOP:", strings.TrimSpace(os.Getenv("XDG_CURRENT_DESKTOP")))
	cfg, err := config.Path()
	if err != nil {
		return err
	}
	fmt.Println("Configuration:", cfg)
	fontsDir, err := install.Dir()
	if err != nil {
		return err
	}
	fmt.Println("Manual font directory:", fontsDir)
	fmt.Println("Note: Terminal font fallback and Arabic RTL shaping vary between emulators.")
	return nil
}
