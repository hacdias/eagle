package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.hacdias.com/eagle/core"
)

func init() {
	rootCmd.AddCommand(brokenLinksCmd)
}

var brokenLinksCmd = &cobra.Command{
	Use: "broken-links",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := core.ParseConfig()
		if err != nil {
			return err
		}

		co, err := core.NewCore(c)
		if err != nil {
			return err
		}

		err = co.Build(true)
		if err != nil {
			return err
		}

		redirects, err := co.GetRedirects(false)
		if err != nil {
			return err
		}

		var resolveRedirect func(u string) string
		resolveRedirect = func(u string) string {
			if r, ok := redirects[u]; ok {
				return resolveRedirect(r)
			}

			return u
		}

		ee, err := co.GetEntries(false)
		if err != nil {
			return err
		}

		exists := func(urlStr string) (bool, error) {
			if !strings.HasPrefix(urlStr, "/") && !strings.HasPrefix(urlStr, c.BaseURL) {
				return true, nil
			}

			return co.IsLinkValid(urlStr)
		}

		printBroken := func(e *core.Entry, what string, urls []string) {
			if len(urls) != 0 {
				fmt.Println(what, e.ID)
				for _, l := range urls {
					r := resolveRedirect(l)
					if r != l {
						fmt.Println("R", l, "->", r)
					} else {
						fmt.Println("D", l)
					}
				}

				fmt.Println("")
			}
		}

		for _, e := range ee {
			markdownURLs, err := co.GetEntryLinks(e.Permalink)
			if err != nil {
				return err
			}
			brokenLinks := []string{}
			for _, urlStr := range markdownURLs {
				exists, err := exists(urlStr)
				if err != nil {
					return err
				}
				if !exists {
					brokenLinks = append(brokenLinks, urlStr)
				}
			}
			printBroken(e, "Entry", brokenLinks)
		}

		return nil
	},
}
