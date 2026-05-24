package profile

type Config struct {
	DBPath     string `yaml:"db_path"`
	PDFBackend string `yaml:"pdf_backend"`
}
