package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/server"
	"github.com/hacdias/eagle/services"
)

func main() {
	c, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}

	log := config.NewLogger(c)
	defer func() {
		_ = log.Sync()
	}()

	s, err := services.NewServices(c, log)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	server := server.NewServer(log.Named("server"), c, s)

	go func() {
		log.Info("starting server")
		err := server.StartHTTP()
		if err != nil {
			log.Errorf("failed to start server: %s", err)
		}
		quit <- os.Interrupt
	}()

	log.Info("starting bot")
	bot, err := server.StartBot()
	if err != nil {
		log.Errorf("failed to start bot: %s", err)
		quit <- os.Interrupt
	}

	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Info("stopping bot")
	bot.Stop()
}
