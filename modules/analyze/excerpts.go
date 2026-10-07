package analyze

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"psycho/modules/ingest"
)

const maxValueExcerpts = 2
const maxExcerptRunes = 240

// valueExcerpts samples context using the scorer's exact tokenizer and lookup.
// Punctuation and paragraph boundaries only select excerpts, never scores.
func valueExcerpts(text string, dict Dictionary) map[string][]ingest.TextExcerpt {
	spans := ingest.TokenizeWordSpans(text)
	out := make(map[string][]ingest.TextExcerpt)
	seen := make(map[string]map[string]bool)
	index := 0
	for _, bounds := range excerptRanges(text) {
		matches := make(map[Category][]ingest.WordSpan)
		for index < len(spans) && spans[index].Start < bounds[1] {
			span := spans[index]
			index++
			if span.Start < bounds[0] {
				continue
			}
			for _, cat := range dict.Lookup(span.Word) {
				if slices.Contains(schwartzValueCategories, cat) && len(out[string(cat)]) < maxValueExcerpts {
					matches[cat] = append(matches[cat], span)
				}
			}
		}
		for _, cat := range schwartzValueCategories {
			if len(matches[cat]) == 0 {
				continue
			}
			excerpt := makeValueExcerpt(text, bounds, spans, matches[cat])
			if len(excerpt.Segments) == 0 {
				continue
			}
			key, plain := string(cat), excerpt.PlainText()
			if seen[key] == nil {
				seen[key] = make(map[string]bool)
			}
			if !seen[key][plain] {
				out[key] = append(out[key], excerpt)
				seen[key][plain] = true
			}
		}
	}
	return out
}

func excerptRanges(text string) [][2]int {
	var ranges [][2]int
	start := 0
	appendRange := func(end int) {
		if end > start && strings.TrimSpace(text[start:end]) != "" {
			ranges = append(ranges, [2]int{start, end})
		}
		start = end
	}
	for i, r := range text {
		if i < start {
			continue
		}
		if r == '\n' {
			appendRange(i)
			start = i + 1
			continue
		}
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		end := i + utf8.RuneLen(r)
		for end < len(text) {
			next, size := utf8.DecodeRuneInString(text[end:])
			if !strings.ContainsRune(".!?\"'’”)]}", next) {
				break
			}
			end += size
		}
		if end == len(text) {
			appendRange(end)
		} else {
			next, _ := utf8.DecodeRuneInString(text[end:])
			if unicode.IsSpace(next) {
				appendRange(end)
			}
		}
	}
	appendRange(len(text))
	return ranges
}

func makeValueExcerpt(text string, bounds [2]int, spans, matches []ingest.WordSpan) ingest.TextExcerpt {
	lo, hi := bounds[0], bounds[1]
	if utf8.RuneCountInString(text[lo:hi]) > maxExcerptRunes {
		// Reserve one rune for each possible ellipsis. Center near the first
		// match, then move cuts out of tokens so words are never split.
		runeOffsets := []int{}
		for i := range text[lo:hi] {
			runeOffsets = append(runeOffsets, lo+i)
		}
		runeOffsets = append(runeOffsets, hi)
		matchIndex := slices.Index(runeOffsets, matches[0].Start)
		first := max(0, matchIndex-80)
		last := min(len(runeOffsets)-1, first+maxExcerptRunes-2)
		lo, hi = runeOffsets[first], runeOffsets[last]
		for _, span := range spans {
			if span.Start < lo && lo < span.End {
				lo = span.End
			}
			if span.Start < hi && hi < span.End {
				hi = span.Start
			}
			if span.Start >= hi {
				break
			}
		}
	}
	for lo < hi {
		r, n := utf8.DecodeRuneInString(text[lo:hi])
		if !unicode.IsSpace(r) {
			break
		}
		lo += n
	}
	for lo < hi {
		r, n := utf8.DecodeLastRuneInString(text[lo:hi])
		if !unicode.IsSpace(r) {
			break
		}
		hi -= n
	}
	var segments []ingest.TextSegment
	if strings.TrimSpace(text[bounds[0]:lo]) != "" {
		segments = append(segments, ingest.TextSegment{Text: "…"})
	}
	pos, matched := lo, false
	for _, span := range matches {
		if span.Start < lo || span.End > hi {
			continue
		}
		if span.Start > pos {
			segments = append(segments, ingest.TextSegment{Text: text[pos:span.Start]})
		}
		segments = append(segments, ingest.TextSegment{Text: text[span.Start:span.End], Matched: true})
		pos, matched = span.End, true
	}
	if !matched {
		return ingest.TextExcerpt{}
	}
	if pos < hi {
		segments = append(segments, ingest.TextSegment{Text: text[pos:hi]})
	}
	if strings.TrimSpace(text[hi:bounds[1]]) != "" {
		segments = append(segments, ingest.TextSegment{Text: "…"})
	}
	return ingest.TextExcerpt{Segments: segments}
}
