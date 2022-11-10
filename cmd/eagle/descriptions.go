package main

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(descriptionsCmd)
	descriptionsCmd.Flags().BoolP("force", "f", false, "Force generation.")
}

var descriptionsCmd = &cobra.Command{
	Use: "descriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		// c, err := config.Parse()
		// if err != nil {
		// 	return err
		// }

		// e, err := eagle.NewEagle(c)
		// if err != nil {
		// 	return err
		// }
		// defer e.Close()

		// entries, err := e.GetEntries(false)
		// if err != nil {
		// 	return err
		// }

		// force, _ := cmd.Flags().GetBool("force")
		// generator := &hooks.DescriptionGenerator{}

		// for _, ee := range entries {
		// 	err = generator.GenerateDescription(ee, force)
		// 	if err != nil {
		// 		return err
		// 	}

		// 	err = e.SaveEntry(ee)
		// 	if err != nil {
		// 		return err
		// 	}
		// }

		return nil
	},
}
