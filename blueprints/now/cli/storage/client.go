package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	Message    string
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
		return nil, &CLIError{
			Code: ExitNetwork,
			Msg:  "network error",
			Hint: "Could not reach " + c.Endpoint + "\nCheck your internet connection and try again",
		}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &CLIError{Code: ExitError, Msg: "failed to read response"}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return data, nil
	}

	// Parse error: try {message} or {error} fields
	var errBody map[string]any
	if json.Unmarshal(data, &errBody) == nil {
		msg := ""
		if m, ok := errBody["message"].(string); ok && m != "" {
			msg = m
		} else if e, ok := errBody["error"].(string); ok && e != "" {
			msg = e
		}
		if msg != "" {
			return nil, &APIError{StatusCode: resp.StatusCode, Message: msg}
		}
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

// Upload streams a file to the API (legacy — use UploadPresigned for new uploads).
func (c *Client) Upload(path string, r io.Reader, contentType string) ([]byte, error) {
	return c.do("PUT", path, r, map[string]string{
		"Content-Type": contentType,
	})
}

// UploadResult contains the result of a presigned upload.
type UploadResult struct {
	Path         string `json:"path"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	Tx           int64  `json:"tx"`
	Time         int64  `json:"time"`
	ContentHash  string `json:"content_hash,omitempty"`
	Deduplicated bool   `json:"deduplicated,omitempty"`
}

// SHA256Hex computes the SHA-256 hex digest of data.
func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// UploadPresigned performs a 3-step presigned upload with content-hash dedup.
//
//  1. POST /files/uploads with path, content_type, content_hash
//     → if deduplicated: true, the blob already exists — done
//  2. PUT content to the presigned URL returned in step 1
//  3. POST /files/uploads/complete with path, content_hash
func (c *Client) UploadPresigned(filePath string, data []byte, contentType string) (*UploadResult, error) {
	contentHash := SHA256Hex(data)

	// Step 1: Initiate upload
	initResp, err := c.DoJSON("POST", "/files/uploads", map[string]string{
		"path":         filePath,
		"content_type": contentType,
		"content_hash": contentHash,
	})
	if err != nil {
		return nil, err
	}

	// Check if deduplicated (blob already exists, no upload needed)
	var initBody struct {
		Deduplicated bool   `json:"deduplicated"`
		Path         string `json:"path"`
		Name         string `json:"name"`
		Size         int64  `json:"size"`
		Tx           int64  `json:"tx"`
		Time         int64  `json:"time"`
		URL          string `json:"url"`
		ContentType  string `json:"content_type"`
		ContentHash  string `json:"content_hash"`
	}
	if err := json.Unmarshal(initResp, &initBody); err != nil {
		return nil, &CLIError{Code: ExitError, Msg: "failed to parse upload response"}
	}

	if initBody.Deduplicated {
		return &UploadResult{
			Path:         initBody.Path,
			Name:         initBody.Name,
			Size:         initBody.Size,
			Tx:           initBody.Tx,
			Time:         initBody.Time,
			ContentHash:  contentHash,
			Deduplicated: true,
		}, nil
	}

	// Step 2: PUT content to presigned URL
	if initBody.URL == "" {
		return nil, &CLIError{Code: ExitError, Msg: "server did not return presigned URL"}
	}

	putReq, err := http.NewRequest("PUT", initBody.URL, bytes.NewReader(data))
	if err != nil {
		return nil, &CLIError{Code: ExitNetwork, Msg: "failed to create upload request"}
	}
	putReq.Header.Set("Content-Type", contentType)

	putResp, err := c.HTTPClient.Do(putReq)
	if err != nil {
		return nil, &CLIError{Code: ExitNetwork, Msg: "failed to upload to storage", Hint: err.Error()}
	}
	defer putResp.Body.Close()

	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		return nil, &CLIError{Code: ExitError, Msg: fmt.Sprintf("upload failed: HTTP %d", putResp.StatusCode)}
	}

	// Step 3: Complete upload
	completeResp, err := c.DoJSON("POST", "/files/uploads/complete", map[string]string{
		"path":         filePath,
		"content_hash": contentHash,
	})
	if err != nil {
		return nil, err
	}

	var result UploadResult
	if err := json.Unmarshal(completeResp, &result); err != nil {
		return nil, &CLIError{Code: ExitError, Msg: "failed to parse complete response"}
	}
	result.ContentHash = contentHash
	return &result, nil
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
		var errBody map[string]any
		if json.Unmarshal(data, &errBody) == nil {
			msg := ""
			if m, ok := errBody["message"].(string); ok && m != "" {
				msg = m
			} else if e, ok := errBody["error"].(string); ok && e != "" {
				msg = e
			}
			if msg != "" {
				return &APIError{StatusCode: resp.StatusCode, Message: msg}
			}
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
		return guessFromExtension(path)
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n == 0 {
		return guessFromExtension(path)
	}

	ct := http.DetectContentType(buf[:n])
	if ct == "application/octet-stream" || ct == "text/plain; charset=utf-8" {
		if ext := guessFromExtension(path); ext != "" {
			return ext
		}
	}

	return ct
}

func guessFromExtension(path string) string {
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
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".mp4"):
		return "video/mp4"
	case strings.HasSuffix(lower, ".mp3"):
		return "audio/mpeg"
	case strings.HasSuffix(lower, ".zip"):
		return "application/zip"
	case strings.HasSuffix(lower, ".gz"):
		return "application/gzip"
	case strings.HasSuffix(lower, ".tar"):
		return "application/x-tar"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	case strings.HasSuffix(lower, ".go"):
		return "text/x-go"
	case strings.HasSuffix(lower, ".ts"), strings.HasSuffix(lower, ".tsx"):
		return "text/typescript"
	case strings.HasSuffix(lower, ".py"):
		return "text/x-python"
	case strings.HasSuffix(lower, ".rs"):
		return "text/x-rust"
	case strings.HasSuffix(lower, ".sh"):
		return "text/x-shellscript"
	default:
		return "application/octet-stream"
	}
}
