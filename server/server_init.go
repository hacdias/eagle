package server

import (
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"time"

	"github.com/maypok86/otter/v2"
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

func (s *Server) initMediaCache() error {
	cache, err := otter.New(&otter.Options[string, []byte]{
		MaximumWeight:    2e8, // 200 MB
		ExpiryCalculator: otter.ExpiryWriting[string, []byte](time.Hour),
		Weigher: func(key string, value []byte) uint32 {
			return uint32(len(value))
		},
	})

	s.mediaCache = cache
	return err
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
	htmlTemplates, err := template.
		New("").
		Funcs(template.FuncMap{
			"urlParse": url.Parse,
		}).
		ParseGlob(filepath.Join(s.c.SourceDirectory, "eagle", "*.html"))
	if err != nil {
		return err
	}

	for _, template := range []string{errorTemplate} {
		if htmlTemplates.Lookup(template) == nil {
			return fmt.Errorf("template %s missing", template)
		}
	}

	s.templates = htmlTemplates
	return nil
}

func (s *Server) initBolt() error {
	var err error
	s.bolt, err = database.NewDatabase(filepath.Join(s.c.DataDirectory, "bolt.db"))
	return err
}

func (s *Server) initMeilisearch() error {
	var err error
	if s.c.Meilisearch != nil {
		s.meilisearch, err = meilisearch.NewMeilisearch(s.c.Meilisearch.Endpoint, s.c.Meilisearch.Key, s.c.Meilisearch.Taxonomies, s.core)
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

func (s *Server) initSyndicators() error {
	s.syndicators = map[string]SyndicationPlugin{}
	for _, plugin := range s.plugins {
		syndicationPlugin, ok := plugin.(SyndicationPlugin)
		if !ok {
			continue
		}

		config := syndicationPlugin.Syndication()
		s.syndicators[config.UID] = syndicationPlugin
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

	for pluginName, plugin := range s.plugins {
		actionPlugin, ok := plugin.(ActionPlugin)
		if !ok {
			continue
		}

		actionName := actionPlugin.ActionName()
		if actionName == "" {
			return fmt.Errorf("plugin %s has no action name", pluginName)
		}

		if _, ok := actions[actionName]; ok {
			return fmt.Errorf("action %s already registered", actionName)
		}

		actions[actionName] = func() error {
			err := actionPlugin.Action()
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
			cronPlugin, ok := plugin.(CronPlugin)
			if !ok {
				continue
			}

			if err := cronPlugin.DailyCron(); err != nil {
				s.log.Errorw("plugin cron job execution failed", "plugin", name, "err", err)
			}
		}

		s.syncStorage()
	})
	return err
}
