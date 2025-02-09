package log

import "go.uber.org/zap"

type LogNotifier struct {
	log *zap.SugaredLogger
}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{
		log: S().Named("notifier"),
	}
}

func (n *LogNotifier) Notify(msg string) {
	n.log.Info(msg)
}
