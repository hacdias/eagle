package notifier

import (
	"github.com/hacdias/eagle/v3/config"
	tb "gopkg.in/tucnak/telebot.v2"
)

type TelegramNotifier struct {
	errNotifier Notifier
	chat        int64
	bot         *tb.Bot
}

func NewTelegramNotifier(c *config.Telegram) (Notifier, error) {
	n := &TelegramNotifier{
		chat:        c.ChatID,
		errNotifier: NewLogNotifier(),
	}
	bot, err := tb.NewBot(tb.Settings{Token: c.Token})
	if err != nil {
		return nil, err
	}

	n.bot = bot
	return n, nil
}

func (n *TelegramNotifier) Info(msg string) {
	_, err := n.bot.Send(&tb.Chat{ID: n.chat}, msg, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err != nil {
		n.errNotifier.Error(err)
	}
}

func (n *TelegramNotifier) Error(err error) {
	n.errNotifier.Error(err)

	_, botErr := n.bot.Send(&tb.Chat{ID: n.chat}, "An error occurred:\n"+err.Error(), &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if botErr != nil {
		n.errNotifier.Error(err)
	}
}
