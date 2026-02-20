package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newKeysCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keys",
		Short: "Generate COOKIE_HASH_KEY and COOKIE_BLOCK_KEY values (base64)",
		RunE: func(cmd *cobra.Command, args []string) error {
			hash := make([]byte, 32)
			block := make([]byte, 32)
			if _, err := rand.Read(hash); err != nil {
				return err
			}
			if _, err := rand.Read(block); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "export COOKIE_HASH_KEY=%s\n", base64.StdEncoding.EncodeToString(hash))
			fmt.Fprintf(os.Stdout, "export COOKIE_BLOCK_KEY=%s\n", base64.StdEncoding.EncodeToString(block))
			return nil
		},
	}
}
