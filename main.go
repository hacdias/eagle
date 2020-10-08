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

	s, err := services.NewServices(c)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)

	go func() {
		log.Println("Starting server...")
		err := server.Start(c, s)
		if err != nil {
			log.Println("Failed to start server:")
			log.Println(err)
		}
		quit <- os.Interrupt
	}()

	log.Println("Starting bot...")
	bot, err := server.StartBot(&c.Telegram, s)
	if err != nil {
		log.Println("Failed to start bot:")
		log.Println(err)
		quit <- os.Interrupt
	}

	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Stopping bot...")
	bot.Stop()
}
