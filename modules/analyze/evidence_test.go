package analyze

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestContributionMatchedWordSamplesAreBoundedAndSerialized(t *testing.T) {
	words := make([]string, MaxEvidenceWords+3)
	for i := range words {
		words[i] = "word" + string(rune('a'+i))
	}
	fv := FeatureVector{
		CategoryPercents: map[Category]float64{"article": 5.5},
		Evidence:         map[Category][]string{"article": words},
	}

	evidence := BigFiveEvidence("openness", fv)
	if len(evidence) != 1 || evidence[0].Category != "article" {
		t.Fatalf("unexpected contribution rows: %+v", evidence)
	}
	if len(evidence[0].MatchedWords) != MaxEvidenceWords {
		t.Fatalf("sample has %d words; want %d", len(evidence[0].MatchedWords), MaxEvidenceWords)
	}
	if evidence[0].MatchedWords[0] != words[0] || evidence[0].MatchedWords[MaxEvidenceWords-1] != words[MaxEvidenceWords-1] {
		t.Fatalf("sample should preserve the first %d normalized words: %v", MaxEvidenceWords, evidence[0].MatchedWords)
	}
	// The evidence row owns its sample and cannot mutate the FeatureVector.
	evidence[0].MatchedWords[0] = "changed"
	if fv.Evidence["article"][0] != words[0] {
		t.Fatal("contribution sample aliases feature evidence")
	}

	encoded, err := json.Marshal(evidence[0])
	if err != nil {
		t.Fatalf("marshal contribution: %v", err)
	}
	if !strings.Contains(string(encoded), `"matched_words":[`) {
		t.Fatalf("matched word sample missing from JSON: %s", encoded)
	}
}
