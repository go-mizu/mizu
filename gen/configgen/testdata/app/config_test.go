package app

import (
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/config"
)

// sources are the layers the round trip reads, with the secret in the
// database DSN written to a file so that the file: indirection is exercised
// through generated code rather than only in the config package's own tests.
func sources(t *testing.T) config.Sources {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "dsn")
	if err := os.WriteFile(dsn, []byte("postgres://localhost/app\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return config.Sources{
		Env:      "local",
		Files:    []string{"config/app.toml", "config/local.toml"},
		DotEnv:   []string{"config/test.env"},
		Environ:  []string{"APP_KEY=c2VjcmV0", "DATABASE_URL=file:" + dsn},
		Args:     []string{"--config.queue.driver=redis"},
		Override: map[string]string{"billing.currency": "gbp"},
	}
}

func load(t *testing.T) *Config {
	t.Helper()
	c, err := LoadConfig(sources(t))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLoadConfig(t *testing.T) {
	c := load(t)

	prefixes := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("192.168.0.0/16"),
	}

	equal(t, "app.name", c.App.Name, "example")
	equal(t, "app.env", c.App.Env, Env("local"))
	equal(t, "app.debug", c.App.Debug, true)
	equal(t, "app.key", string(c.App.Key), "secret")
	equal(t, "app.lang", c.App.Locale, "fr")
	equal(t, "app.internal", c.App.Internal, "")

	equal(t, "http.addr", c.HTTP.Addr, netip.MustParseAddrPort("0.0.0.0:8080"))
	equal(t, "http.host", c.HTTP.Host, "example.test")
	equal(t, "http.read_timeout", c.HTTP.ReadTimeout, 45*time.Second)
	equal(t, "http.write_timeout", c.HTTP.WriteTimeout, 30*time.Second)
	equal(t, "http.max_header_bytes", c.HTTP.MaxHeaderBytes, 2097152)
	equalSlice(t, "http.trusted_proxies", c.HTTP.TrustedProxies, prefixes)
	equal(t, "http.bind_to", c.HTTP.BindTo, netip.MustParseAddr("0.0.0.0"))
	equalMap(t, "http.timeouts", c.HTTP.Timeouts, map[string]time.Duration{
		"upload":  5 * time.Minute,
		"webhook": 3 * time.Second,
	})
	equalSlice(t, "http.origins", c.HTTP.Origins, []string{
		"https://example.test",
		"https://admin.example.test",
	})

	equal(t, "log.level", c.Log.Level, slog.LevelWarn)
	equal(t, "log.format", c.Log.Format, "json")
	equal(t, "log.sample", c.Log.Sample, 0.25)
	equalMap(t, "log.fields", c.Log.Fields, map[string]string{"service": "web", "region": "eu"})

	equal(t, "database.dsn", c.Database.DSN, "postgres://localhost/app")
	equal(t, "database.max_open_conns", c.Database.MaxOpenConns, 10)
	equal(t, "database.max_idle_conns", c.Database.MaxIdleConns, 5)
	equal(t, "database.conn_max_lifetime", c.Database.ConnMaxLifetime, 2*time.Hour)
	equal(t, "database.slow_query", c.Database.SlowQuery, 200*time.Millisecond)
	equalSlice(t, "database.replicas", c.Database.Replicas, []string{"replica-1", "replica-2"})
	equal(t, "database.migrated", c.Database.Migrated.Format(time.RFC3339), "2026-02-03T04:05:06Z")

	equal(t, "cache.driver", c.Cache.Driver, "memory")
	equal(t, "cache.prefix", c.Cache.Prefix, "blog:")
	equal(t, "cache.ttl", c.Cache.TTL, 5*time.Minute)
	equal(t, "cache.max_bytes", c.Cache.MaxBytes, Size(128<<20))
	equal(t, "cache.shards", c.Cache.Shards, uint8(32))

	equal(t, "queue.driver", c.Queue.Driver, "redis")
	equal(t, "queue.workers", c.Queue.Workers, uint(8))
	equal(t, "queue.retries", c.Queue.Retries, int8(5))
	equal(t, "queue.backoff", c.Queue.Backoff, 10*time.Second)
	equalSlice(t, "queue.queues", c.Queue.Queues, []string{"high", "low"})
	equalMap(t, "queue.weights", c.Queue.Weights, map[string]int{"high": 3, "low": 1})

	equal(t, "mail.from", c.Mail.From, "noreply@example.test")
	equal(t, "mail.host", c.Mail.Host, "localhost")
	equal(t, "mail.port", c.Mail.Port, uint16(587))
	equal(t, "mail.password", c.Mail.Password, "hunter2")
	equal(t, "mail.timeout", c.Mail.Timeout, 10*time.Second)

	equal(t, "billing.key", string(c.Billing.Key), "pay")
	equal(t, "billing.currency", c.Billing.Currency, "gbp")
	equal(t, "billing.minimum", c.Billing.Minimum, Money(125))
	equal(t, "billing.rate", c.Billing.Rate, float32(0.015))
	equal(t, "billing.enabled", c.Billing.Enabled, true)
}

