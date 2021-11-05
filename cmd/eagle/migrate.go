package main

import (
	"github.com/hacdias/eagle/v2/migrate"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate from the Hugo based website",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrate.Migrate()
	},
}
