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
	scrobblesCmd.Flags().StringP("mode", "m", "day", "The mode of the reports to create (day, week, month, year).")
	scrobblesCmd.MarkFlagRequired("from")
	scrobblesCmd.MarkFlagRequired("to")
}

var scrobblesCmd = &cobra.Command{
	Use: "scrobbles",
	RunE: func(cmd *cobra.Command, args []string) error {
		fromStr, _ := cmd.Flags().GetString("from")
		toStr, _ := cmd.Flags().GetString("to")
		mode, _ := cmd.Flags().GetString("mode")

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

		switch mode {
		case "day":
			for cur := from; !cur.Equal(to); cur = cur.AddDate(0, 0, 1) {
				year, month, day := cur.Date()

				fmt.Println("Downloading", cur.Format("2006-01-02"))

				err := e.FetchLastfmScrobbles(year, month, day)
				if err != nil {
					return err
				}

				time.Sleep(time.Second)
			}
		case "week":
			for cur := from; cur.Before(to); cur = cur.AddDate(0, 0, 7) {
				year, month, day := cur.Date()

				fmt.Println("Making Week Report", cur.Format("2006-01-02"))

				err := e.MakeWeeklyScrobblesReport(year, month, day)
				if err != nil {
					return err
				}
			}
		case "month":

		case "year":

		}

		return nil
	},
}
