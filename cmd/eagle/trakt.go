package main

import (
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/services/trakt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(traktCmd)
	traktCmd.AddCommand(traktLoginCmd)
	traktCmd.AddCommand(traktFetchCmd)
	traktCmd.AddCommand(traktSummaryCmd)
}

var traktCmd = &cobra.Command{
	Use: "trakt",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var traktLoginCmd = &cobra.Command{
	Use: "login",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := eagle.ParseConfig()
		if err != nil {
			return err
		}

		t, err := trakt.NewTrakt(c.Trakt, nil)
		if err != nil {
			return err
		}

		return t.InteractiveLogin(8050)
	},
}

var traktFetchCmd = &cobra.Command{
	Use: "fetch",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := eagle.ParseConfig()
		if err != nil {
			return err
		}

		fs := fs.NewFS(c.Source.Directory, c.Server.BaseURL, &fs.NopSync{})
		t, err := trakt.NewTrakt(c.Trakt, fs)
		if err != nil {
			return err
		}

		return t.FetchAll(cmd.Context())
	},
}

var traktSummaryCmd = &cobra.Command{
	Use: "summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := eagle.ParseConfig()
		if err != nil {
			return err
		}

		fs := fs.NewFS(c.Source.Directory, c.Server.BaseURL, &fs.NopSync{})
		t, err := trakt.NewTrakt(c.Trakt, fs)
		if err != nil {
			return err
		}

		return t.UpdateWatches()
	},
}
