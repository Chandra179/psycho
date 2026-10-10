package analyze

import (
	"fmt"
	"os"
)

type Dependencies struct {
	Config      Config
	Dict        Dictionary
	Extractor   *FeatureExtractor
	Model       TraitModel
	Calibration *Calibration
}

func NewDependencies(cfg Config) (*Dependencies, error) {
	data, err := os.ReadFile(cfg.DictionaryPath)
	if err != nil {
		return nil, fmt.Errorf("read dictionary: %w", err)
	}

	// Calibration is optional: an empty path leaves scores uncalibrated
	// (fixed 0.50 intercepts, normal-approximation percentiles).
	var calData []byte
	if cfg.CalibrationPath != "" {
		calData, err = os.ReadFile(cfg.CalibrationPath)
		if err != nil {
			return nil, fmt.Errorf("read calibration: %w", err)
		}
	}

	deps, err := NewDependenciesFromData(data, calData)
	if err != nil {
		return nil, err
	}
	deps.Config = cfg
	return deps, nil
}

// NewDependenciesFromData builds the analysis services from dictionary and
// calibration JSON already in memory, so the browser build can embed both
// files instead of reading them from disk. Empty calibration data leaves
// scores uncalibrated.
func NewDependenciesFromData(dictData, calData []byte) (*Dependencies, error) {
	dict, err := LoadDictionaryFromJSON(dictData)
	if err != nil {
		return nil, fmt.Errorf("load dictionary: %w", err)
	}

	var cal *Calibration
	if len(calData) > 0 {
		cal, err = LoadCalibration(calData)
		if err != nil {
			return nil, fmt.Errorf("load calibration: %w", err)
		}
		if cal.DictionarySHA256 != DictionaryFingerprint(dictData) {
			return nil, fmt.Errorf("calibration dictionary fingerprint mismatch; rerun cmd/calibrate for the configured dictionary")
		}
		if cal.ModelFingerprint != ModelFingerprint() {
			return nil, fmt.Errorf("calibration model fingerprint mismatch; rerun cmd/calibrate")
		}
	}

	return &Dependencies{
		Dict:        dict,
		Extractor:   NewFeatureExtractor(dict),
		Model:       NewBigFiveModel(),
		Calibration: cal,
	}, nil
}
