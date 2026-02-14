package dcrawler

import (
	"net/url"
	"testing"
)

func mustParseURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

func TestAnchorTextCapture(t *testing.T) {
	html := `<html><body>
		<a href="/page1">First Article</a>
		<a href="/page2">Second <em>Article</em></a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(meta.Links))
	}
	if meta.Links[0].AnchorText != "First Article" {
		t.Errorf("link[0] anchor text = %q, want %q", meta.Links[0].AnchorText, "First Article")
	}
	if meta.Links[1].AnchorText != "Second Article" {
		t.Errorf("link[1] anchor text = %q, want %q", meta.Links[1].AnchorText, "Second Article")
	}
}

func TestAnchorTitleFallback(t *testing.T) {
	html := `<html><body>
		<a href="/page" title="Full Title"><img src="pic.jpg"></a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(meta.Links))
	}
	if meta.Links[0].AnchorText != "Full Title" {
		t.Errorf("anchor text = %q, want %q (title fallback)", meta.Links[0].AnchorText, "Full Title")
	}
}

func TestAnchorTextOverridesTitle(t *testing.T) {
	html := `<html><body>
		<a href="/page" title="Title Attr">Visible Text</a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(meta.Links))
	}
	// Text content should take priority over title attribute
	if meta.Links[0].AnchorText != "Visible Text" {
		t.Errorf("anchor text = %q, want %q", meta.Links[0].AnchorText, "Visible Text")
	}
}

func TestBaseHref(t *testing.T) {
	html := `<html><head>
		<base href="https://cdn.example.com/pages/">
	</head><body>
		<a href="article.html">Article</a>
		<a href="/absolute">Absolute</a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(meta.Links))
	}
	// Relative URL resolved against <base href>
	if meta.Links[0].TargetURL != "https://cdn.example.com/pages/article.html" {
		t.Errorf("link[0] = %q, want resolved against base", meta.Links[0].TargetURL)
	}
	// Absolute path resolved against <base href> host
	if meta.Links[1].TargetURL != "https://cdn.example.com/absolute" {
		t.Errorf("link[1] = %q, want resolved against base host", meta.Links[1].TargetURL)
	}
}

func TestAreaHref(t *testing.T) {
	html := `<html><body>
		<map name="nav">
			<area href="/section1" alt="Section One">
			<area href="/section2" alt="Section Two" rel="nofollow">
			<area href="https://external.com/page" alt="External">
		</map>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(meta.Links))
	}
	if meta.Links[0].TargetURL != "https://example.com/section1" {
		t.Errorf("area[0] url = %q", meta.Links[0].TargetURL)
	}
	if meta.Links[0].AnchorText != "Section One" {
		t.Errorf("area[0] text = %q, want %q", meta.Links[0].AnchorText, "Section One")
	}
	if !meta.Links[0].IsInternal {
		t.Error("area[0] should be internal")
	}
	if meta.Links[1].Rel != "nofollow" {
		t.Errorf("area[1] rel = %q, want nofollow", meta.Links[1].Rel)
	}
	if meta.Links[2].IsInternal {
		t.Error("area[2] should be external")
	}
}

func TestLinkRelNextPrevAlternate(t *testing.T) {
	html := `<html><head>
		<link rel="canonical" href="https://example.com/page">
		<link rel="next" href="/page?p=2">
		<link rel="prev" href="/page?p=0">
		<link rel="alternate" href="/page/vi" hreflang="vi">
		<link rel="stylesheet" href="/style.css">
	</head><body></body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/page"), "example.com", false)

	if meta.Canonical != "https://example.com/page" {
		t.Errorf("canonical = %q", meta.Canonical)
	}

	// Should have 3 links: next, prev, alternate (not stylesheet, not canonical)
	if len(meta.Links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(meta.Links))
	}

	rels := map[string]string{}
	for _, l := range meta.Links {
		rels[l.Rel] = l.TargetURL
	}
	if _, ok := rels["next"]; !ok {
		t.Error("missing next link")
	}
	if _, ok := rels["prev"]; !ok {
		t.Error("missing prev link")
	}
	if _, ok := rels["alternate"]; !ok {
		t.Error("missing alternate link")
	}
}

