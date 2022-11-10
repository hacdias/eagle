package telegram

import (
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/log"
	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"
)

type Telegram struct {
	chat   int64
	errLog *zap.SugaredLogger
	bot    *tb.Bot
}

func NewTelegram(c *eagle.Telegram) (*Telegram, error) {
	n := &Telegram{
		chat:   c.ChatID,
		errLog: log.S().Named("telegram"),
	}
	bot, err := tb.NewBot(tb.Settings{Token: c.Token})
	if err != nil {
		return nil, err
	}

	n.bot = bot
	return n, nil
}

func (n *Telegram) Info(msg string) {
	_, err := n.bot.Send(&tb.Chat{ID: n.chat}, msg, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err != nil {
		n.errLog.Error(err)
	}
}

func (n *Telegram) Error(err error) {
	n.errLog.Error(err)

	_, botErr := n.bot.Send(&tb.Chat{ID: n.chat}, "An error occurred:\n"+err.Error(), &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if botErr != nil {
		n.errLog.Error(err)
	}
}
