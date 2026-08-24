package config

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestDiscover(t *testing.T) {
	join := func(parts ...string) string { return filepath.Join(parts...) }

	tests := []struct {
		name    string
		environ []string
		args    []string
		wantEnv string
		files   []string
		dotenv  []string
	}{
		{
			name:    "nothing says which environment",
			wantEnv: "local",
			files:   []string{join("app", "config", "local.toml")},
			dotenv: []string{
				join("app", ".env"),
				join("app", ".env.local"),
			},
		},
		{
			name:    "the environment says",
			environ: []string{"MIZU_ENV=production"},
			wantEnv: "production",
			files: []string{
				join("app", "config", "production.toml"),
				join("app", "config", "local.toml"),
			},
		},
		{
			name:    "the command line beats the environment",
			environ: []string{"MIZU_ENV=production"},
			args:    []string{"serve", "--config.env=staging"},
			wantEnv: "staging",
			files: []string{
				join("app", "config", "staging.toml"),
				join("app", "config", "local.toml"),
			},
		},
		{
			name:    "testing reads the dotenv files too",
			environ: []string{"MIZU_ENV=testing"},
			wantEnv: "testing",
			files: []string{
				join("app", "config", "testing.toml"),
				join("app", "config", "local.toml"),
			},
			dotenv: []string{
				join("app", ".env"),
				join("app", ".env.testing"),
				join("app", ".env.testing.local"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Discover("app", tt.environ, tt.args)
			if s.Env != tt.wantEnv {
				t.Errorf("Env is %q, want %q", s.Env, tt.wantEnv)
			}
			if !slices.Equal(s.Files, tt.files) {
				t.Errorf("Files are %q, want %q", s.Files, tt.files)
			}
			if !slices.Equal(s.DotEnv, tt.dotenv) {
				t.Errorf("DotEnv is %q, want %q", s.DotEnv, tt.dotenv)
			}
		})
	}
}

func TestEnvName(t *testing.T) {
	tests := []struct {
		name    string
		environ []string
		args    []string
		want    string
	}{
		{"nothing", nil, nil, "local"},
		{"the variable", []string{"MIZU_ENV=production"}, nil, "production"},
		{"an empty variable", []string{"MIZU_ENV="}, nil, "local"},
		{"another variable", []string{"HOME=/root"}, nil, "local"},
		{"a flag", nil, []string{"--config.env=staging"}, "staging"},
		{"one dash", nil, []string{"-config.env=staging"}, "staging"},
		{"an empty flag", nil, []string{"--config.env="}, "local"},
		{"a flag with no value", nil, []string{"--config.env"}, "local"},
		{"another flag", nil, []string{"--config.app.name=x"}, "local"},
		{"the last flag wins", nil, []string{"--config.env=a", "--config.env=b"}, "b"},
		{"something else last", nil, []string{"--config.env=staging", "serve"}, "staging"},
		{"the last variable wins", []string{"MIZU_ENV=a", "MIZU_ENV=b"}, nil, "b"},
		{"a value with an equals in it", []string{"OTHER=a=b"}, nil, "local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EnvName(tt.environ, tt.args); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigFlag(t *testing.T) {
	tests := []struct {
		arg  string
		rest string
		ok   bool
	}{
		{"--config.a.b=1", "a.b=1", true},
		{"-config.a.b=1", "a.b=1", true},
		{"--config.a", "a", true},
		{"--config.", "", false},
		{"--config", "", false},
		{"--other=1", "", false},
		{"serve", "", false},
		{"", "", false},
		{"-", "", false},
		{"--", "", false},
	}

	for _, tt := range tests {
		rest, ok := configFlag(tt.arg)
		if rest != tt.rest || ok != tt.ok {
			t.Errorf("configFlag(%q) is %q, %v, want %q, %v", tt.arg, rest, ok, tt.rest, tt.ok)
		}
	}
}

func TestFlagKey(t *testing.T) {
	tests := []struct{ in, want string }{
		{"database.max_open_conns", "database.max_open_conns"},
		{"database.max-open-conns", "database.max_open_conns"},
		{"Database.Max-Open-Conns", "database.max_open_conns"},
		{"env", "env"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := flagKey(tt.in); got != tt.want {
			t.Errorf("flagKey(%q) is %q, want %q", tt.in, got, tt.want)
		}
	}
}
