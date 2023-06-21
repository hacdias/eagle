package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/hacdias/eagle/core"
	"github.com/spf13/cobra"
	"golang.org/x/net/publicsuffix"
)

func init() {
	rootCmd.AddCommand(externalLinksCmd)
}

var externalLinksCmd = &cobra.Command{
	Use: "external-links",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := core.ParseConfig()
		if err != nil {
			return err
		}
		c.BaseURL = "https://hacdias.com"

		fs := core.NewFS(c.SourceDirectory, c.BaseURL, &core.NopSync{})
		ee, err := fs.GetEntries(false)
		if err != nil {
			return err
		}

		linksMap := map[string][]Link{}
		for _, e := range ee {
			urls, err := getMarkdownURLs(e)
			if err != nil {
				return err
			}

			for _, urlStr := range urls {
				if strings.HasPrefix(urlStr, "/") || strings.HasPrefix(urlStr, c.BaseURL) {
					continue
				}

				u, err := url.Parse(urlStr)
				if err != nil {
					return err
				}

				hostname := u.Hostname()
				if hostname == "" {
					continue
				}

				hostname, err = publicsuffix.EffectiveTLDPlusOne(hostname)
				if err != nil {
					return err
				}

				if _, ok := linksMap[hostname]; !ok {
					linksMap[hostname] = []Link{}
				}

				linksMap[hostname] = append(linksMap[hostname], Link{
					SourceURL: e.Permalink,
					TargetURL: u.String(),
				})
			}
		}

		links := []Links{}

		for domain, lnks := range linksMap {
			sort.Slice(lnks, func(i, j int) bool {
				return lnks[i].SourceURL < lnks[j].SourceURL
			})

			links = append(links, Links{
				Domain: domain,
				Count:  len(lnks),
				Links:  lnks,
			})
		}

		sort.Slice(links, func(i, j int) bool {
			if links[i].Count == links[j].Count {
				return links[i].Domain > links[j].Domain
			}

			return links[i].Count > links[j].Count
		})

		raw, err := json.MarshalIndent(links, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(raw))
		return nil
	},
}

type Link struct {
	SourceURL string `json:"sourceUrl"`
	TargetURL string `json:"targetUrl"`
}

type Links struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
	Links  []Link `json:"links"`
}
