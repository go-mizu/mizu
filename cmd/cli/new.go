package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var newFlags struct {
	template string
	sub      string
	list     bool
	force    bool
	dryRun   bool
	name     string
	module   string
	license  string
	vars     []string
}

var newCmd = &cobra.Command{
	Use:   "new [path]",
	Short: "Create a new project from a template",
	Long: `Create a new project from a template.

Scaffolds a new Mizu project with the specified template into the target directory.
If no path is specified, the current directory is used.`,
	Example: `  # Create minimal project in current directory
  mizu new . --template minimal

  # Create API project in new directory
  mizu new ./myapp --template api

  # Create React project using sub-template
  mizu new ./myapp --template frontend --sub react

  # Preview what would be created
  mizu new ./myapp --template api --dry-run

  # List available templates
  mizu new --list`,
	Args: cobra.MaximumNArgs(1),
	RunE: wrapRunE(runNewCmd),
}

func init() {
	newCmd.Flags().StringVarP(&newFlags.template, "template", "t", "", "Template to render")
	newCmd.Flags().StringVarP(&newFlags.sub, "sub", "s", "", "Sub-template to use (e.g., react, vue)")
	newCmd.Flags().BoolVar(&newFlags.list, "list", false, "List available templates")
	newCmd.Flags().BoolVar(&newFlags.force, "force", false, "Overwrite existing files")
	newCmd.Flags().BoolVar(&newFlags.dryRun, "dry-run", false, "Print plan without writing")
	newCmd.Flags().StringVar(&newFlags.name, "name", "", "Project name")
	newCmd.Flags().StringVar(&newFlags.module, "module", "", "Go module path")
	newCmd.Flags().StringVar(&newFlags.license, "license", "MIT", "License identifier")
	newCmd.Flags().StringArrayVar(&newFlags.vars, "var", nil, "Template variable (k=v, repeatable)")
}

func runNewCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	// Handle --list
	if newFlags.list {
		return listTemplatesCommand(out)
	}

	// Template is required
	if newFlags.template == "" {
		if Flags.JSON {
			out.WriteJSONError("missing_template", "template is required (use --template or --list)")
		} else {
			out.PrintError("template is required")
			out.Print("Run 'mizu new --list' to see available templates.\n")
		}
		return fmt.Errorf("template is required")
	}

	// Resolve template name with sub-template
	templateName := newFlags.template
	if err := resolveSubTemplate(&templateName, newFlags.sub, out); err != nil {
		return err
	}

	// Check template exists
	if !templateExists(templateName) {
		if Flags.JSON {
			out.WriteJSONError("unknown_template", fmt.Sprintf("unknown template: %s", templateName))
		} else {
			out.PrintError("unknown template %q", templateName)
			out.Print("Run 'mizu new --list' to see available templates.\n")
		}
		return fmt.Errorf("unknown template: %s", templateName)
	}

	// Get target path (positional argument or current dir)
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		if Flags.JSON {
			out.WriteJSONError("path_error", err.Error())
		} else {
			out.PrintError("%v", err)
		}
		return err
	}

	// Derive project name from path
	projectName := newFlags.name
	if projectName == "" {
		projectName = filepath.Base(absPath)
	}

	// Derive module path
	modulePath := newFlags.module
	if modulePath == "" {
		modulePath = "example.com/" + projectName
	}

	// Parse custom variables
	customVars := make(map[string]string)
	for _, v := range newFlags.vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			customVars[parts[0]] = parts[1]
		}
	}

	// Create template variables
	vars := newTemplateVars(projectName, modulePath, newFlags.license, customVars)

	// Build plan
	p, err := buildPlan(templateName, absPath, vars)
	if err != nil {
		if Flags.JSON {
			out.WriteJSONError("plan_error", err.Error())
		} else {
			out.PrintError("%v", err)
		}
		return err
	}

	// Dry run output
	if newFlags.dryRun {
		if Flags.JSON {
			out.WriteJSON(p.toJSON())
		} else {
			p.printHuman(out)
		}
		return nil
	}

	// Check conflicts
	if !newFlags.force {
		conflicts := p.checkConflicts()
		if len(conflicts) > 0 {
			if Flags.JSON {
				out.WriteJSONError("conflicts", fmt.Sprintf("files exist: %v", conflicts))
			} else {
				out.PrintError("files already exist:")
				for _, c := range conflicts {
					out.Print("  %s\n", c)
				}
				out.Print("Use --force to overwrite.\n")
			}
			return fmt.Errorf("files already exist")
		}
	}

	// Calculate summary before apply
	mkdir, write, _, _ := p.summary()

	// Apply plan
	if err := p.apply(newFlags.force); err != nil {
		if Flags.JSON {
			out.WriteJSONError("apply_error", err.Error())
		} else {
			out.PrintError("%v", err)
		}
		return err
	}

	// Success output
	if Flags.JSON {
		out.WriteJSON(p.toJSON())
	} else {
		out.Print("Created %s from template %s\n", out.Cyan(targetPath), out.Bold(templateName))
		out.Print("  %d %s, %d %s\n",
			mkdir, pluralize(mkdir, "directory", "directories"),
			write, pluralize(write, "file", "files"))

		out.Print("\nNext steps:\n")
		out.Print("  cd %s\n", targetPath)
		out.Print("  go mod tidy\n")
		out.Print("  mizu dev\n")
	}

	return nil
}

