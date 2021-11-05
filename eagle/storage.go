package eagle

import (
	"encoding/json"
)

func (e *Eagle) Sync() ([]string, error) {
	return e.srcGit.pullAndPush()
}

func (e *Eagle) Persist(filename string, data []byte, message string) error {
	err := e.SrcFs.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	// TODO: reenable
	// err = e.srcGit.addAndCommit(message, filename)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (e *Eagle) PersistJSON(filename string, data interface{}, msg string) error {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return e.Persist(filename, json, msg)
}

func (e *Eagle) ReadFile(filename string) ([]byte, error) {
	return e.SrcFs.ReadFile(filename)
}

func (e *Eagle) ReadJSON(filename string, v interface{}) error {
	data, err := e.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}
