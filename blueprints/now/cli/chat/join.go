package chat

import (
	pkgchat "now/pkg/chat"

	"github.com/spf13/cobra"
)

func newJoinCmd(d *deps) *cobra.Command {
	var token string

	cmd := &cobra.Command{
		Use:   "join <id>",
		Short: "Join a chat",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ar, err := signAndVerify(cmd, d, "join", map[string]string{
				"chat": args[0],
			})
			if err != nil {
				return err
			}

			return d.svc.Join(cmd.Context(), pkgchat.JoinInput{
				Chat:  args[0],
				Token: token,
			}, ar.Actor)
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "join token")

	return cmd
}
