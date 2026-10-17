package install

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleTTF() []byte { return append([]byte{0, 1, 0, 0}, bytes.Repeat([]byte{0}, 64)...) }
func TestImportTTF(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	result, err := ImportBytes("my-font.ttf", sampleTTF())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("got %d files", len(result))
	}
	if !strings.HasPrefix(result[0], os.Getenv("XDG_DATA_HOME")) {
		t.Fatalf("bad destination: %s", result[0])
	}
	if _, err := os.Stat(result[0]); err != nil {
		t.Fatal(err)
	}
}
func TestZipNestedPathsSafe(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, e := w.Create("../../unsafe.ttf")
	if e != nil {
		t.Fatal(e)
	}
	f.Write(sampleTTF())
	w.Close()
	result, err := ImportBytes("bundle.zip", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || filepath.Base(result[0]) == "unsafe.ttf" {
		t.Fatalf("unexpected output: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("XDG_DATA_HOME"), "unsafe.ttf")); !os.IsNotExist(err) {
		t.Fatal("escaped destination")
	}
}
func TestRejectFakeFont(t *testing.T) {
	if _, err := ImportBytes("fake.ttf", []byte("html not a font")); err == nil {
		t.Fatal("accepted fake font")
	}
}
func TestRejectHTTP(t *testing.T) {
	if _, err := Download("http://example.org/test.ttf"); err == nil {
		t.Fatal("accepted HTTP")
	}
}
