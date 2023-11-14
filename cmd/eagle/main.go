package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"go.hacdias.com/eagle/config"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/server"
)

func init() {
	mainCmd.PersistentFlags().StringP("configDir", "c", ".", "directory with eagle configuration")
}

func main() {
	if err := mainCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var mainCmd = &cobra.Command{
	Use:               "eagle",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	Short:             "Eagle is a website CMS",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := cmd.Flags().GetString("configDir")
		if err != nil {
			return err
		}

		c, err := config.ReadConfig(dir)
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
