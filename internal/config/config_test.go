package config

import "testing"

func TestRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c := Default()
	c.LanguageFonts["ar"] = map[string]string{"sans": "Noto Sans Arabic"}
	if err := Save(c); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.LanguageFonts["ar"]["sans"] != "Noto Sans Arabic" {
		t.Fatalf("bad round trip: %#v", got)
	}
}
func TestInvalidFamily(t *testing.T) {
	c := Default()
	c.Terminals["kitty"] = TerminalSettings{Family: "bad\ninclude hack", Size: 12}
	if err := c.Validate(); err == nil {
		t.Fatal("expected rejection")
	}
}
