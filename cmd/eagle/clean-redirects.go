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

		for src, r := range redirects {
			r.To, err = resolveRedirect(redirects, src)
			if err != nil {
				return err
			}
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

// resolveRedirect follows the redirects from u until reaching a URL that is not
// redirected. It returns an error if the redirects loop.
func resolveRedirect(redirects map[string]core.Redirect, u string) (string, error) {
	seen := map[string]bool{}
	for {
		r, ok := redirects[u]
		if !ok {
			return u, nil
		}

		if seen[u] {
			return "", fmt.Errorf("redirect loop at %s", u)
		}

		seen[u] = true
		u = r.To
	}
}
