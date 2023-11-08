package main

import (
	"github.com/spf13/cobra"
	"go.hacdias.com/eagle/core"
)

func init() {
	rootCmd.AddCommand(externalLinksCmd)
}

var externalLinksCmd = &cobra.Command{
	Use: "external-links",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := core.ParseConfig()
		if err != nil {
			return err
		}

		fs := core.NewFS(c.SourceDirectory, c.BaseURL, &core.NopSync{})
		return fs.UpdateExternalLinks()
	},
}
