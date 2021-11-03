package main

import (
	"encoding/json"
	"io/fs"

	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(reparseXray)
}

var reparseXray = &cobra.Command{
	Use:  "reparse-xray",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Parse()
		if err != nil {
			return err
		}

		e, err := eagle.NewEagle(c)
		if err != nil {
			return err
		}

		return e.SrcFs.Walk(eagle.XRayDirectory, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			data, err := e.ReadFile(path)
			if err != nil {
				return err
			}

			jf2 := map[string]interface{}{}
			err = json.Unmarshal(data, &jf2)
			if err != nil {
				return err
			}

			jf2 = e.ParseXRayResponse(jf2)

			err = e.PersistJSON(path, jf2, "")
			if err != nil {
				return err
			}

			return nil
		})
	},
}
