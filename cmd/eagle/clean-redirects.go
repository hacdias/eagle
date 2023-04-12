package main

import (
	"fmt"
	"os"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cleanRedirectsCmd)
}

var cleanRedirectsCmd = &cobra.Command{
	Use: "clean-redirects",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := eagle.ParseConfig()
		if err != nil {
			return err
		}

		fs := fs.NewFS(c.SourceDirectory, c.Server.BaseURL, &fs.NopSync{})

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

		for src := range redirects {
			redirects[src] = resolveRedirect(src)

			_, err := fs.GetEntry(redirects[src])
			if os.IsNotExist(err) {
				delete(redirects, src)
			}
		}

		for src, dst := range redirects {
			fmt.Printf("%s %s\n", src, dst)
		}

		return nil
	},
}
