package ingest

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func TestWordSpansPreserveExistingTokenizer(t *testing.T) {
	for _, text := range []string{"", "spaces\tand\nlines", "Culture, CULTURE!", "élève 東京 １２ café", "don't ' quoted 'words'", "a—b…c & d<e>f", "I’m not happy", "\u2003\u3000", "Caseİ À Ö"} {
		var legacy []string
		for _, word := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '\'' }) {
			if word = strings.ToLower(strings.TrimSpace(word)); word != "" {
				legacy = append(legacy, word)
			}
		}
		got := TokenizeWords(text)
		if !reflect.DeepEqual(legacy, got) {
			t.Errorf("tokenization changed for %q: %v vs %v", text, legacy, got)
		}
		spans := TokenizeWordSpans(text)
		if len(spans) != len(got) {
			t.Fatal("span count differs from scoring tokens")
		}
		for i, span := range spans {
			if span.Word != got[i] || strings.ToLower(text[span.Start:span.End]) != span.Word {
				t.Fatalf("invalid word offsets: %+v", span)
			}
		}
	}
}
