package main

import (
	"fmt"
	"net/url"
	"path/filepath"
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

		fs := core.NewFS(c.SourceDirectory, c.BaseURL, &core.NopSync{})

		redirects, err := fs.LoadRedirects(false)
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

		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		isBroken := func(urlStr string) (bool, string, error) {
			if !strings.HasPrefix(urlStr, "/") && !strings.HasPrefix(urlStr, c.BaseURL) {
				return false, "", nil
			}

			u, err := url.Parse(urlStr)
			if err != nil {
				return false, "", err
			}

			if strings.HasPrefix(u.Path, "/tags") {
				return false, "", nil
			}

			u.Path = strings.TrimSuffix(u.Path, "/")

			_, err = fs.GetEntry(u.Path)
			if err == nil {
				return false, "", nil
			}

			parts := strings.Split(u.Path, "/")
			if len(parts) == 5 {
				for _, section := range core.Sections {
					_, err = fs.GetEntry("/" + section + "/" + parts[1] + "/" + parts[4])
					if err == nil {
						return false, "", nil
					}
				}
			}

			_, err = fs.ReadFile(filepath.Join("content", u.Path))
			if err == nil {
				return false, "", nil
			}

			_, err = fs.ReadFile(filepath.Join("static", u.Path))
			if err == nil {
				return false, "", nil
			}

			return true, u.Path, nil
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
			markdownURLs, err := core.GetMarkdownURLs(e)
			if err != nil {
				return err
			}
			brokenLinks := []string{}
			for _, urlStr := range markdownURLs {
				broken, canonical, err := isBroken(urlStr)
				if err != nil {
					return err
				}
				if broken {
					brokenLinks = append(brokenLinks, canonical)
				}
			}
			printBroken(e, "Entry", brokenLinks)
		}

		return nil
	},
}
