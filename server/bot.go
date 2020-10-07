package server

import (
	"time"

	"github.com/hacdias/eagle/config"
	tb "gopkg.in/tucnak/telebot.v2"
)

func StartBot(s *config.Telegram) (*tb.Bot, error) {
	b, err := tb.NewBot(tb.Settings{
		Token:  s.Token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		return nil, err
	}

	checkUser := func(fn func(m *tb.Message)) func(m *tb.Message) {
		return func(m *tb.Message) {
			if m.Chat.ID != s.ChatID {
				b.Send(m.Sender, "This bot is not intended to you. Bye!")
			} else {
				fn(m)
			}
		}
	}

	b.Handle("/ping", checkUser(func(m *tb.Message) {

		b.Send(m.Sender, "pong")
	}))

	b.Handle("/push", checkUser(func(m *tb.Message) {
		// TODO
		b.Send(m.Sender, "push")
	}))

	b.Handle("/pull", checkUser(func(m *tb.Message) {
		// TODO
		b.Send(m.Sender, "pull")
	}))

	b.Handle("/build", checkUser(func(m *tb.Message) {
		// TODO
		b.Send(m.Sender, "build")
	}))

	go b.Start()
	return b, nil
}
