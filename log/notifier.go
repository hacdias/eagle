package log

import "go.uber.org/zap"

type LogNotifier struct {
	log *zap.SugaredLogger
}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{
		log: S().Named("notify"),
	}
}

func (n *LogNotifier) Info(msg string) {
	n.log.Info(msg)
}

func (n *LogNotifier) Error(err error) {
	n.log.Error(err)
}
