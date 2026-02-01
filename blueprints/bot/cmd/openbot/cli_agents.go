package main

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// runAgents dispatches agents subcommands: list, add, delete
func runAgents() error {
	sub := ""
	if len(os.Args) > 2 {
		sub = os.Args[2]
	}

	switch sub {
	case "list", "":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tMODEL\tSTATUS")
		fmt.Fprintln(w, "main\tmain\tclaude-sonnet-4-20250514\tactive")
		w.Flush()
		return nil

	case "add":
		fmt.Println("Agent creation not yet implemented. Currently only the 'main' agent is supported.")
		return nil

	case "delete":
		fmt.Println("Agent deletion not yet implemented.")
		return nil

	default:
		fmt.Println("Usage: openbot agents [list|add|delete]")
		return nil
	}
}
