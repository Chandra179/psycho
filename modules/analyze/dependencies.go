package analyze

import (
	"context"
	"fmt"
	"os"

	"psycho/zlogger"
)

type Dependencies struct {
	Config      Config
	Logger      *zlogger.Logger
	Dict        Dictionary
	Extractor   *FeatureExtractor
	Model       TraitModel
	Calibration *Calibration
}

func NewDependencies(cfg Config, logger *zlogger.Logger) (*Dependencies, error) {
	// Load dictionary from JSON file
	data, err := os.ReadFile(cfg.DictionaryPath)
	if err != nil {
		return nil, fmt.Errorf("read dictionary: %w", err)
	}
	dict, err := LoadDictionaryFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("load dictionary: %w", err)
	}

	extractor := NewFeatureExtractor(dict)
	model := NewBigFiveModel()

	// Calibration is optional: an empty path leaves scores uncalibrated
	// (fixed 0.50 intercepts, normal-approximation percentiles).
	var cal *Calibration
	if cfg.CalibrationPath != "" {
		calData, err := os.ReadFile(cfg.CalibrationPath)
		if err != nil {
			return nil, fmt.Errorf("read calibration: %w", err)
		}
		cal, err = LoadCalibration(calData)
		if err != nil {
			return nil, fmt.Errorf("load calibration: %w", err)
		}
		if cal.DictionarySHA256 != DictionaryFingerprint(data) {
			return nil, fmt.Errorf("calibration dictionary fingerprint mismatch; rerun cmd/calibrate for the configured dictionary")
		}
		if cal.ModelFingerprint != ModelFingerprint() {
			return nil, fmt.Errorf("calibration model fingerprint mismatch; rerun cmd/calibrate")
		}
		logger.Info(context.Background(), "calibration loaded",
			zlogger.Field{Key: "path", Value: cfg.CalibrationPath},
			zlogger.Field{Key: "corpus", Value: cal.Corpus},
		)
	}

	return &Dependencies{
		Config:      cfg,
		Logger:      logger,
		Dict:        dict,
		Extractor:   extractor,
		Model:       model,
		Calibration: cal,
	}, nil
}
