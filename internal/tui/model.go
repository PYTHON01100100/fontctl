// Package tui provides a compact font manager UI using Bubble Tea v2.
package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/PYTHON01100100/fontctl/internal/config"
	"github.com/PYTHON01100100/fontctl/internal/engine"
	"github.com/PYTHON01100100/fontctl/internal/fonts"
	"github.com/PYTHON01100100/fontctl/internal/install"
	"github.com/PYTHON01100100/fontctl/internal/preview"
)

var (
	gold  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FABD2F")).Bold(true)
	green = lipgloss.NewStyle().Foreground(lipgloss.Color("#B8BB26"))
	muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#928374"))
	cream = lipgloss.NewStyle().Foreground(lipgloss.Color("#EBDBB2"))
)
var targets = []string{"Language: sans", "Language: serif", "Language: mono", "Terminal: kitty", "Terminal: alacritty", "Desktop: GNOME interface", "Desktop: GNOME monospace"}
var languages = []string{"en", "ar", "ja", "zh-cn", "zh-tw", "ko"}

type resultMsg struct {
	err    error
	desc   string
	reload bool
}
type model struct {
	cfg         config.Config
	all         []string
	filtered    []string
	cursor      int
	langIndex   int
	targetIndex int
	search      string
	inputMode   string // search, file, url, confirm-apply, confirm-restore
	input       string
	sample      string
	status      string
	width       int
	busy        bool
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	list, err := fonts.List("")
	if err != nil {
		return err
	}
	sample, err := preview.NextArabic()
	if err != nil {
		return err
	}
	m := model{cfg: cfg, all: list, filtered: list, sample: sample, status: "Ready. Set a font with Enter, inspect with d, apply with a.", width: 84}
	_, err = tea.NewProgram(m).Run()
	return err
}
func (m model) Init() tea.Cmd { return nil }
func (m *model) refresh() {
	m.filtered = nil
	for _, name := range m.all {
		if strings.Contains(strings.ToLower(name), strings.ToLower(m.search)) {
			m.filtered = append(m.filtered, name)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}
func (m *model) selected() string {
	if len(m.filtered) == 0 {
		return ""
	}
	return m.filtered[m.cursor]
}
func (m *model) stage() error {
	family := m.selected()
	if family == "" {
		return fmt.Errorf("no matching font selected")
	}
	target := targets[m.targetIndex]
	switch m.targetIndex {
	case 0, 1, 2:
		lang := languages[m.langIndex]
		if m.cfg.LanguageFonts[lang] == nil {
			m.cfg.LanguageFonts[lang] = map[string]string{}
		}
		kind := []string{"sans", "serif", "mono"}[m.targetIndex]
		m.cfg.LanguageFonts[lang][kind] = family
	case 3:
		m.cfg.Terminals["kitty"] = config.TerminalSettings{Family: family, Size: 12}
	case 4:
		m.cfg.Terminals["alacritty"] = config.TerminalSettings{Family: family, Size: 12}
	case 5:
		m.cfg.GNOME.Interface = family
	case 6:
		m.cfg.GNOME.Monospace = family
	}
	if err := config.Save(m.cfg); err != nil {
		return err
	}
	m.status = "Staged " + target + " = " + family + " (not yet applied)"
	return nil
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
	case resultMsg:
		m.busy = false
		if v.err != nil {
			m.status = "ERROR: " + v.err.Error()
		} else {
			m.status = v.desc
		}
		if v.reload {
			list, err := fonts.List("")
			if err != nil {
				m.status = "ERROR loading fonts: " + err.Error()
			} else {
				m.all = list
				m.refresh()
			}
		}
	case tea.KeyPressMsg:
		key := v.String()
		if m.busy {
			if key == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.inputMode != "" {
			switch m.inputMode {
			case "confirm-apply", "confirm-restore":
				if key == "y" || key == "Y" {
					action := m.inputMode
					m.inputMode = ""
					m.busy = true
					if action == "confirm-apply" {
						m.status = "Applying settings..."
						return m, func() tea.Msg {
							id, err := engine.Apply(m.cfg)
							return resultMsg{err: err, desc: "Applied successfully. Snapshot: " + id}
						}
					}
					m.status = "Restoring latest snapshot..."
					return m, func() tea.Msg {
						err := engine.RestoreLatest()
						return resultMsg{err: err, desc: "Latest snapshot restored."}
					}
				}
				m.inputMode = ""
				m.status = "Cancelled"
				return m, nil
			default:
				switch key {
				case "esc":
					m.inputMode = ""
					m.input = ""
					m.status = "Cancelled"
					return m, nil
				case "enter":
					input := strings.TrimSpace(m.input)
					mode := m.inputMode
					m.inputMode = ""
					m.input = ""
					if mode == "search" {
						m.search = input
						m.refresh()
						m.status = fmt.Sprintf("Showing %d font families", len(m.filtered))
						return m, nil
					}
					if input == "" {
						m.status = "No path or URL supplied"
						return m, nil
					}
					m.busy = true
					m.status = "Installing font(s)..."
					return m, func() tea.Msg {
						var paths []string
						var err error
						if mode == "url" {
							paths, err = install.Download(input)
						} else {
							paths, err = install.Import(input)
						}
						return resultMsg{err: err, desc: fmt.Sprintf("Installed %d font file(s) locally. Font list refreshed.", len(paths)), reload: true}
					}
				case "backspace":
					if m.input != "" {
						_, size := utf8.DecodeLastRuneInString(m.input)
						m.input = m.input[:len(m.input)-size]
					}
				default:
					if text := v.Key().Text; text != "" {
						m.input += text
					}
				}
				return m, nil
			}
		}
		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor+1 < len(m.filtered) {
				m.cursor++
			}
		case "tab", "t":
			m.targetIndex = (m.targetIndex + 1) % len(targets)
		case "shift+tab":
			m.targetIndex = (m.targetIndex + len(targets) - 1) % len(targets)
		case "l":
			m.langIndex = (m.langIndex + 1) % len(languages)
		case "p":
			sample, err := preview.NextArabic()
			if err != nil {
				m.status = err.Error()
			} else {
				m.sample = sample
				m.status = "Arabic preview rotated"
			}
		case "/":
			m.inputMode = "search"
			m.input = m.search
		case "i":
			m.inputMode = "file"
			m.input = ""
		case "u":
			m.inputMode = "url"
			m.input = ""
		case "esc":
			m.search = ""
			m.refresh()
		case "enter":
			if err := m.stage(); err != nil {
				m.status = "ERROR: " + err.Error()
			}
		case "a":
			m.inputMode = "confirm-apply"
			m.status = "Apply changes? Type y to confirm, any other key to cancel."
		case "r":
			m.inputMode = "confirm-restore"
			m.status = "Restore latest backup? Type y to confirm."
		case "d":
			plan, err := engine.BuildPlan(m.cfg)
			if err != nil {
				m.status = "ERROR: " + err.Error()
			} else {
				m.status = fmt.Sprintf("Dry run: %d file edits, %d GNOME settings. Use `fontctl plan` for details.", len(plan.Files), len(plan.Settings))
			}
		}
	}
	return m, nil
}
func (m model) View() tea.View {
	var b strings.Builder
	b.WriteString(gold.Render("fontctl  /  cross-distro Linux Font Manager"))
	b.WriteString("\n" + muted.Render(strings.Repeat("─", min(65, max(25, m.width-6)))) + "\n")
	b.WriteString(cream.Render(fmt.Sprintf("Target: %s     Language: %s", targets[m.targetIndex], languages[m.langIndex])) + "\n")
	b.WriteString(muted.Render(fmt.Sprintf("Installed families: %d  |  Matching: %d  |  Filter: %q", len(m.all), len(m.filtered), m.search)) + "\n\n")
	begin := max(0, m.cursor-5)
	end := min(len(m.filtered), begin+11)
	b.WriteString(gold.Render("AVAILABLE FONTS") + "\n")
	for i := begin; i < end; i++ {
		line := "  " + m.filtered[i]
		if i == m.cursor {
			line = "❯ " + m.filtered[i]
			b.WriteString(green.Render(line))
		} else {
			b.WriteString(cream.Render(line))
		}
		b.WriteString("\n")
	}
	if len(m.filtered) == 0 {
		b.WriteString(muted.Render("No fonts match. Press / to search again.") + "\n")
	}
	for i := end; i < begin+11; i++ {
		b.WriteString("\n")
	}
	b.WriteString("\n" + gold.Render("PREVIEW / معاينة") + "\n")
	for _, line := range preview.English {
		b.WriteString(cream.Render(line) + "\n")
	}
	b.WriteString(green.Render(m.sample) + "\n")
	b.WriteString(muted.Render("Actual glyph rendering depends on your terminal; previews rotate without changing its font.") + "\n\n")
	if m.inputMode == "file" {
		b.WriteString(gold.Render("LOCAL FONT PATH (.ttf/.otf/.ttc/.zip): ") + m.input + "█\n")
	}
	if m.inputMode == "url" {
		b.WriteString(gold.Render("DIRECT HTTPS FONT URL (.ttf/.otf/.ttc/.zip): ") + m.input + "█\n")
	}
	if m.inputMode == "search" {
		b.WriteString(gold.Render("SEARCH: ") + m.input + "█\n")
	}
	if strings.HasPrefix(m.inputMode, "confirm-") {
		b.WriteString(gold.Render("CONFIRM? [y/N]") + "\n")
	}
	b.WriteString(muted.Render("↑↓ scroll  tab target  l language  / search  enter stage  p rotate preview") + "\n")
	b.WriteString(muted.Render("i import file  u URL download  d dry-run  a apply  r restore  q quit") + "\n")
	b.WriteString("\n" + gold.Render("STATUS: ") + cream.Render(m.status) + "\n")
	return tea.NewView(b.String())
}
