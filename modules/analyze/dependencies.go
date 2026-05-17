package analyze

import (
	"fmt"
	"os"

	"psycho/zlogger"
)

type Dependencies struct {
	Config    Config
	Logger    *zlogger.Logger
	Dict      Dictionary
	Extractor *FeatureExtractor
	Model     TraitModel
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

	return &Dependencies{
		Config:    cfg,
		Logger:    logger,
		Dict:      dict,
		Extractor: extractor,
		Model:     model,
	}, nil
}
