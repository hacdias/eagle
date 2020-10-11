package services

import (
	"log"

	"github.com/hacdias/eagle/config"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Notify struct {
	*config.Telegram
	b *tb.Bot
}

func NewNotify(c *config.Telegram) (*Notify, error) {
	n := &Notify{Telegram: c}
	b, err := tb.NewBot(tb.Settings{Token: n.Token})
	if err != nil {
		return nil, err
	}

	n.b = b
	return n, nil
}

func (n *Notify) Info(msg string) {
	_, err := n.b.Send(&tb.Chat{ID: n.ChatID}, msg, &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeMarkdown,
	})

	if err != nil {
		log.Printf("could not notify: %s", err)
	}
}

func (n *Notify) Error(err error) {
	_, err2 := n.b.Send(&tb.Chat{ID: n.ChatID}, "An error occurred:\n"+err.Error(), &tb.SendOptions{
		DisableWebPagePreview: true,
		ParseMode:             tb.ModeDefault,
	})

	if err2 != nil {
		log.Printf("could not notify: %s", err2)
	}
}
