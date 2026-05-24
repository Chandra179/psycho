package profile

type ProfilePDFGenerator interface {
	Generate(p Profile) ([]byte, error)
}
