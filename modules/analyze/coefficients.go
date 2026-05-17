package analyze

// Regression coefficients are MVP approximations based on published
// psycholinguistic literature.  They are intentionally simplified and should
// be replaced with exact coefficients from Yarkoni (2010) when available.
//
// Sources:
//   * Yarkoni, T. (2010). Personality in 100,000 words. Journal of Research in Personality.
//   * Pennebaker, J.W. & King, L.A. (1999). Linguistic styles. JPSP.
//
// Directional relationships used:
//   - cognitive_process, tentative, big_words  -> Openness
//   - achievement, certainty, article          -> Conscientiousness
//   - positive_emotion, social, pronoun         -> Extraversion
//   - positive_emotion, social                  -> Agreeableness
//   - negative_emotion, pronoun, tentative     -> Neuroticism

// Coefficients map a category name to its weight per trait.
var coefficients = map[string]TraitWeights{
	"positive_emotion":  {Extraversion: 0.15, Agreeableness: 0.12},
	"negative_emotion":  {Neuroticism: 0.20},
	"cognitive_process": {Openness: 0.18},
	"tentative":         {Openness: 0.10, Neuroticism: 0.08},
	"certainty":         {Conscientiousness: 0.14},
	"pronoun":           {Extraversion: 0.10, Neuroticism: 0.12},
	"article":           {Conscientiousness: 0.10},
	"achievement":       {Conscientiousness: 0.16},
	"social":            {Extraversion: 0.12, Agreeableness: 0.14},
	"big_words":         {Openness: 0.12},
	"sensation":         {Openness: 0.06, Extraversion: 0.05},
	"time":              {Conscientiousness: 0.05},
	"space":             {Openness: 0.04},
	"motion":            {Extraversion: 0.05},
	"quantitative":      {Conscientiousness: 0.06},
	"cause":             {Conscientiousness: 0.05, Agreeableness: 0.03},
	"inclusive":         {Agreeableness: 0.05, Extraversion: 0.04},
	"exclusive":         {Neuroticism: 0.04},
	"past_focus":        {Neuroticism: 0.03},
	"present_focus":     {Extraversion: 0.04},
	"future_focus":      {Conscientiousness: 0.05, Openness: 0.04},
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
	Openness:          0.55,
	Conscientiousness: 0.55,
	Extraversion:      0.50,
	Agreeableness:     0.55,
	Neuroticism:       0.45,
}
