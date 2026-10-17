// Package fonts discovers fonts using Fontconfig rather than distro-specific APIs.
package fonts

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

func Available() error {
	if _, err := exec.LookPath("fc-list"); err != nil {
		return fmt.Errorf("fontconfig is required: install fontconfig (fc-list)")
	}
	return nil
}

func List(lang string) ([]string, error) {
	if err := Available(); err != nil {
		return nil, err
	}
	if lang != "" {
		// Lang should come from config's supported-language allowlist.
		switch lang {
		case "ar", "en", "ja", "zh-cn", "zh-tw", "ko":
		default:
			return nil, fmt.Errorf("unsupported language %q", lang)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	args := []string{"-f", "%{family}\n"}
	if lang != "" {
		args = append([]string{":lang=" + lang}, args...)
	}
	out, err := exec.CommandContext(ctx, "fc-list", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("fc-list: %w", err)
	}
	names := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 4096), 1024*1024)
	for sc.Scan() {
		for _, family := range strings.Split(sc.Text(), ",") {
			family = strings.TrimSpace(family)
			if family != "" {
				names[family] = true
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	list := make([]string, 0, len(names))
	for family := range names {
		list = append(list, family)
	}
	sort.Slice(list, func(i, j int) bool { return strings.ToLower(list[i]) < strings.ToLower(list[j]) })
	return list, nil
}

func Exists(family string) (bool, error) {
	list, err := List("")
	if err != nil {
		return false, err
	}
	for _, f := range list {
		if strings.EqualFold(f, family) {
			return true, nil
		}
	}
	return false, nil
}

func Match(pattern string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "fc-match", "-f", "%{family}\n", pattern).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