// TestDefaults is the same struct with nothing to read, which is every default
// in the file going through the parser it was written for.
func TestDefaults(t *testing.T) {
	c, err := LoadConfig(config.Sources{Env: "local"})
	if err != nil {
		t.Fatal(err)
	}
	equal(t, "app.name", c.App.Name, "blog")
	equal(t, "app.debug", c.App.Debug, false)
	equal(t, "http.addr", c.HTTP.Addr, netip.MustParseAddrPort("0.0.0.0:8080"))
	equal(t, "log.level", c.Log.Level, slog.LevelInfo)
	equal(t, "cache.max_bytes", c.Cache.MaxBytes, Size(64<<20))
	equal(t, "billing.minimum", c.Billing.Minimum, Money(50))
	equal(t, "database.migrated", c.Database.Migrated.Format(time.RFC3339), "2026-01-01T00:00:00Z")

	// A field with no default and nothing to read is left alone.
	equal(t, "app.internal", c.App.Internal, "")
	if c.HTTP.Timeouts != nil {
		t.Errorf("http.timeouts = %v, want nil", c.HTTP.Timeouts)
	}
	if len(c.Database.Replicas) != 0 {
		t.Errorf("database.replicas = %v, want nothing", c.Database.Replicas)
	}
}

// TestUnknownKey is the check the milestone asks for: a setting nothing reads
// has to say which one it is, which file it is in, and which line.
func TestUnknownKey(t *testing.T) {
	_, err := LoadConfig(config.Sources{Env: "local", Files: []string{"config/broken.toml"}})
	if err == nil {
		t.Fatal("a file with an unknown setting in it loaded without complaint")
	}
	msg := err.Error()
	for _, want := range []string{"database.max_open_conn", "config/broken.toml", ":6:"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the error does not mention %q:\n%s", want, msg)
		}
	}
	// The closest setting that does exist is worth saying, since the whole
	// point is that this is nearly always a typo.
	if !strings.Contains(msg, "max_open_conns") {
		t.Errorf("the error does not suggest the setting that was meant:\n%s", msg)
	}
}

func TestBadValue(t *testing.T) {
	src := sources(t)
	src.Override = map[string]string{
		"http.addr":     "not an address",
		"queue.workers": "-1",
		"log.level":     "loud",
	}
	_, err := LoadConfig(src)
	if err == nil {
		t.Fatal("three settings that will not parse loaded without complaint")
	}
	// All three are reported, and each is named the way the struct names it,
	// which is where the reader has to go and change something.
	msg := err.Error()
	for _, want := range []string{"HTTP.Addr", "Queue.Workers", "Log.Level"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the error does not mention %q, so it stopped at the first one:\n%s", want, msg)
		}
	}
}

