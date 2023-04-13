package main

import (
	"github.com/hacdias/eagle/core"
	"github.com/hacdias/eagle/core/helpers"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(descriptionsCmd)
	descriptionsCmd.Flags().BoolP("force", "f", false, "Force generation.")
}

var descriptionsCmd = &cobra.Command{
	Use: "descriptions",
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

		force, _ := cmd.Flags().GetBool("force")

		for _, e := range ee {
			helpers.GenerateDescription(e, force)
			err = fs.SaveEntry(e)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
