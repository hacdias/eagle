package telegram

import (
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"
)

type Telegram struct {
	chat int64
	log  *zap.SugaredLogger
	bot  *tb.Bot
}

func NewTelegram(c *core.Telegram) (*Telegram, error) {
	n := &Telegram{
		chat: c.ChatID,
		log:  log.S().Named("telegram"),
	}
	bot, err := tb.NewBot(tb.Settings{Token: c.Token})
	if err != nil {
		return nil, err
	}

	n.bot = bot
	return n, nil
}

func (n *Telegram) Notify(msg string) {
	_, err := n.bot.Send(&tb.Chat{ID: n.chat}, msg, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err != nil {
		n.log.Error(err)
	}
}
