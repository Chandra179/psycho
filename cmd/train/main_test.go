package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"psycho/modules/supervised"
)

func TestCommandWritesReproducibleLocalResultsAndClearsStaleModel(t *testing.T) {
	dir := t.TempDir()
	csvPath, dictPath, outDir := filepath.Join(dir, "essays.csv"), filepath.Join(dir, "dictionary.json"), filepath.Join(dir, "results")
	var csvData bytes.Buffer
	writer := csv.NewWriter(&csvData)
	_ = writer.Write([]string{"#AUTHID", "TEXT", "cEXT", "cNEU", "cAGR", "cCON", "cOPN"})
	for i := 0; i < 100; i++ {
		label := "n"
		if i%3 != 0 {
			label = "y"
		}
		text := strings.Repeat("hello ", 2+i%9) + strings.Repeat("word ", 2+i%11) + fmt.Sprintf("unique essay %d", i)
		_ = writer.Write([]string{fmt.Sprint(i), text, label, label, label, label, label})
	}
	writer.Flush()
	if err := os.WriteFile(csvPath, csvData.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	dict := []byte(`{"word":["word"],"hello":["hello"]}`)
	if err := os.WriteFile(dictPath, dict, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-csv", csvPath, "-dictionary", dictPath, "-out", outDir, "-min-words", "1", "-resamples", "20"}
	if err := run(args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(outDir, "model.json")
	if _, err := supervised.LoadArtifact(modelPath, dict); err != nil {
		t.Fatal(err)
	}
	before := map[string][]byte{}
	for _, name := range []string{"model.json", "report.json", "report.md"} {
		data, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatal(err)
		}
		before[name] = data
	}
	if err := run(args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for name, data := range before {
		after, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil || !bytes.Equal(data, after) {
			t.Fatal("command output changed", name, err)
		}
	}
	if err := os.WriteFile(csvPath, []byte("bad,columns\n1,2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(args, &bytes.Buffer{}); err == nil {
		t.Fatal("accepted malformed input")
	}
	if _, err := os.Stat(modelPath); !os.IsNotExist(err) {
		t.Fatal("stale model survived failed input", err)
	}
}
