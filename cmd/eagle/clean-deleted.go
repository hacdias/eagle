package main

import (
	"path/filepath"

	"github.com/hacdias/eagle/eagle"
	eaglefs "github.com/hacdias/eagle/fs"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cleanDeletedCmd)
}

var cleanDeletedCmd = &cobra.Command{
	Use: "clean-deleted",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := eagle.ParseConfig()
		if err != nil {
			return err
		}

		fs := eaglefs.NewFS(c.SourceDirectory, c.Server.BaseURL, &eaglefs.NopSync{})
		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		for _, e := range ee {
			if e.Deleted() {
				err = fs.RemoveAll(filepath.Join(eaglefs.ContentDirectory, e.ID))
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
}
