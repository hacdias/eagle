package main

import (
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

type telegramSettings struct {
	Token  string
	ChatID int64
}

func newBot(s *telegramSettings) {
	b, err := tb.NewBot(tb.Settings{
		Token:  s.Token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
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
	}))

	b.Handle("/pull", checkUser(func(m *tb.Message) {
		// TODO
	}))

	b.Handle("/build", checkUser(func(m *tb.Message) {
		// TODO
	}))

	b.Start()
}
