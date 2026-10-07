package pipeline

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"psycho/modules/analyze"
	"psycho/modules/ingest"
	"psycho/modules/profile"
	"psycho/zlogger"
)

func testPipeline(t *testing.T, calibrated bool) (*Pipeline, *sql.DB) {
	t.Helper()
	logger := zlogger.New("prod")
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	storage := profile.NewStorage(db, logger)
	if err := storage.Migrate(); err != nil {
		t.Fatal(err)
	}
	cfg := analyze.Config{DictionaryPath: "../analyze/dictionary.json"}
	if calibrated {
		cfg.CalibrationPath = "../../config/calibration.json"
	}
	a, err := analyze.NewDependencies(cfg, logger)
	if err != nil {
		t.Fatal(err)
	}
	agg := profile.NewScoreAggregator()
	agg.UseCalibration(a.Calibration)
	return New(a.Extractor, a.Model, agg, profile.NewTemplateNarrativeGenerator(), storage, a.Calibration), db
}

func TestInvalidInputNeverSaved(t *testing.T) {
	pipe, db := testPipeline(t, false)
	for _, text := range []string{"", "          ", "\t\n\u2003\u3000", "...............", "<p>\t </p>", "too short"} {
		if _, err := pipe.Run(t.Context(), text); !errors.Is(err, ingest.ErrInvalidText) {
			t.Errorf("invalid text %q: %v", text, err)
		}
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM analyses").Scan(&count); err != nil || count != 0 {
		t.Fatalf("invalid analyses saved: %d, %v", count, err)
	}
}

func TestCalculationDetailsPersistAndScoresRepeat(t *testing.T) {
	for _, calibrated := range []bool{false, true} {
		t.Run(map[bool]string{false: "fallback", true: "empirical"}[calibrated], func(t *testing.T) {
			pipe, _ := testPipeline(t, calibrated)
			text, err := os.ReadFile("../../samples/01-analytical.txt")
			if err != nil {
				text = []byte("The plan is to study and think about the research, because it is important. I feel happy with friends.")
			}
			out, err := pipe.Run(t.Context(), string(text))
			if err != nil {
				t.Fatal(err)
			}
			details, ok := out.CalculationDetails.(*analyze.CalculationDetails)
			if !ok || len(details.Traits) != 9 || len(details.Summary) != 4 || len(details.Values) != 10 || len(details.RangeBounds) != 9 {
				t.Fatalf("incomplete details: %+v", details)
			}
			saved, err := pipe.storage.GetAnalysis(out.AnalysisID)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(saved.CalculationDetails, details) {
				t.Fatal("details changed in persistence")
			}
			data, err := json.Marshal(out)
			if err != nil {
				t.Fatal(err)
			}
			if !json.Valid(data) {
				t.Fatal("invalid analysis JSON")
			}
			for name, c := range details.Traits {
				trait := out.Traits[name].(profile.TraitResult)
				if c.FinalScore != trait.Score || c.CalibrationApplied != calibrated || details.Percentiles[name].Percentile != trait.Percentile {
					t.Fatalf("%s trace/result mismatch", name)
				}
			}
			again, err := pipe.Run(t.Context(), string(text))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(out.Traits, again.Traits) || !reflect.DeepEqual(out.CalculationDetails, again.CalculationDetails) {
				t.Fatal("same input produces different output")
			}
		})
	}
}

func TestPercentileReferenceReportsActiveMethodAndSample(t *testing.T) {
	if got := percentileReference(nil); got == nil || got.Method != ingest.PercentileMethodNormalApproximation {
		t.Fatalf("nil calibration should report normal approximation: %+v", got)
	}

	calibration := &analyze.Calibration{
		Corpus: "Reference essays",
		Dimensions: map[string]analyze.DimensionCalibration{
			"openness": {N: 321},
		},
	}
	got := percentileReference(calibration)
	if got == nil || got.Method != ingest.PercentileMethodEmpirical || got.Corpus != "Reference essays" || got.SampleSize != 321 {
		t.Fatalf("empirical reference metadata is incomplete: %+v", got)
	}
}

func TestValueExcerptsPersistAndReturnThroughPipeline(t *testing.T) {
	pipe, _ := testPipeline(t, true)
	out, err := pipe.Run(t.Context(), "Our cultural traditions influence culture. We question tradition and authority.")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ValueExcerpts["value_tradition"]) == 0 {
		t.Fatal("pipeline omitted text excerpts")
	}
	saved, err := pipe.storage.GetAnalysis(out.AnalysisID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.ValueExcerpts, out.ValueExcerpts) {
		t.Fatal("text excerpts changed in storage")
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var restored ingest.AnalysisOutput
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.ValueExcerpts, out.ValueExcerpts) {
		t.Fatal("text excerpts changed across API JSON")
	}
}
