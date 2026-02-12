package goodreads

import "testing"

func TestParseBookPageExtractsExpandedMetadata(t *testing.T) {
	body := `
<html>
  <head>
    <link rel="canonical" href="https://www.goodreads.com/book/show/123-test-book" />
    <script type="application/ld+json">
      {
        "@type":"Book",
        "name":"Test Book",
        "image":"https://images.example.com/book.jpg",
        "bookFormat":"Hardcover",
        "numberOfPages":384,
        "inLanguage":"English",
        "isbn":"9781234567890",
        "author":[{"name":"Jane Doe","url":"https://www.goodreads.com/author/show/42.Jane_Doe"}],
        "aggregateRating":{"ratingValue":4.21,"ratingCount":1200,"reviewCount":210}
      }
    </script>
  </head>
  <body>
    <div data-testid="description"><span>Short</span><span>Long description for the full book page.</span></div>
    <a href="/genres/fantasy">Fantasy</a>
    <a href="/genres/young-adult">Young Adult</a>
    <div>12,345 people are currently reading</div>
    <div>67,890 people want to read</div>
    <div>5 star 1,000</div>
    <div>4 star 900</div>
    <div>3 star 800</div>
    <div>2 star 700</div>
    <div>1 star 600</div>
    <a href="/series/1-test-series">Test Series #2</a>
    <dl>
      <dt>Original title</dt><dd>The Test Book Original</dd>
      <dt>Edition language</dt><dd>English</dd>
      <dt>Setting</dt><dd>District 12, Panem</dd>
      <dt>Characters</dt><dd>Katniss Everdeen, Peeta Mellark</dd>
      <dt>Literary awards</dt><dd>Locus Award, Hugo Award</dd>
      <dt>Published</dt><dd>Sep 1, 2024 by Acme Press</dd>
      <dt>Publisher</dt><dd>Acme Press</dd>
      <dt>ASIN</dt><dd>B00TEST123</dd>
    </dl>
    <div>75 editions</div>
    <a href="/author/show/42.Jane_Doe">Jane Doe</a>
    <a href="/work/quotes/999888">Quotes</a>
    <article class="ReviewCard">
      <div class="ReviewerProfile__name"><a>Reader One</a></div>
      <div>Rating 4 out of 5</div>
      <span class="Text__body3">May 10, 2021</span>
      <span class="Formatted">Loved it.</span>
      <div>12 likes</div>
      <div>Shelves: favorites, dystopia</div>
    </article>
    <article class="ReviewCard">
      <div class="ReviewerProfile__name"><a>Reader Two</a></div>
      <div>Rating 2 out of 5</div>
      <span class="Text__body3">Jun 1, 2022</span>
      <div>This entire review has been hidden because of spoilers.</div>
    </article>
  </body>
</html>
`

	book, err := parseBookPage(body)
	if err != nil {
		t.Fatalf("parseBookPage: %v", err)
	}
	if book.URL != "https://www.goodreads.com/book/show/123-test-book" {
		t.Fatalf("expected canonical URL, got %q", book.URL)
	}
	if book.Title != "Test Book" {
		t.Fatalf("expected title from JSON-LD, got %q", book.Title)
	}
	if book.AuthorName != "Jane Doe" {
		t.Fatalf("expected author name, got %q", book.AuthorName)
	}
	if book.AuthorURL != "https://www.goodreads.com/author/show/42.Jane_Doe" {
		t.Fatalf("expected author URL, got %q", book.AuthorURL)
	}
	if book.OriginalTitle != "The Test Book Original" {
		t.Fatalf("expected original title, got %q", book.OriginalTitle)
	}
	if book.EditionLanguage != "English" {
		t.Fatalf("expected edition language, got %q", book.EditionLanguage)
	}
	if book.Publisher != "Acme Press" {
		t.Fatalf("expected publisher, got %q", book.Publisher)
	}
	if book.ASIN != "B00TEST123" {
		t.Fatalf("expected ASIN, got %q", book.ASIN)
	}
	if book.EditionCount != 75 {
		t.Fatalf("expected edition count 75, got %d", book.EditionCount)
	}
	if len(book.Characters) != 2 || book.Characters[0] != "Katniss Everdeen" {
		t.Fatalf("expected characters list, got %#v", book.Characters)
	}
	if len(book.Settings) != 2 || book.Settings[0] != "District 12" {
		t.Fatalf("expected settings list, got %#v", book.Settings)
	}
	if len(book.LiteraryAwards) != 2 || book.LiteraryAwards[0] != "Locus Award" {
		t.Fatalf("expected awards list, got %#v", book.LiteraryAwards)
	}
	if book.CurrentlyReading != 12345 || book.WantToRead != 67890 {
		t.Fatalf("expected community stats, got currently=%d want=%d", book.CurrentlyReading, book.WantToRead)
	}
	if book.RatingDist != [5]int{1000, 900, 800, 700, 600} {
		t.Fatalf("expected rating dist, got %#v", book.RatingDist)
	}
	if book.WorkID != "999888" {
		t.Fatalf("expected work id 999888, got %q", book.WorkID)
	}
	if len(book.Reviews) != 2 {
		t.Fatalf("expected 2 reviews, got %d", len(book.Reviews))
	}
	if book.Reviews[0].Shelves != "favorites, dystopia" {
		t.Fatalf("expected parsed shelves, got %q", book.Reviews[0].Shelves)
	}
	if !book.Reviews[1].IsSpoiler {
		t.Fatal("expected spoiler review to be marked")
	}
}

func TestParseQuotesPage(t *testing.T) {
	body := `
<div class="quoteText">
  &ldquo;Stay hungry, stay foolish.&rdquo;
  <span class="authorOrTitle">Steve Jobs</span>
</div>
<div>123 likes</div>
`
	quotes := parseQuotesPage(body)
	if len(quotes) != 1 {
		t.Fatalf("expected one quote, got %d", len(quotes))
	}
	if quotes[0].AuthorName != "Steve Jobs" {
		t.Fatalf("expected author name, got %q", quotes[0].AuthorName)
	}
	if quotes[0].Text != "Stay hungry, stay foolish." {
		t.Fatalf("expected quote text, got %q", quotes[0].Text)
	}
	if quotes[0].LikesCount != 123 {
		t.Fatalf("expected likes 123, got %d", quotes[0].LikesCount)
	}
}
