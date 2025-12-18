package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/transport/async"
	"github.com/go-mizu/mizu/contract/v2/transport/jsonrpc"
	"github.com/go-mizu/mizu/contract/v2/transport/rest"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// v2 command definitions
var contractInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Create a new contract definition",
	Long: `Create a new contract definition file.

Creates an api.yaml file with a basic contract structure.
Use --template to start from a predefined template.
Use --from to import from an existing OpenAPI/OpenRPC spec.`,
	Example: `  mizu contract init myapi
  mizu contract init --template minimal
  mizu contract init --from http://localhost:8080/openapi.json`,
	RunE: wrapRunE(runContractInitCmd),
}

var contractValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate a contract definition",
	Long: `Validate a contract definition file.

Checks syntax, type references, and semantic rules.
Returns exit code 0 if valid, 1 if errors found.`,
	Example: `  mizu contract validate api.yaml
  mizu contract validate --strict api.yaml`,
	RunE: wrapRunE(runContractValidateCmd),
}

var contractGenCmd = &cobra.Command{
	Use:   "gen [file]",
	Short: "Generate code from contract definition",
	Long: `Generate client/server code from a contract definition.

Supports TypeScript and Go code generation.`,
	Example: `  mizu contract gen api.yaml --lang typescript
  mizu contract gen api.yaml --lang go --package api
  mizu contract gen api.yaml --lang typescript --output ./client`,
	Args: cobra.MinimumNArgs(1),
	RunE: wrapRunE(runContractGenCmd),
}

// v2 command flags
var contractV2Flags struct {
	template string
	from     string
	output   string
	strict   bool
	lang     string
	pkg      string
	client   bool
	server   bool
	types    bool
}

func init() {
	// Add v2 subcommands
	contractCmd.AddCommand(contractInitCmd)
	contractCmd.AddCommand(contractValidateCmd)
	contractCmd.AddCommand(contractGenCmd)

	// Flags for init
	contractInitCmd.Flags().StringVar(&contractV2Flags.template, "template", "", "Use template (minimal, openai, github)")
	contractInitCmd.Flags().StringVar(&contractV2Flags.from, "from", "", "Import from OpenAPI/OpenRPC URL")
	contractInitCmd.Flags().StringVarP(&contractV2Flags.output, "output", "o", "api.yaml", "Output file")

	// Flags for validate
	contractValidateCmd.Flags().BoolVar(&contractV2Flags.strict, "strict", false, "Fail on warnings")

	// Flags for gen
	contractGenCmd.Flags().StringVar(&contractV2Flags.lang, "lang", "typescript", "Target language (typescript, go)")
	contractGenCmd.Flags().StringVarP(&contractV2Flags.output, "output", "o", "", "Output directory")
	contractGenCmd.Flags().StringVar(&contractV2Flags.pkg, "package", "api", "Package name (Go)")
	contractGenCmd.Flags().BoolVar(&contractV2Flags.client, "client", false, "Generate client code")
	contractGenCmd.Flags().BoolVar(&contractV2Flags.server, "server", false, "Generate server code")
	contractGenCmd.Flags().BoolVar(&contractV2Flags.types, "types", true, "Generate types")
}

func runContractInitCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	// Determine name
	name := "MyAPI"
	if len(args) > 0 {
		name = args[0]
	}

	var svc *contract.Service

	// Import from URL if specified
	if contractV2Flags.from != "" {
		var err error
		svc, err = importContractFromURL(contractV2Flags.from)
		if err != nil {
			out.PrintError("import failed: %v", err)
			return err
		}
		if name != "MyAPI" {
			svc.Name = name
		}
	} else if contractV2Flags.template != "" {
		// Use template
		var err error
		svc, err = loadContractTemplate(contractV2Flags.template)
		if err != nil {
			out.PrintError("template not found: %s", contractV2Flags.template)
			return err
		}
		svc.Name = name
	} else {
		// Create minimal contract
		svc = &contract.Service{
			Name:        name,
			Description: fmt.Sprintf("%s API", name),
			Defaults: &contract.Defaults{
				BaseURL: "http://localhost:8080",
			},
			Resources: []*contract.Resource{
				{
					Name:        "items",
					Description: "Manage items",
					Methods: []*contract.Method{
						{
							Name:        "list",
							Description: "List all items",
							Output:      "ItemList",
							HTTP: &contract.MethodHTTP{
								Method: "GET",
								Path:   "/items",
							},
						},
						{
							Name:        "create",
							Description: "Create a new item",
							Input:       "CreateItemRequest",
							Output:      "Item",
							HTTP: &contract.MethodHTTP{
								Method: "POST",
								Path:   "/items",
							},
						},
						{
							Name:        "get",
							Description: "Get an item by ID",
							Input:       "GetItemRequest",
							Output:      "Item",
							HTTP: &contract.MethodHTTP{
								Method: "GET",
								Path:   "/items/{id}",
							},
						},
					},
				},
			},
			Types: []*contract.Type{
				{
					Name: "Item",
					Kind: contract.KindStruct,
					Fields: []contract.Field{
						{Name: "id", Type: "int64"},
						{Name: "name", Type: "string"},
						{Name: "created_at", Type: "time.Time", Optional: true},
					},
				},
				{
					Name: "ItemList",
					Kind: contract.KindSlice,
					Elem: "Item",
				},
				{
					Name: "CreateItemRequest",
					Kind: contract.KindStruct,
					Fields: []contract.Field{
						{Name: "name", Type: "string"},
					},
				},
				{
					Name: "GetItemRequest",
					Kind: contract.KindStruct,
					Fields: []contract.Field{
						{Name: "id", Type: "int64"},
					},
				},
			},
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(svc)
	if err != nil {
		out.PrintError("marshal failed: %v", err)
		return err
	}

	// Write file
	outputPath := contractV2Flags.output
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		out.PrintError("write failed: %v", err)
		return err
	}

	out.Print("%s %s\n", out.Success("Created"), outputPath)
	out.Print("\nNext steps:\n")
	out.Print("  mizu contract validate %s\n", outputPath)
	out.Print("  mizu contract gen %s --lang typescript\n", outputPath)
	out.Print("  mizu contract spec %s --format openapi\n", outputPath)

	return nil
}

func runContractValidateCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	// Default to api.yaml
	file := "api.yaml"
	if len(args) > 0 {
		file = args[0]
	}

	svc, err := loadContractV2(file)
	if err != nil {
		out.PrintError("parse failed: %v", err)
		return err
	}

	errors, warnings := validateContract(svc)

	if len(warnings) > 0 {
		for _, w := range warnings {
			out.Print("%s %s\n", out.Warn("warning:"), w)
		}
	}

	if len(errors) > 0 {
		for _, e := range errors {
			out.PrintError("%s", e)
		}
		return fmt.Errorf("validation failed with %d errors", len(errors))
	}

	if contractV2Flags.strict && len(warnings) > 0 {
		return fmt.Errorf("validation failed with %d warnings (strict mode)", len(warnings))
	}

	out.Print("%s\n", out.Success("Contract is valid"))
	out.Print("  %d resources, %d methods, %d types\n",
		len(svc.Resources),
		countMethods(svc),
		len(svc.Types),
	)

	return nil
}

func runContractGenCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	file := args[0]
	svc, err := loadContractV2(file)
	if err != nil {
		out.PrintError("parse failed: %v", err)
		return err
	}

	// Validate first
	errors, _ := validateContract(svc)
	if len(errors) > 0 {
		for _, e := range errors {
			out.PrintError("%s", e)
		}
		return fmt.Errorf("contract has errors, cannot generate")
	}

	// Generate code
	var code string
	var filename string

	switch contractV2Flags.lang {
	case "typescript", "ts":
		code = generateTypeScript(svc)
		filename = "api.gen.ts"
	case "go", "golang":
		code = generateGo(svc, contractV2Flags.pkg)
		filename = "api.gen.go"
	default:
		out.PrintError("unsupported language: %s", contractV2Flags.lang)
		return fmt.Errorf("unsupported language: %s", contractV2Flags.lang)
	}

	// Determine output path
	outputDir := contractV2Flags.output
	if outputDir == "" {
		outputDir = filepath.Dir(file)
	}

	outputPath := filepath.Join(outputDir, filename)

	// Ensure directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		out.PrintError("create directory failed: %v", err)
		return err
	}

	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		out.PrintError("write failed: %v", err)
		return err
	}

	out.Print("%s %s\n", out.Success("Generated"), outputPath)

	return nil
}

