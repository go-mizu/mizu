package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// contractFlags holds flags for the contract command.
type contractFlags struct {
	url     string
	timeout time.Duration
}

// Contract subcommands
var contractSubcommands = []*command{
	{name: "ls", short: "List services and methods", run: runContractLs, usage: usageContractLs},
	{name: "show", short: "Show method details", run: runContractShow, usage: usageContractShow},
	{name: "call", short: "Call a method", run: runContractCall, usage: usageContractCall},
	{name: "spec", short: "Export API specification", run: runContractSpec, usage: usageContractSpec},
	{name: "types", short: "List types and schemas", run: runContractTypes, usage: usageContractTypes},
}

func runContract(args []string, gf *globalFlags) int {
	if len(args) == 0 || (len(args) > 0 && (args[0] == "-h" || args[0] == "--help")) {
		usageContract()
		return exitOK
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	for _, cmd := range contractSubcommands {
		if cmd.name == cmdName {
			return cmd.run(cmdArgs, gf)
		}
	}

	fmt.Fprintf(os.Stderr, "error: unknown contract subcommand %q\n", cmdName)
	fmt.Fprintf(os.Stderr, "Run 'mizu contract --help' for usage.\n")
	return exitUsage
}

func usageContract() {
	fmt.Println("mizu contract - Work with service contracts")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mizu contract <command> [url] [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, cmd := range contractSubcommands {
		fmt.Printf("  %-10s %s\n", cmd.name, cmd.short)
	}
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List all methods")
	fmt.Println("  mizu contract ls")
	fmt.Println()
	fmt.Println("  # Call a method")
	fmt.Println("  mizu contract call todo.Create '{\"title\":\"Buy milk\"}'")
	fmt.Println()
	fmt.Println("  # Export OpenAPI spec")
	fmt.Println("  mizu contract spec > openapi.json")
	fmt.Println()
	fmt.Println("  # Use a different server")
	fmt.Println("  mizu contract ls http://api.example.com")
	fmt.Println("  MIZU_URL=http://api.example.com mizu contract ls")
	fmt.Println()
	fmt.Println("Global Flags:")
	fmt.Println("      --url string    Server URL (default \"http://localhost:8080\")")
	fmt.Println("      --json          Output as JSON")
	fmt.Println("      --no-color      Disable colored output")
	fmt.Println("  -v, --verbose       Show request/response details")
	fmt.Println("  -h, --help          Show help")
}

// --- Discovery Types ---

// ContractInfo represents discovered contract information.
type ContractInfo struct {
	Services []ServiceInfo `json:"services"`
	Types    []TypeInfo    `json:"types"`
}

// ServiceInfo describes a service.
type ServiceInfo struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Version     string       `json:"version,omitempty"`
	Methods     []MethodInfo `json:"methods"`
}

// MethodInfo describes a method.
type MethodInfo struct {
	Name        string   `json:"name"`
	FullName    string   `json:"fullName"`
	Description string   `json:"description,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	HTTPMethod  string   `json:"httpMethod,omitempty"`
	HTTPPath    string   `json:"httpPath,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty"`
	Input       *TypeRef `json:"input,omitempty"`
	Output      *TypeRef `json:"output,omitempty"`
}

// TypeRef references a type.
type TypeRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TypeInfo describes a type.
type TypeInfo struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
}

// --- URL Resolution ---

func resolveURL(args []string, cf *contractFlags) string {
	// Check positional argument first
	for _, arg := range args {
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			return arg
		}
	}

	// Check explicit flag
	if cf.url != "" {
		return cf.url
	}

	// Check environment variable
	if url := os.Getenv("MIZU_URL"); url != "" {
		return url
	}

	// Default
	return "http://localhost:8080"
}

// --- Discovery ---

func discoverContract(url string, out *output) (*ContractInfo, error) {
	// Try JSON-RPC discovery
	info, err := discoverJSONRPC(url)
	if err == nil {
		return info, nil
	}
	out.verbosef(1, "JSON-RPC discovery failed: %v\n", err)

	// Try OpenRPC spec
	info, err = discoverOpenRPC(url)
	if err == nil {
		return info, nil
	}
	out.verbosef(1, "OpenRPC discovery failed: %v\n", err)

	// Try OpenAPI spec
	info, err = discoverOpenAPI(url)
	if err == nil {
		return info, nil
	}
	out.verbosef(1, "OpenAPI discovery failed: %v\n", err)

	// Try contract endpoint
	info, err = discoverContractEndpoint(url)
	if err == nil {
		return info, nil
	}
	out.verbosef(1, "Contract endpoint discovery failed: %v\n", err)

	return nil, fmt.Errorf("cannot discover contract at %s", url)
}

func discoverJSONRPC(url string) (*ContractInfo, error) {
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  "rpc.discover",
		"id":      1,
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var rpcResp struct {
		Result *ContractInfo `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", rpcResp.Error.Message)
	}
	if rpcResp.Result == nil {
		return nil, fmt.Errorf("no result")
	}

	return rpcResp.Result, nil
}

