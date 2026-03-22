package chat

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"now/pkg/auth"

	"github.com/spf13/cobra"
)

func newInitCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize chat configuration",
		Long:  "Generate an ed25519 keypair and save identity to ~/.config/now/config.json.",
		RunE: func(cmd *cobra.Command, args []string) error {
			actor, _ := cmd.Flags().GetString("actor")
			if actor == "" && d.identity != nil {
				actor = d.identity.Actor
			}
			if actor == "" {
				fmt.Print("Enter actor (e.g. u/alice, a/bot1): ")
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					actor = strings.TrimSpace(scanner.Text())
				}
				if actor == "" {
					return fmt.Errorf("actor required")
				}
			}

			id, err := auth.GenerateIdentity(actor)
			if err != nil {
				return err
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			dir := filepath.Join(home, ".config", "now")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}

			cfg := chatConfig{
				Actor:       id.Actor,
				PublicKey:   base64.URLEncoding.EncodeToString(id.PublicKey),
				PrivateKey:  base64.URLEncoding.EncodeToString(id.PrivateKey),
				Fingerprint: id.Fingerprint,
			}

			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}

			path := filepath.Join(dir, "config.json")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				return err
			}

			fmt.Println(path)
			return nil
		},
	}
}
