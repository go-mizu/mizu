package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewLocal_QMDAliasesAndSubcommands(t *testing.T) {
	cmd := NewLocal()

	vsearch := findSubcommand(t, cmd, "vsearch")
	if !hasAlias(vsearch, "vector-search") {
		t.Fatalf("vsearch aliases=%v; want vector-search", vsearch.Aliases)
	}

	query := findSubcommand(t, cmd, "query")
	if !hasAlias(query, "deep-search") {
		t.Fatalf("query aliases=%v; want deep-search", query.Aliases)
	}

	collection := findSubcommand(t, cmd, "collection")
	updateCmd := findSubcommand(t, collection, "update-cmd")
	if !hasAlias(updateCmd, "set-update") {
		t.Fatalf("collection update-cmd aliases=%v; want set-update", updateCmd.Aliases)
	}

	mcp := findSubcommand(t, cmd, "mcp")
	_ = findSubcommand(t, mcp, "stop")
	_ = findSubcommand(t, mcp, "status")
}

func findSubcommand(t *testing.T, parent interface{ Commands() []*cobra.Command }, name string) *cobra.Command {
	t.Helper()
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

func hasAlias(cmd *cobra.Command, alias string) bool {
	for _, a := range cmd.Aliases {
		if a == alias {
			return true
		}
	}
	return false
}
