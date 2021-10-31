package main

import (
	"errors"
	"net/http"
	"os"
	"os/signal"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/logging"
	"github.com/hacdias/eagle/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:               "eagle",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	Short:             "Eagle is a website CMS built around Hugo",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Parse()
		if err != nil {
			return err
		}

		defer func() {
			_ = logging.L().Sync()
		}()

		e, err := eagle.NewEagle(c)
		if err != nil {
			return err
		}

		quit := make(chan os.Signal, 1)
		server, err := server.NewServer(e)
		if err != nil {
			return err
		}

		log := logging.S()

		go func() {
			log.Info("starting server")
			err := server.Start()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Errorf("failed to start server: %s", err)
			}
			quit <- os.Interrupt
		}()

		signal.Notify(quit, os.Interrupt)
		<-quit

		log.Info("stopping server")
		_ = server.Stop()
		return nil
	},
}
