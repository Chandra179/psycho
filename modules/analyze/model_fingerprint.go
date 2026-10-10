package analyze

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// ModelFingerprint identifies the parameters and feature semantics that affect
// the calibrated dimensions. Change the rules version whenever their semantics
// change. JSON sorts map keys so the fingerprint is stable across processes.
func ModelFingerprint() string {
	spec := modelSpecification()
	data, err := json.Marshal(spec)
	if err != nil {
		panic(err) // the compiled specification contains only finite constants
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func modelSpecification() map[string]any {
	baselines := make(map[string]float64, len(dimensionKeys))
	for _, dim := range dimensionKeys {
		baselines[dim] = additionalScoreBaseline
	}
	for _, dim := range dimensionKeys[:5] {
		baselines[dim] = dimensionValue(dim, &intercepts)
	}
	baselines["cognitive_style"] = cognitiveStyleBaseline
	return map[string]any{
		"rules_version":      9,
		"baselines":          baselines,
		"big_five":           coefficients,
		"regulatory_focus":   regFocusCoefficients,
		"need_for_cognition": needCogCoefficients,
		"cognitive_style":    cognitiveStyleCoefficients,
		"need_for_closure":   needClosureCoefficients,
		"features":           "strip tags using letter/slash delimiter; collapse Unicode whitespace within lines, drop bare page numbers, figure/table captions and lines repeated 3+ times that look like running headers, rejoin words hyphenated across long lines, preserve blank-line paragraphs; tokenize on non-Unicode-letter/non-number except ASCII apostrophe; lowercase; dictionary exact matches; trim/lowercase dictionary entries, deduplicate category membership; category percent=count/token_count*100; long_word_ratio=letter_count>6/token_count (apostrophes excluded); value-category matches within 3 tokens after a negator are not counted; words that a concordance audit showed to be mostly used in another sense are removed from the scored content-word lists; per-measure score standard error from the spread of per-word contributions",
		"score_policy":       "baseline plus weighted category percentages in sorted category order; long-word term=weight*ratio*100; clamp [0,1] then round half away from zero to 2 decimals; add calibration offset then clamp and round again",
	}
}

func DictionaryFingerprint(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// FeatureProcessingFingerprint identifies shared normalization/extraction
// semantics without tying offline learned models to heuristic coefficients.
func FeatureProcessingFingerprint() string {
	spec := map[string]any{"features": modelSpecification()["features"], "offline_inputs": "sorted dictionary category percentages only; no summary proxies"}
	data, _ := json.Marshal(spec)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
