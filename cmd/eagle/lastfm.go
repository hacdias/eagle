package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hacdias/eagle/v4/config"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(lastfmCmd)
	lastfmCmd.Flags().StringP("from", "f", "", "Start date to start fetching scrobbles.")
	lastfmCmd.Flags().StringP("to", "t", "", "End date to start fetching scrobbles (not included).")
	lastfmCmd.Flags().Bool("no-fetch", false, "Skip fetching from Last.fm.`")
	lastfmCmd.Flags().Bool("no-generate", false, "Skip generating daily post.`")
	_ = lastfmCmd.MarkFlagRequired("from")
	_ = lastfmCmd.MarkFlagRequired("to")
}

var lastfmCmd = &cobra.Command{
	Use: "lastfm",
	RunE: func(cmd *cobra.Command, args []string) error {
		fromStr, _ := cmd.Flags().GetString("from")
		toStr, _ := cmd.Flags().GetString("to")
		noFetch, _ := cmd.Flags().GetBool("no-fetch")
		noGenerate, _ := cmd.Flags().GetBool("no-generate")

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

		for cur := from; !cur.Equal(to); cur = cur.AddDate(0, 0, 1) {
			year, month, day := cur.Date()
			fmt.Printf("Processing %04d-%02d-%02d\n", year, month, day)

			created := true
			var err error

			if !noFetch {
				created, err = e.FetchLastFmListens(year, month, day)
				if err != nil {
					return err
				}
			}

			if !noGenerate && created {
				err := e.CreateDailyListensEntry(year, month, day)
				if err != nil && !os.IsNotExist(err) {
					return err
				}
			}

		}

		return nil
	},
}
