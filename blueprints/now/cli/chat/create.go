package chat

import (
	"encoding/json"
	"fmt"
	"os"

	pkgchat "now/pkg/chat"

	"github.com/spf13/cobra"
)

func newCreateCmd(d *deps) *cobra.Command {
	var (
		kind    string
		title   string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a chat",
		RunE: func(cmd *cobra.Command, args []string) error {
			if kind != pkgchat.KindDirect && kind != pkgchat.KindRoom {
				return fmt.Errorf("kind must be \"room\" or \"direct\"")
			}

			ar, err := signAndVerify(cmd, d, "create", map[string]string{
				"kind": kind, "title": title,
			})
			if err != nil {
				return err
			}

			c, err := d.svc.Create(cmd.Context(), pkgchat.CreateInput{
				Kind:  kind,
				Title: title,
			}, ar.Actor)
			if err != nil {
				return err
			}

			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(c)
			}
			fmt.Println(c.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "chat kind (room or direct)")
	cmd.Flags().StringVar(&title, "title", "", "chat title")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output full JSON")

	return cmd
}
