package server

import (
	"errors"
	"fmt"
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

func initFS(c *core.Config) *core.FS {
	var srcSync core.Sync
	if c.Development {
		srcSync = &core.NopSync{}
	} else {
		srcSync = core.NewGitSync(c.SourceDirectory)
	}

	return core.NewFS(c.SourceDirectory, c.BaseURL, srcSync)
}

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

func (s *Server) initBadger() error {
	var err error
	s.badger, err = database.NewDatabase(filepath.Join(s.c.DataDirectory, "bolt.db"))
	return err
}

func (s *Server) initMeiliSearch() error {
	var err error
	if s.c.MeiliSearch != nil {
		s.meilisearch, err = meilisearch.NewMeiliSearch(s.c.MeiliSearch.Endpoint, s.c.MeiliSearch.Key, s.fs)
	}
	return err
}

func (s *Server) initPlugins() error {
	s.plugins = map[string]Plugin{}
	for pluginName, pluginInitializer := range pluginRegistry {
		cfg, ok := s.c.Plugins[pluginName]
		if ok {
			plugin, err := pluginInitializer(s.fs, cfg)
			if err != nil {
				return err
			}
			s.plugins[pluginName] = plugin
		}
	}
	return nil
}

func (s *Server) initActions() error {
	err := errors.Join(
		s.registerAction("Build Website", func() error {
			return s.hugo.Build(false)
		}),
		s.registerAction("Build Website (Clean)", func() error {
			return s.hugo.Build(true)
		}),
		s.registerAction("Sync Storage", func() error {
			go s.syncStorage()
			return nil
		}),
		s.registerAction("Reset Index", func() error {
			s.indexAll()
			return nil
		}),
		s.registerAction("Reload Redirects", s.loadRedirects),
		s.registerAction("Reload Gone", s.loadGone),
	)

	for _, plugin := range s.plugins {
		name, action := plugin.GetAction()
		if name != "" && action != nil {
			err = errors.Join(err, s.registerAction(name, action))
		}
	}

	return err
}

func (s *Server) initExternalLinks() error {
	// TODO: make this a plugin. Allow plugins to have their own HTTP handler.
	s.cronJobs = append(s.cronJobs, s.fs.UpdateExternalLinks)
	return s.registerActionWithRebuild("Update External Links", func() error {
		err := s.fs.UpdateExternalLinks()
		if err != nil {
			return err
		}
		return s.loadLinks()
	})
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
