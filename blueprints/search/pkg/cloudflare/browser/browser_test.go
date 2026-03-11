package browser_test

import (
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cloudflare/browser"
)

const testURL = "https://sqlite.org"

func newTestClient(t *testing.T) *browser.Client {
	t.Helper()
	creds, err := browser.LoadCredentials()
	if err != nil {
		t.Skipf("no CF credentials (%v); skipping integration test", err)
	}
	return browser.NewClient(creds)
}

func TestContent(t *testing.T) {
	c := newTestClient(t)
	html, err := c.Content(browser.ContentRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(html), "sqlite") {
		t.Errorf("expected 'sqlite' in content, got first 200 chars: %.200s", html)
	}
	t.Logf("content length: %d bytes", len(html))
}

func TestMarkdown(t *testing.T) {
	c := newTestClient(t)
	md, err := c.Markdown(browser.MarkdownRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(md), "sqlite") {
		t.Errorf("expected 'sqlite' in markdown, got first 200 chars: %.200s", md)
	}
	t.Logf("markdown length: %d bytes", len(md))
}

func TestLinks(t *testing.T) {
	c := newTestClient(t)
	links, err := c.Links(browser.LinksRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(links) == 0 {
		t.Error("expected at least one link")
	}
	t.Logf("found %d links; first: %s", len(links), links[0])
}

func TestScreenshot(t *testing.T) {
	c := newTestClient(t)
	img, err := c.Screenshot(browser.ScreenshotRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(img) == 0 {
		t.Fatal("expected non-empty screenshot")
	}
	// PNG magic bytes: 89 50 4E 47
	if len(img) < 4 || img[0] != 0x89 || img[1] != 0x50 || img[2] != 0x4E || img[3] != 0x47 {
		t.Errorf("expected PNG magic bytes, got: %x", img[:min(4, len(img))])
	}
	t.Logf("screenshot size: %d bytes", len(img))
}

func TestPDF(t *testing.T) {
	c := newTestClient(t)
	pdf, err := c.PDF(browser.PDFRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) == 0 {
		t.Fatal("expected non-empty PDF")
	}
	// PDF magic bytes: %PDF
	if len(pdf) < 4 || pdf[0] != '%' || pdf[1] != 'P' || pdf[2] != 'D' || pdf[3] != 'F' {
		t.Errorf("expected PDF magic bytes %%PDF, got: %q", pdf[:min(8, len(pdf))])
	}
	t.Logf("PDF size: %d bytes", len(pdf))
}

func TestSnapshot(t *testing.T) {
	c := newTestClient(t)
	snap, err := c.Snapshot(browser.SnapshotRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
	})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Screenshot == "" {
		t.Error("expected non-empty screenshot in snapshot")
	}
	if !strings.Contains(strings.ToLower(snap.Content), "sqlite") {
		t.Error("expected 'sqlite' in snapshot content")
	}
	t.Logf("snapshot screenshot length: %d chars (base64)", len(snap.Screenshot))
}

func TestScrape(t *testing.T) {
	c := newTestClient(t)
	result, err := c.Scrape(browser.ScrapeRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
		Elements:      []browser.ScrapeElement{{Selector: "a"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least one selector result")
	}
	if len(result[0].Results) == 0 {
		t.Error("expected at least one scraped element for selector 'a'")
	}
	t.Logf("scraped %d <a> elements", len(result[0].Results))
}

func TestJSON(t *testing.T) {
	c := newTestClient(t)
	result, err := c.JSON(browser.JSONRequest{
		CommonRequest: browser.CommonRequest{URL: testURL},
		Prompt:        "Extract the page title and list the main navigation links",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty JSON result")
	}
	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	t.Logf("JSON result keys: %v", keys)
}

func TestCrawl(t *testing.T) {
	c := newTestClient(t)

	renderFalse := false
	job, err := c.StartCrawl(browser.CrawlRequest{
		URL:    testURL,
		Limit:  3,
		Render: &renderFalse,
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.ID == "" {
		t.Fatal("expected non-empty job ID")
	}
	t.Logf("started crawl job %s (status=%s)", job.ID, job.Status)

	// Poll until complete (max 3 min).
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)
		result, err := c.GetCrawl(job.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("job %s: status=%s finished=%d/%d", result.ID, result.Status, result.Finished, result.Total)
		if result.Status != "running" {
			if len(result.Records) == 0 {
				t.Error("expected at least one crawl record")
			}
			t.Logf("crawl done: %d records, status=%s", len(result.Records), result.Status)
			return
		}
	}
	// Attempt cleanup on timeout.
	_ = c.DeleteCrawl(job.ID)
	t.Error("crawl did not complete within 3 minutes")
}