// Enhanced spec command with v2 support
func runContractSpecV2(file string) ([]byte, error) {
	svc, err := loadContractV2(file)
	if err != nil {
		return nil, err
	}

	switch contractFlags.format {
	case "openapi", "":
		return rest.OpenAPIDocument(svc)
	case "openrpc":
		return jsonrpc.OpenRPCDocument(svc)
	case "asyncapi":
		return async.AsyncAPIDocument(svc)
	default:
		return nil, fmt.Errorf("unsupported format: %s", contractFlags.format)
	}
}

// Helper functions

func loadContractV2(path string) (*contract.Service, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var svc contract.Service
	if err := yaml.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	return &svc, nil
}

func isContractFile(arg string) bool {
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		return false
	}
	ext := filepath.Ext(arg)
	return ext == ".yaml" || ext == ".yml" || ext == ".json"
}

func validateContract(svc *contract.Service) (errors []string, warnings []string) {
	// Build type index
	typeIndex := make(map[string]*contract.Type)
	for _, t := range svc.Types {
		if t != nil && t.Name != "" {
			typeIndex[t.Name] = t
		}
	}

	// Check type references
	checkTypeRef := func(ref contract.TypeRef, context string) {
		r := string(ref)
		if r == "" {
			return
		}
		// Skip primitives
		if isPrimitive(r) {
			return
		}
		if _, ok := typeIndex[r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: unknown type %q", context, r))
		}
	}

	// Validate resources and methods
	for _, res := range svc.Resources {
		if res == nil {
			continue
		}
		if res.Name == "" {
			errors = append(errors, "resource with empty name")
			continue
		}

		for _, m := range res.Methods {
			if m == nil {
				continue
			}
			ctx := fmt.Sprintf("%s.%s", res.Name, m.Name)

			if m.Name == "" {
				errors = append(errors, fmt.Sprintf("%s: method with empty name", res.Name))
				continue
			}

			checkTypeRef(m.Input, ctx+" input")
			checkTypeRef(m.Output, ctx+" output")

			if m.Stream != nil {
				checkTypeRef(m.Stream.Item, ctx+" stream.item")
				checkTypeRef(m.Stream.Done, ctx+" stream.done")
				checkTypeRef(m.Stream.Error, ctx+" stream.error")
				checkTypeRef(m.Stream.InputItem, ctx+" stream.input_item")
			}

			// Check HTTP binding
			if m.HTTP != nil {
				if m.HTTP.Method == "" {
					errors = append(errors, fmt.Sprintf("%s: http binding missing method", ctx))
				}
				if m.HTTP.Path == "" {
					errors = append(errors, fmt.Sprintf("%s: http binding missing path", ctx))
				} else if !strings.HasPrefix(m.HTTP.Path, "/") {
					errors = append(errors, fmt.Sprintf("%s: http path must start with /", ctx))
				}
			}
		}
	}

	// Validate types
	for _, t := range svc.Types {
		if t == nil {
			continue
		}
		if t.Name == "" {
			errors = append(errors, "type with empty name")
			continue
		}

		ctx := fmt.Sprintf("type %s", t.Name)

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				if f.Name == "" {
					errors = append(errors, fmt.Sprintf("%s: field with empty name", ctx))
					continue
				}
				checkTypeRef(f.Type, fmt.Sprintf("%s.%s", ctx, f.Name))
			}

		case contract.KindSlice, contract.KindMap:
			if t.Elem == "" {
				errors = append(errors, fmt.Sprintf("%s: %s missing elem type", ctx, t.Kind))
			} else {
				checkTypeRef(t.Elem, ctx+" elem")
			}

		case contract.KindUnion:
			if t.Tag == "" {
				warnings = append(warnings, fmt.Sprintf("%s: union missing tag field", ctx))
			}
			if len(t.Variants) == 0 {
				errors = append(errors, fmt.Sprintf("%s: union has no variants", ctx))
			}
			for _, v := range t.Variants {
				if v.Value == "" {
					errors = append(errors, fmt.Sprintf("%s: variant missing value", ctx))
				}
				checkTypeRef(v.Type, fmt.Sprintf("%s variant %q", ctx, v.Value))
			}

		default:
			if t.Kind == "" {
				errors = append(errors, fmt.Sprintf("%s: missing kind", ctx))
			} else {
				errors = append(errors, fmt.Sprintf("%s: unknown kind %q", ctx, t.Kind))
			}
		}
	}

	return errors, warnings
}

