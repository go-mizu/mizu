package cli

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// titleCase converts the first letter of each word to uppercase
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// SeedStorageFiles creates actual file content for seeded storage objects
func SeedStorageFiles(ctx context.Context, store *postgres.Store, dataDir string) error {
	// Get all buckets
	buckets, err := store.Storage().ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list buckets: %w", err)
	}

	for _, bucket := range buckets {
		// Get all objects in this bucket
		objects, err := store.Storage().ListObjects(ctx, bucket.ID, "", 1000, 0)
		if err != nil {
			return fmt.Errorf("failed to list objects in bucket %s: %w", bucket.Name, err)
		}

		for _, obj := range objects {
			// Skip folders (.keep files)
			if strings.HasSuffix(obj.Name, ".keep") {
				// Create empty .keep file
				filePath := filepath.Join(dataDir, bucket.ID, obj.Name)
				if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
					return err
				}
				continue
			}

			// Generate content based on content type
			content := generateFileContent(obj.Name, obj.ContentType)
			if content == nil {
				continue
			}

			// Write file to filesystem
			filePath := filepath.Join(dataDir, bucket.ID, obj.Name)
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", filePath, err)
			}
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", filePath, err)
			}

			// Update the size in database to match actual content
			if err := store.Storage().UpdateObjectSize(ctx, obj.ID, int64(len(content))); err != nil {
				// Non-fatal, just log
				fmt.Printf("Warning: failed to update size for %s: %v\n", obj.Name, err)
			}
		}
	}

	return nil
}

// generateFileContent creates actual file content based on file type
func generateFileContent(name, contentType string) []byte {
	ext := strings.ToLower(filepath.Ext(name))
	baseName := strings.TrimSuffix(filepath.Base(name), ext)

	// Text-based files
	switch {
	case contentType == "text/markdown" || ext == ".md":
		return generateMarkdown(baseName)
	case contentType == "application/json" || ext == ".json":
		return generateJSON(baseName)
	case contentType == "application/x-yaml" || ext == ".yaml" || ext == ".yml":
		return generateYAML(baseName)
	case contentType == "text/x-python" || ext == ".py":
		return generatePython(baseName)
	case contentType == "text/x-go" || ext == ".go":
		return generateGo(baseName)
	case ext == ".ts" || ext == ".tsx":
		return generateTypeScript(baseName)
	case ext == ".js" || ext == ".jsx":
		return generateJavaScript(baseName)
	case contentType == "text/html" || ext == ".html" || ext == ".htm":
		return generateHTML(baseName)
	case contentType == "text/css" || ext == ".css":
		return generateCSS(baseName)
	case contentType == "text/x-rust" || ext == ".rs":
		return generateRust(baseName)
	case ext == ".java":
		return generateJava(baseName)
	case contentType == "text/csv" || ext == ".csv":
		return generateCSV()
	case contentType == "application/xml" || ext == ".xml":
		return generateXML(baseName)
	case ext == ".toml":
		return generateTOML(baseName)
	case ext == ".sql":
		return generateSQL()
	case strings.Contains(name, "Dockerfile"):
		return generateDockerfile()
	case strings.Contains(name, "Makefile"):
		return generateMakefile()
	case contentType == "text/plain" || ext == ".txt":
		return generateText(baseName)

	// Image files - generate actual images
	case contentType == "image/svg+xml" || ext == ".svg":
		return generateSVG(baseName)
	case contentType == "image/png" || ext == ".png":
		return generatePNG(baseName)
	case contentType == "image/jpeg" || ext == ".jpg" || ext == ".jpeg":
		return generateJPEG(baseName)
	case contentType == "image/gif" || ext == ".gif":
		return generateGIF()
	case contentType == "image/webp" || ext == ".webp":
		// WebP needs special handling, return PNG for now
		return generatePNG(baseName)
	case contentType == "image/x-icon" || ext == ".ico":
		return generateICO()

	// PDF - minimal valid PDF
	case contentType == "application/pdf" || ext == ".pdf":
		return generatePDF(baseName)

	// Office documents - minimal valid files
	case strings.Contains(contentType, "wordprocessingml") || ext == ".docx":
		return generateDOCX(baseName)
	case strings.Contains(contentType, "spreadsheetml") || ext == ".xlsx":
		return generateXLSX(baseName)
	case strings.Contains(contentType, "presentationml") || ext == ".pptx":
		return generatePPTX(baseName)

	// Audio files - minimal valid audio
	case contentType == "audio/mpeg" || ext == ".mp3":
		return generateMP3()
	case contentType == "audio/wav" || ext == ".wav":
		return generateWAV()
	case contentType == "audio/ogg" || ext == ".ogg":
		return generateOGG()

	// Video files - minimal valid video
	case contentType == "video/mp4" || ext == ".mp4":
		return generateMP4()
	case contentType == "video/webm" || ext == ".webm":
		return generateWebM()

	// Archive files - minimal valid archives
	case contentType == "application/zip" || ext == ".zip":
		return generateZIP()
	case contentType == "application/gzip" || ext == ".gz":
		return generateGZIP()
	}

	// Default: return nil (will use placeholder)
	return nil
}

