package eagle

type Eagle struct {
}

func NewEagle(conf *Config) (*Eagle, error) {
	eagle := &Eagle{}

	return eagle, nil
}
