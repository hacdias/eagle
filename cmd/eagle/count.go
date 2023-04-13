package main

import (
	"fmt"
	"sort"

	"github.com/hacdias/eagle/core"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(countCmd)
}

var countCmd = &cobra.Command{
	Use: "count",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := core.ParseConfig()
		if err != nil {
			return err
		}

		fs := core.NewFS(c.SourceDirectory, c.BaseURL, &core.NopSync{})
		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		count := map[string]int{}

		for _, e := range ee {
			for _, section := range e.Categories {
				if _, ok := count[section]; !ok {
					count[section] = 0
				}

				count[section]++
			}
		}

		keys := lo.Keys(count)
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Printf("%s: %d\n", k, count[k])
		}

		fmt.Println("\nTotal:", len(ee))
		return nil
	},
}
