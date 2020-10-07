package server

import (
	"strings"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
	tb "gopkg.in/tucnak/telebot.v2"
)

func StartBot(c *config.Telegram, s *services.Services) (*tb.Bot, error) {
	b, err := tb.NewBot(tb.Settings{
		Token:  c.Token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		return nil, err
	}

	checkUser := func(fn func(m *tb.Message)) func(m *tb.Message) {
		return func(m *tb.Message) {
			if m.Chat.ID != c.ChatID {
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
		err := s.Git.Push()
		if err != nil {
			s.Notify.Error(err)
		} else {
			s.Notify.Info("Push was successfull!")
		}
	}))

	b.Handle("/pull", checkUser(func(m *tb.Message) {
		err := s.Git.Pull()
		if err != nil {
			s.Notify.Error(err)
		} else {
			s.Notify.Info("Pull was successfull!")
		}
	}))

	b.Handle("/build", checkUser(func(m *tb.Message) {
		clean := strings.Contains(m.Text, "clean")
		err := s.Hugo.Build(clean)
		if err != nil {
			s.Notify.Error(err)
		} else {
			s.Notify.Info("Build was successfull!")
		}
	}))

	go b.Start()
	return b, nil
}
