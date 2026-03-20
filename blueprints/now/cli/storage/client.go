package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Client wraps HTTP calls to the storage API.
type Client struct {
	Endpoint   string
	Token      string
	HTTPClient *http.Client
}

// APIError represents a structured API error.
type APIError struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("HTTP %d", e.StatusCode)
}

// ExitCode returns the CLI exit code for this API error.
func (e *APIError) ExitCode() int {
	switch e.StatusCode {
	case 401:
		return ExitAuth
	case 403:
		return ExitPermission
	case 404:
		return ExitNotFound
	case 409:
		return ExitConflict
	default:
		return ExitError
	}
}

// do performs an HTTP request and returns the response body.
func (c *Client) do(method, path string, body io.Reader, headers map[string]string) ([]byte, error) {
	url := c.Endpoint + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, &CLIError{Code: ExitNetwork, Msg: "failed to create request", Hint: err.Error()}
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &CLIError{Code: ExitNetwork, Msg: "network error", Hint: "Could not reach " + c.Endpoint + "\nCheck your internet connection and try again"}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &CLIError{Code: ExitError, Msg: "failed to read response"}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return data, nil
	}

	// Parse error
	var errBody struct {
		Error APIError `json:"error"`
	}
	if json.Unmarshal(data, &errBody) == nil && errBody.Error.Message != "" {
		errBody.Error.StatusCode = resp.StatusCode
		return nil, &errBody.Error
	}

	return nil, &APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
}

// DoJSON performs a request with JSON body and returns parsed response.
func (c *Client) DoJSON(method, path string, reqBody any) ([]byte, error) {
	var body io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	return c.do(method, path, body, map[string]string{
		"Content-Type": "application/json",
	})
}

// Get performs a GET request.
func (c *Client) Get(path string) ([]byte, error) {
	return c.do("GET", path, nil, nil)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) ([]byte, error) {
	return c.do("DELETE", path, nil, nil)
}

// Upload streams a file to the API.
func (c *Client) Upload(method, path string, r io.Reader, contentType string) ([]byte, error) {
	return c.do(method, path, r, map[string]string{
		"Content-Type": contentType,
	})
}

// Download streams a file from the API to a writer.
func (c *Client) Download(path string, w io.Writer) error {
	url := c.Endpoint + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &CLIError{Code: ExitNetwork, Msg: "failed to create request"}
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return &CLIError{Code: ExitNetwork, Msg: "network error", Hint: "Could not reach " + c.Endpoint}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		var errBody struct {
			Error APIError `json:"error"`
		}
		if json.Unmarshal(data, &errBody) == nil && errBody.Error.Message != "" {
			errBody.Error.StatusCode = resp.StatusCode
			return &errBody.Error
		}
		return &APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

// DetectContentType returns the MIME type for a file path.
func DetectContentType(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n == 0 {
		return "application/octet-stream"
	}

	ct := http.DetectContentType(buf[:n])

	// Improve detection for common text types based on extension
	if ct == "application/octet-stream" || ct == "text/plain; charset=utf-8" {
		lower := strings.ToLower(path)
		switch {
		case strings.HasSuffix(lower, ".json"):
			return "application/json"
		case strings.HasSuffix(lower, ".md"):
			return "text/markdown"
		case strings.HasSuffix(lower, ".html"), strings.HasSuffix(lower, ".htm"):
			return "text/html"
		case strings.HasSuffix(lower, ".css"):
			return "text/css"
		case strings.HasSuffix(lower, ".js"):
			return "application/javascript"
		case strings.HasSuffix(lower, ".xml"):
			return "application/xml"
		case strings.HasSuffix(lower, ".csv"):
			return "text/csv"
		case strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
			return "application/yaml"
		case strings.HasSuffix(lower, ".pdf"):
			return "application/pdf"
		}
	}

	return ct
}
