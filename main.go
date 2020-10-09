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

	s, err := services.NewServices(c)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)

	go func() {
		log.Info("Starting server")
		err := server.Start(log.Named("server"), c, s)
		if err != nil {
			log.Errorf("Failed to start server: %s", err)
		}
		quit <- os.Interrupt
	}()

	log.Info("Starting bot")
	bot, err := server.StartBot(log.Named("bot"), &c.Telegram, s)
	if err != nil {
		log.Errorf("Failed to start bot: %s", err)
		quit <- os.Interrupt
	}

	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Info("Stopping bot")
	bot.Stop()
}
