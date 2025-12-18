package cli

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

// newFlags holds flags for the new command.
type newFlags struct {
	template string
	list     bool
	force    bool
	dryRun   bool
	name     string
	module   string
	license  string
	vars     varList
}

// varList is a flag.Value for repeatable --var flags.
type varList []string

func (v *varList) String() string { return strings.Join(*v, ",") }
func (v *varList) Set(s string) error {
	*v = append(*v, s)
	return nil
}

//nolint:cyclop // CLI command with sequential logic
func runNew(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)
	nf := &newFlags{}

	// Parse flags
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.StringVar(&nf.template, "template", "", "Template to render")
	fs.StringVar(&nf.template, "t", "", "Template to render (shorthand)")
	fs.BoolVar(&nf.list, "list", false, "List available templates")
	fs.BoolVar(&nf.force, "force", false, "Overwrite existing files")
	fs.BoolVar(&nf.dryRun, "dry-run", false, "Print plan without writing")
	fs.StringVar(&nf.name, "name", "", "Project name")
	fs.StringVar(&nf.module, "module", "", "Go module path")
	fs.StringVar(&nf.license, "license", "MIT", "License identifier")
	fs.Var(&nf.vars, "var", "Template variable (k=v, repeatable)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageNew()
			return exitOK
		}
		return exitUsage
	}

	// Handle --list
	if nf.list {
		return listTemplatesCmd(out)
	}

	// Template is required
	if nf.template == "" {
		if out.json {
			out.writeJSONError("missing_template", "template is required (use --template or --list)")
		} else {
			out.errorf("error: template is required\n")
			out.errorf("Run 'mizu new --list' to see available templates.\n")
		}
		return exitUsage
	}

	// Check template exists
	if !templateExists(nf.template) {
		if out.json {
			out.writeJSONError("unknown_template", fmt.Sprintf("unknown template: %s", nf.template))
		} else {
			out.errorf("error: unknown template %q\n", nf.template)
			out.errorf("Run 'mizu new --list' to see available templates.\n")
		}
		return exitError
	}

	// Get target path (positional argument or current dir)
	targetPath := "."
	if fs.NArg() > 0 {
		targetPath = fs.Arg(0)
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		if out.json {
			out.writeJSONError("path_error", err.Error())
		} else {
			out.errorf("error: %v\n", err)
		}
		return exitError
	}

	// Derive project name from path
	projectName := nf.name
	if projectName == "" {
		projectName = filepath.Base(absPath)
	}

	// Derive module path
	modulePath := nf.module
	if modulePath == "" {
		// Default to example.com/projectname
		modulePath = "example.com/" + projectName
	}

	// Parse custom variables
	customVars := make(map[string]string)
	for _, v := range nf.vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			customVars[parts[0]] = parts[1]
		}
	}

	// Create template variables
	vars := newTemplateVars(projectName, modulePath, nf.license, customVars)

	// Build plan
	p, err := buildPlan(nf.template, absPath, vars)
	if err != nil {
		if out.json {
			out.writeJSONError("plan_error", err.Error())
		} else {
			out.errorf("error: %v\n", err)
		}
		return exitError
	}

	// Dry run output
	if nf.dryRun {
		if out.json {
			out.writeJSON(p.toJSON())
		} else {
			p.printHuman(out)
		}
		return exitOK
	}

	// Check conflicts
	if !nf.force {
		conflicts := p.checkConflicts()
		if len(conflicts) > 0 {
			if out.json {
				out.writeJSONError("conflicts", fmt.Sprintf("files exist: %v", conflicts))
			} else {
				out.errorf("error: files already exist:\n")
				for _, c := range conflicts {
					out.errorf("  %s\n", c)
				}
				out.errorf("Use --force to overwrite.\n")
			}
			return exitError
		}
	}

	// Calculate summary before apply (files don't exist yet)
	mkdir, write, _, _ := p.summary()

	// Apply plan
	if err := p.apply(nf.force); err != nil {
		if out.json {
			out.writeJSONError("apply_error", err.Error())
		} else {
			out.errorf("error: %v\n", err)
		}
		return exitError
	}

	// Success output
	if out.json {
		out.writeJSON(p.toJSON())
	} else {
		out.print("Created %s from template %s\n", out.cyan(targetPath), out.bold(nf.template))
		out.print("  %d %s, %d %s\n",
			mkdir, pluralize(mkdir, "directory", "directories"),
			write, pluralize(write, "file", "files"))

		out.print("\nNext steps:\n")
		out.print("  cd %s\n", targetPath)
		out.print("  go mod tidy\n")
		out.print("  mizu dev\n")
	}

	return exitOK
}

func listTemplatesCmd(out *output) int {
	templates, err := listTemplates()
	if err != nil {
		if out.json {
			out.writeJSONError("list_error", err.Error())
		} else {
			out.errorf("error: %v\n", err)
		}
		return exitError
	}

	if out.json {
		out.writeJSON(map[string]any{"templates": templates})
		return exitOK
	}

	// Human output
	tbl := newTable("Template", "Purpose")
	for _, t := range templates {
		tbl.addRow(t.Name, t.Description)
	}
	tbl.write(out.stdout)

	return exitOK
}

//nolint:cyclop // plan building with multiple steps
func buildPlan(templateName, root string, vars templateVars) (*plan, error) {
	files, err := loadTemplateFiles(templateName)
	if err != nil {
		return nil, err
	}

	p := newPlan(templateName, root)

	// Collect all directories we need to create
	dirs := make(map[string]bool)

	for _, tf := range files {
		// Render template content
		content, err := renderTemplateFile(tf, vars)
		if err != nil {
			return nil, err
		}

		// Validate path
		if err := validatePath(tf.path); err != nil {
			p.addSkip(tf.path, err.Error())
			continue
		}

		// Track directory
		dir := filepath.Dir(tf.path)
		if dir != "." && dir != "" {
			dirs[dir] = true
			// Also track parent directories
			for d := dir; d != "." && d != ""; d = filepath.Dir(d) {
				dirs[d] = true
			}
		}

		// Add write operation
		if err := p.addWrite(tf.path, content, defaultFileMode); err != nil {
			p.addSkip(tf.path, err.Error())
			continue
		}
	}

	// Add mkdir operations
	for dir := range dirs {
		_ = p.addMkdir(dir)
	}

	p.sort()
	return p, nil
}

func usageNew() {
	fmt.Println("Usage:")
	fmt.Println("  mizu new [path] [flags]")
	fmt.Println()
	fmt.Println("Create a new project from a template.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mizu new . --template minimal")
	fmt.Println("  mizu new ./myapp --template api")
	fmt.Println("  mizu new ./myapp --template api --dry-run")
	fmt.Println("  mizu new ./myapp --template api --json")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -t, --template <name>    Template to render (required unless --list)")
	fmt.Println("      --list               List available templates")
	fmt.Println("      --force              Overwrite existing files")
	fmt.Println("      --dry-run            Print plan, do not write files")
	fmt.Println("      --json               Emit plan as JSON")
	fmt.Println("      --var k=v            Override template variables (repeatable)")
	fmt.Println("      --name <value>       Project name (default: derived from path)")
	fmt.Println("      --module <value>     Go module path (default: detected or derived)")
	fmt.Println("      --license <value>    License identifier (default: MIT)")
	fmt.Println("  -h, --help               Show help")
}