func discoverOpenRPC(baseURL string) (*ContractInfo, error) {
	specURL := strings.TrimSuffix(baseURL, "/") + "/openrpc.json"
	resp, err := http.Get(specURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var doc struct {
		Info struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Version     string `json:"version"`
		} `json:"info"`
		Methods []struct {
			Name        string `json:"name"`
			Summary     string `json:"summary"`
			Description string `json:"description"`
			Deprecated  bool   `json:"deprecated"`
			Params      []struct {
				Name   string         `json:"name"`
				Schema map[string]any `json:"schema"`
			} `json:"params"`
			Result *struct {
				Name   string         `json:"name"`
				Schema map[string]any `json:"schema"`
			} `json:"result"`
		} `json:"methods"`
		Components *struct {
			Schemas map[string]map[string]any `json:"schemas"`
		} `json:"components"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	info := &ContractInfo{
		Types: make([]TypeInfo, 0),
	}

	// Extract service name from title or first method
	serviceName := "service"
	if doc.Info.Title != "" {
		serviceName = strings.TrimSuffix(doc.Info.Title, " API")
	}

	svc := ServiceInfo{
		Name:        serviceName,
		Description: doc.Info.Description,
		Version:     doc.Info.Version,
		Methods:     make([]MethodInfo, 0, len(doc.Methods)),
	}

	for _, m := range doc.Methods {
		method := MethodInfo{
			Name:        m.Name,
			FullName:    m.Name,
			Description: m.Description,
			Summary:     m.Summary,
			Deprecated:  m.Deprecated,
		}

		// Parse Service.Method name format
		if parts := strings.SplitN(m.Name, ".", 2); len(parts) == 2 {
			svc.Name = parts[0]
			method.Name = parts[1]
		}

		if len(m.Params) > 0 && m.Params[0].Schema != nil {
			ref := extractSchemaRef(m.Params[0].Schema)
			if ref != "" {
				method.Input = &TypeRef{ID: ref, Name: ref}
			}
		}

		if m.Result != nil && m.Result.Schema != nil {
			ref := extractSchemaRef(m.Result.Schema)
			if ref != "" {
				method.Output = &TypeRef{ID: ref, Name: ref}
			}
		}

		svc.Methods = append(svc.Methods, method)
	}

	info.Services = append(info.Services, svc)

	// Extract types
	if doc.Components != nil {
		for id, schema := range doc.Components.Schemas {
			info.Types = append(info.Types, TypeInfo{
				ID:     id,
				Name:   id,
				Schema: schema,
			})
		}
	}

	return info, nil
}

func discoverOpenAPI(baseURL string) (*ContractInfo, error) {
	specURL := strings.TrimSuffix(baseURL, "/") + "/openapi.json"
	resp, err := http.Get(specURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var doc struct {
		Info struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Version     string `json:"version"`
		} `json:"info"`
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
			Summary     string `json:"summary"`
			Description string `json:"description"`
			Deprecated  bool   `json:"deprecated"`
			RequestBody *struct {
				Content map[string]struct {
					Schema map[string]any `json:"schema"`
				} `json:"content"`
			} `json:"requestBody"`
			Responses map[string]struct {
				Content map[string]struct {
					Schema map[string]any `json:"schema"`
				} `json:"content"`
			} `json:"responses"`
		} `json:"paths"`
		Components *struct {
			Schemas map[string]map[string]any `json:"schemas"`
		} `json:"components"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	info := &ContractInfo{
		Types: make([]TypeInfo, 0),
	}

	// Extract service name
	serviceName := "service"
	if doc.Info.Title != "" {
		serviceName = strings.TrimSuffix(doc.Info.Title, " API")
	}

	svc := ServiceInfo{
		Name:        serviceName,
		Description: doc.Info.Description,
		Version:     doc.Info.Version,
		Methods:     make([]MethodInfo, 0),
	}

	for path, methods := range doc.Paths {
		for httpMethod, op := range methods {
			if op.OperationID == "" {
				continue
			}

			method := MethodInfo{
				FullName:    op.OperationID,
				Description: op.Description,
				Summary:     op.Summary,
				HTTPMethod:  strings.ToUpper(httpMethod),
				HTTPPath:    path,
				Deprecated:  op.Deprecated,
			}

			// Parse Service.Method name format
			if parts := strings.SplitN(op.OperationID, ".", 2); len(parts) == 2 {
				svc.Name = parts[0]
				method.Name = parts[1]
			} else {
				method.Name = op.OperationID
			}

			// Extract input type
			if op.RequestBody != nil {
				if content, ok := op.RequestBody.Content["application/json"]; ok {
					ref := extractSchemaRef(content.Schema)
					if ref != "" {
						method.Input = &TypeRef{ID: ref, Name: ref}
					}
				}
			}

			// Extract output type from 200 response
			if resp, ok := op.Responses["200"]; ok {
				if content, ok := resp.Content["application/json"]; ok {
					ref := extractSchemaRef(content.Schema)
					if ref != "" {
						method.Output = &TypeRef{ID: ref, Name: ref}
					}
				}
			}

			svc.Methods = append(svc.Methods, method)
		}
	}

	// Sort methods by name
	sort.Slice(svc.Methods, func(i, j int) bool {
		return svc.Methods[i].Name < svc.Methods[j].Name
	})

	info.Services = append(info.Services, svc)

	// Extract types
	if doc.Components != nil {
		for id, schema := range doc.Components.Schemas {
			info.Types = append(info.Types, TypeInfo{
				ID:     id,
				Name:   id,
				Schema: schema,
			})
		}
	}

	return info, nil
}

func discoverContractEndpoint(baseURL string) (*ContractInfo, error) {
	specURL := strings.TrimSuffix(baseURL, "/") + "/_contract"
	resp, err := http.Get(specURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var info ContractInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

func extractSchemaRef(schema map[string]any) string {
	if ref, ok := schema["$ref"].(string); ok {
		// Extract type name from #/components/schemas/TypeName
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
}

// --- ls subcommand ---

func runContractLs(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)
	cf := &contractFlags{}

	fs := flag.NewFlagSet("contract ls", flag.ContinueOnError)
	fs.StringVar(&cf.url, "url", "", "Server URL")
	showAll := fs.Bool("all", false, "Include deprecated methods")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageContractLs()
			return exitOK
		}
		return exitUsage
	}

	url := resolveURL(fs.Args(), cf)
	info, err := discoverContract(url, out)
	if err != nil {
		if out.json {
			emitContractError(out, err)
		} else {
			out.errorf("error: %v\n", err)
			out.errorf("\nhint: is the server running? try: mizu dev\n")
		}
		return exitError
	}

	if out.json {
		enc := newJSONEncoder(out.stdout)
		_ = enc.encode(info)
		return exitOK
	}

	// Pretty print
	for _, svc := range info.Services {
		out.print("%s\n", out.bold(svc.Name))
		if svc.Description != "" && gf.verbose > 0 {
			out.print("  %s\n", out.gray(svc.Description))
		}

		for _, m := range svc.Methods {
			if m.Deprecated && !*showAll {
				continue
			}

			httpInfo := ""
			if m.HTTPMethod != "" && m.HTTPPath != "" {
				httpInfo = fmt.Sprintf("%-6s %-20s", m.HTTPMethod, m.HTTPPath)
			}

			deprecated := ""
			if m.Deprecated {
				deprecated = out.gray(" [deprecated]")
			}

			desc := m.Summary
			if desc == "" {
				desc = m.Description
			}
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}

			out.print("  %-20s %s %s%s\n",
				out.cyan(m.FullName),
				out.gray(httpInfo),
				desc,
				deprecated,
			)
		}
		out.print("\n")
	}

	return exitOK
}

