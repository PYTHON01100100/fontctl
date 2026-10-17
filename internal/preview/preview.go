// Package preview contains exact-language samples and an Arabic sample rotation.
package preview

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var English = []string{
	"Hello World My friend",
	"one cli to rule them all",
	"I am the king of the world",
}

var Arabic = []string{
	"صِفْ خَلْقَ خَوْدٍ كَمِثْلِ الشَّمْسِ إِذْ بَزَغَتْ يَحْظَى ضَجِيعٌ بِهَا نَجْلَاءَ مِعْطَارِ",
	"بَلاغَةُ الحَرْفِ تُغْنِي عَنِ الكَلامِ",
	"العَرَبِيَّةُ: أَصَالَةُ المَاضِي وَأُفُقُ المُسْتَقْبَلِ",
}

func StateDir() (string, error) {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		if !filepath.IsAbs(d) {
			return "", fmt.Errorf("XDG_STATE_HOME must be absolute")
		}
		return filepath.Join(d, "fontctl"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "fontctl"), nil
}

// NextArabic persists the next index. It cycles 0 -> 1 -> 2 across CLI launches.
func NextArabic() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "preview.json")
	var state struct {
		Next int `json:"next"`
	}
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return "", fmt.Errorf("invalid preview state: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	i := state.Next % len(Arabic)
	if i < 0 {
		i = 0
	}
	state.Next = (i + 1) % len(Arabic)
	data, err = json.Marshal(state)
	if err != nil {
		return "", err
	}
	// Preserve previous state if writing fails; no root permissions required.
	tmp, err := os.CreateTemp(dir, ".preview-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}
	return Arabic[i], nil
}
