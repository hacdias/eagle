package main

import (
	"github.com/hacdias/eagle/core"
	"github.com/hacdias/eagle/hooks"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(locationsCmd)
}

var locationsCmd = &cobra.Command{
	Use: "locations",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := core.ParseConfig()
		if err != nil {
			return err
		}

		fs := core.NewFS(c.SourceDirectory, c.Server.BaseURL, &core.NopSync{})
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
