package main

import (
	"os"
	"path/filepath"

	"github.com/hacdias/eagle/v4/config"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mapsCmd)
	mapsCmd.Flags().BoolP("skip", "s", false, "Skip already existent maps.")
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

		skip, _ := cmd.Flags().GetBool("skip")

		for _, ee := range entries {
			if skip {
				if _, err := os.Stat(filepath.Join(e.Config.SourceDirectory, eagle.ContentDirectory, ee.ID, "map.png")); err == nil {
					continue
				}

				if _, err := os.Stat(filepath.Join(e.Config.SourceDirectory, eagle.ContentDirectory, ee.ID, "map.jpeg")); err == nil {
					continue
				}
			}

			err = e.ProcessLocationMap(ee)
			if err != nil {
				e.Error(err)
			}
		}

		return nil
	},
}
