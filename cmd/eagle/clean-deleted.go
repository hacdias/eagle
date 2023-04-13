package main

import (
	"path/filepath"

	"github.com/hacdias/eagle/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cleanDeletedCmd)
}

var cleanDeletedCmd = &cobra.Command{
	Use: "clean-deleted",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := core.ParseConfig()
		if err != nil {
			return err
		}

		fs := core.NewFS(c.SourceDirectory, c.BaseURL, &core.NopSync{})
		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		for _, e := range ee {
			if e.Deleted() {
				err = fs.RemoveAll(filepath.Join(core.ContentDirectory, e.ID))
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
}