func usageContractLs() {
	fmt.Println("Usage:")
	fmt.Println("  mizu contract ls [url] [flags]")
	fmt.Println()
	fmt.Println("List all services and methods.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mizu contract ls")
	fmt.Println("  mizu contract ls http://api.example.com")
	fmt.Println("  mizu contract ls --json")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --url string    Server URL (default \"http://localhost:8080\")")
	fmt.Println("      --all           Include deprecated methods")
	fmt.Println("      --json          Output as JSON")
	fmt.Println("  -h, --help          Show help")
}

// --- show subcommand ---

func runContractShow(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)
	cf := &contractFlags{}

	fs := flag.NewFlagSet("contract show", flag.ContinueOnError)
	fs.StringVar(&cf.url, "url", "", "Server URL")
	showSchema := fs.Bool("schema", false, "Show full JSON schema")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageContractShow()
			return exitOK
		}
		return exitUsage
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		out.errorf("error: method name required\n")
		out.errorf("\nUsage: mizu contract show <method>\n")
		return exitUsage
	}

	methodName := remaining[0]
	url := resolveURL(remaining[1:], cf)

	info, err := discoverContract(url, out)
	if err != nil {
		if out.json {
			emitContractError(out, err)
		} else {
			out.errorf("error: %v\n", err)
		}
		return exitError
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
			// Collect similar names for suggestions
			if strings.Contains(strings.ToLower(m.FullName), strings.ToLower(methodName)) {
				suggestions = append(suggestions, m.FullName)
			}
		}
		if method != nil {
			break
		}
	}

	if method == nil {
		out.errorf("error: method not found: %s\n", methodName)
		if len(suggestions) > 0 {
			out.errorf("\ndid you mean?\n")
			for _, s := range suggestions {
				out.errorf("  %s\n", s)
			}
		}
		return exitError
	}

	if *showSchema || out.json {
		// Build schema response
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

		enc := newJSONEncoder(out.stdout)
		_ = enc.encode(result)
		return exitOK
	}

	// Pretty print
	out.print("%s\n", out.bold(method.FullName))
	if method.HTTPMethod != "" && method.HTTPPath != "" {
		out.print("  %s %s\n", method.HTTPMethod, method.HTTPPath)
	}
	if method.Description != "" {
		out.print("\n  %s\n", method.Description)
	}
	if method.Deprecated {
		out.print("\n  %s\n", out.yellow("DEPRECATED"))
	}

	// Show input schema
	if method.Input != nil {
		out.print("\n%s (%s):\n", out.bold("Input"), method.Input.Name)
		for _, t := range info.Types {
			if t.ID == method.Input.ID || t.Name == method.Input.Name {
				printSchemaFields(out, t.Schema, "  ")
				break
			}
		}
	}

	// Show output schema
	if method.Output != nil {
		out.print("\n%s (%s):\n", out.bold("Output"), method.Output.Name)
		for _, t := range info.Types {
			if t.ID == method.Output.ID || t.Name == method.Output.Name {
				printSchemaFields(out, t.Schema, "  ")
				break
			}
		}
	}

	return exitOK
}

