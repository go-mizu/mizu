package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fields are what an application with a full mizu.Base and a few modules of
// its own asks for, which is the work this package does once at startup.
var fields = []Field{
	{Path: "app.name", Env: "APP_NAME", Default: "mizu"},
	{Path: "app.env", Env: "MIZU_ENV", Default: "local"},
	{Path: "app.debug", Env: "APP_DEBUG", Default: "false"},
	{Path: "app.url", Env: "APP_URL", Default: "http://localhost:8080"},
	{Path: "app.key", Env: "APP_KEY", Secret: true},
	{Path: "http.addr", Env: "HTTP_ADDR", Default: ":8080"},
	{Path: "http.read_timeout", Env: "HTTP_READ_TIMEOUT", Default: "15s"},
	{Path: "http.write_timeout", Env: "HTTP_WRITE_TIMEOUT", Default: "15s"},
	{Path: "http.idle_timeout", Env: "HTTP_IDLE_TIMEOUT", Default: "60s"},
	{Path: "http.trusted_proxies", Env: "HTTP_TRUSTED_PROXIES"},
	{Path: "log.format", Env: "LOG_FORMAT", Default: "json"},
	{Path: "log.level", Env: "LOG_LEVEL", Default: "info"},
	{Path: "database.driver", Env: "DB_DRIVER", Default: "postgres"},
	{Path: "database.dsn", Env: "DATABASE_URL", Secret: true},
	{Path: "database.max_open_conns", Env: "DB_MAX_OPEN_CONNS", Default: "25"},
	{Path: "database.max_idle_conns", Env: "DB_MAX_IDLE_CONNS", Default: "25"},
	{Path: "cache.store", Env: "CACHE_STORE", Default: "redis"},
	{Path: "cache.ttl", Env: "CACHE_TTL", Default: "1h"},
	{Path: "queue.driver", Env: "QUEUE_DRIVER", Default: "database"},
	{Path: "queue.workers", Env: "QUEUE_WORKERS", Default: "8"},
	{Path: "mail.host", Env: "MAIL_HOST", Default: "localhost"},
	{Path: "mail.port", Env: "MAIL_PORT", Default: "1025"},
	{Path: "billing.trial_days", Env: "BILLING_TRIAL_DAYS", Default: "14"},
	{Path: "billing.webhook_secret", Env: "BILLING_WEBHOOK_SECRET", Secret: true},
}

const benchConfig = `
[app]
name = "mizu"
debug = false
url = "https://example.com"

[http]
addr = ":8080"
read_timeout = "15s"
write_timeout = "15s"
idle_timeout = "60s"
trusted_proxies = ["10.0.0.0/8", "172.16.0.0/12"]

[log]
format = "json"
level = "info"

[database]
driver = "postgres"
max_open_conns = 25
max_idle_conns = 25

[cache]
store = "redis"
ttl = "1h"

[queue]
driver = "database"
workers = 8

[mail]
host = "smtp.example.com"
port = 587

[billing]
trial_days = 30
`

const benchDotEnv = `
APP_KEY=base64:0000000000000000000000000000000000000000000=
DB_HOST=db.internal
DATABASE_URL=postgres://user:pass@${DB_HOST}:5432/app?sslmode=disable
BILLING_WEBHOOK_SECRET=whsec_0000000000000000000000000000000000
`

// benchProject writes a project that has all three kinds of file in it and
// returns the sources for reading it.
func benchProject(b *testing.B) Sources {
	b.Helper()
	dir := b.TempDir()
	files := map[string]string{
		filepath.Join("config", "production.toml"): benchConfig,
		".env": benchDotEnv,
	}
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o666); err != nil {
			b.Fatal(err)
		}
	}
	s := Discover(dir, []string{"MIZU_ENV=production", "LOG_LEVEL=debug"}, []string{"--config.http.addr=:9000"})
	s.DotEnv = []string{filepath.Join(dir, ".env")}
	return s
}

// BenchmarkOpen is the part that touches the disk, which happens once.
func BenchmarkOpen(b *testing.B) {
	s := benchProject(b)
	for b.Loop() {
		if _, err := Open(s); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFind is the search through the layers, which runs once per field
// and is what grows with the size of an application's configuration.
//
// It measures find rather than Lookup because Lookup also records the field,
// and a benchmark that called it a million times would be measuring a slice
// growing to a million settings. What startup really does is BenchmarkLoad.
func BenchmarkFind(b *testing.B) {
	l, err := Open(benchProject(b))
	if err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		for _, f := range fields {
			if _, ok := l.find(f); !ok {
				b.Fatalf("%s is not set", f.Path)
			}
		}
	}
}

// BenchmarkLoad is what startup actually costs: read everything, then ask for
// every field, then check for settings nobody wanted.
func BenchmarkLoad(b *testing.B) {
	s := benchProject(b)
	for b.Loop() {
		l, err := Open(s)
		if err != nil {
			b.Fatal(err)
		}
		for _, f := range fields {
			l.Lookup(f)
		}
		if err := l.Check(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCheck(b *testing.B) {
	l, err := Open(benchProject(b))
	if err != nil {
		b.Fatal(err)
	}
	for _, f := range fields {
		l.Lookup(f)
	}
	for b.Loop() {
		if err := l.Check(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCheckTypo is the same walk when there is something to report, which
// is where the suggestions get worked out.
func BenchmarkCheckTypo(b *testing.B) {
	s := benchProject(b)
	l, err := Open(s)
	if err != nil {
		b.Fatal(err)
	}
	for _, f := range fields {
		l.Lookup(f)
	}
	l.Lookup(Field{Path: "database.max_conns"}) // not in any file, so nothing to find
	for b.Loop() {
		l.Check()
	}
}

func BenchmarkParseDotEnv(b *testing.B) {
	src := []byte(strings.Repeat(benchDotEnv, 8))
	resolve := func(string) (string, bool) { return "", false }
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		if _, err := parseDotEnv(".env", src, resolve); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNearest(b *testing.B) {
	known := make([]string, 0, len(fields))
	for _, f := range fields {
		known = append(known, f.Path)
	}
	for b.Loop() {
		if got := nearest("database.max_conns", known); got == "" {
			b.Fatal("found nothing to suggest")
		}
	}
}

func ExampleLoader() {
	dir, err := os.MkdirTemp("", "mizu")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)

	os.MkdirAll(filepath.Join(dir, "config"), 0o777)
	os.WriteFile(filepath.Join(dir, "config", "production.toml"), []byte(
		"[database]\nmax_open_conns = 25\n"), 0o666)
	os.WriteFile(filepath.Join(dir, ".env"), []byte(
		"DATABASE_URL=postgres://localhost/app\n"), 0o666)

	s := Discover(dir, []string{"MIZU_ENV=production"}, nil)
	s.DotEnv = []string{filepath.Join(dir, ".env")} // production does not read one
	l, err := Open(s)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, f := range []Field{
		{Path: "database.dsn", Env: "DATABASE_URL", Secret: true},
		{Path: "database.max_open_conns", Env: "DB_MAX_OPEN_CONNS", Default: "10"},
		{Path: "database.max_idle_conns", Env: "DB_MAX_IDLE_CONNS", Default: "10"},
	} {
		l.Lookup(f)
	}
	for _, s := range l.Settings() {
		fmt.Printf("%-24s %-8s %s\n", s.Path, s.Display(), s.Value.Source.From)
	}
	// Output:
	// database.dsn             ***      dotenv
	// database.max_open_conns  25       file
	// database.max_idle_conns  10       default
}
