package analyze

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"psycho/modules/ingest"
)

func TestCalculationsReplayAndDeterminism(t *testing.T) {
	data, err := os.ReadFile("dictionary.json")
	if err != nil {
		t.Fatal(err)
	}
	dict, err := LoadDictionaryFromJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	fe := NewFeatureExtractor(dict)
	for _, text := range []string{"The research about the history is thoughtful, curious and important. I feel happy with my friends.", strings.Repeat("the ", 40), strings.Repeat("I ", 40)} {
		fv, _ := fe.Extract(ingest.NewNormalizer().Normalize(text))
		model := NewBigFiveModel()
		first := model.Infer(fv)
		for i := 0; i < 100; i++ {
			if !reflect.DeepEqual(first, model.Infer(fv)) {
				t.Fatal("model varies between runs")
			}
		}
		calculations := first.Calculations
		calculations["regulatory_focus"] = ComputeRegulatoryFocusCalculation(fv)
		calculations["need_for_cognition"] = ComputeNeedForCognitionCalculation(fv)
		calculations["cognitive_style"] = ComputeCognitiveStyleCalculation(fv)
		calculations["need_for_closure"] = ComputeNeedForClosureCalculation(fv)
		for name, c := range calculations {
			accumulator, total := c.Baseline, 0.0
			for _, term := range c.Terms {
				percent := float64(term.MatchedCount) / float64(term.TotalWords) * 100
				if math.Abs(percent-term.WordPercent) > 1e-12 {
					t.Fatalf("%s percent mismatch: %+v", name, term)
				}
				product := term.Weight * percent
				if term.Category == "long_word_ratio" {
					product = term.Weight * (float64(term.MatchedCount) / float64(term.TotalWords)) * 100
				}
				if product != term.Contribution {
					t.Fatalf("%s product mismatch", name)
				}
				accumulator += term.Contribution
				total += term.Contribution
			}
			if accumulator != c.UnroundedScore || total != c.ContributionTotal || clamp(accumulator) != c.ModelScore {
				t.Fatalf("%s cannot replay: %+v", name, c)
			}
		}
	}
}

