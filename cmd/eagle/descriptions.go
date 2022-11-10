package main

import (
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/hooks"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(descriptionsCmd)
	descriptionsCmd.Flags().BoolP("force", "f", false, "Force generation.")
}

var descriptionsCmd = &cobra.Command{
	Use: "descriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := eagle.ParseConfig()
		if err != nil {
			return err
		}

		fs := fs.NewFS(c.Source.Directory, c.Server.BaseURL, &fs.NopSync{})
		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		gen := &hooks.DescriptionGenerator{}
		force, _ := cmd.Flags().GetBool("force")

		for _, e := range ee {
			err = gen.GenerateDescription(e, force)
			if err != nil {
				return err
			}

			err = fs.SaveEntry(e)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
