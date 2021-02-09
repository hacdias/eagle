package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/logging"
	"github.com/hacdias/eagle/server"
	"github.com/hacdias/eagle/services"
)

func main() {
	c, err := config.Parse()
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		_ = logging.L().Sync()
	}()

	e, err := services.NewEagle(c)
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
		if err != nil {
			log.Errorf("failed to start server: %s", err)
		}
		quit <- os.Interrupt
	}()

	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Info("stopping server")
	server.Stop()
}
