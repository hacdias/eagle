package server

import (
	"fmt"
	"html/template"
	"path/filepath"

	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/services/bunny"
	"go.hacdias.com/eagle/services/database"
	"go.hacdias.com/eagle/services/imgproxy"
	"go.hacdias.com/eagle/services/media"
	"go.hacdias.com/eagle/services/meilisearch"
	"go.hacdias.com/eagle/services/telegram"
)

func initMedia(c *core.Config) *media.Media {
	var (
		storage     media.Storage
		transformer media.Transformer
	)
	if c.BunnyCDN != nil {
		storage = bunny.NewBunny(c.BunnyCDN)
	}
	if c.ImgProxy != nil {
		transformer = imgproxy.NewImgProxy(c.ImgProxy)
	}
	if storage != nil {
		return media.NewMedia(storage, transformer)
	}
	return nil
}

func (s *Server) initNotifier() error {
	var err error
	if s.c.Notifications.Telegram != nil {
		s.n, err = telegram.NewTelegram(s.c.Notifications.Telegram)
	} else {
		s.n = log.NewLogNotifier()
	}
	return err
}

func (s *Server) initTemplates() error {
	htmlTemplates, err := template.ParseGlob(filepath.Join(s.c.SourceDirectory, "eagle", "*.html"))
	if err != nil {
		return err
	}
	for _, template := range templates {
		if htmlTemplates.Lookup(template) == nil {
			return fmt.Errorf("template %s missing", template)
		}
	}

	s.templates = htmlTemplates
	return nil
}

func (s *Server) initBadger() error {
	var err error
	s.badger, err = database.NewDatabase(filepath.Join(s.c.DataDirectory, "bolt.db"))
	return err
}

func (s *Server) initMeiliSearch() error {
	var err error
	if s.c.MeiliSearch != nil {
		s.meilisearch, err = meilisearch.NewMeiliSearch(s.c.MeiliSearch.Endpoint, s.c.MeiliSearch.Key, s.core)
	}
	return err
}

func (s *Server) initPlugins() error {
	s.plugins = map[string]Plugin{}
	for pluginName, pluginInitializer := range pluginRegistry {
		cfg, ok := s.c.Plugins[pluginName]
		if ok {
			plugin, err := pluginInitializer(s.core, cfg)
			if err != nil {
				return err
			}
			s.plugins[pluginName] = plugin
		}
	}
	return nil
}

func (s *Server) initActions() error {
	actions := map[string]func() error{
		"Build Website": func() error {
			return s.core.Build(false)
		},
		"Build Website (Clean)": func() error {
			return s.core.Build(true)
		},
		"Sync Storage": func() error {
			go s.syncStorage()
			return nil
		},
		"Reset Index": func() error {
			s.indexAll()
			return nil
		},
		"Reload Redirects": s.loadRedirects,
		"Reload Gone":      s.loadGone,
	}

	for _, plugin := range s.plugins {
		name, action := plugin.GetAction()
		if name == "" || action == nil {
			continue
		}

		if _, ok := actions[name]; ok {
			return fmt.Errorf("action %s already registered", name)
		}

		actions[name] = func() error {
			err := action()
			if err != nil {
				return err
			}
			return s.core.Build(false)
		}
	}

	s.actions = actions
	return nil
}

func (s *Server) initCron() error {
	_, err := s.cron.AddFunc("00 05 * * *", func() {
		for name, plugin := range s.plugins {
			if job := plugin.GetDailyCron(); job != nil {
				if err := job(); err != nil {
					s.n.Error(fmt.Errorf("cron job (plugin %s): %w", name, err))
				}
			}
		}

		for _, job := range s.cronJobs {
			if err := job(); err != nil {
				s.n.Error(fmt.Errorf("cron job: %w", err))
			}
		}

		s.syncStorage()
	})
	return err
}
