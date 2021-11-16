package main

import (
	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mapsCmd)
}

var mapsCmd = &cobra.Command{
	Use: "maps",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Parse()
		if err != nil {
			return err
		}

		e, err := eagle.NewEagle(c)
		if err != nil {
			return err
		}
		defer e.Close()

		entries, err := e.GetEntries(false)
		if err != nil {
			return err
		}

		for _, ee := range entries {
			err = e.ProcessLocationMap(ee)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
