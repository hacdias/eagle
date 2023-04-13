package main

import (
	"github.com/hacdias/eagle/core"
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
		c, err := core.ParseConfig()
		if err != nil {
			return err
		}

		fs := core.NewFS(c.SourceDirectory, c.Server.BaseURL, &core.NopSync{})
		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		gen := hooks.NewDescriptionGenerator(fs)
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
