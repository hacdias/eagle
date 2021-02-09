package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
)

func main() {
	moveWebmentions()
}

func moveWebmentions() {
	c, err := config.Get()
	if err != nil {
		panic(err)
	}

	defer func() {
		_ = c.L().Sync()
	}()

	eagle, err := services.NewEagle(c)
	if err != nil {
		panic(err)
	}

	entries, err := eagle.GetAll()
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(filepath.Join(c.Hugo.Source, "data", "interactions"), 0777)
	if err != nil {
		panic(err)
	}

	fmt.Printf("got %d entries\n", len(entries))

	for _, entry := range entries {
		wmpath := filepath.Dir(entry.Path)
		wmpath = filepath.Join(wmpath, "mentions.json")

	}
}

func testSave() {
	c, err := config.Get()
	if err != nil {
		panic(err)
	}

	defer func() {
		_ = c.L().Sync()
	}()

	eagle, err := services.NewEagle(c)
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
