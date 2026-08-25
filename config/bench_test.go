package config

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/toml"
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

// benchConf is the struct a generated decoder fills in, using every parser
// that has any work to do.
type benchConf struct {
	App struct {
		Name  string
		Env   string
		Debug bool
		URL   string
		Key   []byte
	}
	HTTP struct {
		Addr           string
		ReadTimeout    time.Duration
		WriteTimeout   time.Duration
		IdleTimeout    time.Duration
		TrustedProxies []netip.Prefix
	}
	Log struct {
		Format string
		Level  slog.Level
	}
	Database struct {
		Driver       string
		DSN          string
		MaxOpenConns int
		MaxIdleConns int
	}
	Cache struct {
		Store string
		TTL   time.Duration
	}
	Queue struct {
		Driver  string
		Workers int
	}
	Mail struct {
		Host string
		Port uint16
	}
	Billing struct {
		TrialDays     int
		WebhookSecret string
	}
}

// decode is what mizu gen config emits, written out by hand. Every line is one
// field, and the order is the order the errors come back in.
func decode(l *Loader, c *benchConf) {
	Get(l, &c.App.Name, fields[0], String)
	Get(l, &c.App.Env, fields[1], String)
	Get(l, &c.App.Debug, fields[2], Bool)
	Get(l, &c.App.URL, fields[3], String)
	Get(l, &c.App.Key, fields[4], Bytes)
	Get(l, &c.HTTP.Addr, fields[5], String)
	Get(l, &c.HTTP.ReadTimeout, fields[6], Duration)
	Get(l, &c.HTTP.WriteTimeout, fields[7], Duration)
	Get(l, &c.HTTP.IdleTimeout, fields[8], Duration)
	Get(l, &c.HTTP.TrustedProxies, fields[9], Slice(Prefix))
	Get(l, &c.Log.Format, fields[10], String)
	Get(l, &c.Log.Level, fields[11], Level)
	Get(l, &c.Database.Driver, fields[12], String)
	Get(l, &c.Database.DSN, fields[13], String)
	Get(l, &c.Database.MaxOpenConns, fields[14], Int)
	Get(l, &c.Database.MaxIdleConns, fields[15], Int)
	Get(l, &c.Cache.Store, fields[16], String)
	Get(l, &c.Cache.TTL, fields[17], Duration)
	Get(l, &c.Queue.Driver, fields[18], String)
	Get(l, &c.Queue.Workers, fields[19], Int)
	Get(l, &c.Mail.Host, fields[20], String)
	Get(l, &c.Mail.Port, fields[21], Uint)
	Get(l, &c.Billing.TrialDays, fields[22], Int)
	Get(l, &c.Billing.WebhookSecret, fields[23], String)
}

