package main

import (
	"errors"
	"net/http"
	"os"
	"os/signal"

	"github.com/hacdias/eagle/core"
	"github.com/hacdias/eagle/log"
	"github.com/hacdias/eagle/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:               "eagle",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	Short:             "Eagle is a website CMS",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := core.ParseConfig()
		if err != nil {
			return err
		}

		defer func() {
			_ = log.L().Sync()
		}()

		quit := make(chan os.Signal, 1)
		server, err := server.NewServer(c)
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