// Text file generators

func generateMarkdown(name string) []byte {
	return []byte(fmt.Sprintf(`# %s

This is a sample markdown file generated for testing.

## Features

- **Bold text** and *italic text*
- Code blocks with syntax highlighting
- Lists and tables

## Code Example

`+"```go"+`
func main() {
    fmt.Println("Hello, World!")
}
`+"```"+`

## Table

| Name | Description |
|------|-------------|
| Item 1 | First item |
| Item 2 | Second item |

---

Generated by Localbase seeder.
`, titleCase(strings.ReplaceAll(name, "-", " "))))
}

func generateJSON(name string) []byte {
	return []byte(fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "Sample JSON file for testing",
  "settings": {
    "enabled": true,
    "count": 42,
    "tags": ["sample", "test", "demo"]
  },
  "items": [
    {"id": 1, "name": "First"},
    {"id": 2, "name": "Second"},
    {"id": 3, "name": "Third"}
  ]
}
`, name))
}

func generateYAML(name string) []byte {
	return []byte(fmt.Sprintf(`# %s Configuration
name: %s
version: "1.0.0"

settings:
  enabled: true
  debug: false
  log_level: info

database:
  host: localhost
  port: 5432
  name: mydb

features:
  - authentication
  - storage
  - realtime
`, name, name))
}

func generatePython(name string) []byte {
	return []byte(fmt.Sprintf(`#!/usr/bin/env python3
"""
%s - Sample Python script for testing.
"""

import json
from typing import List, Dict


def greet(name: str) -> str:
    """Return a greeting message."""
    return f"Hello, {name}!"


def process_data(items: List[Dict]) -> Dict:
    """Process a list of items and return statistics."""
    return {
        "count": len(items),
        "first": items[0] if items else None,
        "last": items[-1] if items else None,
    }


if __name__ == "__main__":
    print(greet("World"))

    data = [
        {"id": 1, "name": "Alice"},
        {"id": 2, "name": "Bob"},
        {"id": 3, "name": "Charlie"},
    ]

    result = process_data(data)
    print(json.dumps(result, indent=2))
`, name))
}

func generateGo(name string) []byte {
	return []byte(fmt.Sprintf(`package %s

import (
	"fmt"
	"log"
)

// Item represents a sample data structure.
type Item struct {
	ID   int
	Name string
}

// Process handles a list of items.
func Process(items []Item) error {
	for _, item := range items {
		fmt.Printf("Processing: %%s\n", item.Name)
	}
	return nil
}

func main() {
	items := []Item{
		{ID: 1, Name: "First"},
		{ID: 2, Name: "Second"},
		{ID: 3, Name: "Third"},
	}

	if err := Process(items); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done!")
}
`, name))
}

func generateTypeScript(name string) []byte {
	return []byte(fmt.Sprintf(`/**
 * %s - Sample TypeScript module
 */

interface User {
  id: number;
  name: string;
  email: string;
}

interface ApiResponse<T> {
  data: T;
  status: number;
  message: string;
}

export async function fetchUser(id: number): Promise<User> {
  const response = await fetch('/api/users/' + id);
  const data: ApiResponse<User> = await response.json();
  return data.data;
}

export function formatUser(user: User): string {
  return '${user.name} <${user.email}>';
}

// Example usage
const users: User[] = [
  { id: 1, name: 'Alice', email: 'alice@example.com' },
  { id: 2, name: 'Bob', email: 'bob@example.com' },
];

users.forEach(user => console.log(formatUser(user)));
`, name))
}

func generateJavaScript(name string) []byte {
	return []byte(fmt.Sprintf(`/**
 * %s - Sample JavaScript module
 */

const API_BASE = '/api/v1';

async function fetchData(endpoint) {
  const response = await fetch(API_BASE + endpoint);
  if (!response.ok) {
    throw new Error('HTTP error: ' + response.status);
  }
  return response.json();
}

function debounce(func, wait) {
  let timeout;
  return function executedFunction(...args) {
    const later = () => {
      clearTimeout(timeout);
      func(...args);
    };
    clearTimeout(timeout);
    timeout = setTimeout(later, wait);
  };
}

// Example usage
const items = [
  { id: 1, name: 'First', active: true },
  { id: 2, name: 'Second', active: false },
  { id: 3, name: 'Third', active: true },
];

const activeItems = items.filter(item => item.active);
console.log('Active items:', activeItems);

module.exports = { fetchData, debounce };
`, name))
}

func generateHTML(name string) []byte {
	return []byte(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 2rem;
            background: #f5f5f5;
        }
        h1 { color: #333; }
        .card {
            background: white;
            padding: 1.5rem;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin: 1rem 0;
        }
    </style>
</head>
<body>
    <h1>%s</h1>
    <div class="card">
        <h2>Welcome</h2>
        <p>This is a sample HTML file for testing file preview functionality.</p>
    </div>
    <div class="card">
        <h2>Features</h2>
        <ul>
            <li>Responsive design</li>
            <li>Clean styling</li>
            <li>Semantic HTML</li>
        </ul>
    </div>
</body>
</html>
`, name, name))
}

