package main

import (
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/hooks"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(locationsCmd)
}

var locationsCmd = &cobra.Command{
	Use: "locations",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := eagle.ParseConfig()
		if err != nil {
			return err
		}

		fs := fs.NewFS(c.SourceDirectory, c.Server.BaseURL, &fs.NopSync{})
		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		locationsFetcher := hooks.NewLocationFetcher(fs, c.Site.Language)
		for _, e := range ee {
			err = locationsFetcher.FetchLocation(e)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
