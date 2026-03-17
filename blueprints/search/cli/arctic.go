package cli

import "github.com/spf13/cobra"

func NewArctic() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arctic",
		Short: "Arctic Shift Reddit dataset publishing",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newArcticPublish())
	cmd.AddCommand(newArcticCatalogSizes())
	cmd.AddCommand(newArcticBench())
	return cmd
}