func TestMetaRefresh(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		wantURL string
	}{
		{
			name:    "standard",
			html:    `<html><head><meta http-equiv="refresh" content="5; url=https://example.com/new"></head></html>`,
			wantURL: "https://example.com/new",
		},
		{
			name:    "quoted",
			html:    `<html><head><meta http-equiv="refresh" content="0;URL='/redirect'"></head></html>`,
			wantURL: "https://example.com/redirect",
		},
		{
			name:    "double-quoted",
			html:    `<html><head><meta http-equiv="Refresh" content='3; url="https://example.com/page"'></head></html>`,
			wantURL: "https://example.com/page",
		},
		{
			name:    "no-url",
			html:    `<html><head><meta http-equiv="refresh" content="30"></head></html>`,
			wantURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := ExtractLinksAndMeta([]byte(tt.html), mustParseURL("https://example.com/"), "example.com", false)

			if tt.wantURL == "" {
				if len(meta.Links) != 0 {
					t.Errorf("expected no links, got %d", len(meta.Links))
				}
				return
			}
			if len(meta.Links) != 1 {
				t.Fatalf("expected 1 link, got %d", len(meta.Links))
			}
			if meta.Links[0].TargetURL != tt.wantURL {
				t.Errorf("url = %q, want %q", meta.Links[0].TargetURL, tt.wantURL)
			}
			if meta.Links[0].Rel != "meta-refresh" {
				t.Errorf("rel = %q, want meta-refresh", meta.Links[0].Rel)
			}
		})
	}
}

func TestIframeSrc(t *testing.T) {
	html := `<html><body>
		<iframe src="/embed/video"></iframe>
		<iframe src="https://youtube.com/embed/abc"></iframe>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	// Only internal iframe should be extracted
	if len(meta.Links) != 1 {
		t.Fatalf("expected 1 link (internal iframe only), got %d", len(meta.Links))
	}
	if meta.Links[0].TargetURL != "https://example.com/embed/video" {
		t.Errorf("iframe url = %q", meta.Links[0].TargetURL)
	}
	if meta.Links[0].Rel != "iframe" {
		t.Errorf("iframe rel = %q, want iframe", meta.Links[0].Rel)
	}
	if !meta.Links[0].IsInternal {
		t.Error("iframe should be internal")
	}
}

func TestNestedHTMLInAnchors(t *testing.T) {
	html := `<html><body>
		<a href="/article">
			<span class="title">Breaking News:</span>
			<span class="desc">Something happened</span>
		</a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(meta.Links))
	}
	want := "Breaking News: Something happened"
	if meta.Links[0].AnchorText != want {
		t.Errorf("anchor text = %q, want %q", meta.Links[0].AnchorText, want)
	}
}

func TestConsecutiveAnchors(t *testing.T) {
	html := `<html><body>
		<a href="/p1">First</a>
		<a href="/p2">Second</a>
		<a href="/p3">Third</a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(meta.Links))
	}
	texts := []string{"First", "Second", "Third"}
	for i, want := range texts {
		if meta.Links[i].AnchorText != want {
			t.Errorf("link[%d] text = %q, want %q", i, meta.Links[i].AnchorText, want)
		}
	}
}

func TestUnclosedAnchor(t *testing.T) {
	// Malformed HTML: anchor never closed
	html := `<html><body>
		<a href="/page">Unclosed link text
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(meta.Links))
	}
	if meta.Links[0].AnchorText == "" {
		t.Error("unclosed anchor should still capture text")
	}
}

func TestJavascriptAndMailtoFiltered(t *testing.T) {
	html := `<html><body>
		<a href="javascript:void(0)">JS</a>
		<a href="mailto:user@example.com">Email</a>
		<a href="tel:+1234567890">Phone</a>
		<a href="/real-page">Real</a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(meta.Links))
	}
	if meta.Links[0].TargetURL != "https://example.com/real-page" {
		t.Errorf("url = %q", meta.Links[0].TargetURL)
	}
}

func TestInternalExternalDetection(t *testing.T) {
	html := `<html><body>
		<a href="/local">Local</a>
		<a href="https://example.com/same">Same</a>
		<a href="https://www.example.com/www">WWW</a>
		<a href="https://sub.example.com/sub">Sub</a>
		<a href="https://other.com/ext">External</a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 5 {
		t.Fatalf("expected 5 links, got %d", len(meta.Links))
	}

	expects := []bool{true, true, true, true, false}
	for i, want := range expects {
		if meta.Links[i].IsInternal != want {
			t.Errorf("link[%d] %q internal=%v, want %v", i, meta.Links[i].TargetURL, meta.Links[i].IsInternal, want)
		}
	}
}

