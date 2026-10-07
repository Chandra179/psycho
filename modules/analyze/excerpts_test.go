package analyze

import (
	"strings"
	"testing"
	"unicode/utf8"

	"psycho/modules/ingest"
)

func excerptDictionary(t *testing.T) Dictionary {
	t.Helper()
	dict, err := LoadDictionaryFromJSON([]byte(`{"value_tradition":["culture","tradition","cultural"],"value_security":["safe"],"negation":["not"]}`))
	if err != nil {
		t.Fatal(err)
	}
	return dict
}

func TestValueExcerptsPreserveContextAndAllCounts(t *testing.T) {
	doc := ingest.NewNormalizer().Normalize("I do not follow tradition. CULTURE, culture!\n\nOur cultural customs are safe.")
	fv, _ := NewFeatureExtractor(excerptDictionary(t)).Extract(doc)
	if fv.CategoryCounts["value_tradition"] != 4 {
		t.Fatalf("all occurrences must count: %+v", fv.CategoryCounts)
	}
	excerpts := fv.ValueExcerpts["value_tradition"]
	if len(excerpts) != 2 || excerpts[0].PlainText() != "I do not follow tradition." || excerpts[1].PlainText() != "CULTURE, culture!" {
		t.Fatalf("wrong sampled context: %+v", excerpts)
	}
	var highlighted []string
	for _, segment := range excerpts[1].Segments {
		if segment.Matched {
			highlighted = append(highlighted, segment.Text)
		}
	}
	if strings.Join(highlighted, ",") != "CULTURE,culture" {
		t.Fatalf("repeated matches/case lost: %v", highlighted)
	}
	if got := fv.ValueExcerpts["value_security"][0].PlainText(); got != "Our cultural customs are safe." {
		t.Fatalf("paragraph context lost: %q", got)
	}
}

func TestValueExcerptsDistinctAndUnicodeBounded(t *testing.T) {
	dict := excerptDictionary(t)
	text := "Culture matters. Culture matters. Tradition matters."
	got := valueExcerpts(text, dict)["value_tradition"]
	if len(got) != 2 || got[1].PlainText() != "Tradition matters." {
		t.Fatalf("duplicate sample retained: %+v", got)
	}
	text = strings.Repeat("élève ", 90) + "TRADITION " + strings.Repeat("東京 reader ", 90)
	got = valueExcerpts(text, dict)["value_tradition"]
	if len(got) != 1 {
		t.Fatalf("long excerpt lost: %+v", got)
	}
	plain := got[0].PlainText()
	if !utf8.ValidString(plain) || utf8.RuneCountInString(plain) > maxExcerptRunes || !strings.HasPrefix(plain, "…") || !strings.HasSuffix(plain, "…") {
		t.Fatalf("invalid clipping: %q", plain)
	}
	for _, s := range got[0].Segments {
		if s.Matched && s.Text != "TRADITION" {
			t.Fatalf("split matching token: %q", s.Text)
		}
	}
	if !strings.Contains(plain, "TRADITION") {
		t.Fatal("clipping lost the matching word")
	}
}

func TestValueExcerptsPunctuationAndParagraphs(t *testing.T) {
	for _, text := range []string{"Culture? Tradition!", "Culture\n\nTradition", "Culture... Tradition.", "Culture.” Tradition."} {
		doc := ingest.NewNormalizer().Normalize(text)
		got := valueExcerpts(doc.RawText, excerptDictionary(t))["value_tradition"]
		if len(got) != 2 {
			t.Errorf("expected two excerpts for %q: %+v", text, got)
		}
	}
	if got := valueExcerpts("No matching value words.", excerptDictionary(t)); len(got) != 0 {
		t.Fatalf("invented examples: %+v", got)
	}
}

func TestAllValuesHaveDictionaryTopicDescriptions(t *testing.T) {
	for _, category := range SchwartzValueKeys() {
		if ValueDescription(category) == "" {
			t.Errorf("missing description: %s", category)
		}
	}
}
