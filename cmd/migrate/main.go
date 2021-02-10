package main

import (
	"fmt"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/eagle"
)

func main() {
	c, err := config.Parse()
	if err != nil {
		panic(err)
	}

	eagle, err := eagle.NewEagle(c)
	if err != nil {
		panic(err)
	}

	entries, err := eagle.GetAll()
	if err != nil {
		panic(err)
	}

	fmt.Printf("got %d entries\n", len(entries))

	for _, entry := range entries {
		entry.Path = ""
		err = eagle.SaveEntry(entry)
		if err != nil {
			panic(err)
		}
	}
}
