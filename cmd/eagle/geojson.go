package main

import (
	"github.com/hacdias/eagle/v4/config"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(geoJson)
}

var geoJson = &cobra.Command{
	Use: "geojson",
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
			err = e.GenerateGeoJSON(ee)
			if err != nil {
				e.Error(err)
			}
		}

		return nil
	},
}
