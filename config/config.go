package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Example    ExampleConfig    `yaml:"example"`
	Middleware MiddlewareConfig `yaml:"middleware"`
}

type ExampleConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Port               string `yaml:"port"`
	ReadTimeoutInSec   int    `yaml:"read_timeout_in_second"`
	WriteTimeoutInSec  int    `yaml:"write_timeout_in_second"`
	IdleTimeoutInSec   int    `yaml:"idle_timeout_in_second"`
}

type MiddlewareConfig struct {
	TimeoutInSec int          `yaml:"timeout_in_second"`
	Logger       LoggerConfig `yaml:"logger"`
}

type LoggerConfig struct {
	Level string `yaml:"level"`
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
	return &cfg, nil
}
