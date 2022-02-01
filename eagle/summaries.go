package eagle

import "path/filepath"

const ReadsSummary = "_summary.reads.json"
const WatchesSummary = "_summary.watches.json"

func (e *Eagle) UpdateReadsSummary() error {
	stats, err := e.db.ReadsSummary()
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, ReadsSummary)
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

func (e *Eagle) UpdateWatchesSummary() error {
	stats, err := e.db.WatchesSummary()
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, WatchesSummary)
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
