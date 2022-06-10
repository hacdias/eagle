package main

import (
	"errors"
	"net/http"
	"os"
	"os/signal"

	"github.com/hacdias/eagle/v4/config"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/log"
	"github.com/hacdias/eagle/v4/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:               "eagle",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	Short:             "Eagle is a website CMS",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Parse()
		if err != nil {
			return err
		}

		defer func() {
			_ = log.L().Sync()
		}()

		e, err := eagle.NewEagle(c)
		if err != nil {
			return err
		}
		defer e.Close()

		quit := make(chan os.Signal, 1)
		server, err := server.NewServer(e)
		if err != nil {
			return err
		}

		log := log.S()

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
