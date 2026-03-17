package chat

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	pkgchat "now/pkg/chat"

	"github.com/spf13/cobra"
)

func newSendCmd(d *deps) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "send <id> <text>",
		Short: "Send a message",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ar, err := signAndVerify(cmd, d, "send", map[string]string{
				"chat": args[0], "text": args[1],
			})
			if err != nil {
				return err
			}

			m, err := d.svc.Send(cmd.Context(), pkgchat.SendInput{
				Chat:      args[0],
				Text:      args[1],
				Signature: base64.URLEncoding.EncodeToString(ar.Signature),
			}, ar.Actor)
			if err != nil {
				return err
			}

			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(m)
			}
			fmt.Println(m.ID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "output full JSON")

	return cmd
}