func isPrimitive(t string) bool {
	switch t {
	case "string", "int", "int32", "int64", "uint", "uint32", "uint64",
		"float32", "float64", "bool", "boolean", "number",
		"time.Time", "json.RawMessage", "any":
		return true
	}
	return false
}

func countMethods(svc *contract.Service) int {
	count := 0
	for _, r := range svc.Resources {
		if r != nil {
			count += len(r.Methods)
		}
	}
	return count
}

// TypeScript code generator
func generateTypeScript(svc *contract.Service) string {
	var sb strings.Builder

	sb.WriteString("// Generated from contract definition\n")
	sb.WriteString("// Do not edit manually\n\n")

	// Sort types for stable output
	types := make([]*contract.Type, len(svc.Types))
	copy(types, svc.Types)
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})

	// Build type index
	typeIndex := make(map[string]*contract.Type)
	for _, t := range types {
		if t != nil {
			typeIndex[t.Name] = t
		}
	}

	for _, t := range types {
		if t == nil {
			continue
		}

		switch t.Kind {
		case contract.KindStruct:
			writeTSStruct(&sb, t, typeIndex)
		case contract.KindSlice:
			writeTSSlice(&sb, t, typeIndex)
		case contract.KindMap:
			writeTSMap(&sb, t, typeIndex)
		case contract.KindUnion:
			writeTSUnion(&sb, t, typeIndex)
		}
	}

	return sb.String()
}

func writeTSStruct(sb *strings.Builder, t *contract.Type, typeIndex map[string]*contract.Type) {
	if t.Description != "" {
		fmt.Fprintf(sb, "/** %s */\n", t.Description)
	}
	fmt.Fprintf(sb, "export type %s = {\n", t.Name)

	for _, f := range t.Fields {
		optional := ""
		if f.Optional {
			optional = "?"
		}
		tsType := typeRefToTS(f.Type, f.Nullable, f.Enum, f.Const, typeIndex)

		if f.Description != "" {
			fmt.Fprintf(sb, "  /** %s */\n", f.Description)
		}
		fmt.Fprintf(sb, "  %s%s: %s\n", f.Name, optional, tsType)
	}

	fmt.Fprintf(sb, "}\n\n")
}

func writeTSSlice(sb *strings.Builder, t *contract.Type, typeIndex map[string]*contract.Type) {
	elemType := typeRefToTS(t.Elem, false, nil, "", typeIndex)
	fmt.Fprintf(sb, "export type %s = %s[]\n\n", t.Name, elemType)
}

func writeTSMap(sb *strings.Builder, t *contract.Type, typeIndex map[string]*contract.Type) {
	elemType := typeRefToTS(t.Elem, false, nil, "", typeIndex)
	fmt.Fprintf(sb, "export type %s = Record<string, %s>\n\n", t.Name, elemType)
}

