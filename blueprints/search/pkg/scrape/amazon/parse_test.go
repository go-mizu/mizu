package amazon

import "testing"

func TestParseSearchResults(t *testing.T) {
	html := `
<html><body>
<div class="s-main-slot">
  <div data-component-type="s-search-result" data-asin="B000111">
    <h2><a href="/dp/B000111"><span>Widget One</span></a></h2>
    <img class="s-image" src="https://img/1.jpg" />
    <span class="a-price"><span class="a-offscreen">$19.99</span></span>
    <span class="a-icon-alt">4.6 out of 5 stars</span>
    <span class="a-size-base s-underline-text">1,245</span>
    <span class="a-badge-text">Best Seller</span>
    <i aria-label="Amazon Prime"></i>
  </div>
  <div data-component-type="s-search-result" data-asin="B000222">
    <h2><a href="https://www.amazon.com/dp/B000222"><span>Widget Two</span></a></h2>
    <img class="s-image" src="https://img/2.jpg" />
    <span class="a-price-whole">29</span><span class="a-price-fraction">50</span>
    <span class="a-icon-alt">3.9 out of 5 stars</span>
    <span class="a-size-base s-underline-text">77</span>
    Sponsored
  </div>
</div>
<a class="s-pagination-next" href="/s?k=widget&page=2">Next</a>
</body></html>`

	got, hasNext, err := ParseSearchResults("https://www.amazon.com", "widget", 1, []byte(html))
	if err != nil {
		t.Fatalf("ParseSearchResults() error = %v", err)
	}
	if !hasNext {
		t.Fatalf("expected hasNext=true")
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 products, got %d", len(got))
	}
	if got[0].ASIN != "B000111" || got[0].Title != "Widget One" {
		t.Fatalf("unexpected first product: %+v", got[0])
	}
	if got[0].PriceValue != 19.99 || got[0].Currency != "USD" || !got[0].IsPrime {
		t.Fatalf("unexpected first pricing/flags: %+v", got[0])
	}
	if got[1].URL != "https://www.amazon.com/dp/B000222" || !got[1].IsSponsored {
		t.Fatalf("unexpected second URL/flags: %+v", got[1])
	}
	if got[1].PriceValue != 29.50 || got[1].ReviewCount != 77 {
		t.Fatalf("unexpected second pricing/reviews: %+v", got[1])
	}
}

func TestValidateSearchHTML(t *testing.T) {
	if err := ValidateSearchHTML(nil); err == nil {
		t.Fatal("expected error for empty body")
	}
	if err := ValidateSearchHTML([]byte("please solve CAPTCHA")); err == nil {
		t.Fatal("expected captcha error")
	}
	if err := ValidateSearchHTML([]byte("<html></html>")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
