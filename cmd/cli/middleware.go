package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var middlewareCmd = &cobra.Command{
	Use:   "middleware",
	Short: "Explore available middlewares",
	Long: `Explore available middlewares for Mizu applications.

List all middlewares by category or show detailed information about specific ones.`,
	Example: `  # List all middlewares
  mizu middleware ls

  # Filter by category
  mizu middleware ls -c security

  # Show middleware details
  mizu middleware show helmet`,
	RunE: wrapRunE(func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}),
}

var middlewareLsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all middlewares",
	RunE:    wrapRunE(runMiddlewareLsCmd),
}

var middlewareShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details about a middleware",
	Args:  cobra.ExactArgs(1),
	RunE:  wrapRunE(runMiddlewareShowCmd),
}

var middlewareFlags struct {
	category string
}

func init() {
	middlewareCmd.AddCommand(middlewareLsCmd)
	middlewareCmd.AddCommand(middlewareShowCmd)

	middlewareLsCmd.Flags().StringVarP(&middlewareFlags.category, "category", "c", "", "Filter by category")
}

func runMiddlewareLsCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	allMiddlewares := getMiddlewares()

	// Filter by category if specified
	if middlewareFlags.category != "" {
		cat := strings.ToLower(middlewareFlags.category)
		allMiddlewares = filterByCategory(allMiddlewares, cat)
		if len(allMiddlewares) == 0 {
			if Flags.JSON {
				out.WriteJSONError("unknown_category", fmt.Sprintf("unknown category: %s", middlewareFlags.category))
			} else {
				out.PrintError("unknown category %q", middlewareFlags.category)
				out.Print("Available categories: %s\n", strings.Join(categories, ", "))
			}
			return fmt.Errorf("unknown category: %s", middlewareFlags.category)
		}
	}

	if Flags.JSON {
		return printMiddlewaresJSONNew(out, allMiddlewares, middlewareFlags.category)
	}
	return printMiddlewaresHumanNew(out, allMiddlewares, middlewareFlags.category)
}

func runMiddlewareShowCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	name := strings.ToLower(args[0])
	mw := findMiddleware(name)
	if mw == nil {
		if Flags.JSON {
			out.WriteJSONError("unknown_middleware", fmt.Sprintf("unknown middleware: %s", name))
		} else {
			out.PrintError("unknown middleware %q", name)
			out.Print("Run 'mizu middleware ls' to see available middlewares.\n")
		}
		return fmt.Errorf("unknown middleware: %s", name)
	}

	if Flags.JSON {
		return printMiddlewareJSONNew(out, mw)
	}
	return printMiddlewareHumanNew(out, mw)
}

func printMiddlewaresHumanNew(out *Output, mws []middlewareInfo, category string) error {
	if category != "" {
		out.Print("%s (%d)\n\n", out.Bold(strings.ToUpper(category)), len(mws))
		tbl := newTable("Name", "Description")
		for _, mw := range mws {
			tbl.addRow(mw.Name, mw.Description)
		}
		tbl.write(out.Stdout)
	} else {
		grouped := groupByCategory(mws)

		for _, cat := range categories {
			catMws := grouped[cat]
			if len(catMws) == 0 {
				continue
			}

			sort.Slice(catMws, func(i, j int) bool {
				return catMws[i].Name < catMws[j].Name
			})

			desc := categoryDescriptions[cat]
			out.Print("%s (%d) - %s\n", out.Bold(strings.ToUpper(cat)), len(catMws), desc)

			for _, mw := range catMws {
				out.Print("  %-16s %s\n", out.Cyan(mw.Name), mw.Description)
			}
			out.Print("\n")
		}
	}

	out.Print("%d middlewares available. Use 'mizu middleware show <name>' for details.\n", len(mws))
	return nil
}

func printMiddlewaresJSONNew(out *Output, mws []middlewareInfo, category string) error {
	type mwJSON struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}

	type catJSON struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Middlewares []mwJSON `json:"middlewares"`
	}

	if category != "" {
		items := make([]mwJSON, 0, len(mws))
		for _, mw := range mws {
			items = append(items, mwJSON{
				Name:        mw.Name,
				Description: mw.Description,
				Category:    mw.Category,
			})
		}
		out.WriteJSON(map[string]any{
			"category":    category,
			"middlewares": items,
			"total":       len(items),
		})
	} else {
		grouped := groupByCategory(mws)
		cats := make([]catJSON, 0, len(categories))

		for _, cat := range categories {
			catMws := grouped[cat]
			if len(catMws) == 0 {
				continue
			}

			items := make([]mwJSON, 0, len(catMws))
			for _, mw := range catMws {
				items = append(items, mwJSON{
					Name:        mw.Name,
					Description: mw.Description,
					Category:    mw.Category,
				})
			}

			cats = append(cats, catJSON{
				Name:        cat,
				Description: categoryDescriptions[cat],
				Middlewares: items,
			})
		}

		out.WriteJSON(map[string]any{
			"categories": cats,
			"total":      len(mws),
		})
	}
	return nil
}

func printMiddlewareHumanNew(out *Output, mw *middlewareInfo) error {
	out.Print("%s\n\n", out.Bold(strings.ToUpper(mw.Name)))
	out.Print("%s\n\n", mw.Description)

	out.Print("%s: %s\n\n", out.Dim("CATEGORY"), mw.Category)

	out.Print("%s:\n", out.Dim("IMPORT"))
	out.Print("  %s\n\n", out.Cyan(mw.Import))

	out.Print("%s:\n", out.Dim("QUICK START"))
	out.Print("  %s\n\n", mw.QuickStart)

	if len(mw.Related) > 0 {
		out.Print("%s:\n", out.Dim("RELATED"))
		out.Print("  %s\n", strings.Join(mw.Related, ", "))
	}

	return nil
}

func printMiddlewareJSONNew(out *Output, mw *middlewareInfo) error {
	out.WriteJSON(map[string]any{
		"name":        mw.Name,
		"description": mw.Description,
		"category":    mw.Category,
		"import":      mw.Import,
		"quick_start": mw.QuickStart,
		"related":     mw.Related,
	})
	return nil
}
