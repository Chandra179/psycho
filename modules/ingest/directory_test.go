package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectoryReaderBoundsBytes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), []byte(strings.Repeat("a", 100000)), 0600); err != nil {
		t.Fatal(err)
	}
	text, files, err := ReadDir(dir, 100)
	if err != nil || files != 1 || len(text) != 101 {
		t.Fatalf("reader must retain only limit + 1 bytes: %d, %d, %v", len(text), files, err)
	}
}