func TestTitleAndDescription(t *testing.T) {
	html := `<html lang="en"><head>
		<title>Page Title</title>
		<meta name="description" content="Page description here">
	</head><body></body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if meta.Title != "Page Title" {
		t.Errorf("title = %q", meta.Title)
	}
	if meta.Description != "Page description here" {
		t.Errorf("description = %q", meta.Description)
	}
	if meta.Language != "en" {
		t.Errorf("language = %q", meta.Language)
	}
}

func TestParseMetaRefreshURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"5; url=https://example.com/page", "https://example.com/page"},
		{"0;URL='https://example.com/page'", "https://example.com/page"},
		{`3; url="https://example.com/page"`, "https://example.com/page"},
		{"30", ""},
		{"", ""},
		{"5; url=", ""},
	}
	for _, tt := range tests {
		got := parseMetaRefreshURL(tt.input)
		if got != tt.want {
			t.Errorf("parseMetaRefreshURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestKenh14StyleHTML(t *testing.T) {
	// Simulates kenh14.vn article structure
	html := `<html lang="vi">
<head>
	<title>Tin tức giải trí - Kenh14.vn</title>
	<meta name="description" content="Kênh tin tức giải trí">
	<link rel="canonical" href="https://kenh14.vn/star.chn">
</head>
<body>
	<a href="/" title="Trang chủ">TRANG CHỦ</a>
	<a href="/star.chn" title="Star">Star</a>
	<a href="/cine.chn" title="Ciné">Ciné</a>
	<a href="/tron-ven-truong-doan-xuc-dong-215260208170836147.chn">
		<span class="knswli-title">Trọn vẹn trường đoạn xúc động</span>
	</a>
	<a href="https://kenh14.vn/wechoice-awards-2025.html">WeChoice Awards 2025</a>
	<a href="https://www.facebook.com/K14vn">Fanpage</a>
</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://kenh14.vn/"), "kenh14.vn", false)

	if meta.Title != "Tin tức giải trí - Kenh14.vn" {
		t.Errorf("title = %q", meta.Title)
	}
	if meta.Language != "vi" {
		t.Errorf("language = %q", meta.Language)
	}
	if meta.Canonical != "https://kenh14.vn/star.chn" {
		t.Errorf("canonical = %q", meta.Canonical)
	}

	if len(meta.Links) != 6 {
		t.Fatalf("expected 6 links, got %d", len(meta.Links))
	}

	// Check anchor text for article with nested span
	articleLink := meta.Links[3]
	if articleLink.AnchorText != "Trọn vẹn trường đoạn xúc động" {
		t.Errorf("article anchor text = %q, want nested span text", articleLink.AnchorText)
	}

	// Homepage link uses title fallback? No — "TRANG CHỦ" is text content
	if meta.Links[0].AnchorText != "TRANG CHỦ" {
		t.Errorf("home anchor text = %q", meta.Links[0].AnchorText)
	}

	// Facebook is external
	fbLink := meta.Links[5]
	if fbLink.IsInternal {
		t.Error("facebook link should be external")
	}

	// wechoice link is internal (kenh14.vn domain)
	wcLink := meta.Links[4]
	if !wcLink.IsInternal {
		t.Error("wechoice link should be internal")
	}
}

func TestRelativeURLResolution(t *testing.T) {
	html := `<html><body>
		<a href="sibling.html">Sibling</a>
		<a href="../parent.html">Parent</a>
		<a href="//cdn.example.com/page">Protocol-relative</a>
		<a href="?q=search">Query-only</a>
		<a href="#section">Fragment-only</a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/dir/page.html"), "example.com", false)

	expects := []string{
		"https://example.com/dir/sibling.html",
		"https://example.com/parent.html",
		"https://cdn.example.com/page",
		"https://example.com/dir/page.html?q=search",
		"https://example.com/dir/page.html#section",
	}

	if len(meta.Links) != len(expects) {
		t.Fatalf("expected %d links, got %d", len(expects), len(meta.Links))
	}
	for i, want := range expects {
		if meta.Links[i].TargetURL != want {
			t.Errorf("link[%d] = %q, want %q", i, meta.Links[i].TargetURL, want)
		}
	}
}

func TestAnchorWithNoHref(t *testing.T) {
	html := `<html><body>
		<a name="section">Named anchor</a>
		<a href="/real">Real link</a>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	if len(meta.Links) != 1 {
		t.Fatalf("expected 1 link (skip named anchor), got %d", len(meta.Links))
	}
	if meta.Links[0].AnchorText != "Real link" {
		t.Errorf("text = %q", meta.Links[0].AnchorText)
	}
}

