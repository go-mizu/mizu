package amazon

import (
	"crypto/md5"
	"fmt"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestExtractASINSupportsCommonAmazonURLShapes(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"https://www.amazon.com/dp/B08N5WRWNW":                              "B08N5WRWNW",
		"https://www.amazon.com/gp/product/B08N5WRWNW/ref=something":        "B08N5WRWNW",
		"https://www.amazon.com/product-reviews/B08N5WRWNW?pageNumber=2":    "B08N5WRWNW",
		"https://www.amazon.com/ask/questions/asin/B08N5WRWNW/1/ref=ask_ql": "B08N5WRWNW",
		"https://www.amazon.com/ask/B08N5WRWNW/ref=cm_ask_ql_ql_al_hza":     "B08N5WRWNW",
	}

	for rawURL, want := range cases {
		rawURL := rawURL
		want := want
		t.Run(rawURL, func(t *testing.T) {
			t.Parallel()
			if got := ExtractASIN(rawURL); got != want {
				t.Fatalf("ExtractASIN(%q) = %q, want %q", rawURL, got, want)
			}
		})
	}
}

func TestParseCategoryFallsBackToBestsellerLinks(t *testing.T) {
	t.Parallel()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<html><body>
			<ul id="zg-ordered-list">
				<li><a href="/dp/B08N5WRWNW">One</a></li>
				<li><a href="/dp/B07FZ8S74R">Two</a></li>
			</ul>
		</body></html>`))
	if err != nil {
		t.Fatalf("NewDocumentFromReader: %v", err)
	}

	category, err := ParseCategory(doc, "https://www.amazon.com/b?node=172282")
	if err != nil {
		t.Fatalf("ParseCategory: %v", err)
	}

	if len(category.TopASINs) != 2 {
		t.Fatalf("expected 2 top ASINs, got %d (%v)", len(category.TopASINs), category.TopASINs)
	}
}

func TestParseQABuildsContentStableIDs(t *testing.T) {
	t.Parallel()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<html><body>
			<div id="question-123">
				<div class="a-expander-content">Does it support Matter?</div>
				<span class="a-profile-name">Buyer</span>
				<div id="answer-456">
					<div class="a-expander-content">Yes, over Wi-Fi.</div>
					<span class="a-profile-name">Amazon</span>
				</div>
			</div>
		</body></html>`))
	if err != nil {
		t.Fatalf("NewDocumentFromReader: %v", err)
	}

	qas, _, err := ParseQA(doc, "B08N5WRWNW", "https://www.amazon.com/ask/B08N5WRWNW")
	if err != nil {
		t.Fatalf("ParseQA: %v", err)
	}
	if len(qas) != 1 {
		t.Fatalf("expected 1 QA, got %d", len(qas))
	}

	want := fmt.Sprintf("%x", md5.Sum([]byte("B08N5WRWNW|Does it support Matter?|Yes, over Wi-Fi.")))
	if qas[0].QAID != want {
		t.Fatalf("QAID = %q, want %q", qas[0].QAID, want)
	}
}