// BenchmarkDecode is the whole of startup as an application sees it: read the
// files, then fill in a struct, then check for settings nobody wanted.
func BenchmarkDecode(b *testing.B) {
	s := benchProject(b)
	for b.Loop() {
		l, err := Open(s)
		if err != nil {
			b.Fatal(err)
		}
		var c benchConf
		decode(l, &c)
		if err := l.Err(); err != nil {
			b.Fatal(err)
		}
		if err := l.Check(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGet is the same fields without the reading, which is the part that
// grows with the number of settings rather than the number of files.
func BenchmarkGet(b *testing.B) {
	l, err := Open(benchProject(b))
	if err != nil {
		b.Fatal(err)
	}
	var c benchConf
	for b.Loop() {
		// Every Get records the field it read, so without this the benchmark
		// would spend its time growing a slice to a million settings. Cutting
		// it back keeps the capacity, which is what a real run has after the
		// first few fields anyway.
		l.settings = l.settings[:0]
		decode(l, &c)
	}
	if err := l.Err(); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkParse(b *testing.B) {
	file := func(body string) Value {
		doc, err := toml.Parse("bench.toml", []byte("x = "+body+"\n"))
		if err != nil {
			b.Fatal(err)
		}
		return Value{Source: Source{From: FromFile, Name: "bench.toml:1:5"}, TOML: doc.Get("x")}
	}
	env := func(s string) Value {
		return Value{Source: Source{From: FromEnv, Name: "X"}, Text: s}
	}

	b.Run("String", func(b *testing.B) {
		v, dst := file(`"postgres"`), ""
		for b.Loop() {
			String(&dst, v)
		}
	})
	b.Run("Int", func(b *testing.B) {
		v, dst := env("25"), 0
		for b.Loop() {
			Int(&dst, v)
		}
	})
	b.Run("Duration", func(b *testing.B) {
		v, dst := env("15s"), time.Duration(0)
		for b.Loop() {
			Duration(&dst, v)
		}
	})
	b.Run("Level", func(b *testing.B) {
		v, dst := env("info"), slog.Level(0)
		for b.Loop() {
			Level(&dst, v)
		}
	})
	b.Run("Bytes", func(b *testing.B) {
		v, dst := env("base64:0000000000000000000000000000000000000000000="), []byte(nil)
		for b.Loop() {
			Bytes(&dst, v)
		}
	})
	b.Run("SliceFromFile", func(b *testing.B) {
		v, dst := file(`["10.0.0.0/8", "172.16.0.0/12"]`), []netip.Prefix(nil)
		parse := Slice(Prefix)
		for b.Loop() {
			parse(&dst, v)
		}
	})
	b.Run("SliceFromText", func(b *testing.B) {
		v, dst := env("10.0.0.0/8, 172.16.0.0/12"), []netip.Prefix(nil)
		parse := Slice(Prefix)
		for b.Loop() {
			parse(&dst, v)
		}
	})
}

func ExampleGet() {
	dir, err := os.MkdirTemp("", "mizu")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)

	os.MkdirAll(filepath.Join(dir, "config"), 0o777)
	os.WriteFile(filepath.Join(dir, "config", "local.toml"), []byte(
		"[http]\naddr = \":9000\"\nread_timeout = \"soon\"\n"), 0o666)

	l, err := Open(Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})
	if err != nil {
		fmt.Println(err)
		return
	}

	var c struct {
		Addr    string
		Timeout time.Duration
		Workers int
	}
	Get(l, &c.Addr, Field{Name: "HTTP.Addr", Path: "http.addr"}, String)
	Get(l, &c.Timeout, Field{Name: "HTTP.ReadTimeout", Path: "http.read_timeout"}, Duration)
	Get(l, &c.Workers, Field{Name: "Queue.Workers", Path: "queue.workers", Default: "8"}, Int)

	fmt.Println(c.Addr, c.Workers)
	for _, e := range l.Errors() {
		// The file is in a temporary directory, and on Windows it is separated
		// by backslashes, so cut the directory off and write what is left the
		// way a project writes it down.
		msg := strings.TrimPrefix(e.Error(), dir+string(filepath.Separator))
		fmt.Println(filepath.ToSlash(msg))
	}
	// Output:
	// :9000 8
	// config/local.toml:3:16: HTTP.ReadTimeout: want a length of time such as 30s or 2h45m, got "soon"
}

// megabytes is a setting written the way a person writes it, as 64M rather
// than 67108864. A type that reads itself like this one is a [Parser], and
// [Config] is the parser that hands it the value.
type megabytes int64

func (m *megabytes) ParseConfig(v Value) error {
	// Str is the text of the value from whichever layer had it. Reading
	// v.Text instead works everywhere except a file, which is the one place
	// most settings come from.
	s, err := v.Str()
	if err != nil {
		return err
	}
	// The error says what it wanted and nothing about where, because Get puts
	// the field and the line it was written on in front of it.
	n, ok := strings.CutSuffix(s, "M")
	if !ok {
		return fmt.Errorf("want a size in megabytes such as 64M, got %q", s)
	}
	i, err := strconv.Atoi(n)
	if err != nil {
		return fmt.Errorf("want a size in megabytes such as 64M, got %q", s)
	}
	*m = megabytes(i) << 20
	return nil
}

func ExampleParser() {
	dir, err := os.MkdirTemp("", "mizu")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)

	os.MkdirAll(filepath.Join(dir, "config"), 0o777)
	os.WriteFile(filepath.Join(dir, "config", "local.toml"), []byte(
		"[cache]\nmax_bytes = \"64M\"\nspill_at = 512\nfloor = \"tiny\"\n"), 0o666)

	l, err := Open(Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})
	if err != nil {
		fmt.Println(err)
		return
	}

	var c struct{ MaxBytes, SpillAt, Floor megabytes }
	Get(l, &c.MaxBytes, Field{Name: "Cache.MaxBytes", Path: "cache.max_bytes"}, Config)
	Get(l, &c.SpillAt, Field{Name: "Cache.SpillAt", Path: "cache.spill_at"}, Config)
	Get(l, &c.Floor, Field{Name: "Cache.Floor", Path: "cache.floor"}, Config)

	fmt.Println(c.MaxBytes)
	for _, e := range l.Errors() {
		msg := strings.TrimPrefix(e.Error(), dir+string(filepath.Separator))
		fmt.Println(filepath.ToSlash(msg))
	}
	// Output:
	// 67108864
	// config/local.toml:3:12: Cache.SpillAt: want a string, got integer
	// config/local.toml:4:9: Cache.Floor: want a size in megabytes such as 64M, got "tiny"
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
