package eagle

func (e *Eagle) Sync() ([]string, error) {
	return e.srcGit.pullAndPush()
}

func (e *Eagle) Persist(filename string, data []byte, message string) error {
	err := e.srcFs.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	err = e.srcGit.addAndCommit(message, filename)
	if err != nil {
		return err
	}

	return nil
}

func (e *Eagle) ReadFile(filename string) ([]byte, error) {
	return e.srcFs.ReadFile(filename)
}