func generateCSS(name string) []byte {
	return []byte(fmt.Sprintf(`/**
 * %s - Sample CSS stylesheet
 */

:root {
  --primary-color: #3ecf8e;
  --secondary-color: #1c1c1c;
  --text-color: #444;
  --border-radius: 8px;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  line-height: 1.6;
  color: var(--text-color);
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
}

.button {
  display: inline-flex;
  align-items: center;
  padding: 0.75rem 1.5rem;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: opacity 0.2s;
}

.button:hover {
  opacity: 0.9;
}

.card {
  background: white;
  border-radius: var(--border-radius);
  padding: 1.5rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

@media (max-width: 768px) {
  .container {
    padding: 0 0.5rem;
  }
}
`, name))
}

func generateRust(name string) []byte {
	return []byte(fmt.Sprintf(`//! %s - Sample Rust module

use std::collections::HashMap;

/// A simple item structure
#[derive(Debug, Clone)]
pub struct Item {
    pub id: u32,
    pub name: String,
}

impl Item {
    /// Create a new item
    pub fn new(id: u32, name: &str) -> Self {
        Self {
            id,
            name: name.to_string(),
        }
    }
}

/// Process a list of items
pub fn process_items(items: &[Item]) -> HashMap<u32, String> {
    items.iter()
        .map(|item| (item.id, item.name.clone()))
        .collect()
}

fn main() {
    let items = vec![
        Item::new(1, "First"),
        Item::new(2, "Second"),
        Item::new(3, "Third"),
    ];

    let map = process_items(&items);

    for (id, name) in &map {
        println!("Item {}: {}", id, name);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_process_items() {
        let items = vec![Item::new(1, "Test")];
        let map = process_items(&items);
        assert_eq!(map.get(&1), Some(&"Test".to_string()));
    }
}
`, name))
}