func writeTSUnion(sb *strings.Builder, t *contract.Type, typeIndex map[string]*contract.Type) {
	if t.Description != "" {
		fmt.Fprintf(sb, "/** %s */\n", t.Description)
	}

	variants := make([]string, 0, len(t.Variants))
	for _, v := range t.Variants {
		variants = append(variants, string(v.Type))
	}

	fmt.Fprintf(sb, "export type %s = %s\n\n", t.Name, strings.Join(variants, " | "))
}

func typeRefToTS(ref contract.TypeRef, nullable bool, enum []string, constVal string, typeIndex map[string]*contract.Type) string {
	// Handle const
	if constVal != "" {
		return fmt.Sprintf("%q", constVal)
	}

	// Handle enum
	if len(enum) > 0 {
		quoted := make([]string, len(enum))
		for i, v := range enum {
			quoted[i] = fmt.Sprintf("%q", v)
		}
		return strings.Join(quoted, " | ")
	}

	r := string(ref)
	base := primitiveToTS(r, typeIndex)

	if nullable {
		return base + " | null"
	}
	return base
}

func primitiveToTS(t string, typeIndex map[string]*contract.Type) string {
	switch t {
	case "string":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64", "float32", "float64", "number":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "time.Time":
		return "string"
	case "json.RawMessage", "any":
		return "unknown"
	default:
		// Check if it's a declared type
		if _, ok := typeIndex[t]; ok {
			return t
		}
		return "unknown"
	}
}

// Go code generator
func generateGo(svc *contract.Service, pkg string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "// Code generated from contract definition. DO NOT EDIT.\n\n")
	fmt.Fprintf(&sb, "package %s\n\n", pkg)

	// Check if we need imports
	needsTime := false
	needsJSON := false

	for _, t := range svc.Types {
		if t == nil {
			continue
		}
		if t.Kind == contract.KindStruct {
			for _, f := range t.Fields {
				if string(f.Type) == "time.Time" {
					needsTime = true
				}
				if string(f.Type) == "json.RawMessage" {
					needsJSON = true
				}
			}
		}
	}

	if needsTime || needsJSON {
		sb.WriteString("import (\n")
		if needsJSON {
			sb.WriteString("\t\"encoding/json\"\n")
		}
		if needsTime {
			sb.WriteString("\t\"time\"\n")
		}
		sb.WriteString(")\n\n")
	}

	// Sort types for stable output
	types := make([]*contract.Type, len(svc.Types))
	copy(types, svc.Types)
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})

	// Build type index
	typeIndex := make(map[string]*contract.Type)
	for _, t := range types {
		if t != nil {
			typeIndex[t.Name] = t
		}
	}

	for _, t := range types {
		if t == nil {
			continue
		}

		switch t.Kind {
		case contract.KindStruct:
			writeGoStruct(&sb, t, typeIndex)
		case contract.KindSlice:
			writeGoSlice(&sb, t, typeIndex)
		case contract.KindMap:
			writeGoMap(&sb, t, typeIndex)
		case contract.KindUnion:
			// Unions in Go are typically handled with interfaces
			writeGoUnion(&sb, t, typeIndex)
		}
	}

	return sb.String()
}

func writeGoStruct(sb *strings.Builder, t *contract.Type, typeIndex map[string]*contract.Type) {
	if t.Description != "" {
		fmt.Fprintf(sb, "// %s %s\n", t.Name, t.Description)
	}
	fmt.Fprintf(sb, "type %s struct {\n", t.Name)

	for _, f := range t.Fields {
		goType := typeRefToGo(f.Type, f.Nullable, f.Optional, typeIndex)
		jsonTag := f.Name
		if f.Optional {
			jsonTag += ",omitempty"
		}

		if f.Description != "" {
			fmt.Fprintf(sb, "\t// %s\n", f.Description)
		}
		fmt.Fprintf(sb, "\t%s %s `json:\"%s\"`\n", exportName(f.Name), goType, jsonTag)
	}

	fmt.Fprintf(sb, "}\n\n")
}

