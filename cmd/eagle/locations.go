package main

import (
	"github.com/hacdias/eagle/v3/config"
	"github.com/hacdias/eagle/v3/eagle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(locationsCmd)
}

var locationsCmd = &cobra.Command{
	Use: "locations",
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
			err = e.ProcessLocation(ee)
			if err != nil {
				e.Error(err)
			}
		}

		return nil
	},
}