func generateJava(name string) []byte {
	className := strings.ReplaceAll(titleCase(name), "-", "")
	return []byte(fmt.Sprintf(`package com.example;

import java.util.List;
import java.util.ArrayList;
import java.util.stream.Collectors;

/**
 * %s - Sample Java class
 */
public class %s {
    private int id;
    private String name;

    public %s(int id, String name) {
        this.id = id;
        this.name = name;
    }

    public int getId() { return id; }
    public String getName() { return name; }

    public static List<%s> createSampleData() {
        List<%s> items = new ArrayList<>();
        items.add(new %s(1, "First"));
        items.add(new %s(2, "Second"));
        items.add(new %s(3, "Third"));
        return items;
    }

    public static void main(String[] args) {
        List<%s> items = createSampleData();

        items.stream()
            .map(%s::getName)
            .forEach(System.out::println);
    }
}
`, name, className, className, className, className, className, className, className, className, className))
}

func generateCSV() []byte {
	return []byte(`id,name,email,department,salary,start_date
1,Alice Johnson,alice@example.com,Engineering,95000,2020-03-15
2,Bob Smith,bob@example.com,Marketing,75000,2019-07-22
3,Charlie Brown,charlie@example.com,Engineering,88000,2021-01-10
4,Diana Prince,diana@example.com,Sales,82000,2020-11-05
5,Edward Norton,edward@example.com,Engineering,92000,2018-06-30
6,Fiona Apple,fiona@example.com,HR,70000,2022-02-14
7,George Lucas,george@example.com,Marketing,78000,2021-08-20
8,Helen Troy,helen@example.com,Sales,85000,2019-12-01
9,Ivan Petrov,ivan@example.com,Engineering,90000,2020-04-18
10,Julia Roberts,julia@example.com,HR,72000,2021-09-25
`)
}

func generateXML(name string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <metadata>
        <name>%s</name>
        <version>1.0</version>
        <created>2025-01-18T00:00:00Z</created>
    </metadata>
    <items>
        <item id="1">
            <name>First Item</name>
            <description>This is the first item</description>
            <active>true</active>
        </item>
        <item id="2">
            <name>Second Item</name>
            <description>This is the second item</description>
            <active>false</active>
        </item>
        <item id="3">
            <name>Third Item</name>
            <description>This is the third item</description>
            <active>true</active>
        </item>
    </items>
</root>
`, name))
}

func generateTOML(name string) []byte {
	return []byte(fmt.Sprintf(`# %s Configuration

[package]
name = "%s"
version = "1.0.0"
authors = ["Developer <dev@example.com>"]

[settings]
debug = false
log_level = "info"
max_connections = 100

[database]
host = "localhost"
port = 5432
name = "mydb"
pool_size = 10

[features]
authentication = true
storage = true
realtime = true

[[servers]]
name = "primary"
host = "server1.example.com"
port = 8080

[[servers]]
name = "backup"
host = "server2.example.com"
port = 8080
`, name, name))
}

func generateSQL() []byte {
	return []byte(`-- Sample SQL queries for demonstration

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create posts table
CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data
INSERT INTO users (name, email) VALUES
    ('Alice', 'alice@example.com'),
    ('Bob', 'bob@example.com'),
    ('Charlie', 'charlie@example.com');

-- Query: Get all users with their post counts
SELECT
    u.name,
    u.email,
    COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.id, u.name, u.email
ORDER BY post_count DESC;

-- Query: Get recent published posts
SELECT
    p.title,
    p.content,
    u.name as author,
    p.created_at
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.published = true
ORDER BY p.created_at DESC
LIMIT 10;
`)
}

func generateDockerfile() []byte {
	return []byte(`# Dockerfile for sample application
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]
`)
}

func generateMakefile() []byte {
	return []byte(`.PHONY: all build test clean run

# Variables
BINARY_NAME=app
GO=go
GOFLAGS=-ldflags="-s -w"

all: build

build:
	$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) ./cmd/server

test:
	$(GO) test -v ./...

test-coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

run: build
	./bin/$(BINARY_NAME)

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

