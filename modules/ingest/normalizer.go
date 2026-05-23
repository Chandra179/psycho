package ingest

import (
	"strings"
	"unicode"
)

// Document holds normalized text with metadata.
type Document struct {
	RawText     string
	WordCount   int
	SentenceCount int
	ParagraphCount int
	TypeTokenRatio float64
}

// Normalizer cleans raw text into a standardized form.
type Normalizer struct{}

func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// Normalize strips markup, normalizes whitespace, and computes basic stats.
func (n *Normalizer) Normalize(raw string) Document {
	// Simple HTML stripping: remove tags
	clean := stripHTMLTags(raw)
	// Normalize whitespace
	clean = normalizeWhitespace(clean)
	// Compute stats
	words := tokenizeWords(clean)
	wordCount := len(words)
	sentences := countSentences(clean)
	paragraphs := countParagraphs(clean)
	uniqueWords := uniqueWordCount(words)
	var ttr float64
	if wordCount > 0 {
		ttr = float64(uniqueWords) / float64(wordCount)
	}
	return Document{
		RawText:        clean,
		WordCount:      wordCount,
		SentenceCount:  sentences,
		ParagraphCount: paragraphs,
		TypeTokenRatio: ttr,
	}
}

func stripHTMLTags(s string) string {
	runes := []rune(s)
	var result strings.Builder
	inTag := false
	for i, r := range runes {
		if r == '<' && i+1 < len(runes) && (unicode.IsLetter(runes[i+1]) || runes[i+1] == '/') {
			inTag = true
			continue
		}
		if r == '>' && inTag {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	// Group consecutive non-blank lines into paragraphs separated by blank lines.
	var paragraphs []string
	var current strings.Builder
	for _, line := range lines {
		if line == "" {
			if current.Len() > 0 {
				paragraphs = append(paragraphs, current.String())
				current.Reset()
			}
			continue
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		paragraphs = append(paragraphs, current.String())
	}
	return strings.Join(paragraphs, "\n\n")
}

func tokenizeWords(s string) []string {
	var words []string
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '\''
	})
	for _, w := range fields {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" {
			words = append(words, w)
		}
	}
	return words
}

func countSentences(s string) int {
	count := 0
	for _, r := range s {
		if r == '.' || r == '!' || r == '?' {
			count++
		}
	}
	if count == 0 && len(strings.TrimSpace(s)) > 0 {
		return 1
	}
	return count
}

func countParagraphs(s string) int {
	parts := strings.Split(s, "\n\n")
	count := 0
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			count++
		}
	}
	if count == 0 && len(strings.TrimSpace(s)) > 0 {
		return 1
	}
	return count
}

func uniqueWordCount(words []string) int {
	seen := make(map[string]struct{}, len(words))
	for _, w := range words {
		seen[w] = struct{}{}
	}
	return len(seen)
}
