package crawler

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"simple", "https://example.com/page", "https://example.com/page", false},
		{"trailing slash", "https://example.com/page/", "https://example.com/page", false},
		{"root path", "https://example.com/", "https://example.com/", false},
		{"no path", "https://example.com", "https://example.com/", false},
		{"uppercase host", "https://EXAMPLE.COM/Page", "https://example.com/Page", false},
		{"uppercase scheme", "HTTPS://example.com/page", "https://example.com/page", false},
		{"default port http", "http://example.com:80/page", "http://example.com/page", false},
		{"default port https", "https://example.com:443/page", "https://example.com/page", false},
		{"non-default port", "https://example.com:8080/page", "https://example.com:8080/page", false},
		{"fragment removed", "https://example.com/page#section", "https://example.com/page", false},
		{"query sorted", "https://example.com/page?z=1&a=2", "https://example.com/page?a=2&z=1", false},
		{"dot segments", "https://example.com/a/b/../c", "https://example.com/a/c", false},
		{"ftp rejected", "ftp://example.com/file", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsSameScope(t *testing.T) {
	tests := []struct {
		name   string
		start  string
		target string
		scope  ScopePolicy
		want   bool
	}{
		{"same domain yes", "https://example.com/a", "https://example.com/b", ScopeSameDomain, true},
		{"same domain no", "https://example.com/a", "https://other.com/b", ScopeSameDomain, false},
		{"same host subdomain yes", "https://example.com/a", "https://sub.example.com/b", ScopeSameHost, true},
		{"same host exact yes", "https://example.com/a", "https://example.com/b", ScopeSameHost, true},
		{"same host no", "https://example.com/a", "https://other.com/b", ScopeSameHost, false},
		{"subpath yes", "https://example.com/docs/", "https://example.com/docs/page", ScopeSubpath, true},
		{"subpath no", "https://example.com/docs/", "https://example.com/blog/page", ScopeSubpath, false},
		{"subpath different domain", "https://example.com/docs/", "https://other.com/docs/page", ScopeSubpath, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSameScope(tt.start, tt.target, tt.scope)
			if got != tt.want {
				t.Errorf("IsSameScope(%q, %q, %v) = %v, want %v", tt.start, tt.target, tt.scope, got, tt.want)
			}
		})
	}
}

func TestIsValidCrawlURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com/page", true},
		{"http://example.com/", true},
		{"https://example.com/page.html", true},
		{"https://example.com/image.jpg", false},
		{"https://example.com/style.css", false},
		{"ftp://example.com/file", false},
		{"mailto:test@example.com", false},
		{"javascript:void(0)", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := IsValidCrawlURL(tt.url)
			if got != tt.want {
				t.Errorf("IsValidCrawlURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestResolveURL(t *testing.T) {
	base := "https://example.com/docs/guide/"
	tests := []struct {
		ref  string
		want string
	}{
		{"/about", "https://example.com/about"},
		{"page2", "https://example.com/docs/guide/page2"},
		{"../other", "https://example.com/docs/other"},
		{"https://other.com/page", "https://other.com/page"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got, err := ResolveURL(base, tt.ref)
			if err != nil {
				t.Fatalf("ResolveURL(%q, %q) error = %v", base, tt.ref, err)
			}
			if got != tt.want {
				t.Errorf("ResolveURL(%q, %q) = %q, want %q", base, tt.ref, got, tt.want)
			}
		})
	}
}

func TestDomainOf(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/page", "example.com"},
		{"https://sub.example.com:8080/page", "sub.example.com"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := DomainOf(tt.url)
			if got != tt.want {
				t.Errorf("DomainOf(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
