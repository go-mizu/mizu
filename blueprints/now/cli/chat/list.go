package chat

import (
	"encoding/json"
	"os"

	pkgchat "now/pkg/chat"

	"github.com/spf13/cobra"
)

func newListCmd(d *deps) *cobra.Command {
	var (
		kind  string
		limit int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List chats",
		RunE: func(cmd *cobra.Command, args []string) error {
			ar, err := signAndVerify(cmd, d, "list", map[string]string{
				"kind": kind,
			})
			if err != nil {
				return err
			}

			chats, err := d.svc.List(cmd.Context(), pkgchat.ListInput{
				Kind:  kind,
				Limit: limit,
			}, ar.Actor)
			if err != nil {
				return err
			}

			return json.NewEncoder(os.Stdout).Encode(chats)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "filter by kind (room or direct)")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")

	return cmd
}
