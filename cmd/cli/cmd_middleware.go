package cli

import (
	"flag"
	"fmt"
	"sort"
	"strings"
)

// middlewareFlags holds flags for the middleware command.
type middlewareFlags struct {
	category string
}

func runMiddleware(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)

	if len(args) == 0 {
		usageMiddleware()
		return exitUsage
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "ls", "list":
		return runMiddlewareLs(subargs, gf, out)
	case "show":
		return runMiddlewareShow(subargs, gf, out)
	default:
		out.errorf("error: unknown subcommand %q\n", subcmd)
		out.errorf("Run 'mizu middleware --help' for usage.\n")
		return exitUsage
	}
}

func runMiddlewareLs(args []string, gf *globalFlags, out *output) int {
	mf := &middlewareFlags{}
	fs := flag.NewFlagSet("middleware ls", flag.ContinueOnError)
	fs.StringVar(&mf.category, "category", "", "Filter by category")
	fs.StringVar(&mf.category, "c", "", "Filter by category (shorthand)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageMiddlewareLs()
			return exitOK
		}
		return exitUsage
	}

	allMiddlewares := getMiddlewares()

	// Filter by category if specified
	if mf.category != "" {
		cat := strings.ToLower(mf.category)
		allMiddlewares = filterByCategory(allMiddlewares, cat)
		if len(allMiddlewares) == 0 {
			if out.json {
				out.writeJSONError("unknown_category", fmt.Sprintf("unknown category: %s", mf.category))
			} else {
				out.errorf("error: unknown category %q\n", mf.category)
				out.errorf("Available categories: %s\n", strings.Join(categories, ", "))
			}
			return exitError
		}
	}

	if out.json {
		return printMiddlewaresJSON(out, allMiddlewares, mf.category)
	}
	return printMiddlewaresHuman(out, allMiddlewares, mf.category)
}

func runMiddlewareShow(args []string, gf *globalFlags, out *output) int {
	fs := flag.NewFlagSet("middleware show", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageMiddlewareShow()
			return exitOK
		}
		return exitUsage
	}

	if fs.NArg() == 0 {
		if out.json {
			out.writeJSONError("missing_name", "middleware name required")
		} else {
			out.errorf("error: middleware name required\n")
			out.errorf("Run 'mizu middleware ls' to see available middlewares.\n")
		}
		return exitUsage
	}

	name := strings.ToLower(fs.Arg(0))
	mw := findMiddleware(name)
	if mw == nil {
		if out.json {
			out.writeJSONError("unknown_middleware", fmt.Sprintf("unknown middleware: %s", name))
		} else {
			out.errorf("error: unknown middleware %q\n", name)
			out.errorf("Run 'mizu middleware ls' to see available middlewares.\n")
		}
		return exitError
	}

	if gf.json {
		return printMiddlewareJSON(out, mw)
	}
	return printMiddlewareHuman(out, mw)
}

func printMiddlewaresHuman(out *output, mws []middlewareInfo, category string) int {
	if category != "" {
		// Single category output
		out.print("%s (%d)\n\n", out.bold(strings.ToUpper(category)), len(mws))
		tbl := newTable("Name", "Description")
		for _, mw := range mws {
			tbl.addRow(mw.Name, mw.Description)
		}
		tbl.write(out.stdout)
	} else {
		// Group by category
		grouped := groupByCategory(mws)

		for _, cat := range categories {
			catMws := grouped[cat]
			if len(catMws) == 0 {
				continue
			}

			// Sort middlewares by name within category
			sort.Slice(catMws, func(i, j int) bool {
				return catMws[i].Name < catMws[j].Name
			})

			desc := categoryDescriptions[cat]
			out.print("%s (%d) - %s\n", out.bold(strings.ToUpper(cat)), len(catMws), desc)

			for _, mw := range catMws {
				out.print("  %-16s %s\n", out.cyan(mw.Name), mw.Description)
			}
			out.print("\n")
		}
	}

	out.print("%d middlewares available. Use 'mizu middleware show <name>' for details.\n", len(mws))
	return exitOK
}

func printMiddlewaresJSON(out *output, mws []middlewareInfo, category string) int {
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
		// Single category
		items := make([]mwJSON, 0, len(mws))
		for _, mw := range mws {
			items = append(items, mwJSON{
				Name:        mw.Name,
				Description: mw.Description,
				Category:    mw.Category,
			})
		}
		out.writeJSON(map[string]any{
			"category":    category,
			"middlewares": items,
			"total":       len(items),
		})
	} else {
		// All categories
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

		out.writeJSON(map[string]any{
			"categories": cats,
			"total":      len(mws),
		})
	}
	return exitOK
}

func printMiddlewareHuman(out *output, mw *middlewareInfo) int {
	out.print("%s\n\n", out.bold(strings.ToUpper(mw.Name)))
	out.print("%s\n\n", mw.Description)

	out.print("%s: %s\n\n", out.gray("CATEGORY"), mw.Category)

	out.print("%s:\n", out.gray("IMPORT"))
	out.print("  %s\n\n", out.cyan(mw.Import))

	out.print("%s:\n", out.gray("QUICK START"))
	out.print("  %s\n\n", mw.QuickStart)

	if len(mw.Related) > 0 {
		out.print("%s:\n", out.gray("RELATED"))
		out.print("  %s\n", strings.Join(mw.Related, ", "))
	}

	return exitOK
}

func printMiddlewareJSON(out *output, mw *middlewareInfo) int {
	out.writeJSON(map[string]any{
		"name":        mw.Name,
		"description": mw.Description,
		"category":    mw.Category,
		"import":      mw.Import,
		"quick_start": mw.QuickStart,
		"related":     mw.Related,
	})
	return exitOK
}

func usageMiddleware() {
	fmt.Println("Usage:")
	fmt.Println("  mizu middleware <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Explore available middlewares for Mizu applications.")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  ls, list    List all middlewares")
	fmt.Println("  show        Show details about a middleware")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mizu middleware ls")
	fmt.Println("  mizu middleware ls -c security")
	fmt.Println("  mizu middleware show helmet")
	fmt.Println("  mizu middleware show cors --json")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h, --help    Show help")
}

func usageMiddlewareLs() {
	fmt.Println("Usage:")
	fmt.Println("  mizu middleware ls [flags]")
	fmt.Println()
	fmt.Println("List all available middlewares, optionally filtered by category.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mizu middleware ls")
	fmt.Println("  mizu middleware ls -c security")
	fmt.Println("  mizu middleware ls --category logging")
	fmt.Println("  mizu middleware ls --json")
	fmt.Println()
	fmt.Println("Categories:")
	for _, cat := range categories {
		fmt.Printf("  %-14s %s\n", cat, categoryDescriptions[cat])
	}
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -c, --category <name>    Filter by category")
	fmt.Println("      --json               Output as JSON")
	fmt.Println("  -h, --help               Show help")
}

func usageMiddlewareShow() {
	fmt.Println("Usage:")
	fmt.Println("  mizu middleware show <name> [flags]")
	fmt.Println()
	fmt.Println("Show detailed information about a middleware.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mizu middleware show helmet")
	fmt.Println("  mizu middleware show cors")
	fmt.Println("  mizu middleware show ratelimit --json")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --json    Output as JSON")
	fmt.Println("  -h, --help    Show help")
}
