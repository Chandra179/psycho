package analyze

type Config struct {
	DictionaryPath  string `yaml:"dictionary_path"`
	CalibrationPath string `yaml:"calibration_path"` // optional; empty disables calibration
}