func TestRedact(t *testing.T) {
	c := load(t)
	r := c.Redact()

	equal(t, "database.dsn", r.Database.DSN, config.Redacted)
	equal(t, "mail.password", r.Mail.Password, config.Redacted)
	if r.App.Key != nil {
		t.Errorf("app.key = %v, want nothing", r.App.Key)
	}
	if r.Billing.Key != nil {
		t.Errorf("billing.key = %v, want nothing", r.Billing.Key)
	}

	// Everything that is not a secret comes through unchanged.
	equal(t, "app.name", r.App.Name, c.App.Name)
	equal(t, "queue.driver", r.Queue.Driver, c.Queue.Driver)
	equalSlice(t, "http.origins", r.HTTP.Origins, c.HTTP.Origins)
	equalMap(t, "log.fields", r.Log.Fields, c.Log.Fields)

	// The original is untouched, and the copy does not share anything with it.
	equal(t, "database.dsn", c.Database.DSN, "postgres://localhost/app")
	equal(t, "app.key", string(c.App.Key), "secret")
	r.HTTP.Origins[0] = "https://elsewhere.test"
	r.Log.Fields["service"] = "worker"
	equal(t, "http.origins", c.HTTP.Origins[0], "https://example.test")
	equal(t, "log.fields", c.Log.Fields["service"], "web")
}

func TestDescribe(t *testing.T) {
	c := load(t)
	docs := c.Describe()
	if len(docs) != len(configFields) {
		t.Fatalf("got %d settings, want %d", len(docs), len(configFields))
	}

	byPath := map[string]config.FieldDoc{}
	for _, d := range docs {
		byPath[d.Path] = d
	}
	equal(t, "app.name", byPath["app.name"].Value, "example")
	equal(t, "http.read_timeout", byPath["http.read_timeout"].Value, "45s")
	equal(t, "http.trusted_proxies", byPath["http.trusted_proxies"].Value, "[10.0.0.0/8, 192.168.0.0/16]")
	equal(t, "log.fields", byPath["log.fields"].Value, "{region = eu, service = web}")
	equal(t, "cache.max_bytes", byPath["cache.max_bytes"].Value, "128M")
	equal(t, "billing.minimum", byPath["billing.minimum"].Value, "1.25")
	equal(t, "database.migrated", byPath["database.migrated"].Value, "2026-02-03T04:05:06Z")

	// A secret that is set says so and does not say what it is.
	equal(t, "database.dsn", byPath["database.dsn"].Value, config.Redacted)
	equal(t, "mail.password", byPath["mail.password"].Value, config.Redacted)
	equal(t, "app.internal", byPath["app.internal"].Value, "")

	// The explanation comes from the doc comment on the field.
	if got := byPath["cache.ttl"].Doc; !strings.HasPrefix(got, "TTL is how long") {
		t.Errorf("cache.ttl doc = %q", got)
	}
	if got := byPath["queue.workers"].Type; got != "uint" {
		t.Errorf("queue.workers type = %q, want uint", got)
	}
}

func TestDiff(t *testing.T) {
	from := load(t)
	to := load(t)
	to.App.Name = "other"
	to.Queue.Workers = 16
	to.Mail.Password = "hunter3"

	changes := from.Diff(to)
	if len(changes) != 3 {
		t.Fatalf("got %d changes, want 3:\n%v", len(changes), changes)
	}
	want := []string{
		"app.name: example -> other",
		"queue.workers: 8 -> 16",
		"mail.password: " + config.Redacted + " -> " + config.Redacted,
	}
	for i, c := range changes {
		if c.String() != want[i] {
			t.Errorf("change %d = %q, want %q", i, c.String(), want[i])
		}
	}

	if len(from.Diff(from)) != 0 {
		t.Error("a configuration differs from itself")
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	src := config.Sources{
		Env:   "local",
		Files: []string{"config/app.toml", "config/local.toml"},
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := LoadConfig(src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDescribe(b *testing.B) {
	c, err := LoadConfig(config.Sources{Env: "local"})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		c.Describe()
	}
}

func BenchmarkRedact(b *testing.B) {
	c, err := LoadConfig(config.Sources{Env: "local"})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		c.Redact()
	}
}

func equal[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func equalSlice[T comparable](t *testing.T, name string, got, want []T) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s = %v, want %v", name, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s = %v, want %v", name, got, want)
			return
		}
	}
}

func equalMap[T comparable](t *testing.T, name string, got, want map[string]T) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s = %v, want %v", name, got, want)
		return
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", name, got, want)
			return
		}
	}
}
