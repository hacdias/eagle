package eagle

import (
	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/logging"
	"go.uber.org/zap"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Notifications interface {
	// Notify should notify the administrator of a certain message.
	Notify(msg string)
	// NotifyError should notify the administrator of the error and log it.
	NotifyError(err error)
}

type tgNotifications struct {
	logNotifications Notifications
	chat             int64
	bot              *tb.Bot
}

func newTgNotifications(c *config.Telegram) (Notifications, error) {
	n := &tgNotifications{
		chat:             c.ChatID,
		logNotifications: newLogNotifications(),
	}
	bot, err := tb.NewBot(tb.Settings{Token: c.Token})
	if err != nil {
		return nil, err
	}

	n.bot = bot
	return n, nil
}

func (n *tgNotifications) Notify(msg string) {
	_, err := n.bot.Send(&tb.Chat{ID: n.chat}, msg, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err != nil {
		n.logNotifications.NotifyError(err)
	}
}

func (n *tgNotifications) NotifyError(err error) {
	n.logNotifications.NotifyError(err)

	_, botErr := n.bot.Send(&tb.Chat{ID: n.chat}, "An error occurred:\n"+err.Error(), &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if botErr != nil {
		n.logNotifications.NotifyError(err)
	}
}

type logNotifications struct {
	*zap.SugaredLogger
}

func newLogNotifications() Notifications {
	return &logNotifications{
		SugaredLogger: logging.S().Named("notify"),
	}
}

func (n *logNotifications) Notify(msg string) {
	n.Info(msg)
}

func (n *logNotifications) NotifyError(err error) {
	n.Error(err)
}
