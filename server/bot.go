package server

import (
	"strings"
	"time"

	"github.com/prometheus/common/log"
	tb "gopkg.in/tucnak/telebot.v2"
)

func (s *Server) StartBot() (*tb.Bot, error) {
	b, err := tb.NewBot(tb.Settings{
		Token:  s.c.Telegram.Token,
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
		err := s.Store.Sync()
		if err != nil {
			s.Notify.Error(err)
		} else {
			s.Notify.Info("Sync was successfull! ⚡️")
		}
	}))

	b.Handle("/build", checkUser(func(m *tb.Message) {
		clean := strings.Contains(m.Text, "clean")
		err := s.Hugo.Build(clean)
		if err != nil {
			s.Notify.Error(err)
		} else {
			s.Notify.Info("Build was successfull! 💪")
		}
	}))

	b.Handle("/build_index", checkUser(func(m *tb.Message) {
		if s.MeiliSearch == nil {
			s.Notify.Info("MeiliSearch is not implemented!")
			return
		}

		s.Lock()
		entries, err := s.Hugo.GetAll()
		if err != nil {
			s.Notify.Error(err)
			s.Unlock()
			return
		}
		s.Unlock()

		err = s.MeiliSearch.Add(entries...)
		if err != nil {
			s.Notify.Error(err)
			return
		}

		s.Notify.Info("Successfully indexed! 🔎")
	}))

	b.Handle("/delete_index", checkUser(func(m *tb.Message) {
		if s.MeiliSearch == nil {
			s.Notify.Info("MeiliSearch is not implemented!")
			return
		}

		err = s.MeiliSearch.Wipe()
		if err != nil {
			s.Notify.Error(err)
			return
		}

		s.Notify.Info("Search index wiped! 🔎")
	}))

	b.Handle("/webmentions", checkUser(func(m *tb.Message) {
		id := strings.TrimSpace(strings.TrimPrefix(m.Text, "/webmentions"))

		entry, err := s.Hugo.GetEntry(id)
		if err != nil {
			s.Notify.Error(err)
			return
		}

		s.sendWebmentions(entry)
	}))

	b.Handle("/activity", checkUser(func(m *tb.Message) {
		id := strings.TrimSpace(strings.TrimPrefix(m.Text, "/activity"))

		entry, err := s.Hugo.GetEntry(id)
		if err != nil {
			s.Notify.Error(err)
			return
		}

		s.activity(entry)
	}))

	go b.Start()
	return b, nil
}
