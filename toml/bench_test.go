package toml

import (
	"fmt"
	"strings"
	"testing"
)

// config is about the size of a real application's config/production.toml,
// which is the document this parser exists to read.
var config = []byte(`
[app]
name = "mizu"
env = "production"
debug = false
url = "https://example.com"
key = "base64:0000000000000000000000000000000000000000000="
timezone = "UTC"
locale = "en"

[http]
addr = ":8080"
read_timeout = "15s"
write_timeout = "15s"
idle_timeout = "60s"
max_header_bytes = 1_048_576
trusted_proxies = ["10.0.0.0/8", "172.16.0.0/12"]

[log]
format = "json"
level = "info"
add_source = false

[database]
driver = "postgres"
dsn = "postgres://user:pass@localhost:5432/mizu?sslmode=disable"
max_open_conns = 25
max_idle_conns = 25
conn_max_lifetime = "5m"

[cache]
store = "redis"
prefix = "mizu"
ttl = "1h"

[queue]
driver = "database"
workers = 8
retry_after = "90s"

[[queue.connections]]
name = "default"
tries = 3

[[queue.connections]]
name = "emails"
tries = 5

[mail]
driver = "smtp"
host = "smtp.example.com"
port = 587
from = { address = "hello@example.com", name = "Example" }

[deploy]
at = 2026-08-24T09:15:00Z
`)

func BenchmarkParse(b *testing.B) {
	b.SetBytes(int64(len(config)))
	for b.Loop() {
		if _, err := Parse("config.toml", config); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseLarge is the shape of a generated or exported document: many
// keys, few tables, nothing clever.
func BenchmarkParseLarge(b *testing.B) {
	var doc strings.Builder
	for i := range 200 {
		fmt.Fprintf(&doc, "[section_%d]\nname = \"value %d\"\ncount = %d\nratio = %d.5\nenabled = true\n\n", i, i, i, i)
	}
	src := []byte(doc.String())

	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for b.Loop() {
		if _, err := Parse("large.toml", src); err != nil {
			b.Fatal(err)
		}
	}
}

// repeat builds a document of n lines from a template, which takes the line
// number so that every key is different.
func repeat(template string, n int) []byte {
	var doc strings.Builder
	for i := range n {
		fmt.Fprintf(&doc, template, i)
	}
	return []byte(doc.String())
}

func BenchmarkParseStrings(b *testing.B) {
	src := repeat("a%[1]d = \"one two three\"\nb%[1]d = 'four five six'\nc%[1]d = \"seven \\u00e9 eight\"\n", 50)
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		if _, err := Parse("strings.toml", src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseNumbers(b *testing.B) {
	src := repeat("a%[1]d = 1_234_567\nb%[1]d = -3.141_592e-7\nc%[1]d = 0xdead_beef\n", 50)
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		if _, err := Parse("numbers.toml", src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseDates(b *testing.B) {
	src := repeat("a%[1]d = 1979-05-27T07:32:00Z\nb%[1]d = 1979-05-27\nc%[1]d = 07:32:00\n", 50)
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		if _, err := Parse("dates.toml", src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLookup(b *testing.B) {
	doc, err := Parse("config.toml", config)
	if err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if v := doc.Lookup("database", "max_open_conns"); v == nil {
			b.Fatal("not found")
		}
	}
}
