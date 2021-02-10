package eagle

import (
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/logging"
	"go.uber.org/zap"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Notifications struct {
	b    *tb.Bot
	conf *config.Telegram
	log  *zap.SugaredLogger
}

func NewNotifications(c *config.Telegram) (*Notifications, error) {
	n := &Notifications{
		conf: c,
		log:  logging.S().Named("notify"),
	}
	b, err := tb.NewBot(tb.Settings{Token: n.conf.Token})
	if err != nil {
		return nil, err
	}

	n.b = b
	return n, nil
}

func (n *Notifications) Notify(msg string) {
	_, err := n.b.Send(&tb.Chat{ID: n.conf.ChatID}, msg, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err != nil {
		n.log.Error(err)
	}
}

func (n *Notifications) NotifyError(not error) {
	_, err := n.b.Send(&tb.Chat{ID: n.conf.ChatID}, "An error occurred:\n"+not.Error(), &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err != nil {
		n.log.Error(err)
	}
}
