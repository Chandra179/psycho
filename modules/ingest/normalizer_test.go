package ingest

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizedTextValidation(t *testing.T) {
	n := NewNormalizer()
	for _, text := range []string{"", "          ", "\t\t\t", "\n\n\n", "\u00a0\u2003\u202f\u3000", "...!!!???---", "<p> </p><div>\t</div>", "ééééééééé"} {
		if err := ValidateDocument(n.Normalize(text)); !errors.Is(err, ErrInvalidText) {
			t.Errorf("accepted invalid text %q: %v", text, err)
		}
	}
	for _, text := range []string{"abcdefghij", "éééééééééé", "1234567890", "  This\t is   valid\n\nAnother paragraph.  "} {
		if err := ValidateDocument(n.Normalize(text)); err != nil {
			t.Errorf("rejected valid text %q: %v", text, err)
		}
	}
	doc := n.Normalize(" \u2003first\t\t line \r\nsecond  line\r\n \t\r\n next\u00a0\u00a0paragraph ")
	if doc.RawText != "first line second line\n\nnext paragraph" || doc.ParagraphCount != 2 {
		t.Fatalf("whitespace/paragraph handling: %+v", doc)
	}
}

func TestStripHTMLTags(t *testing.T) {
	html := "<p>Hello <b>world</b>!</p>"
	got := stripHTMLTags(html)
	want := "Hello world!"
	if got != want {
		t.Errorf("stripHTMLTags(%q) = %q; want %q", html, got, want)
	}
}

func TestNormalizeWhitespacePreservesParagraphs(t *testing.T) {
	input := "Hello\nworld\n\nThis is a test\n\nGoodbye"
	got := normalizeWhitespace(input)
	// Paragraph breaks (blank lines) preserved as double newline
	want := "Hello world\n\nThis is a test\n\nGoodbye"
	if got != want {
		t.Errorf("normalizeWhitespace = %q; want %q", got, want)
	}
}

func TestNormalizer(t *testing.T) {
	raw := "Hello world. This is a test. Goodbye!"
	n := NewNormalizer()
	doc := n.Normalize(raw)

	if doc.WordCount != 7 {
		t.Errorf("WordCount = %d; want 7", doc.WordCount)
	}
	if doc.SentenceCount != 3 {
		t.Errorf("SentenceCount = %d; want 3", doc.SentenceCount)
	}
	if doc.ParagraphCount != 1 {
		t.Errorf("ParagraphCount = %d; want 1", doc.ParagraphCount)
	}
	if doc.TypeTokenRatio != 1.0 {
		t.Errorf("TypeTokenRatio = %f; want 1.0", doc.TypeTokenRatio)
	}
}

func TestNormalizerUniqueWords(t *testing.T) {
	raw := "hello hello world world test"
	n := NewNormalizer()
	doc := n.Normalize(raw)
	if doc.WordCount != 5 {
		t.Fatalf("WordCount = %d; want 5", doc.WordCount)
	}
	if doc.TypeTokenRatio != 0.6 {
		t.Errorf("TypeTokenRatio = %f; want 0.6", doc.TypeTokenRatio)
	}
}

func TestNormalizerLargeText(t *testing.T) {
	words := make([]string, 1000)
	for i := range words {
		words[i] = "word"
	}
	raw := strings.Join(words, " ") + "."
	n := NewNormalizer()
	doc := n.Normalize(raw)
	if doc.WordCount != 1000 {
		t.Errorf("WordCount = %d; want 1000", doc.WordCount)
	}
}
