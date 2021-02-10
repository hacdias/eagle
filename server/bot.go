package server

import (
	"strings"
	"time"

	"github.com/prometheus/common/log"
	tb "gopkg.in/tucnak/telebot.v2"
)

func (s *Server) buildBot() error {
	b, err := tb.NewBot(tb.Settings{
		Token:  s.c.Telegram.Token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		return err
	}

	logIfErr := func(err error) {
		if err != nil {
			log.Warn(err)
		}
	}

	checkUser := func(fn func(m *tb.Message)) func(m *tb.Message) {
		return func(m *tb.Message) {
			if m.Chat.ID != s.c.Telegram.ChatID {
				_, err := b.Send(m.Sender, "This bot is not intended to you. 👋")
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
		err := s.e.Sync()
		if err != nil {
			s.e.NotifyError(err)
		} else {
			s.e.Notify("Sync was successfull! ⚡️")
		}
	}))

	b.Handle("/build", checkUser(func(m *tb.Message) {
		clean := strings.Contains(m.Text, "clean")
		err := s.e.Build(clean)
		if err != nil {
			s.e.NotifyError(err)
		} else {
			s.e.Notify("Build was successfull! 💪")
		}
	}))

	b.Handle("/rebuild_index", checkUser(func(m *tb.Message) {
		err = s.e.RebuildIndex()
		if err != nil {
			s.e.NotifyError(err)
			return
		}

		s.e.Notify("Search index rebuilt! 🔎")
	}))

	b.Handle("/webmentions", checkUser(func(m *tb.Message) {
		id := strings.TrimSpace(strings.TrimPrefix(m.Text, "/webmentions"))

		entry, err := s.e.GetEntry(id)
		if err != nil {
			s.e.NotifyError(err)
			return
		}

		s.sendWebmentions(entry)
	}))

	b.Handle("/activity", checkUser(func(m *tb.Message) {
		id := strings.TrimSpace(strings.TrimPrefix(m.Text, "/activity"))

		entry, err := s.e.GetEntry(id)
		if err != nil {
			s.e.NotifyError(err)
			return
		}

		s.activity(entry)
	}))

	s.bot = b
	return nil
}