deps:
	$(GO) mod download
	$(GO) mod tidy

docker-build:
	docker build -t $(BINARY_NAME) .

docker-run:
	docker run -p 8080:8080 $(BINARY_NAME)
`)
}

func generateText(name string) []byte {
	return []byte(fmt.Sprintf(`%s

This is a sample text file generated for testing file preview functionality.

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim
veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea
commodo consequat.

Features:
- Plain text format
- Easy to read and edit
- Universal compatibility
- Lightweight file size

Notes:
1. This file was automatically generated
2. Content is for demonstration purposes
3. Feel free to modify as needed

Generated by Localbase seeder.
`, titleCase(strings.ReplaceAll(name, "-", " "))))
}

// Image generators

func generateSVG(name string) []byte {
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200" width="200" height="200">
  <defs>
    <linearGradient id="grad" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" style="stop-color:#3ecf8e;stop-opacity:1" />
      <stop offset="100%%" style="stop-color:#1c1c1c;stop-opacity:1" />
    </linearGradient>
  </defs>
  <rect width="200" height="200" fill="url(#grad)" rx="20"/>
  <text x="100" y="90" font-family="Arial, sans-serif" font-size="14" fill="white" text-anchor="middle">%s</text>
  <text x="100" y="120" font-family="Arial, sans-serif" font-size="10" fill="rgba(255,255,255,0.7)" text-anchor="middle">Sample SVG</text>
</svg>`, name))
}

func generatePNG(_ string) []byte {
	// Create a simple colored PNG image
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))

	// Fill with gradient-like pattern
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			r := uint8(62 + (x * 100 / 200))
			g := uint8(207 - (y * 50 / 200))
			b := uint8(142 - (x * 50 / 200))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func generateJPEG(_ string) []byte {
	// Create a simple colored JPEG image
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))

	// Fill with gradient-like pattern
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			r := uint8(100 + (x * 100 / 200))
			g := uint8(150 - (y * 50 / 200))
			b := uint8(200 - (x * 50 / 200))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return buf.Bytes()
}

func generateGIF() []byte {
	// Minimal valid GIF (1x1 pixel, transparent)
	// GIF89a header + minimal image data
	gif := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // GIF89a
		0x01, 0x00, 0x01, 0x00, // 1x1 dimensions
		0x80, 0x00, 0x00, // Global color table flag
		0xFF, 0xFF, 0xFF, // White
		0x00, 0x00, 0x00, // Black
		0x21, 0xF9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, // Graphics control
		0x2C, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // Image descriptor
		0x02, 0x02, 0x44, 0x01, 0x00, // Image data
		0x3B, // Trailer
	}
	return gif
}

func generateICO() []byte {
	// Create a simple 16x16 ICO file
	// ICO header + PNG data
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	// Fill with brand color
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{62, 207, 142, 255})
		}
	}

	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)
	pngData := pngBuf.Bytes()

	// ICO header
	ico := []byte{
		0x00, 0x00, // Reserved
		0x01, 0x00, // ICO type
		0x01, 0x00, // 1 image
		// Image entry
		0x10,                   // Width (16)
		0x10,                   // Height (16)
		0x00,                   // Color palette
		0x00,                   // Reserved
		0x01, 0x00,             // Color planes
		0x20, 0x00,             // Bits per pixel (32)
	}

	// Size of PNG data (4 bytes, little endian)
	size := uint32(len(pngData))
	ico = append(ico, byte(size), byte(size>>8), byte(size>>16), byte(size>>24))

	// Offset to image data (22 bytes = header size)
	ico = append(ico, 0x16, 0x00, 0x00, 0x00)

	// Append PNG data
	ico = append(ico, pngData...)

	return ico
}

// PDF generator - minimal valid PDF

func generatePDF(name string) []byte {
	content := fmt.Sprintf(`%%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>
endobj
4 0 obj
<< /Length 120 >>
stream
BT
/F1 24 Tf
100 700 Td
(%s) Tj
/F1 12 Tf
0 -30 Td
(Sample PDF document for testing) Tj
ET
endstream
endobj
5 0 obj
<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>
endobj
xref
0 6
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000266 00000 n
0000000437 00000 n
trailer
<< /Size 6 /Root 1 0 R >>
startxref
522
%%%%EOF
`, name)
	return []byte(content)
}

