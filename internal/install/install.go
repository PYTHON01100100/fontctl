// Package install imports user-supplied fonts from local files or HTTPS URLs.
// No font binary is bundled or silently downloaded by fontctl.
package install

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxDownload = 64 << 20
const maxFontSize = 32 << 20
const maxFonts = 80
const maxExpanded = 128 << 20

func Dir() (string, error) {
	if p := os.Getenv("XDG_DATA_HOME"); p != "" {
		if !filepath.IsAbs(p) {
			return "", errors.New("XDG_DATA_HOME must be an absolute path")
		}
		return filepath.Join(p, "fonts", "fontctl"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".local", "share", "fonts", "fontctl"), nil
}

// Import copies verified fonts to a user-local font directory. Archives are not extracted to disk.
// ZIP entry paths are discarded; every output name is derived from its safe basename and digest.
func Import(path string) ([]string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("source must be a regular file (not a symlink): %s", path)
	}
	if info.Size() > maxDownload {
		return nil, errors.New("source is over the 64 MiB limit")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ImportBytes(filepath.Base(path), data)
}

// ImportBytes handles a local archive or verified font data. Bytes must come from an explicit user action.
func ImportBytes(filename string, data []byte) ([]string, error) {
	if len(data) > maxDownload {
		return nil, errors.New("archive or font exceeds 64 MiB")
	}
	var sources []struct {
		name string
		data []byte
	}
	if strings.EqualFold(filepath.Ext(filename), ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, fmt.Errorf("invalid zip archive: %w", err)
		}
		if len(zr.File) > 3000 {
			return nil, errors.New("too many ZIP entries")
		}
		var total uint64
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			if f.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if !isSupportedName(f.Name) {
				continue
			}
			if len(sources) >= maxFonts {
				return nil, errors.New("ZIP contains more than 80 font files")
			}
			if f.UncompressedSize64 > maxFontSize {
				return nil, errors.New("ZIP contains oversized font")
			}
			total += f.UncompressedSize64
			if total > maxExpanded {
				return nil, errors.New("ZIP expands to over 128 MiB")
			}
			r, err := f.Open()
			if err != nil {
				return nil, err
			}
			b, err := io.ReadAll(io.LimitReader(r, maxFontSize+1))
			closeErr := r.Close()
			if err != nil {
				return nil, err
			}
			if closeErr != nil {
				return nil, closeErr
			}
			if len(b) > maxFontSize {
				return nil, errors.New("font exceeds 32 MiB limit")
			}
			if !fontSignatureOK(f.Name, b) {
				return nil, fmt.Errorf("invalid font header: %s", f.Name)
			}
			sources = append(sources, struct {
				name string
				data []byte
			}{filepath.Base(f.Name), b})
		}
	} else {
		if !isSupportedName(filename) || !fontSignatureOK(filename, data) {
			return nil, errors.New("supported formats are valid TTF, OTF and TTC files, or a ZIP containing them")
		}
		sources = append(sources, struct {
			name string
			data []byte
		}{filepath.Base(filename), data})
	}
	if len(sources) == 0 {
		return nil, errors.New("no TTF, OTF or TTC font files found")
	}
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	var installed []string
	// Keep previously installed files intact if a later one fails; report partial success to caller.
	for _, font := range sources {
		digest := sha256.Sum256(font.data)
		stem := safeName(strings.TrimSuffix(font.name, filepath.Ext(font.name)))
		target := filepath.Join(dir, fmt.Sprintf("%s-%x%s", stem, digest[:6], strings.ToLower(filepath.Ext(font.name))))
		if err := put(target, font.data); err != nil {
			return installed, fmt.Errorf("installed %d file(s) before failure: %w", len(installed), err)
		}
		installed = append(installed, target)
	}
	if bin, err := exec.LookPath("fc-cache"); err == nil {
		// Cache generation may fail in sandbox; installed files are still present.
		if out, e := exec.Command(bin, "-f", dir).CombinedOutput(); e != nil {
			return installed, fmt.Errorf("installed %d file(s), but fc-cache failed: %s (%w)", len(installed), strings.TrimSpace(string(out)), e)
		}
	}
	return installed, nil
}

func safeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
		if b.Len() >= 60 {
			break
		}
	}
	n := strings.Trim(b.String(), "-_")
	if n == "" {
		return "font"
	}
	return n
}
func isSupportedName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".ttf", ".otf", ".ttc":
		return true
	}
	return false
}
func fontSignatureOK(name string, b []byte) bool {
	if len(b) < 4 {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".ttf":
		return bytes.Equal(b[:4], []byte{0, 1, 0, 0}) || string(b[:4]) == "true"
	case ".otf":
		return string(b[:4]) == "OTTO"
	case ".ttc":
		return string(b[:4]) == "ttcf"
	}
	return false
}
func put(path string, data []byte) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsafe existing target: %s", path)
		}
		existing, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Equal(existing, data) {
			return nil
		}
		return fmt.Errorf("font already exists with differing content: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".fontctl-font-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0644); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

// Download accepts only explicit HTTPS direct-download URLs (not arbitrary web pages).
// Redirects to non-HTTPS addresses and oversized responses are rejected.
func Download(rawURL string) ([]string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" || u.Host == "" || u.User != nil {
		return nil, errors.New("a direct HTTPS font file or ZIP URL is required")
	}
	if !isSupportedName(u.Path) && !strings.EqualFold(filepath.Ext(u.Path), ".zip") {
		return nil, errors.New("URL must end in .ttf, .otf, .ttc or .zip (download pages are not supported)")
	}
	client := http.Client{Timeout: 40 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" {
			return errors.New("refusing redirect to non-HTTPS address")
		}
		return nil
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fontctl/0.1 (+manual-font-download)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d while downloading", resp.StatusCode)
	}
	if resp.ContentLength > maxDownload {
		return nil, errors.New("download is larger than 64 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownload {
		return nil, errors.New("download is larger than 64 MiB")
	}
	return ImportBytes(filepath.Base(u.Path), data)
}