func TestImageExtraction(t *testing.T) {
	html := `<html><body>
		<img src="https://i.pinimg.com/236x/abc.jpg" alt="Gouache painting">
		<img src="/local/img.png" alt="Local image">
		<a href="/page">Link</a>
	</body></html>`

	// Without extractImages: only the <a> link
	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)
	if len(meta.Links) != 1 {
		t.Fatalf("without extractImages: expected 1 link, got %d", len(meta.Links))
	}

	// With extractImages: <a> link + 2 images
	meta = ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", true)
	if len(meta.Links) != 3 {
		t.Fatalf("with extractImages: expected 3 links, got %d", len(meta.Links))
	}

	// First two should be images
	if meta.Links[0].Rel != "image" {
		t.Errorf("link[0] rel = %q, want image", meta.Links[0].Rel)
	}
	if meta.Links[0].AnchorText != "Gouache painting" {
		t.Errorf("link[0] alt = %q, want %q", meta.Links[0].AnchorText, "Gouache painting")
	}
	if meta.Links[0].IsInternal {
		t.Error("pinimg link should be external")
	}
	if !meta.Links[1].IsInternal {
		t.Error("local image should be internal")
	}
}

func TestSrcsetExtraction(t *testing.T) {
	html := `<html><body>
		<img src="https://i.pinimg.com/236x/abc.jpg"
		     srcset="https://i.pinimg.com/474x/abc.jpg 2x, https://i.pinimg.com/736x/abc.jpg 4x"
		     alt="Pin">
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", true)

	// 1 src + 2 srcset = 3 image links
	if len(meta.Links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(meta.Links))
	}
	if meta.Links[0].Rel != "image" {
		t.Errorf("link[0] rel = %q, want image", meta.Links[0].Rel)
	}
	if meta.Links[1].Rel != "image-srcset" {
		t.Errorf("link[1] rel = %q, want image-srcset", meta.Links[1].Rel)
	}
	if meta.Links[2].Rel != "image-srcset" {
		t.Errorf("link[2] rel = %q, want image-srcset", meta.Links[2].Rel)
	}
}

func TestOGImageExtraction(t *testing.T) {
	html := `<html><head>
		<meta property="og:image" content="https://i.pinimg.com/originals/abc.jpg">
		<meta name="description" content="A painting">
	</head><body></body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", true)

	if meta.Description != "A painting" {
		t.Errorf("description = %q", meta.Description)
	}

	if len(meta.Links) != 1 {
		t.Fatalf("expected 1 link (og:image), got %d", len(meta.Links))
	}
	if meta.Links[0].Rel != "og:image" {
		t.Errorf("rel = %q, want og:image", meta.Links[0].Rel)
	}
	if meta.Links[0].TargetURL != "https://i.pinimg.com/originals/abc.jpg" {
		t.Errorf("url = %q", meta.Links[0].TargetURL)
	}
}

func TestNextDataExtraction(t *testing.T) {
	html := `<html><head>
		<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"posts":[{"slug":"/blog/openai-o3"},{"slug":"/blog/chatgpt"}]}},"page":"/blog","query":{}}</script>
	</head><body></body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://openai.com/"), "openai.com", false)

	foundPaths := map[string]bool{}
	for _, l := range meta.Links {
		if l.Rel == "next-data" {
			foundPaths[l.TargetURL] = true
		}
	}
	if !foundPaths["https://openai.com/blog/openai-o3"] {
		t.Error("expected /blog/openai-o3 from __NEXT_DATA__")
	}
	if !foundPaths["https://openai.com/blog/chatgpt"] {
		t.Error("expected /blog/chatgpt from __NEXT_DATA__")
	}
	if !foundPaths["https://openai.com/blog"] {
		t.Error("expected /blog from __NEXT_DATA__")
	}
}

func TestJSONLDExtraction(t *testing.T) {
	html := `<html><head>
		<script type="application/ld+json">{"@type":"Article","url":"https://openai.com/blog/post","mainEntityOfPage":"https://openai.com/blog","sameAs":["https://twitter.com/openai"]}</script>
	</head><body></body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://openai.com/"), "openai.com", false)

	foundURLs := map[string]bool{}
	for _, l := range meta.Links {
		if l.Rel == "json-ld" {
			foundURLs[l.TargetURL] = true
		}
	}
	if !foundURLs["https://openai.com/blog/post"] {
		t.Error("expected blog/post from JSON-LD url field")
	}
	if !foundURLs["https://openai.com/blog"] {
		t.Error("expected blog from JSON-LD mainEntityOfPage field")
	}
	if !foundURLs["https://twitter.com/openai"] {
		t.Error("expected twitter from JSON-LD sameAs field")
	}
}

