package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"go.hacdias.com/eagle/core"
)

func init() {
	rootCmd.AddCommand(duplicateDateCmd)
}

var duplicateDateCmd = &cobra.Command{
	Use: "duplicate-date",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := core.ParseConfig("")
		if err != nil {
			return err
		}

		co, err := core.NewCore(c)
		if err != nil {
			return err
		}

		ee, err := co.GetEntries(false)
		if err != nil {
			return err
		}

		mapDates := make(map[string][]*core.Entry)

		for _, e := range ee {
			if e.Date.IsZero() {
				continue
			}

			mapDates[e.Date.Format(time.RFC3339)] = append(mapDates[e.Date.Format(time.RFC3339)], e)
		}

		for date, entries := range mapDates {
			if len(entries) > 1 {
				fmt.Printf("%s: %d entries\n", date, len(entries))
				for _, e := range entries {
					fmt.Printf("  - %s\n", e.ID)
				}
			}
		}

		return nil
	},
}
