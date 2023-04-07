package main

import (
	"github.com/hacdias/eagle/eagle"
	eaglefs "github.com/hacdias/eagle/fs"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cleanSidecarCmd)
}

var cleanSidecarCmd = &cobra.Command{
	Use: "clean-sidecar",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := eagle.ParseConfig()
		if err != nil {
			return err
		}

		fs := eaglefs.NewFS(c.Source.Directory, c.Server.BaseURL, &eaglefs.NopSync{})
		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		for _, e := range ee {
			err = fs.UpdateSidecar(e, func(s *eagle.Sidecar) (*eagle.Sidecar, error) {
				return s, nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	},
}
