package cli

import (
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE:  wrapRunE(runVersionCmd),
}

func runVersionCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	info := map[string]string{
		"version":    Version,
		"go_version": runtime.Version(),
		"commit":     Commit,
		"built_at":   BuildTime,
	}

	if Flags.JSON {
		return out.WriteJSON(info)
	}

	out.Print("mizu version %s\n", info["version"])
	out.Print("go version: %s\n", info["go_version"])
	out.Print("commit: %s\n", info["commit"])
	out.Print("built: %s\n", info["built_at"])

	return nil
}
