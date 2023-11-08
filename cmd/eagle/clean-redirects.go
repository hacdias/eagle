package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.hacdias.com/eagle/core"
)

func init() {
	rootCmd.AddCommand(cleanRedirectsCmd)
}

var cleanRedirectsCmd = &cobra.Command{
	Use: "clean-redirects",
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
