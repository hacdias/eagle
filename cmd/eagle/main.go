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

	defer func() {
		_ = c.L().Sync()
	}()

	e, err := services.NewEagle(c)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	server := server.NewServer(c, e)

	go func() {
		c.S().Info("starting server")
		err := server.StartHTTP()
		if err != nil {
			c.S().Errorf("failed to start server: %s", err)
		}
		quit <- os.Interrupt
	}()

	c.S().Info("starting bot")
	bot, err := server.StartBot()
	if err != nil {
		c.S().Errorf("failed to start bot: %s", err)
		quit <- os.Interrupt
	}

	signal.Notify(quit, os.Interrupt)
	<-quit

	c.S().Info("stopping bot")
	bot.Stop()
}