func listTemplatesCommand(out *Output) error {
	templates, err := listTemplates()
	if err != nil {
		if Flags.JSON {
			out.WriteJSONError("list_error", err.Error())
		} else {
			out.PrintError("%v", err)
		}
		return err
	}

	if Flags.JSON {
		out.WriteJSON(map[string]any{"templates": templates})
		return nil
	}

	// Human output
	tbl := newTable("Template", "Purpose")
	for _, t := range templates {
		desc := t.Description
		if len(t.SubTemplates) > 0 {
			// Add sub-template names to description
			var names []string
			for _, st := range t.SubTemplates {
				names = append(names, st.Name)
			}
			desc += " (--sub: " + strings.Join(names, ", ") + ")"
		}
		tbl.addRow(t.Name, desc)
	}
	tbl.write(out.Stdout)

	return nil
}

// buildPlan creates a template plan (forwarded to existing implementation).
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

// resolveSubTemplate checks if a template requires a sub-template and validates the selection.
// If --sub is provided, it appends the sub-template to the template name.
// If the template has subTemplates but --sub is not provided, it returns an error.
func resolveSubTemplate(templateName *string, sub string, out *Output) error {
	meta, err := loadTemplateMeta(*templateName)
	if err != nil {
		return nil // Let templateExists handle this
	}

	// If template has sub-templates
	if len(meta.SubTemplates) > 0 {
		if sub == "" {
			// No sub-template provided, show available options
			if Flags.JSON {
				out.WriteJSONError("missing_sub", fmt.Sprintf("template %q requires --sub flag", *templateName))
			} else {
				out.PrintError("template %q requires --sub flag", *templateName)
				out.Print("Available sub-templates:\n")
				for _, st := range meta.SubTemplates {
					out.Print("  %s - %s\n", st.Name, st.Description)
				}
				out.Print("\nExample: mizu new ./myapp --template %s --sub %s\n", *templateName, meta.SubTemplates[0].Name)
			}
			return fmt.Errorf("missing sub-template")
		}

		// Validate the sub-template exists
		valid := false
		for _, st := range meta.SubTemplates {
			if st.Name == sub {
				valid = true
				break
			}
		}
		if !valid {
			if Flags.JSON {
				out.WriteJSONError("invalid_sub", fmt.Sprintf("invalid sub-template %q for %s", sub, *templateName))
			} else {
				out.PrintError("invalid sub-template %q for %s", sub, *templateName)
				out.Print("Available sub-templates:\n")
				for _, st := range meta.SubTemplates {
					out.Print("  %s - %s\n", st.Name, st.Description)
				}
			}
			return fmt.Errorf("invalid sub-template: %s", sub)
		}

		// Append sub-template to template name
		*templateName = *templateName + "/" + sub
	} else if sub != "" {
		// --sub provided but template doesn't have sub-templates
		if Flags.JSON {
			out.WriteJSONError("invalid_sub", fmt.Sprintf("template %q does not support sub-templates", *templateName))
		} else {
			out.PrintError("template %q does not support sub-templates", *templateName)
		}
		return fmt.Errorf("template does not support sub-templates")
	}

	return nil
}
