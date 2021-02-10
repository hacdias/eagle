package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/logging"
	"github.com/hacdias/eagle/server"
)

func main() {
	c, err := config.Parse()
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		_ = logging.L().Sync()
	}()

	e, err := eagle.NewEagle(c)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	server, err := server.NewServer(c, e)
	if err != nil {
		log.Fatal(err)
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
}
