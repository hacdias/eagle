package services

type Notify struct {
}

func (n *Notify) Info(msg string) error {
	return nil
}

func (n *Notify) Error(err error) error {
	return nil
}