func writeGoSlice(sb *strings.Builder, t *contract.Type, typeIndex map[string]*contract.Type) {
	elemType := typeRefToGo(t.Elem, false, false, typeIndex)
	if t.Description != "" {
		fmt.Fprintf(sb, "// %s %s\n", t.Name, t.Description)
	}
	fmt.Fprintf(sb, "type %s []%s\n\n", t.Name, elemType)
}

func writeGoMap(sb *strings.Builder, t *contract.Type, typeIndex map[string]*contract.Type) {
	elemType := typeRefToGo(t.Elem, false, false, typeIndex)
	if t.Description != "" {
		fmt.Fprintf(sb, "// %s %s\n", t.Name, t.Description)
	}
	fmt.Fprintf(sb, "type %s map[string]%s\n\n", t.Name, elemType)
}

func writeGoUnion(sb *strings.Builder, t *contract.Type, typeIndex map[string]*contract.Type) {
	// For unions, generate an interface and marker method
	if t.Description != "" {
		fmt.Fprintf(sb, "// %s %s\n", t.Name, t.Description)
	}
	fmt.Fprintf(sb, "type %s interface {\n", t.Name)
	fmt.Fprintf(sb, "\tis%s()\n", t.Name)
	fmt.Fprintf(sb, "}\n\n")

	// Note: Variant types should implement the marker method
	// This would typically be done in a separate implementation file
}

func typeRefToGo(ref contract.TypeRef, nullable bool, optional bool, typeIndex map[string]*contract.Type) string {
	r := string(ref)
	base := primitiveToGo(r, typeIndex)

	if nullable || optional {
		// Use pointer for optional/nullable
		if !strings.HasPrefix(base, "*") && !strings.HasPrefix(base, "[]") && !strings.HasPrefix(base, "map[") {
			return "*" + base
		}
	}
	return base
}

func primitiveToGo(t string, typeIndex map[string]*contract.Type) string {
	switch t {
	case "string":
		return "string"
	case "int":
		return "int"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "uint":
		return "uint"
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "float32":
		return "float32"
	case "float64", "number":
		return "float64"
	case "bool", "boolean":
		return "bool"
	case "time.Time":
		return "time.Time"
	case "json.RawMessage":
		return "json.RawMessage"
	case "any":
		return "any"
	default:
		// Check if it's a declared type
		if _, ok := typeIndex[t]; ok {
			return t
		}
		return "any"
	}
}