func printSchemaFields(out *output, schema map[string]any, indent string) {
	props, _ := schema["properties"].(map[string]any)
	required := make(map[string]bool)
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}

	// Sort field names
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
			reqStr = out.gray("required")
		}
		desc := ""
		if d, ok := prop["description"].(string); ok {
			desc = out.gray(d)
		}
		out.print("%s%-15s %-10s %s  %s\n", indent, out.cyan(name), typeStr, reqStr, desc)
	}
}

func schemaTypeString(schema map[string]any) string {
	if enum, ok := schema["enum"].([]any); ok {
		return fmt.Sprintf("enum(%d)", len(enum))
	}
	if t, ok := schema["type"].(string); ok {
		if format, ok := schema["format"].(string); ok {
			return t + ":" + format
		}
		return t
	}
	return "any"
}

func usageContractShow() {
	fmt.Println("Usage:")
	fmt.Println("  mizu contract show <method> [url] [flags]")
	fmt.Println()
	fmt.Println("Show detailed information about a method.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mizu contract show todo.Create")
	fmt.Println("  mizu contract show todo.Create --schema")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --url string    Server URL")
	fmt.Println("      --schema        Show full JSON schema")
	fmt.Println("      --json          Output as JSON")
	fmt.Println("  -h, --help          Show help")
}

