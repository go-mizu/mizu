package chat

import (
	"encoding/json"
	"os"

	pkgchat "now/pkg/chat"

	"github.com/spf13/cobra"
)

func newMessagesCmd(d *deps) *cobra.Command {
	var (
		limit  int
		before string
	)

	cmd := &cobra.Command{
		Use:   "messages <id>",
		Short: "List messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ar, err := signAndVerify(cmd, d, "messages", map[string]string{
				"chat": args[0],
			})
			if err != nil {
				return err
			}

			msgs, err := d.svc.Messages(cmd.Context(), pkgchat.MessagesInput{
				Chat:   args[0],
				Before: before,
				Limit:  limit,
			}, ar.Actor)
			if err != nil {
				return err
			}

			return json.NewEncoder(os.Stdout).Encode(msgs)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "max messages")
	cmd.Flags().StringVar(&before, "before", "", "cursor: message ID to paginate before")

	return cmd
}
