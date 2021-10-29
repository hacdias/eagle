package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	rootCmd.AddCommand(pwdCmd)
}

var pwdCmd = &cobra.Command{
	Use:   "pwd",
	Short: "Generate a password hash to use on the configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pwd := args[0]
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		fmt.Println(string(hash))
		return nil
	},
}
