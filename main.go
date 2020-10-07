package main

import (
	"log"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/server"
)

func main() {
	conf, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}

	server.Start(conf)
}
