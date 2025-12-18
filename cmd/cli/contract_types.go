package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

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

// headerFlags implements flag.Value for repeatable headers.
type headerFlags []string

func (h *headerFlags) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

// Discovery functions

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

			if parts := strings.SplitN(op.OperationID, ".", 2); len(parts) == 2 {
				svc.Name = parts[0]
				method.Name = parts[1]
			} else {
				method.Name = op.OperationID
			}

			if op.RequestBody != nil {
				if content, ok := op.RequestBody.Content["application/json"]; ok {
					ref := extractSchemaRef(content.Schema)
					if ref != "" {
						method.Input = &TypeRef{ID: ref, Name: ref}
					}
				}
			}

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

	sort.Slice(svc.Methods, func(i, j int) bool {
		return svc.Methods[i].Name < svc.Methods[j].Name
	})

	info.Services = append(info.Services, svc)

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
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
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
	_ = serviceName
	return data, nil
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
