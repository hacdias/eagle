package services

import (
	"github.com/hacdias/eagle/config"
	"go.uber.org/zap"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Notifications struct {
	*zap.SugaredLogger
	*config.Telegram
	b *tb.Bot
}

func NewNotifications(c *config.Telegram, log *zap.SugaredLogger) (*Notifications, error) {
	n := &Notifications{
		Telegram:      c,
		SugaredLogger: log,
	}
	b, err := tb.NewBot(tb.Settings{Token: n.Token})
	if err != nil {
		return nil, err
	}

	n.b = b
	return n, nil
}

func (n *Notifications) Notify(msg string) {
	_, err := n.b.Send(&tb.Chat{ID: n.ChatID}, msg, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err != nil {
		n.Errorf("could not notify: %s", err)
	}
}

func (n *Notifications) NotifyError(err error) {
	_, err2 := n.b.Send(&tb.Chat{ID: n.ChatID}, "An error occurred:\n"+err.Error(), &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err2 != nil {
		n.Errorf("could not notify: %s", err2)
	}
}
