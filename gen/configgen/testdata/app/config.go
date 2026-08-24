// Package app is a configuration struct for the generator's tests. It holds
// one field of every type the generator claims to read, so that regenerating
// it and building the result is a check on all of them at once.
package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/config"
)

// Env is the name of a deployment, as a named string, so that a defined type
// over a basic one is covered.
type Env string

// Size is a byte count that reads itself, which is the way in for a type the
// config package has never heard of.
type Size int64

// ParseConfig reads a size written as a number of bytes, or with a K, M or G
// after it.
func (s *Size) ParseConfig(v config.Value) error {
	text, err := v.Str()
	if err != nil {
		return err
	}
	mult := int64(1)
	if n := len(text); n > 0 {
		switch text[n-1] {
		case 'K':
			mult, text = 1<<10, text[:n-1]
		case 'M':
			mult, text = 1<<20, text[:n-1]
		case 'G':
			mult, text = 1<<30, text[:n-1]
		}
	}
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return fmt.Errorf("want a size such as 64M, got %q", v.Display())
	}
	*s = Size(n * mult)
	return nil
}

// String writes a size back out the way it was written.
func (s Size) String() string {
	switch {
	case s != 0 && s%(1<<30) == 0:
		return strconv.FormatInt(int64(s)/(1<<30), 10) + "G"
	case s != 0 && s%(1<<20) == 0:
		return strconv.FormatInt(int64(s)/(1<<20), 10) + "M"
	case s != 0 && s%(1<<10) == 0:
		return strconv.FormatInt(int64(s)/(1<<10), 10) + "K"
	}
	return strconv.FormatInt(int64(s), 10)
}

// Money is an amount in the smallest unit of a currency, written the way
// people write it, so 4.99 rather than 499.
type Money int64

// UnmarshalText makes Money an encoding.TextUnmarshaler, which is the other
// way a type the config package does not know about gets read.
func (m *Money) UnmarshalText(text []byte) error {
	whole, frac, _ := strings.Cut(string(text), ".")
	for len(frac) < 2 {
		frac += "0"
	}
	if len(frac) > 2 {
		return errors.New("want an amount with at most two decimal places")
	}
	n, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil {
		return errors.New("want an amount such as 4.99")
	}
	*m = Money(n)
	return nil
}

// String writes an amount back out with its decimal point.
func (m Money) String() string {
	return strconv.FormatInt(int64(m)/100, 10) + "." + strconv.FormatInt(int64(m)%100+100, 10)[1:]
}

//mizu:config
type Config struct {
	App struct {
		// Name is what the application calls itself in logs and mail.
		Name string `default:"blog"`

		// Env is which deployment this is.
		Env Env `default:"local"`

		// Debug turns on the developer error page.
		Debug bool `default:"false"`

		// Key signs cookies and sessions.
		Key []byte `env:"APP_KEY" secret:"true"`

		// Locale is the language to answer in.
		Locale string `toml:"lang" default:"en"`

		// Internal is set in code and never read from the environment.
		Internal string `env:"-"`
	}

	HTTP struct {
		// Addr is the address to listen on.
		Addr netip.AddrPort `default:"0.0.0.0:8080"`

		// Host is the name the application is reached by.
		Host string `default:"localhost"`

		// ReadTimeout is how long a request has to arrive.
		ReadTimeout time.Duration `default:"30s"`

		// WriteTimeout is how long a response has to be written.
		WriteTimeout time.Duration `default:"30s"`

		// MaxHeaderBytes is the largest set of headers accepted.
		MaxHeaderBytes int `default:"1048576"`

		// TrustedProxies are the networks whose forwarded headers are believed.
		TrustedProxies []netip.Prefix `default:"10.0.0.0/8,127.0.0.1/32"`

		// BindTo is a single address, as opposed to an address and a port.
		BindTo netip.Addr `default:"127.0.0.1"`

		// Timeouts are limits for individual routes, by name.
		Timeouts map[string]time.Duration

		// Origins are the sites allowed to call this one from a browser.
		Origins []string `default:"http://localhost:5173"`
	}

	Log struct {
		// Level is the lowest severity written out.
		Level slog.Level `default:"info"`

		// Format is either text or json.
		Format string `default:"text"`

		// Sample is the share of debug lines kept.
		Sample float64 `default:"1.0"`

		// Fields are extra values attached to every line.
		Fields map[string]string
	}

	Database struct {
		// DSN is how to reach the database.
		DSN string `env:"DATABASE_URL" secret:"true" default:"sqlite:app.db"`

		// MaxOpenConns is the most connections opened at once.
		MaxOpenConns int `default:"25"`

		// MaxIdleConns is how many are kept when nothing is happening.
		MaxIdleConns int `default:"5"`

		// ConnMaxLifetime is how long a connection is reused for.
		ConnMaxLifetime time.Duration `default:"1h"`

		// SlowQuery is the duration above which a query is written to the log.
		SlowQuery time.Duration `default:"200ms"`

		// Replicas are read only copies to send selects to.
		Replicas []string

		// Migrated is when the schema was last brought up to date.
		Migrated time.Time `default:"2026-01-01T00:00:00Z"`
	}

	Cache struct {
		// Driver is where cached values are kept.
		Driver string `default:"memory"`

		// Prefix goes in front of every key.
		Prefix string `default:"blog:"`

		// TTL is how long a value lives without being asked for.
		TTL time.Duration `default:"5m"`

		// MaxBytes is how much memory the cache may hold.
		MaxBytes Size `default:"64M"`

		// Shards is how many independent maps the cache is split into.
		Shards uint8 `default:"16"`
	}

	Queue struct {
		// Driver is where jobs are kept.
		Driver string `default:"database"`

		// Workers is how many jobs run at once.
		Workers uint `default:"4"`

		// Retries is how many times a failed job is tried again.
		Retries int8 `default:"3"`

		// Backoff is how long to wait before trying again.
		Backoff time.Duration `default:"10s"`

		// Queues are the names to take work from, in order.
		Queues []string `default:"high,default,low"`

		// Weights are how much of the worker pool each queue gets.
		Weights map[string]int
	}

	Mail struct {
		// From is the address messages are sent as.
		From string `default:"hello@example.com"`

		// Host is the mail server.
		Host string `default:"localhost"`

		// Port is the port on it.
		Port uint16 `default:"1025"`

		// Password authenticates to the mail server.
		Password string `env:"MAIL_PASSWORD" secret:"true"`

		// Timeout is how long sending one message may take.
		Timeout time.Duration `default:"10s"`
	}

	Billing struct {
		// Key authenticates to the payment provider.
		Key []byte `secret:"true"`

		// Currency is what prices are in.
		Currency string `default:"usd"`

		// Minimum is the smallest charge accepted.
		Minimum Money `default:"0.50"`

		// Rate is how much of a charge is kept as commission.
		Rate float32 `default:"0.029"`

		// Enabled turns charging on.
		Enabled bool `default:"false"`
	}
}
