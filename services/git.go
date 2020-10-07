package services

type Git struct {
	Directory string
}

func (g *Git) Commit(msg string) error {
	return nil
}

func (g *Git) CommitFile(msg string, files ...string) error {
	return nil
}

func (g *Git) Push() error {
	return nil
}

func (g *Git) Pull() error {
	return nil
}
