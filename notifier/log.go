package notifier

import (
	"github.com/hacdias/eagle/v2/logging"
	"go.uber.org/zap"
)

type LogNotifier struct {
	log *zap.SugaredLogger
}

func NewLogNotifier() Notifier {
	return &LogNotifier{
		log: logging.S().Named("notify"),
	}
}

func (n *LogNotifier) Info(msg string) {
	n.log.Info(msg)
}

func (n *LogNotifier) Error(err error) {
	n.log.Error(err)
}
