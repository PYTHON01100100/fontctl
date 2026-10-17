package preview

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArabicRotation(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for i := 0; i < 6; i++ {
		s, err := NextArabic()
		if err != nil {
			t.Fatal(err)
		}
		if s != Arabic[i%3] {
			t.Fatalf("at %d got %q", i, s)
		}
	}
}
func TestNoStateOutsideXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	_, _ = NextArabic()
	if _, err := os.Stat(filepath.Join(dir, "fontctl", "preview.json")); err != nil {
		t.Fatal(err)
	}
}