// --- call subcommand ---

func runContractCall(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)
	cf := &contractFlags{timeout: 30 * time.Second}

	fs := flag.NewFlagSet("contract call", flag.ContinueOnError)
	fs.StringVar(&cf.url, "url", "", "Server URL")
	fs.DurationVar(&cf.timeout, "timeout", 30*time.Second, "Request timeout")
	pathID := fs.String("id", "", "Path parameter ID")
	raw := fs.Bool("raw", false, "Output raw response")
	var headers headerFlags
	fs.Var(&headers, "H", "Add header (key:value)")
	fs.Var(&headers, "header", "Add header (key:value)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageContractCall()
			return exitOK
		}
		return exitUsage
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		out.errorf("error: method name required\n")
		out.errorf("\nUsage: mizu contract call <method> [input]\n")
		return exitUsage
	}

	methodName := remaining[0]
	var inputData string
	urlArgs := remaining[1:]

	// Check if second arg is input data or URL
	if len(remaining) > 1 {
		arg := remaining[1]
		if strings.HasPrefix(arg, "{") || strings.HasPrefix(arg, "[") || arg == "-" || strings.HasPrefix(arg, "@") {
			inputData = arg
			urlArgs = remaining[2:]
		}
	}

	url := resolveURL(urlArgs, cf)

	// Read input
	var input []byte
	if inputData != "" {
		var err error
		input, err = readInput(inputData)
		if err != nil {
			out.errorf("error: %v\n", err)
			return exitError
		}
	}

	info, err := discoverContract(url, out)
	if err != nil {
		if out.json {
			emitContractError(out, err)
		} else {
			out.errorf("error: %v\n", err)
		}
		return exitError
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
		out.errorf("error: method not found: %s\n", methodName)
		return exitError
	}

	// Validate input requirement
	if method.Input != nil && len(input) == 0 {
		out.errorf("error: missing required input\n")
		out.errorf("  the %s method requires input\n", method.FullName)
		out.errorf("\nhint: mizu contract call %s '{...}'\n", method.FullName)
		out.errorf("      mizu contract show %s  # see input schema\n", method.FullName)
		return exitUsage
	}

	// Make the call
	var result []byte
	if method.HTTPMethod != "" && method.HTTPPath != "" {
		// REST call
		result, err = callREST(url, method, input, *pathID, headers, cf.timeout)
	} else {
		// JSON-RPC call
		result, err = callJSONRPC(url, method.FullName, input, headers, cf.timeout)
	}

	if err != nil {
		if out.json {
			emitContractError(out, err)
		} else {
			out.errorf("error: %v\n", err)
		}
		return exitError
	}

	// Output result
	if *raw {
		fmt.Print(string(result))
	} else {
		// Pretty print JSON
		var buf bytes.Buffer
		if err := json.Indent(&buf, result, "", "  "); err != nil {
			fmt.Print(string(result))
		} else {
			fmt.Println(buf.String())
		}
	}

	return exitOK
}

