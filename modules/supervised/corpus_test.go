package supervised

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"
)

func csvBytes(rows [][]string) []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.WriteAll(rows)
	return b.Bytes()
}

func sampleRecords() [][]string {
	return [][]string{{"#AUTHID", "TEXT", "cEXT", "cNEU", "cAGR", "cCON", "cOPN"}, {"author1", "  I\t write  words.\n\nNew\u00a0 ideas grow. ", "y", "n", "y", "n", "y"}}
}

func TestCorpusNormalizationAndEncoding(t *testing.T) {
	c, err := ParseCorpus(csvBytes(sampleRecords()))
	if err != nil {
		t.Fatal(err)
	}
	if c.Encoding != "UTF-8" || c.RawRows != 1 || !validHash(c.Checksum) || c.Essays[0].Document.RawText != "I write words.\n\nNew ideas grow." {
		t.Fatalf("unexpected normalized metadata: %+v", c)
	}
	if !c.Essays[0].Labels[0] || c.Essays[0].Labels[1] {
		t.Fatal("labels changed")
	}
	_, encoding, err := DecodeCorpus([]byte("don\x92t \x93quoted\x94"))
	if err != nil || encoding != "Windows-1252" {
		t.Fatal(encoding, err)
	}
	text, _, _ := DecodeCorpus([]byte("don\x92t \x93quoted\x94"))
	if text != "don’t “quoted”" {
		t.Fatal(text)
	}
	utf, enc, err := DecodeCorpus([]byte("\ufeffdéjà café"))
	if err != nil || enc != "UTF-8" || utf != "déjà café" {
		t.Fatal(utf, enc, err)
	}
	if _, err := ParseCorpus(append([]byte("\xef\xbb\xbf"), csvBytes(sampleRecords())...)); err != nil {
		t.Fatal(err)
	}
}

func TestCorpusRejectsMalformedRowsWithoutLeakingText(t *testing.T) {
	cases := []struct {
		name   string
		modify func([][]string) [][]string
	}{
		{"missing column", func(r [][]string) [][]string { r[0][2] = "wrong"; return r }},
		{"duplicate header", func(r [][]string) [][]string { r[0][2] = "TEXT"; return r }},
		{"short record", func(r [][]string) [][]string { r[1] = r[1][:6]; return r }},
		{"long record", func(r [][]string) [][]string { r[1] = append(r[1], "extra"); return r }},
		{"blank author", func(r [][]string) [][]string { r[1][0] = " \t"; return r }},
		{"bad label", func(r [][]string) [][]string { r[1][2] = "yes"; return r }},
		{"empty label", func(r [][]string) [][]string { r[1][3] = ""; return r }},
		{"duplicate author", func(r [][]string) [][]string {
			next := append([]string(nil), r[1]...)
			next[1] = "another separate participant essay"
			return append(r, next)
		}},
		{"duplicate normalized text", func(r [][]string) [][]string {
			next := append([]string(nil), r[1]...)
			next[0] = "author2"
			next[1] = "I WRITE WORDS. NEW IDEAS GROW."
			return append(r, next)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseCorpus(csvBytes(c.modify(sampleRecords())))
			if err == nil {
				t.Fatal("accepted malformed corpus")
			}
			if strings.Contains(err.Error(), "participant essay") || strings.Contains(err.Error(), "author1") {
				t.Fatal("participant data leaked in error", err)
			}
		})
	}
	for _, text := range []string{"", " \t\n\u2003\u00a0 ", ".............", "<b></b>", "short"} {
		rows := sampleRecords()
		rows[1][1] = text
		if _, err := ParseCorpus(csvBytes(rows)); err == nil {
			t.Fatalf("accepted invalid text %q", text)
		}
	}
	if _, err := ParseCorpus([]byte("#AUTHID,TEXT\n")); err == nil {
		t.Fatal("accepted empty corpus")
	}
}

func syntheticCorpus(t *testing.T, n int) Corpus {
	t.Helper()
	rows := sampleRecords()[:1]
	for i := 0; i < n; i++ {
		positive := i%3 != 0
		words := strings.Repeat("hello idea ", 2+i%9) + strings.Repeat("work word ", 2+(i*7)%11) + fmt.Sprintf("unique essay number %d", i)
		label := "n"
		if positive {
			label = "y"
		}
		rows = append(rows, []string{fmt.Sprintf("author%d", i), words, label, label, label, label, label})
	}
	c, err := ParseCorpus(csvBytes(rows))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
