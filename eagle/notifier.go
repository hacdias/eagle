package eagle

type Notifier interface {
	Info(msg string)
	Error(err error)
}