func validTestCalibration(t *testing.T) *Calibration {
	t.Helper()
	c, err := BuildCalibration("test corpus", "date", []BigFiveScores{{}, {Openness: 1, Conscientiousness: 1, Extraversion: 1, Agreeableness: 1, Neuroticism: 1, RegulatoryFocus: 1, NeedForCognition: 1, CognitiveStyle: 1, NeedForClosure: 1}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCalibrationRejectsIncompleteAndMalformed(t *testing.T) {
	cases := map[string]func(*Calibration){
		"legacy":            func(c *Calibration) { c.ModelFingerprint = "" },
		"missing dimension": func(c *Calibration) { delete(c.Dimensions, "openness") },
		"extra dimension":   func(c *Calibration) { c.Dimensions["unknown"] = c.Dimensions["openness"] },
		"short": func(c *Calibration) {
			d := c.Dimensions["openness"]
			d.Quantiles = d.Quantiles[:98]
			c.Dimensions["openness"] = d
		},
		"unsorted": func(c *Calibration) {
			d := c.Dimensions["openness"]
			d.Quantiles[0] = 1
			d.Quantiles[1] = 0
			c.Dimensions["openness"] = d
		},
		"out of range": func(c *Calibration) {
			d := c.Dimensions["openness"]
			d.Quantiles[98] = 1.1
			c.Dimensions["openness"] = d
		},
		"sample size": func(c *Calibration) { d := c.Dimensions["openness"]; d.N = 1; c.Dimensions["openness"] = d },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := validTestCalibration(t)
			mutate(c)
			data, _ := json.Marshal(c)
			if _, err := LoadCalibration(data); err == nil {
				t.Fatal("accepted malformed calibration")
			}
		})
	}
	// JSON numbers cannot encode NaN or Infinity; reject overflowing exponents too.
	data, _ := json.Marshal(validTestCalibration(t))
	data = []byte(strings.Replace(string(data), `"offset":0`, `"offset":1e999`, 1))
	if _, err := LoadCalibration(data); err == nil {
		t.Fatal("accepted non-finite number")
	}
}

func TestStartupRejectsFingerprintMismatches(t *testing.T) {
	dictionary, err := os.ReadFile("dictionary.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, which := range []string{"dictionary", "model"} {
		t.Run(which, func(t *testing.T) {
			c := validTestCalibration(t)
			c.DictionarySHA256 = DictionaryFingerprint(dictionary)
			if which == "dictionary" {
				c.DictionarySHA256 = "stale"
			} else {
				c.ModelFingerprint = "stale"
			}
			data, _ := json.Marshal(c)
			path := filepath.Join(t.TempDir(), "calibration.json")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := NewDependencies(Config{DictionaryPath: "dictionary.json", CalibrationPath: path}); err == nil {
				t.Fatal("accepted stale calibration")
			}
		})
	}
}

func TestCalibrationTraceClampingAndPercentileBranches(t *testing.T) {
	c := validTestCalibration(t)
	d := c.Dimensions["openness"]
	d.Offset = .2
	c.Dimensions["openness"] = d
	for _, raw := range []float64{.01, .50, .99} {
		trace := newScoreCalculation(raw)
		trace.finish(raw)
		s := BigFiveScores{Openness: raw, Calculations: map[string]*ScoreCalculation{"openness": trace}}
		c.AdjustScores(&s)
		if trace.FinalScore != s.Openness || trace.FinalScore != clamp(trace.ModelScore+trace.CalibrationOffset) {
			t.Fatal("calibration cannot replay")
		}
	}
	for _, score := range []float64{-.1, 0, .125, .5, 1, 1.1} {
		p, trace, ok := c.PercentileWithDetails("openness", score)
		if !ok || trace.Formula == "" || p != trace.Percentile {
			t.Fatal("missing percentile operands")
		}
	}
	if !finite(0) || finite(math.NaN()) || finite(math.Inf(1)) {
		t.Fatal("finite policy incorrect")
	}
}

func TestSummaryAndValueCalculationsReplay(t *testing.T) {
	data, err := os.ReadFile("dictionary.json")
	if err != nil {
		t.Fatal(err)
	}
	dict, err := LoadDictionaryFromJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	fv, _ := NewFeatureExtractor(dict).Extract(ingest.NewNormalizer().Normalize("I think about the important research because I am happy with friends and the plan."))
	summary, details := ComputeSummaryVariablesWithDetails(fv)
	for name, c := range details {
		p := c.Inputs
		numerator := 0.0
		switch name {
		case "emotional_tone":
			numerator = p["positive_emotion"] - p["negative_emotion"]
		case "analytical_thinking":
			numerator = p["article"] + p["cognitive_process"] + p["cause"] + p["certainty"] + p["exclusive"] + p["quantitative"] - p["pronoun"] - p["tentative"] - p["inclusive"] - p["sensation"] - p["time"]
		case "clout":
			numerator = p["certainty"] + p["social"] + p["achievement"] + p["exclusive"] - p["pronoun"] - p["tentative"] - p["negative_emotion"]
		case "authenticity":
			numerator = p["pronoun"] + p["tentative"] + p["present_focus"] + p["inclusive"] + p["sensation"] - p["cognitive_process"] - p["cause"] - p["past_focus"] - p["exclusive"] - p["certainty"] - authenticityCenter
		}
		if numerator != c.Numerator || math.Round((1/(1+math.Exp(-numerator/c.Divisor)))*100)/100 != c.Score {
			t.Fatalf("summary %s cannot replay", name)
		}
	}
	if summary.AnalyticalThinking != details["analytical_thinking"].Score || summary.Authenticity != details["authenticity"].Score || summary.Clout != details["clout"].Score || summary.EmotionalTone != details["emotional_tone"].Score {
		t.Fatal("summary output differs from trace")
	}
	for name, pct := range ComputeSchwartzValues(fv) {
		if pct != float64(fv.CategoryCounts[Category(name)])/float64(fv.WordCount)*100 {
			t.Fatalf("value %s cannot replay", name)
		}
	}
}

func TestEmpiricalPercentileFormulaReplay(t *testing.T) {
	c := validTestCalibration(t)
	d := c.Dimensions["openness"]
	for i := range d.Quantiles {
		d.Quantiles[i] = float64(i+1) / 100
	}
	d.Quantiles[49] = .49
	c.Dimensions["openness"] = d
	for _, score := range []float64{0, .01, .255, .49, .501, .99, 1} {
		result, trace, _ := c.PercentileWithDetails("openness", score)
		expected := 0
		switch {
		case score < d.Quantiles[0]:
			expected = 1
		case score > d.Quantiles[98]:
			expected = 99
		case trace.EmpiricalLookup.EqualQuantiles > 0:
			expected = trace.EmpiricalLookup.QuantilesBelow + (trace.EmpiricalLookup.EqualQuantiles+1)/2
		default:
			expected = trace.EmpiricalLookup.LowerPercentile + int(math.Round((trace.Score-trace.EmpiricalLookup.LowerScore)/(trace.EmpiricalLookup.UpperScore-trace.EmpiricalLookup.LowerScore)))
		}
		if result != max(1, min(99, expected)) {
			t.Fatalf("percentile %f cannot replay", score)
		}
	}
}
