package main

import (
	"fmt"
	"net/http"
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
		c, err := core.ParseConfig("")
		if err != nil {
			return err
		}

		fs, err := core.NewCore(c)
		if err != nil {
			return err
		}

		redirects, err := fs.GetRedirects(false)
		if err != nil {
			return err
		}

		var resolveRedirect func(u string) string
		resolveRedirect = func(u string) string {
			if r, ok := redirects[u]; ok {
				return resolveRedirect(r.To)
			}

			return u
		}

		for src, r := range redirects {
			r.To = resolveRedirect(src)
			redirects[src] = r

			_, err := fs.GetEntryByPermalink(r.To)
			if os.IsNotExist(err) {
				delete(redirects, src)
			}
		}

		for src, r := range redirects {
			if r.Status == http.StatusMovedPermanently {
				fmt.Printf("%s %s\n", src, r.To)
			} else {
				fmt.Printf("%s %s %d\n", src, r.To, r.Status)
			}
		}

		return nil
	},
}
