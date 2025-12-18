package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var contractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Work with service contracts",
	Long: `Work with service contracts.

Discover, inspect, and call methods on Mizu contract-based services.
Supports JSON-RPC, OpenAPI, and OpenRPC discovery.`,
	Example: `  # List all methods
  mizu contract ls

  # Call a method
  mizu contract call todo.Create '{"title":"Buy milk"}'

  # Export OpenAPI spec
  mizu contract spec > openapi.json

  # Use different server
  mizu contract ls http://api.example.com`,
	RunE: wrapRunE(func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}),
}

var contractLsCmd = &cobra.Command{
	Use:     "ls [file|url]",
	Aliases: []string{"list"},
	Short:   "List services and methods",
	Long: `List services and methods from a contract file or running server.

For v2 contract files (.yaml, .yml), parses and displays the contract.
For running servers, discovers via JSON-RPC, OpenRPC, or OpenAPI.`,
	Example: `  mizu contract ls api.yaml
  mizu contract ls http://localhost:8080`,
	RunE: wrapRunE(runContractLsCmd),
}

var contractShowCmd = &cobra.Command{
	Use:   "show <method> [url]",
	Short: "Show method details",
	Args:  cobra.MinimumNArgs(1),
	RunE:  wrapRunE(runContractShowCmd),
}

var contractCallCmd = &cobra.Command{
	Use:   "call <method> [input] [url]",
	Short: "Call a method",
	Args:  cobra.MinimumNArgs(1),
	RunE:  wrapRunE(runContractCallCmd),
}

var contractSpecCmd = &cobra.Command{
	Use:   "spec [file|url]",
	Short: "Export API specification",
	Long: `Export API specification from a contract file or running server.

For v2 contract files (.yaml, .yml), generates OpenAPI, OpenRPC, or AsyncAPI.
For running servers, fetches the existing specification.`,
	Example: `  mizu contract spec api.yaml --format openapi
  mizu contract spec api.yaml --format openrpc
  mizu contract spec http://localhost:8080`,
	RunE: wrapRunE(runContractSpecCmdEnhanced),
}

var contractTypesCmd = &cobra.Command{
	Use:   "types [type] [url]",
	Short: "List types and schemas",
	RunE:  wrapRunE(runContractTypesCmd),
}

// Contract command flags
var contractFlags struct {
	url     string
	timeout time.Duration
	all     bool
	schema  bool
	format  string
	pretty  bool
	output  string
	service string
	pathID  string
	raw     bool
	headers []string
}

func init() {
	// Add subcommands
	contractCmd.AddCommand(contractLsCmd)
	contractCmd.AddCommand(contractShowCmd)
	contractCmd.AddCommand(contractCallCmd)
	contractCmd.AddCommand(contractSpecCmd)
	contractCmd.AddCommand(contractTypesCmd)

	// Flags for ls
	contractLsCmd.Flags().StringVar(&contractFlags.url, "url", "", "Server URL")
	contractLsCmd.Flags().BoolVar(&contractFlags.all, "all", false, "Include deprecated methods")

	// Flags for show
	contractShowCmd.Flags().StringVar(&contractFlags.url, "url", "", "Server URL")
	contractShowCmd.Flags().BoolVar(&contractFlags.schema, "schema", false, "Show full JSON schema")

	// Flags for call
	contractCallCmd.Flags().StringVar(&contractFlags.url, "url", "", "Server URL")
	contractCallCmd.Flags().DurationVar(&contractFlags.timeout, "timeout", 30*time.Second, "Request timeout")
	contractCallCmd.Flags().StringVar(&contractFlags.pathID, "id", "", "Path parameter ID")
	contractCallCmd.Flags().BoolVar(&contractFlags.raw, "raw", false, "Output raw response")
	contractCallCmd.Flags().StringArrayVarP(&contractFlags.headers, "header", "H", nil, "Add header (key:value)")

	// Flags for spec
	contractSpecCmd.Flags().StringVar(&contractFlags.url, "url", "", "Server URL")
	contractSpecCmd.Flags().StringVar(&contractFlags.format, "format", "", "Output format (openapi, openrpc, asyncapi)")
	contractSpecCmd.Flags().BoolVar(&contractFlags.pretty, "pretty", false, "Pretty print JSON")
	contractSpecCmd.Flags().StringVarP(&contractFlags.output, "output", "o", "", "Output file")
	contractSpecCmd.Flags().StringVar(&contractFlags.service, "service", "", "Export specific service")

	// Flags for types
	contractTypesCmd.Flags().StringVar(&contractFlags.url, "url", "", "Server URL")
	contractTypesCmd.Flags().BoolVar(&contractFlags.schema, "schema", false, "Show full JSON schema")
}

func runContractLsCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	// Check if first arg is a local contract file
	var info *ContractInfo
	var err error

	if len(args) > 0 && isContractFile(args[0]) {
		info, err = discoverContractV2File(args[0], out)
		if err != nil {
			if Flags.JSON {
				emitContractErrorNew(out, err)
			} else {
				out.PrintError("parse failed: %v", err)
			}
			return err
		}
	} else {
		url := resolveContractURL(args)
		info, err = discoverContractNew(url, out)
		if err != nil {
			if Flags.JSON {
				emitContractErrorNew(out, err)
			} else {
				out.PrintError("%v", err)
				out.PrintHint("is the server running? try: mizu dev")
			}
			return err
		}
	}

	if Flags.JSON {
		out.WriteJSON(info)
		return nil
	}

	// Pretty print
	for _, svc := range info.Services {
		out.Print("%s\n", out.Bold(svc.Name))
		if svc.Description != "" && Flags.Verbose > 0 {
			out.Print("  %s\n", out.Dim(svc.Description))
		}

		for _, m := range svc.Methods {
			if m.Deprecated && !contractFlags.all {
				continue
			}

			httpInfo := ""
			if m.HTTPMethod != "" && m.HTTPPath != "" {
				httpInfo = fmt.Sprintf("%-6s %-20s", m.HTTPMethod, m.HTTPPath)
			}

			deprecated := ""
			if m.Deprecated {
				deprecated = out.Dim(" [deprecated]")
			}

			desc := m.Summary
			if desc == "" {
				desc = m.Description
			}
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}

			out.Print("  %-20s %s %s%s\n",
				out.Cyan(m.FullName),
				out.Dim(httpInfo),
				desc,
				deprecated,
			)
		}
		out.Print("\n")
	}

	return nil
}

func runContractShowCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	methodName := args[0]
	url := resolveContractURL(args[1:])

	info, err := discoverContractNew(url, out)
	if err != nil {
		if Flags.JSON {
			emitContractErrorNew(out, err)
		} else {
			out.PrintError("%v", err)
		}
		return err
	}

	// Find method
	var method *MethodInfo
	var suggestions []string
	for _, svc := range info.Services {
		for i := range svc.Methods {
			m := &svc.Methods[i]
			if m.FullName == methodName || m.Name == methodName {
				method = m
				break
			}
			if strings.Contains(strings.ToLower(m.FullName), strings.ToLower(methodName)) {
				suggestions = append(suggestions, m.FullName)
			}
		}
		if method != nil {
			break
		}
	}

	if method == nil {
		out.PrintError("method not found: %s", methodName)
		if len(suggestions) > 0 {
			out.Print("\ndid you mean?\n")
			for _, s := range suggestions {
				out.Print("  %s\n", s)
			}
		}
		return fmt.Errorf("method not found: %s", methodName)
	}

	if contractFlags.schema || Flags.JSON {
		result := map[string]any{
			"name":     method.FullName,
			"fullName": method.FullName,
		}
		if method.Description != "" {
			result["description"] = method.Description
		}
		if method.HTTPMethod != "" {
			result["httpMethod"] = method.HTTPMethod
		}
		if method.HTTPPath != "" {
			result["httpPath"] = method.HTTPPath
		}
		if method.Input != nil {
			for _, t := range info.Types {
				if t.ID == method.Input.ID || t.Name == method.Input.Name {
					result["input"] = t.Schema
					break
				}
			}
		}
		if method.Output != nil {
			for _, t := range info.Types {
				if t.ID == method.Output.ID || t.Name == method.Output.Name {
					result["output"] = t.Schema
					break
				}
			}
		}

		out.WriteJSON(result)
		return nil
	}

	// Pretty print
	out.Print("%s\n", out.Bold(method.FullName))
	if method.HTTPMethod != "" && method.HTTPPath != "" {
		out.Print("  %s %s\n", method.HTTPMethod, method.HTTPPath)
	}
	if method.Description != "" {
		out.Print("\n  %s\n", method.Description)
	}
	if method.Deprecated {
		out.Print("\n  %s\n", out.Warn("DEPRECATED"))
	}

	// Show input schema
	if method.Input != nil {
		out.Print("\n%s (%s):\n", out.Bold("Input"), method.Input.Name)
		for _, t := range info.Types {
			if t.ID == method.Input.ID || t.Name == method.Input.Name {
				printSchemaFieldsNew(out, t.Schema, "  ")
				break
			}
		}
	}

	// Show output schema
	if method.Output != nil {
		out.Print("\n%s (%s):\n", out.Bold("Output"), method.Output.Name)
		for _, t := range info.Types {
			if t.ID == method.Output.ID || t.Name == method.Output.Name {
				printSchemaFieldsNew(out, t.Schema, "  ")
				break
			}
		}
	}

	return nil
}

func runContractCallCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	methodName := args[0]
	var inputData string
	urlArgs := args[1:]

	// Check if second arg is input data or URL
	if len(args) > 1 {
		arg := args[1]
		if strings.HasPrefix(arg, "{") || strings.HasPrefix(arg, "[") || arg == "-" || strings.HasPrefix(arg, "@") {
			inputData = arg
			urlArgs = args[2:]
		}
	}

	url := resolveContractURL(urlArgs)

	// Read input
	var input []byte
	if inputData != "" {
		var err error
		input, err = readInputData(inputData)
		if err != nil {
			out.PrintError("%v", err)
			return err
		}
	}

	info, err := discoverContractNew(url, out)
	if err != nil {
		if Flags.JSON {
			emitContractErrorNew(out, err)
		} else {
			out.PrintError("%v", err)
		}
		return err
	}

	// Find method
	var method *MethodInfo
	for _, svc := range info.Services {
		for i := range svc.Methods {
			m := &svc.Methods[i]
			if m.FullName == methodName || m.Name == methodName {
				method = m
				break
			}
		}
		if method != nil {
			break
		}
	}

	if method == nil {
		out.PrintError("method not found: %s", methodName)
		return fmt.Errorf("method not found: %s", methodName)
	}

	// Validate input requirement
	if method.Input != nil && len(input) == 0 {
		out.PrintError("missing required input")
		out.Print("  the %s method requires input\n", method.FullName)
		out.PrintHint("mizu contract call %s '{...}'", method.FullName)
		out.Print("      mizu contract show %s  # see input schema\n", method.FullName)
		return fmt.Errorf("missing required input")
	}

	// Make the call
	var result []byte
	headers := headerFlags(contractFlags.headers)
	if method.HTTPMethod != "" && method.HTTPPath != "" {
		result, err = callREST(url, method, input, contractFlags.pathID, headers, contractFlags.timeout)
	} else {
		result, err = callJSONRPC(url, method.FullName, input, headers, contractFlags.timeout)
	}

	if err != nil {
		if Flags.JSON {
			emitContractErrorNew(out, err)
		} else {
			out.PrintError("%v", err)
		}
		return err
	}

	// Output result
	// NOTE: Raw fmt.Print is intentional here for unformatted API responses
	// that users may pipe to other tools (jq, etc.)
	if contractFlags.raw {
		fmt.Print(string(result))
	} else {
		var buf bytes.Buffer
		if err := json.Indent(&buf, result, "", "  "); err != nil {
			fmt.Print(string(result))
		} else {
			fmt.Println(buf.String())
		}
	}

	return nil
}

func runContractSpecCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	url := resolveContractURL(args)

	// Determine format and fetch spec
	var specData []byte
	var err error

	if contractFlags.format == "" || contractFlags.format == "openapi" {
		specData, err = fetchSpec(url, "/openapi.json")
		if err != nil && contractFlags.format == "" {
			specData, err = fetchSpec(url, "/openrpc.json")
		}
	} else if contractFlags.format == "openrpc" {
		specData, err = fetchSpec(url, "/openrpc.json")
	}

	if err != nil {
		out.PrintError("%v", err)
		return err
	}

	// Filter by service if requested
	if contractFlags.service != "" {
		specData, err = filterSpecByService(specData, contractFlags.service)
		if err != nil {
			out.PrintError("%v", err)
			return err
		}
	}

	// Pretty print if requested
	if contractFlags.pretty {
		var buf bytes.Buffer
		if err := json.Indent(&buf, specData, "", "  "); err == nil {
			specData = buf.Bytes()
		}
	}

	// Output
	// NOTE: Raw fmt.Print is intentional for spec output that users pipe to files/tools
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

func runContractTypesCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	// Check if specific type requested
	var typeName string
	var urlArgs []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			urlArgs = append(urlArgs, arg)
		} else if typeName == "" {
			typeName = arg
		}
	}

	url := resolveContractURL(urlArgs)

	info, err := discoverContractNew(url, out)
	if err != nil {
		if Flags.JSON {
			emitContractErrorNew(out, err)
		} else {
			out.PrintError("%v", err)
		}
		return err
	}

	// Sort types
	sort.Slice(info.Types, func(i, j int) bool {
		return info.Types[i].Name < info.Types[j].Name
	})

	// Show specific type
	if typeName != "" {
		for _, t := range info.Types {
			if t.Name == typeName || t.ID == typeName {
				if contractFlags.schema || Flags.JSON {
					out.WriteJSON(t.Schema)
				} else {
					out.Print("%s (%s)\n", out.Bold(t.Name), schemaTypeString(t.Schema))
					printSchemaFieldsNew(out, t.Schema, "  ")
				}
				return nil
			}
		}
		out.PrintError("type not found: %s", typeName)
		return fmt.Errorf("type not found: %s", typeName)
	}

	// List all types
	if Flags.JSON {
		out.WriteJSON(info.Types)
		return nil
	}

	for _, t := range info.Types {
		typeStr := schemaTypeString(t.Schema)
		fieldCount := 0
		if props, ok := t.Schema["properties"].(map[string]any); ok {
			fieldCount = len(props)
		}
		enumCount := 0
		if enum, ok := t.Schema["enum"].([]any); ok {
			enumCount = len(enum)
		}

		extra := ""
		if fieldCount > 0 {
			extra = fmt.Sprintf("%d fields", fieldCount)
		} else if enumCount > 0 {
			extra = fmt.Sprintf("%d values", enumCount)
		}

		out.Print("%-25s %-10s %s\n", out.Cyan(t.Name), typeStr, out.Dim(extra))
	}

	return nil
}

// Helper functions

func resolveContractURL(args []string) string {
	// Check positional argument first
	for _, arg := range args {
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			return arg
		}
	}

	// Check explicit flag
	if contractFlags.url != "" {
		return contractFlags.url
	}

	// Check environment variable
	if url := os.Getenv("MIZU_URL"); url != "" {
		return url
	}

	// Default
	return "http://localhost:8080"
}

func discoverContractNew(url string, out *Output) (*ContractInfo, error) {
	// Try JSON-RPC discovery
	info, err := discoverJSONRPC(url)
	if err == nil {
		return info, nil
	}
	out.Verbosef(1, "JSON-RPC discovery failed: %v\n", err)

	// Try OpenRPC spec
	info, err = discoverOpenRPC(url)
	if err == nil {
		return info, nil
	}
	out.Verbosef(1, "OpenRPC discovery failed: %v\n", err)

	// Try OpenAPI spec
	info, err = discoverOpenAPI(url)
	if err == nil {
		return info, nil
	}
	out.Verbosef(1, "OpenAPI discovery failed: %v\n", err)

	// Try contract endpoint
	info, err = discoverContractEndpoint(url)
	if err == nil {
		return info, nil
	}
	out.Verbosef(1, "Contract endpoint discovery failed: %v\n", err)

	return nil, fmt.Errorf("cannot discover contract at %s", url)
}

func readInputData(arg string) ([]byte, error) {
	if arg == "-" {
		return io.ReadAll(os.Stdin)
	}
	if strings.HasPrefix(arg, "@") {
		return os.ReadFile(arg[1:])
	}
	return []byte(arg), nil
}

func printSchemaFieldsNew(out *Output, schema map[string]any, indent string) {
	props, _ := schema["properties"].(map[string]any)
	required := make(map[string]bool)
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}

	var names []string
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		prop := props[name].(map[string]any)
		typeStr := schemaTypeString(prop)
		reqStr := ""
		if required[name] {
			reqStr = out.Dim("required")
		}
		desc := ""
		if d, ok := prop["description"].(string); ok {
			desc = out.Dim(d)
		}
		out.Print("%s%-15s %-10s %s  %s\n", indent, out.Cyan(name), typeStr, reqStr, desc)
	}
}

func emitContractErrorNew(out *Output, err error) {
	out.WriteJSON(map[string]any{
		"error": err.Error(),
	})
}
