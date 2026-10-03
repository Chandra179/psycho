package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App        AppConfig        `yaml:"app"`
	Middleware MiddlewareConfig `yaml:"middleware"`
	Ingest     IngestConfig     `yaml:"ingest"`
	Analyze    AnalyzeConfig    `yaml:"analyze"`
	Profile    ProfileConfig    `yaml:"profile"`
}

type AppConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Port              string `yaml:"port"`
	ReadTimeoutInSec  int    `yaml:"read_timeout_in_second"`
	WriteTimeoutInSec int    `yaml:"write_timeout_in_second"`
	IdleTimeoutInSec  int    `yaml:"idle_timeout_in_second"`
}

type MiddlewareConfig struct {
	TimeoutInSec int          `yaml:"timeout_in_second"`
	Logger       LoggerConfig `yaml:"logger"`
}

type LoggerConfig struct {
	Level string `yaml:"level"`
}

type IngestConfig struct {
	MaxTextSize int    `yaml:"max_text_size"`
	DirPath     string `yaml:"dir_path"`
}

type AnalyzeConfig struct {
	DictionaryPath  string `yaml:"dictionary_path"`
	CalibrationPath string `yaml:"calibration_path"`
}

type ProfileConfig struct {
	DBPath     string `yaml:"db_path"`
	PDFBackend string `yaml:"pdf_backend"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// validate fails fast on a zero-value or partial config: without this, an
// empty port makes the server listen on a random one and an empty
// dictionary path fails later with a confusing error.
func (c *Config) validate() error {
	if c.App.HTTP.Port == "" {
		return fmt.Errorf("config: app.http.port is required")
	}
	if c.Analyze.DictionaryPath == "" {
		return fmt.Errorf("config: analyze.dictionary_path is required")
	}
	if c.Profile.DBPath == "" {
		return fmt.Errorf("config: profile.db_path is required")
	}
	return nil
}
