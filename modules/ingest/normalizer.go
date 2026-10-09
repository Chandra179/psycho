package ingest

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Document holds normalized text with metadata.
type Document struct {
	RawText        string
	WordCount      int
	SentenceCount  int
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
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	lines = removeScanNoise(lines)
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
	walkWords(s, func(word string, _, _ int) { words = append(words, word) })
	return words
}

// WordSpan retains byte offsets into the original normalized text. Matching
// uses Word, exactly as TokenizeWords does; excerpts preserve the original case.
type WordSpan struct {
	Word       string
	Start, End int
}

func TokenizeWordSpans(s string) []WordSpan {
	var spans []WordSpan
	walkWords(s, func(word string, start, end int) {
		spans = append(spans, WordSpan{Word: word, Start: start, End: end})
	})
	return spans
}

func walkWords(s string, visit func(string, int, int)) {
	start := -1
	for i, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '\'' {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			visit(strings.ToLower(s[start:i]), start, i)
			start = -1
		}
	}
	if start >= 0 {
		visit(strings.ToLower(s[start:]), start, len(s))
	}
}

// TextExcerpt contains only text, never trusted HTML. Renderers escape every
// segment and supply their own highlighting around the matched segments.
type TextExcerpt struct {
	Segments []TextSegment `json:"segments"`
}

type TextSegment struct {
	Text    string `json:"text"`
	Matched bool   `json:"matched"`
}

func (e TextExcerpt) PlainText() string {
	var out strings.Builder
	for _, segment := range e.Segments {
		out.WriteString(segment.Text)
	}
	return out.String()
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

// ErrInvalidText is shared by all transports. Validation happens after
// normalization in the pipeline, before inference or persistence.
var ErrInvalidText = errors.New("text must contain letters or numbers and at least 10 normalized characters")

func ValidateDocument(doc Document) error {
	if utf8.RuneCountInString(doc.RawText) < 10 {
		return ErrInvalidText
	}
	for _, r := range doc.RawText {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return nil
		}
	}
	return ErrInvalidText
}

// TokenizeWords is the common tokenizer used for document and feature counts.
func TokenizeWords(s string) []string { return tokenizeWords(s) }

var (
	pageNumberLine = regexp.MustCompile(`^(?i:(page\s+)?\d{1,4}(\s+of\s+\d{1,4})?|[-–—]\s*\d{1,4}\s*[-–—])$`)
	captionLine    = regexp.MustCompile(`^(?i:(figure|fig\.|table|map|plate)\s+[0-9ivx]+\b)`)
	headerDigits   = regexp.MustCompile(`\d+`)
)

// Minimum shape for a repeated line to count as a running page header, and how
// many times it must repeat. Short chat refrains ("haha") and full sentences
// are left alone.
const (
	headerMinRunes   = 20
	headerMinRepeats = 3
	hyphenMinLine    = 40
)

// removeScanNoise drops what scanned or pasted print leaves between sentences,
// so it is not scored as the author's wording: bare page numbers, figure and
// table captions, running page headers (a short unpunctuated line repeated
// three or more times), and it rejoins words hyphenated across line breaks.
func removeScanNoise(lines []string) []string {
	repeats := make(map[string]int, len(lines))
	for _, line := range lines {
		if k := headerKey(line); k != "" {
			repeats[k]++
		}
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, line)
			continue
		}
		if pageNumberLine.MatchString(line) || captionLine.MatchString(line) {
			continue
		}
		if k := headerKey(line); k != "" && repeats[k] >= headerMinRepeats {
			continue
		}
		out = append(out, line)
	}
	// Rejoin "exam-" + "ple" only inside long lines of running text, so a
	// dash that ends a short chat line is kept.
	for i := 0; i+1 < len(out); i++ {
		cur, next := out[i], out[i+1]
		if next == "" || len([]rune(cur)) < hyphenMinLine || !strings.HasSuffix(cur, "-") {
			continue
		}
		prev, _ := utf8.DecodeLastRuneInString(cur[:len(cur)-1])
		first, _ := utf8.DecodeRuneInString(next)
		if unicode.IsLetter(prev) && unicode.IsLower(first) {
			out[i+1] = cur[:len(cur)-1] + next
			out = append(out[:i], out[i+1:]...)
			i--
		}
	}
	return out
}

// headerKey returns a comparison key for a line that could be a running page
// header, or "" when the line is too short, too long, or ends like a sentence.
func headerKey(line string) string {
	if utf8.RuneCountInString(line) < headerMinRunes {
		return ""
	}
	if n := len(strings.Fields(line)); n < 2 || n > 10 {
		return ""
	}
	if last, _ := utf8.DecodeLastRuneInString(line); strings.ContainsRune(".!?:;,\"'”)", last) {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(headerDigits.ReplaceAllString(line, "")))
}