func TestInlineJSExtraction(t *testing.T) {
	html := `<html><body>
		<script>
			var routes = ["/blog", "/research/papers", "/api"];
			window.location = "/about";
		</script>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://openai.com/"), "openai.com", false)

	foundPaths := map[string]bool{}
	for _, l := range meta.Links {
		if l.Rel == "inline-js" {
			foundPaths[l.TargetURL] = true
		}
	}
	if !foundPaths["https://openai.com/blog"] {
		t.Error("expected /blog from inline JS")
	}
	if !foundPaths["https://openai.com/research/papers"] {
		t.Error("expected /research/papers from inline JS")
	}
	if !foundPaths["https://openai.com/api"] {
		t.Error("expected /api from inline JS")
	}
	if !foundPaths["https://openai.com/about"] {
		t.Error("expected /about from inline JS")
	}
}

func TestInlineJSSkipsAssets(t *testing.T) {
	html := `<html><body>
		<script>
			var assets = ["/_next/static/chunk.js", "/assets/logo.png", "/static/style.css"];
			var page = "/real-page";
		</script>
	</body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	for _, l := range meta.Links {
		if l.Rel == "inline-js" {
			for _, junk := range []string{"_next", "assets", "static"} {
				if contains(l.TargetURL, junk) {
					t.Errorf("should skip asset path: %s", l.TargetURL)
				}
			}
		}
	}
}

func contains(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && len(s) >= len(sub) && (s == sub || len(s) > len(sub) && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestPrefetchPreloadLinks(t *testing.T) {
	html := `<html><head>
		<link rel="prefetch" href="/next-page">
		<link rel="preload" href="/critical-resource" as="document">
		<link rel="prerender" href="/upcoming-page">
		<link rel="stylesheet" href="/style.css">
	</head><body></body></html>`

	meta := ExtractLinksAndMeta([]byte(html), mustParseURL("https://example.com/"), "example.com", false)

	rels := map[string]bool{}
	for _, l := range meta.Links {
		rels[l.Rel] = true
	}
	if !rels["prefetch"] {
		t.Error("expected prefetch link")
	}
	if !rels["preload"] {
		t.Error("expected preload link")
	}
	if !rels["prerender"] {
		t.Error("expected prerender link")
	}
	// stylesheet should NOT be extracted
	if rels["stylesheet"] {
		t.Error("stylesheet should not be extracted")
	}
}

func TestTrackingParamStripping(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/page?utm_source=twitter&utm_medium=social", "https://example.com/page"},
		{"https://example.com/page?q=search&utm_source=google", "https://example.com/page?q=search"},
		{"https://example.com/page?fbclid=abc123", "https://example.com/page"},
		{"https://example.com/page?key=val", "https://example.com/page?key=val"},
		{"https://example.com/page", "https://example.com/page"},
	}
	for _, tt := range tests {
		got := NormalizeURL(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseSrcset(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"https://img.com/1.jpg 1x", 1},
		{"https://img.com/1.jpg 236w, https://img.com/2.jpg 474w, https://img.com/3.jpg 736w", 3},
		{"https://i.pinimg.com/474x/abc.jpg 2x, https://i.pinimg.com/736x/abc.jpg 4x", 2},
	}
	for _, tt := range tests {
		got := parseSrcset(tt.input)
		if len(got) != tt.want {
			t.Errorf("parseSrcset(%q) = %d urls, want %d", tt.input, len(got), tt.want)
		}
	}
}