func readInput(arg string) ([]byte, error) {
	if arg == "-" {
		return io.ReadAll(os.Stdin)
	}
	if strings.HasPrefix(arg, "@") {
		return os.ReadFile(arg[1:])
	}
	return []byte(arg), nil
}

func callJSONRPC(url, method string, input []byte, headers headerFlags, timeout time.Duration) ([]byte, error) {
	var params any
	if len(input) > 0 {
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid JSON input: %w", err)
		}
	}

	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"id":      1,
	}
	if params != nil {
		req["params"] = params
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			httpReq.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		Result any `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    any    `json:"data"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return json.Marshal(rpcResp.Result)
}

func callREST(baseURL string, method *MethodInfo, input []byte, pathID string, headers headerFlags, timeout time.Duration) ([]byte, error) {
	// Build URL
	path := method.HTTPPath
	if pathID != "" {
		path = strings.ReplaceAll(path, "{id}", pathID)
	}

	url := strings.TrimSuffix(baseURL, "/") + path

	var bodyReader io.Reader
	if len(input) > 0 && method.HTTPMethod != http.MethodGet {
		bodyReader = bytes.NewReader(input)
	}

	httpReq, err := http.NewRequest(method.HTTPMethod, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if bodyReader != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			httpReq.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Message != "" {
			return nil, fmt.Errorf("%s: %s", errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// headerFlags implements flag.Value for repeatable headers
type headerFlags []string

func (h *headerFlags) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func usageContractCall() {
	fmt.Println("Usage:")
	fmt.Println("  mizu contract call <method> [input] [url] [flags]")
	fmt.Println()
	fmt.Println("Call a method with input data.")
	fmt.Println()
	fmt.Println("Input formats:")
	fmt.Println("  '{\"key\":\"value\"}'   JSON string")
	fmt.Println("  @file.json           Read from file")
	fmt.Println("  -                    Read from stdin")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mizu contract call todo.Create '{\"title\":\"Buy milk\"}'")
	fmt.Println("  mizu contract call todo.List")
	fmt.Println("  mizu contract call todo.Get --id abc123")
	fmt.Println("  echo '{\"title\":\"test\"}' | mizu contract call todo.Create -")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --url string        Server URL")
	fmt.Println("      --id string         Path parameter ID (for REST)")
	fmt.Println("      --timeout duration  Request timeout (default 30s)")
	fmt.Println("  -H, --header key:value  Add request header")
	fmt.Println("      --raw               Output raw response")
	fmt.Println("      --json              Force JSON output")
	fmt.Println("  -h, --help              Show help")
}

// --- spec subcommand ---

func runContractSpec(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)
	cf := &contractFlags{}

	fs := flag.NewFlagSet("contract spec", flag.ContinueOnError)
	fs.StringVar(&cf.url, "url", "", "Server URL")
	format := fs.String("format", "", "Output format (openapi, openrpc)")
	pretty := fs.Bool("pretty", false, "Pretty print JSON")
	outputFile := fs.String("o", "", "Output file")
	fs.StringVar(outputFile, "output", "", "Output file")
	serviceName := fs.String("service", "", "Export specific service")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageContractSpec()
			return exitOK
		}
		return exitUsage
	}

	url := resolveURL(fs.Args(), cf)

	// Determine format and fetch spec
	var specData []byte
	var err error

	if *format == "" || *format == "openapi" {
		specData, err = fetchSpec(url, "/openapi.json")
		if err != nil && *format == "" {
			specData, err = fetchSpec(url, "/openrpc.json")
		}
	} else if *format == "openrpc" {
		specData, err = fetchSpec(url, "/openrpc.json")
	}

	if err != nil {
		out.errorf("error: %v\n", err)
		return exitError
	}

	// Filter by service if requested
	if *serviceName != "" {
		specData, err = filterSpecByService(specData, *serviceName)
		if err != nil {
			out.errorf("error: %v\n", err)
			return exitError
		}
	}

	// Pretty print if requested
	if *pretty {
		var buf bytes.Buffer
		if err := json.Indent(&buf, specData, "", "  "); err == nil {
			specData = buf.Bytes()
		}
	}

	// Output
	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, specData, 0644); err != nil {
			out.errorf("error: %v\n", err)
			return exitError
		}
		out.print("Wrote %s\n", *outputFile)
	} else {
		fmt.Print(string(specData))
	}

	return exitOK
}

