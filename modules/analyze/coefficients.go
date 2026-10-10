package analyze

// Correlation-weighted heuristic inspired by Yarkoni (2010) Table 1 Spearman
// correlations (ρ), converted to per-percentage-point weights on a [0,1]
// trait scale using the formula β ≈ ρ × (SD_trait / SD_category) / trait_range
// where SD_trait ≈ 0.15, SD_category ≈ 2.5 pp, trait_range = 1.
// These scales are project assumptions. Zero-order correlations are not
// fitted multiple-regression coefficients; neither this formula nor these
// scores are validated by that paper.
//
// Source:
//   Yarkoni, T. (2010). Personality in 100,000 words: A large-scale analysis
//   of personality and word use among bloggers. Journal of Research in
//   Personality, 44(3), 363–373. https://doi.org/10.1016/j.jrp.2010.04.001
//
// Verified against Table 1 of the paper (PMC2885844, minimum N = 576) on
// 2026-10-10: every weight below equalled ρ × 0.06 in sign and size (Openness weights are then scaled by opennessScale, see below), and every
// weight rests on a correlation marked significant at p < .05.
//
// Only dictionary.json categories with a clear Yarkoni mapping are included.
// Categories with no Yarkoni basis (quantitative, present_focus,
// future_focus) are kept as zero and should be updated when new research
// provides empirical coefficients.

// Coefficients map a category name to its per-percentage-point weight per trait.
var coefficients = map[string]TraitWeights{
	// Openness (+): articles (ρ=.20), prepositions (.17), inclusive (.11)
	// Openness (-): pronouns (-.21), time (-.22), motion (-.22), past_focus (-.16),
	//               positive_emotion (-.15)
	"positive_emotion":  {Openness: -0.009, Extraversion: 0.006, Agreeableness: 0.011},
	"cognitive_process": {Neuroticism: 0.008, Conscientiousness: -0.007},
	"tentative":         {Neuroticism: 0.007},
	"certainty":         {Neuroticism: 0.008},
	// Total pronouns: only the Openness correlation (ρ = -.21) is significant in
	// Table 1; the Extraversion and Neuroticism correlations (both .06) are not,
	// so they carry no weight.
	"pronoun": {Openness: -0.013},
	// Prepositions are Yarkoni's second-strongest openness signal (ρ = .17).
	"preposition": {Openness: 0.010},
	"article":     {Openness: 0.012, Neuroticism: -0.007},
	"achievement": {Conscientiousness: 0.008},
	"social":      {Extraversion: 0.009},
	"time":        {Openness: -0.013, Agreeableness: 0.007},
	"space":       {Agreeableness: 0.010},
	"motion":      {Openness: -0.013, Agreeableness: 0.008},
	"cause":       {Neuroticism: 0.007, Conscientiousness: -0.007},
	"inclusive":   {Openness: 0.007, Agreeableness: 0.011},
	"exclusive":   {Conscientiousness: -0.010},
	"past_focus":  {Openness: -0.010},
	// Categories mapped via weaker/general associations
	"negative_emotion": {Neuroticism: 0.010, Conscientiousness: -0.011, Agreeableness: -0.009},
	// Negations (ρ: N +.11, O −.13, C −.17 — the largest standalone C
	// correlation in the table). Negative contractions are negation words too.
	"negation": {Neuroticism: 0.007, Openness: -0.008, Conscientiousness: -0.010},
	// "sensation" is intentionally absent: its published correlation with
	// Neuroticism (Sensory Processes, ρ = .05) is non-significant at the
	// paper's sample size, leaving no empirical basis for a weight.
}

// opennessScale shrinks every Openness weight above. Its pre-shrink SD over the
// 3,992 reference posts was 0.098, about four times the 0.015 to 0.03 that a
// trait SD of 0.15 and category correlations of 0.1 to 0.2 allow, because its
// weights (articles and prepositions up; pronouns, time and motion words down)
// all track formal against casual writing and add up. The other traits sit at
// 0.009 to 0.026. 0.3 brings Openness to about 0.03. The factor comes from the
// reference SDs only, not from any label. Rank (AUC) does not depend on it.
const opennessScale = 0.3

func init() {
	for cat, w := range coefficients {
		w.Openness *= opennessScale
		coefficients[cat] = w
	}
}

// TraitWeights holds per-trait heuristic weights for a single category.
type TraitWeights struct {
	Openness          float64
	Conscientiousness float64
	Extraversion      float64
	Agreeableness     float64
	Neuroticism       float64
}

// intercepts provide baseline scores so results sit in a plausible 0-1 range.
// They are a fixed 0.50 rather than calibrated against average category
// rates, so adding a high-frequency category shifts absolute levels; treat
// scores comparatively until a reference-calibration pass.
var intercepts = BigFiveScores{
	Openness:          0.50,
	Conscientiousness: 0.50,
	Extraversion:      0.50,
	Agreeableness:     0.50,
	Neuroticism:       0.50,
}