func exportName(name string) string {
	if name == "" {
		return name
	}
	// Convert snake_case to PascalCase
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

// Template and import helpers

func loadContractTemplate(name string) (*contract.Service, error) {
	switch name {
	case "minimal":
		return &contract.Service{
			Description: "Minimal API template",
			Defaults: &contract.Defaults{
				BaseURL: "http://localhost:8080",
			},
			Resources: []*contract.Resource{
				{
					Name: "health",
					Methods: []*contract.Method{
						{
							Name:        "check",
							Description: "Health check endpoint",
							Output:      "HealthStatus",
							HTTP: &contract.MethodHTTP{
								Method: "GET",
								Path:   "/health",
							},
						},
					},
				},
			},
			Types: []*contract.Type{
				{
					Name: "HealthStatus",
					Kind: contract.KindStruct,
					Fields: []contract.Field{
						{Name: "status", Type: "string"},
						{Name: "timestamp", Type: "time.Time"},
					},
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown template: %s", name)
	}
}

func importContractFromURL(url string) (*contract.Service, error) {
	// Try OpenAPI first
	info, err := discoverOpenAPI(url)
	if err == nil {
		return contractInfoToService(info)
	}

	// Try OpenRPC
	info, err = discoverOpenRPC(url)
	if err == nil {
		return contractInfoToService(info)
	}

	return nil, fmt.Errorf("could not import from %s", url)
}

func contractInfoToService(info *ContractInfo) (*contract.Service, error) {
	svc := &contract.Service{
		Resources: make([]*contract.Resource, 0),
		Types:     make([]*contract.Type, 0),
	}

	// Convert services to resources
	for _, s := range info.Services {
		svc.Name = s.Name
		svc.Description = s.Description

		res := &contract.Resource{
			Name:    s.Name,
			Methods: make([]*contract.Method, 0),
		}

		for _, m := range s.Methods {
			method := &contract.Method{
				Name:        m.Name,
				Description: m.Description,
			}
			if m.Input != nil {
				method.Input = contract.TypeRef(m.Input.Name)
			}
			if m.Output != nil {
				method.Output = contract.TypeRef(m.Output.Name)
			}
			if m.HTTPMethod != "" && m.HTTPPath != "" {
				method.HTTP = &contract.MethodHTTP{
					Method: m.HTTPMethod,
					Path:   m.HTTPPath,
				}
			}
			res.Methods = append(res.Methods, method)
		}

		svc.Resources = append(svc.Resources, res)
	}

	// Convert types
	for _, t := range info.Types {
		ct := schemaToContractType(t.Name, t.Schema)
		if ct != nil {
			svc.Types = append(svc.Types, ct)
		}
	}

	return svc, nil
}

func schemaToContractType(name string, schema map[string]any) *contract.Type {
	t := &contract.Type{
		Name: name,
	}

	schemaType, _ := schema["type"].(string)

	switch schemaType {
	case "object":
		t.Kind = contract.KindStruct
		if props, ok := schema["properties"].(map[string]any); ok {
			required := make(map[string]bool)
			if req, ok := schema["required"].([]any); ok {
				for _, r := range req {
					if s, ok := r.(string); ok {
						required[s] = true
					}
				}
			}

			for fname, fprop := range props {
				fp, _ := fprop.(map[string]any)
				field := contract.Field{
					Name:     fname,
					Optional: !required[fname],
				}

				if desc, ok := fp["description"].(string); ok {
					field.Description = desc
				}

				// Extract type
				if ref, ok := fp["$ref"].(string); ok {
					parts := strings.Split(ref, "/")
					field.Type = contract.TypeRef(parts[len(parts)-1])
				} else if ft, ok := fp["type"].(string); ok {
					field.Type = contract.TypeRef(jsonSchemaTypeToContract(ft, fp))
				}

				// Check nullable
				if anyOf, ok := fp["anyOf"].([]any); ok {
					for _, a := range anyOf {
						if am, ok := a.(map[string]any); ok {
							if am["type"] == "null" {
								field.Nullable = true
							}
						}
					}
				}

				t.Fields = append(t.Fields, field)
			}
		}

	case "array":
		t.Kind = contract.KindSlice
		if items, ok := schema["items"].(map[string]any); ok {
			if ref, ok := items["$ref"].(string); ok {
				parts := strings.Split(ref, "/")
				t.Elem = contract.TypeRef(parts[len(parts)-1])
			} else if it, ok := items["type"].(string); ok {
				t.Elem = contract.TypeRef(jsonSchemaTypeToContract(it, items))
			}
		}

	default:
		// Primitive types don't need to be declared
		return nil
	}

	return t
}

func jsonSchemaTypeToContract(t string, schema map[string]any) string {
	switch t {
	case "string":
		if format, ok := schema["format"].(string); ok {
			if format == "date-time" {
				return "time.Time"
			}
		}
		return "string"
	case "integer":
		if format, ok := schema["format"].(string); ok {
			return format
		}
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	default:
		return "any"
	}
}

// Convert v2 Service to ContractInfo for display
func serviceToContractInfo(svc *contract.Service) *ContractInfo {
	info := &ContractInfo{
		Services: make([]ServiceInfo, 0),
		Types:    make([]TypeInfo, 0),
	}

	for _, res := range svc.Resources {
		if res == nil {
			continue
		}

		svcInfo := ServiceInfo{
			Name:        res.Name,
			Description: res.Description,
			Methods:     make([]MethodInfo, 0),
		}

		for _, m := range res.Methods {
			if m == nil {
				continue
			}

			method := MethodInfo{
				Name:        m.Name,
				FullName:    res.Name + "." + m.Name,
				Description: m.Description,
			}
			if m.HTTP != nil {
				method.HTTPMethod = m.HTTP.Method
				method.HTTPPath = m.HTTP.Path
			}
			if m.Input != "" {
				method.Input = &TypeRef{Name: string(m.Input)}
			}
			if m.Output != "" {
				method.Output = &TypeRef{Name: string(m.Output)}
			}
			svcInfo.Methods = append(svcInfo.Methods, method)
		}

		info.Services = append(info.Services, svcInfo)
	}

	// Convert types to TypeInfo with JSON schema representation
	for _, t := range svc.Types {
		if t == nil {
			continue
		}
		info.Types = append(info.Types, TypeInfo{
			ID:     t.Name,
			Name:   t.Name,
			Schema: contractTypeToSchema(t),
		})
	}

	return info
}

func contractTypeToSchema(t *contract.Type) map[string]any {
	schema := make(map[string]any)

	if t.Description != "" {
		schema["description"] = t.Description
	}

	switch t.Kind {
	case contract.KindStruct:
		schema["type"] = "object"
		props := make(map[string]any)
		required := make([]string, 0)

		for _, f := range t.Fields {
			fp := make(map[string]any)
			if f.Description != "" {
				fp["description"] = f.Description
			}
			fp["type"] = contractTypeToJSONType(string(f.Type))
			if len(f.Enum) > 0 {
				fp["enum"] = f.Enum
			}
			if f.Const != "" {
				fp["const"] = f.Const
			}
			props[f.Name] = fp

			if !f.Optional {
				required = append(required, f.Name)
			}
		}

		schema["properties"] = props
		if len(required) > 0 {
			schema["required"] = required
		}

	case contract.KindSlice:
		schema["type"] = "array"
		schema["items"] = map[string]any{
			"type": contractTypeToJSONType(string(t.Elem)),
		}

	case contract.KindMap:
		schema["type"] = "object"
		schema["additionalProperties"] = map[string]any{
			"type": contractTypeToJSONType(string(t.Elem)),
		}

	case contract.KindUnion:
		variants := make([]map[string]any, 0)
		for _, v := range t.Variants {
			variants = append(variants, map[string]any{
				"$ref": "#/components/schemas/" + string(v.Type),
			})
		}
		schema["oneOf"] = variants
		if t.Tag != "" {
			schema["discriminator"] = map[string]any{
				"propertyName": t.Tag,
			}
		}
	}

	return schema
}

func contractTypeToJSONType(t string) string {
	switch t {
	case "string", "time.Time":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "integer"
	case "float32", "float64", "number":
		return "number"
	case "bool", "boolean":
		return "boolean"
	default:
		return "object"
	}
}

// Update the spec command to support v2 files
func runContractSpecCmdEnhanced(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	// Check if first arg is a local file
	if len(args) > 0 && isContractFile(args[0]) {
		specData, err := runContractSpecV2(args[0])
		if err != nil {
			out.PrintError("%v", err)
			return err
		}

		// Pretty print if requested
		if contractFlags.pretty {
			var buf bytes.Buffer
			if err := json.Indent(&buf, specData, "", "  "); err == nil {
				specData = buf.Bytes()
			}
		}

		// Output
		if contractFlags.output != "" {
			if err := os.WriteFile(contractFlags.output, specData, 0644); err != nil {
				out.PrintError("%v", err)
				return err
			}
			out.Print("Wrote %s\n", contractFlags.output)
		} else {
			fmt.Print(string(specData))
		}

		return nil
	}

	// Fall back to original URL-based behavior
	return runContractSpecCmd(cmd, args)
}

// Discovery function for v2 local files
func discoverContractV2File(path string, out *Output) (*ContractInfo, error) {
	svc, err := loadContractV2(path)
	if err != nil {
		return nil, err
	}
	return serviceToContractInfo(svc), nil
}