func fetchSpec(baseURL, path string) ([]byte, error) {
	specURL := strings.TrimSuffix(baseURL, "/") + path
	resp, err := http.Get(specURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: HTTP %d", specURL, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func filterSpecByService(data []byte, serviceName string) ([]byte, error) {
	// For now, just return the full spec
	// TODO: implement filtering
	_ = serviceName
	return data, nil
}

func usageContractSpec() {
	fmt.Println("Usage:")
	fmt.Println("  mizu contract spec [url] [flags]")
	fmt.Println()
	fmt.Println("Export API specification.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mizu contract spec > openapi.json")
	fmt.Println("  mizu contract spec --format openrpc > openrpc.json")
	fmt.Println("  mizu contract spec --pretty")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --url string       Server URL")
	fmt.Println("      --format string    Output format (openapi, openrpc)")
	fmt.Println("      --service string   Export specific service")
	fmt.Println("      --pretty           Pretty print JSON")
	fmt.Println("  -o, --output string    Write to file")
	fmt.Println("  -h, --help             Show help")
}

// --- types subcommand ---

func runContractTypes(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)
	cf := &contractFlags{}

	fs := flag.NewFlagSet("contract types", flag.ContinueOnError)
	fs.StringVar(&cf.url, "url", "", "Server URL")
	showSchema := fs.Bool("schema", false, "Show full JSON schema")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageContractTypes()
			return exitOK
		}
		return exitUsage
	}

	remaining := fs.Args()
	url := resolveURL(remaining, cf)

	// Check if specific type requested
	var typeName string
	for _, arg := range remaining {
		if !strings.HasPrefix(arg, "http://") && !strings.HasPrefix(arg, "https://") {
			typeName = arg
			break
		}
	}

	info, err := discoverContract(url, out)
	if err != nil {
		if out.json {
			emitContractError(out, err)
		} else {
			out.errorf("error: %v\n", err)
		}
		return exitError
	}

	// Sort types
	sort.Slice(info.Types, func(i, j int) bool {
		return info.Types[i].Name < info.Types[j].Name
	})

	// Show specific type
	if typeName != "" {
		for _, t := range info.Types {
			if t.Name == typeName || t.ID == typeName {
				if *showSchema || out.json {
					enc := newJSONEncoder(out.stdout)
					_ = enc.encode(t.Schema)
				} else {
					out.print("%s (%s)\n", out.bold(t.Name), schemaTypeString(t.Schema))
					printSchemaFields(out, t.Schema, "  ")
				}
				return exitOK
			}
		}
		out.errorf("error: type not found: %s\n", typeName)
		return exitError
	}

	// List all types
	if out.json {
		enc := newJSONEncoder(out.stdout)
		_ = enc.encode(info.Types)
		return exitOK
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

		out.print("%-25s %-10s %s\n", out.cyan(t.Name), typeStr, out.gray(extra))
	}

	return exitOK
}

func usageContractTypes() {
	fmt.Println("Usage:")
	fmt.Println("  mizu contract types [type] [url] [flags]")
	fmt.Println()
	fmt.Println("List types and their schemas.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mizu contract types")
	fmt.Println("  mizu contract types Todo")
	fmt.Println("  mizu contract types Todo --schema")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --url string    Server URL")
	fmt.Println("      --schema        Show full JSON schema")
	fmt.Println("      --json          Output as JSON")
	fmt.Println("  -h, --help          Show help")
}

// --- helpers ---

func emitContractError(out *output, err error) {
	result := map[string]any{
		"error": err.Error(),
	}
	enc := newJSONEncoder(out.stdout)
	_ = enc.encode(result)
}
