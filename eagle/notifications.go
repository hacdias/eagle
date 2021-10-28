package eagle

import (
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/logging"
	"go.uber.org/zap"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Notifications struct {
	chat int64
	bot  *tb.Bot
	log  *zap.SugaredLogger
}

func NewNotifications(c *config.Telegram) (*Notifications, error) {
	n := &Notifications{
		chat: c.ChatID,
		log:  logging.S().Named("notify"),
	}
	bot, err := tb.NewBot(tb.Settings{Token: c.Token})
	if err != nil {
		return nil, err
	}

	n.bot = bot
	return n, nil
}

func (n *Notifications) Notify(msg string) {
	_, err := n.bot.Send(&tb.Chat{ID: n.chat}, msg, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err != nil {
		n.log.Error(err)
	}
}

func (n *Notifications) NotifyError(err error) {
	_, botErr := n.bot.Send(&tb.Chat{ID: n.chat}, "An error occurred:\n"+err.Error(), &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if botErr != nil {
		n.log.Error(botErr)
	}
}
