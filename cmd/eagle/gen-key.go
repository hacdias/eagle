package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(genKey)
}

var genKey = &cobra.Command{
	Use: "gen-key",
	RunE: func(cmd *cobra.Command, args []string) error {
		privKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return err
		}

		err = pem.Encode(os.Stdout, &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privKey),
		})
		if err != nil {
			return err
		}

		err = pem.Encode(os.Stdout, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&privKey.PublicKey),
		})
		if err != nil {
			return err
		}

		return nil
	},
}
