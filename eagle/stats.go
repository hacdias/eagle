package eagle

import "path/filepath"

func (e *Eagle) UpdateReadStatistics() error {
	stats, err := e.db.ReadsSummary()
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

func (e *Eagle) UpdateWatchStatistics() error {
	stats, err := e.db.WatchesSummary()
	if err != nil {
		return err
	}

	// TODO: do not like this hardcoded.
	filename := filepath.Join(ContentDirectory, "watches/summary/_summary.json")

	err = e.fs.WriteJSON(filename, stats, "update watches summary")
	if err != nil {
		return err
	}

	ee, err := e.GetEntry("/watches/summary")
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}
