package crawler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSitemapURLSet(t *testing.T) {
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/page1</loc>
    <lastmod>2024-01-15</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.8</priority>
  </url>
  <url>
    <loc>https://example.com/page2</loc>
    <lastmod>2024-01-10T12:00:00Z</lastmod>
    <priority>0.5</priority>
  </url>
  <url>
    <loc>https://example.com/page3</loc>
  </url>
</urlset>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(sitemapXML))
	}))
	defer srv.Close()

	urls, err := FetchSitemap(srv.Client(), srv.URL+"/sitemap.xml", 0)
	if err != nil {
		t.Fatalf("FetchSitemap error: %v", err)
	}
	if len(urls) != 3 {
		t.Fatalf("got %d URLs, want 3", len(urls))
	}
	if urls[0].URL != "https://example.com/page1" {
		t.Errorf("urls[0].URL = %q, want %q", urls[0].URL, "https://example.com/page1")
	}
	if urls[0].ChangeFreq != "weekly" {
		t.Errorf("urls[0].ChangeFreq = %q, want %q", urls[0].ChangeFreq, "weekly")
	}
	if urls[0].Priority != 0.8 {
		t.Errorf("urls[0].Priority = %f, want 0.8", urls[0].Priority)
	}
	if urls[0].LastMod.IsZero() {
		t.Error("urls[0].LastMod should not be zero")
	}
}

func TestFetchSitemapIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if r.URL.Path == "/sitemap_index.xml" {
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>` + "http://" + r.Host + `/sitemap1.xml</loc>
  </sitemap>
  <sitemap>
    <loc>` + "http://" + r.Host + `/sitemap2.xml</loc>
  </sitemap>
</sitemapindex>`))
		} else if r.URL.Path == "/sitemap1.xml" {
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/a</loc></url>
  <url><loc>https://example.com/b</loc></url>
</urlset>`))
		} else if r.URL.Path == "/sitemap2.xml" {
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/c</loc></url>
</urlset>`))
		}
	}))
	defer srv.Close()

	urls, err := FetchSitemap(srv.Client(), srv.URL+"/sitemap_index.xml", 0)
	if err != nil {
		t.Fatalf("FetchSitemap error: %v", err)
	}
	if len(urls) != 3 {
		t.Fatalf("got %d URLs, want 3", len(urls))
	}
}

func TestFetchSitemapMaxURLs(t *testing.T) {
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/1</loc></url>
  <url><loc>https://example.com/2</loc></url>
  <url><loc>https://example.com/3</loc></url>
  <url><loc>https://example.com/4</loc></url>
  <url><loc>https://example.com/5</loc></url>
</urlset>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sitemapXML))
	}))
	defer srv.Close()

	urls, err := FetchSitemap(srv.Client(), srv.URL+"/sitemap.xml", 3)
	if err != nil {
		t.Fatalf("FetchSitemap error: %v", err)
	}
	if len(urls) != 3 {
		t.Fatalf("got %d URLs, want 3", len(urls))
	}
}

func TestFetchSitemap404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	_, err := FetchSitemap(srv.Client(), srv.URL+"/sitemap.xml", 0)
	if err == nil {
		t.Error("expected error for 404 sitemap")
	}
}
