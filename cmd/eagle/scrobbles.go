package main

import (
	"fmt"
	"time"

	"github.com/hacdias/eagle/v3/config"
	"github.com/hacdias/eagle/v3/eagle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(scrobblesCmd)
	scrobblesCmd.Flags().StringP("from", "f", "", "From date to start fetching scrobbles (including).")
	scrobblesCmd.Flags().StringP("to", "t", "", "To date to start fetching scrobbles (not including).")
	scrobblesCmd.MarkFlagRequired("from")
	scrobblesCmd.MarkFlagRequired("to")
}

var scrobblesCmd = &cobra.Command{
	Use: "scrobbles",
	RunE: func(cmd *cobra.Command, args []string) error {
		fromStr, _ := cmd.Flags().GetString("from")
		toStr, _ := cmd.Flags().GetString("to")

		from, err := time.ParseInLocation("2006-01-02", fromStr, time.UTC)
		if err != nil {
			return err
		}

		to, err := time.ParseInLocation("2006-01-02", toStr, time.UTC)
		if err != nil {
			return err
		}

		c, err := config.Parse()
		if err != nil {
			return err
		}

		e, err := eagle.NewEagle(c)
		if err != nil {
			return err
		}
		defer e.Close()

		for day := from; !day.Equal(to); day = day.AddDate(0, 0, 1) {
			year, month, day := day.Date()

			fmt.Println(day)

			err := e.FetchLastfmScrobbles(year, month, day)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
