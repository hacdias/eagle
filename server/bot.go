package server

import (
	"strings"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
	"go.uber.org/zap"
	tb "gopkg.in/tucnak/telebot.v2"
)

func StartBot(log *zap.SugaredLogger, c *config.Telegram, s *services.Services) (*tb.Bot, error) {
	b, err := tb.NewBot(tb.Settings{
		Token:  c.Token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		return nil, err
	}

	logIfErr := func(err error) {
		if err != nil {
			log.Warn(err)
		}
	}

	checkUser := func(fn func(m *tb.Message)) func(m *tb.Message) {
		return func(m *tb.Message) {
			if m.Chat.ID != c.ChatID {
				_, err := b.Send(m.Sender, "This bot is not intended to you. Bye!")
				logIfErr(err)
			} else {
				fn(m)
			}
		}
	}

	b.Handle("/ping", checkUser(func(m *tb.Message) {
		_, err := b.Send(m.Sender, "pong")
		logIfErr(err)
	}))

	b.Handle("/sync", checkUser(func(m *tb.Message) {
		err := s.Store.Sync()
		if err != nil {
			s.Notify.Error(err)
		} else {
			s.Notify.Info("Sync was successfull!")
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
