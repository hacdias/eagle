package eagle

import "path/filepath"

func (e *Eagle) UpdateReadStatistics() error {
	stats, err := e.db.ReadsStatistics()
	if err != nil {
		return err
	}

	// TODO: do not like this hardcoded.
	filename := filepath.Join(ContentDirectory, "reads/summary/_summary.json")

	err = e.fs.WriteJSON(filename, stats, "update read summary")
	if err != nil {
		return err
	}

	ee, err := e.GetEntry("/reads/summary")
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}
