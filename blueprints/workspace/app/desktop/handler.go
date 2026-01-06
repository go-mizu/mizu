package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ProxyHandler proxies ALL requests to the backend server
type ProxyHandler struct {
	backendAddr string
	client      *http.Client
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(backendAddr string) *ProxyHandler {
	return &ProxyHandler{
		backendAddr: backendAddr,
		client:      &http.Client{},
	}
}

// ServeHTTP proxies all requests to the backend
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root path redirects to login/app
	if path == "/" || path == "" {
		path = "/login"
	}

	// Build backend URL
	backendURL := fmt.Sprintf("http://%s%s", h.backendAddr, path)
	if r.URL.RawQuery != "" {
		backendURL += "?" + r.URL.RawQuery
	}

	// Create new request
	req, err := http.NewRequestWithContext(r.Context(), r.Method, backendURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Send request
	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, "Backend unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set content type for HTML responses to ensure proper rendering
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" && strings.HasSuffix(path, ".html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}

	// Write status and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// AssetHandler is kept for backwards compatibility but not used
type AssetHandler struct {
	backendAddr string
	client      *http.Client
}

// NewAssetHandler creates a new asset handler (deprecated, use NewProxyHandler)
func NewAssetHandler(backendAddr string) *AssetHandler {
	return &AssetHandler{
		backendAddr: backendAddr,
		client:      &http.Client{},
	}
}

// ServeHTTP handles HTTP requests
func (h *AssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Proxy all requests to backend
	backendURL := fmt.Sprintf("http://%s%s", h.backendAddr, r.URL.RequestURI())

	req, err := http.NewRequestWithContext(r.Context(), r.Method, backendURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, "Backend unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
