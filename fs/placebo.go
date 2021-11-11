package fs

type PlaceboSync struct{}

func NewPlaceboSync() FSSync {
	return &PlaceboSync{}
}

func (g *PlaceboSync) Persist(msg string, file string) error {
	return nil
}

func (g *PlaceboSync) Sync() ([]string, error) {
	return []string{}, nil
}
