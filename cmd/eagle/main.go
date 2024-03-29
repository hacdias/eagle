package main

import (
	"fmt"
	"os"

	_ "go.hacdias.com/eagle/plugins/external-links"
	_ "go.hacdias.com/eagle/plugins/linkding"
	_ "go.hacdias.com/eagle/plugins/miniflux"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
