package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v3/config"
	"github.com/hacdias/eagle/v3/eagle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(uploadCmd)
}

var uploadCmd = &cobra.Command{
	Use: "upload",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Parse()
		if err != nil {
			return err
		}

		e, err := eagle.NewEagle(c)
		if err != nil {
			return err
		}
		defer e.Close()

		root := "/Users/hacdias/Documents/CDN/u"

		files, err := ioutil.ReadDir(root)
		if err != nil {
			return err
		}

		for _, file := range files {
			data, err := ioutil.ReadFile(filepath.Join(root, file.Name()))
			if err != nil {
				return err
			}

			ext := filepath.Ext(file.Name())
			filename := strings.TrimSuffix(file.Name(), ext)

			url, err := e.UploadMedia(filename, ext, bytes.NewBuffer(data))
			if err != nil {
				return err
			}

			fmt.Println(url)
		}

		// for _, ee := range entries {
		// 	err = e.ProcessLocation(ee)
		// 	if err != nil {
		// 		e.Error(err)
		// 	}
		// }

		return nil
	},
}
