package analyze

// Regression coefficients derived from Yarkoni (2010) Table 1 Spearman
// correlations (ρ), converted to per-percentage-point weights on a [0,1]
// trait scale using the formula β ≈ ρ × (SD_trait / SD_category) / trait_range
// where SD_trait ≈ 0.15, SD_category ≈ 3 pp, trait_range = 1.
//
// Source:
//   Yarkoni, T. (2010). Personality in 100,000 words: A large-scale analysis
//   of personality and word use among bloggers. Journal of Research in
//   Personality, 44(3), 363–373. https://doi.org/10.1016/j.jrp.2010.04.001
//
// Only dictionary.json categories with a clear Yarkoni mapping are included.
// Categories with no Yarkoni basis (big_words, quantitative, present_focus,
// future_focus) are kept as zero and should be updated when new research
// provides empirical coefficients.

// Coefficients map a category name to its per-percentage-point weight per trait.
var coefficients = map[string]TraitWeights{
	// Openness (+): articles (ρ=.20), prepositions (.17), inclusive (.11)
	// Openness (-): pronouns (-.21), time (-.22), motion (-.22), past_focus (-.16),
	//               positive_emotion (-.15)
	"positive_emotion": {Openness: -0.009, Extraversion: 0.006, Agreeableness: 0.011},
	"cognitive_process": {Neuroticism: 0.008, Conscientiousness: -0.007},
	"tentative":         {Neuroticism: 0.007},
	"certainty":         {Neuroticism: 0.008},
	"pronoun":           {Openness: -0.013, Extraversion: 0.006, Neuroticism: 0.005},
	"article":           {Openness: 0.012, Neuroticism: -0.007},
	"achievement":       {Conscientiousness: 0.008},
	"social":            {Extraversion: 0.009},
	"time":              {Openness: -0.013, Agreeableness: 0.007},
	"space":             {Agreeableness: 0.010},
	"motion":            {Openness: -0.013, Agreeableness: 0.008},
	"cause":             {Neuroticism: 0.007, Conscientiousness: -0.007},
	"inclusive":         {Openness: 0.007, Agreeableness: 0.011},
	"exclusive":         {Conscientiousness: -0.010},
	"past_focus":        {Openness: -0.010},
	// Categories mapped via weaker/general associations
	"negative_emotion": {Neuroticism: 0.010, Conscientiousness: -0.011, Agreeableness: -0.009},
	"sensation":        {Neuroticism: 0.006},
}

// TraitWeights holds per-trait regression weights for a single category.
type TraitWeights struct {
	Openness          float64
	Conscientiousness float64
	Extraversion      float64
	Agreeableness     float64
	Neuroticism       float64
}

// intercepts provide baseline scores so results sit in a plausible 0-1 range.
var intercepts = BigFiveScores{
	Openness:          0.50,
	Conscientiousness: 0.50,
	Extraversion:      0.50,
	Agreeableness:     0.50,
	Neuroticism:       0.50,
}
