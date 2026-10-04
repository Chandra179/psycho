// Package supervised provides offline training and evaluation. It is not wired
// into the production analysis pipeline.
package supervised

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"psycho/modules/ingest"
)

type Trait struct{ Column, Name string }

// Traits returns a fresh, ordered list; callers cannot mutate shared state.
func Traits() []Trait {
	return []Trait{{"cEXT", "extraversion"}, {"cNEU", "neuroticism"}, {"cAGR", "agreeableness"}, {"cCON", "conscientiousness"}, {"cOPN", "openness"}}
}

type Essay struct {
	Author   string
	Text     string // decoded source, retained only in memory for legacy evaluation
	Document ingest.Document
	Labels   [5]bool
}

type Corpus struct {
	Essays   []Essay
	Checksum string
	Encoding string
	RawRows  int
}

// DecodeCorpus detects UTF-8, including a BOM, before falling back to CP1252.
func DecodeCorpus(raw []byte) (string, string, error) {
	if utf8.Valid(raw) {
		return strings.TrimPrefix(string(raw), "\ufeff"), "UTF-8", nil
	}
	decoded, err := charmap.Windows1252.NewDecoder().Bytes(raw)
	return string(decoded), "Windows-1252", err
}

func ReadCorpus(path string) (Corpus, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Corpus{}, fmt.Errorf("read corpus: %w", err)
	}
	return ParseCorpus(raw)
}

// ParseCorpus validates every row before filtering. Errors include row numbers
// and column names, never participant text or identifiers.
func ParseCorpus(raw []byte) (Corpus, error) {
	text, encoding, err := DecodeCorpus(raw)
	if err != nil {
		return Corpus{}, fmt.Errorf("decode corpus: %w", err)
	}
	r := csv.NewReader(strings.NewReader(text))
	records, err := r.ReadAll() // FieldsPerRecord=0 enforces header width.
	if err != nil {
		return Corpus{}, fmt.Errorf("parse corpus: %w", err)
	}
	if len(records) < 2 {
		return Corpus{}, fmt.Errorf("corpus needs a header and data rows")
	}
	columns := make(map[string]int)
	for i, name := range records[0] {
		name = strings.TrimPrefix(strings.TrimSpace(name), "#")
		if name == "" {
			return Corpus{}, fmt.Errorf("empty column name")
		}
		if _, exists := columns[name]; exists {
			return Corpus{}, fmt.Errorf("duplicate column %s", name)
		}
		columns[name] = i
	}
	required := []string{"AUTHID", "TEXT"}
	for _, trait := range Traits() {
		required = append(required, trait.Column)
	}
	for _, name := range required {
		if _, ok := columns[name]; !ok {
			return Corpus{}, fmt.Errorf("missing column %s", name)
		}
	}
	checksum := sha256.Sum256(raw)
	corpus := Corpus{Checksum: hex.EncodeToString(checksum[:]), Encoding: encoding, RawRows: len(records) - 1}
	authors, texts := map[string]bool{}, map[string]bool{}
	normalizer := ingest.NewNormalizer()
	for i, row := range records[1:] {
		author := strings.TrimSpace(row[columns["AUTHID"]])
		if author == "" {
			return Corpus{}, fmt.Errorf("row %d: blank AUTHID", i+2)
		}
		if authors[author] {
			return Corpus{}, fmt.Errorf("row %d: duplicate author", i+2)
		}
		authors[author] = true
		doc := normalizer.Normalize(row[columns["TEXT"]])
		if err := ingest.ValidateDocument(doc); err != nil {
			return Corpus{}, fmt.Errorf("row %d: invalid TEXT: %w", i+2, err)
		}
		// Ignore case and paragraph/space differences when detecting duplicates.
		key := strings.ToLower(strings.Join(strings.Fields(doc.RawText), " "))
		if texts[key] {
			return Corpus{}, fmt.Errorf("row %d: duplicate normalized essay", i+2)
		}
		texts[key] = true
		essay := Essay{Author: author, Text: row[columns["TEXT"]], Document: doc}
		for j, trait := range Traits() {
			label := strings.ToLower(strings.TrimSpace(row[columns[trait.Column]]))
			if label != "y" && label != "n" {
				return Corpus{}, fmt.Errorf("row %d: %s must be y or n", i+2, trait.Column)
			}
			essay.Labels[j] = label == "y"
		}
		corpus.Essays = append(corpus.Essays, essay)
	}
	return corpus, nil
}