// Office document generators - these are actually ZIP files with XML content

func generateDOCX(_ string) []byte {
	// DOCX is a complex ZIP format with XML content
	// Return nil to use placeholder - browser will show "Preview not available"
	return nil
}

func generateXLSX(_ string) []byte {
	// XLSX is a complex ZIP format with XML content
	// Return nil to use placeholder
	return nil
}

func generatePPTX(_ string) []byte {
	// PPTX is a complex ZIP format with XML content
	return nil
}

// Audio generators - minimal valid audio files

func generateMP3() []byte {
	// Minimal valid MP3 frame (silence)
	// MP3 frame header for 128kbps, 44100Hz, stereo
	return []byte{
		0xFF, 0xFB, 0x90, 0x00, // Frame header
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Repeat for minimal valid file
		0xFF, 0xFB, 0x90, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
}

func generateWAV() []byte {
	// Minimal valid WAV file (44 byte header + 1 sample)
	return []byte{
		// RIFF header
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x26, 0x00, 0x00, 0x00, // File size - 8
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		// fmt chunk
		0x66, 0x6D, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Chunk size (16)
		0x01, 0x00,             // Audio format (1 = PCM)
		0x01, 0x00,             // Channels (1 = mono)
		0x44, 0xAC, 0x00, 0x00, // Sample rate (44100)
		0x88, 0x58, 0x01, 0x00, // Byte rate
		0x02, 0x00,             // Block align
		0x10, 0x00,             // Bits per sample (16)
		// data chunk
		0x64, 0x61, 0x74, 0x61, // "data"
		0x02, 0x00, 0x00, 0x00, // Data size
		0x00, 0x00,             // One sample of silence
	}
}

func generateOGG() []byte {
	// Minimal Ogg container is complex, return nil for placeholder
	return nil
}

// Video generators

func generateMP4() []byte {
	// Minimal MP4 is complex, return nil for placeholder
	return nil
}

func generateWebM() []byte {
	// Minimal WebM is complex, return nil for placeholder
	return nil
}

// Archive generators

func generateZIP() []byte {
	// Minimal valid ZIP with one empty file
	return []byte{
		// Local file header
		0x50, 0x4B, 0x03, 0x04, // Signature
		0x0A, 0x00,             // Version
		0x00, 0x00,             // Flags
		0x00, 0x00,             // Compression
		0x00, 0x00,             // Mod time
		0x00, 0x00,             // Mod date
		0x00, 0x00, 0x00, 0x00, // CRC
		0x00, 0x00, 0x00, 0x00, // Compressed size
		0x00, 0x00, 0x00, 0x00, // Uncompressed size
		0x09, 0x00,             // Filename length
		0x00, 0x00,             // Extra field length
		0x52, 0x45, 0x41, 0x44, 0x4D, 0x45, 0x2E, 0x6D, 0x64, // "README.md"
		// Central directory
		0x50, 0x4B, 0x01, 0x02,
		0x14, 0x00, 0x0A, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x09, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x52, 0x45, 0x41, 0x44, 0x4D, 0x45, 0x2E, 0x6D, 0x64,
		// End of central directory
		0x50, 0x4B, 0x05, 0x06,
		0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x00,
		0x37, 0x00, 0x00, 0x00,
		0x27, 0x00, 0x00, 0x00,
		0x00, 0x00,
	}
}

func generateGZIP() []byte {
	// Minimal valid gzip of empty content
	return []byte{
		0x1F, 0x8B, 0x08, 0x00, // Magic + compression + flags
		0x00, 0x00, 0x00, 0x00, // Modification time
		0x00, 0x03,             // Extra flags + OS
		0x03, 0x00,             // Compressed empty block
		0x00, 0x00, 0x00, 0x00, // CRC32
		0x00, 0x00, 0x00, 0x00, // Original size
	}
}
