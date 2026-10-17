package engine

import (
	"encoding/xml"
	"github.com/PYTHON01100100/fontctl/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFontconfigXML(t *testing.T) {
	c := config.Default()
	c.LanguageFonts["ar"] = map[string]string{"sans": "Noto Sans Arabic"}
	c.LanguageFonts["ja"] = map[string]string{"mono": "Noto Sans Mono CJK JP"}
	raw := FontconfigXML(c)
	d := xml.NewDecoder(strings.NewReader(raw))
	for {
		_, err := d.Token()
		if err != nil {
			if err.Error() != "EOF" {
				t.Fatal(err)
			}
			break
		}
	}
	if !strings.Contains(raw, "Noto Sans Arabic") {
		t.Fatal("missing Arabic")
	}
}
func TestKittyPlanOnlyManagedFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c := config.Default()
	c.Terminals["kitty"] = config.TerminalSettings{Family: "DejaVu Sans Mono", Size: 12}
	p, err := BuildPlan(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Files) != 2 {
		t.Fatalf("expected include and overlay, got %d", len(p.Files))
	}
	if _, err := os.Stat(p.Files[0].Path); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote file: %s", p.Files[0].Path)
	}
}
func TestNoSymlinkOverwrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(link, []byte("changed"), 0600); err == nil {
		t.Fatal("should refuse symlink")
	}
}
