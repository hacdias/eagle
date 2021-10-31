package main

import (
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/eagle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateTargetsCmd)
}

var updateTargetsCmd = &cobra.Command{
	Use:   "update-targets",
	Short: "Update the posts data files with the current targets.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Parse()
		if err != nil {
			return err
		}

		e, err := eagle.NewEagle(c)
		if err != nil {
			return err
		}

		err = e.Build(true)
		if err != nil {
			return err
		}

		entries, err := e.GetAllEntries()
		if err != nil {
			return err
		}

		for _, entry := range entries {
			err = e.UpdateTargets(entry)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
