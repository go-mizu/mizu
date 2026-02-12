package goodreads

import "testing"

func TestParseAuthorPageExtractsWebsiteAndInfluences(t *testing.T) {
	body := `
<html>
  <head><title>Jane Doe (Author of Test Book)</title></head>
  <body>
    <h1 class="authorName"><span>Jane Doe</span></h1>
    <div class="aboutAuthorInfo">
      <span>Short bio.</span>
      <span>Jane Doe writes books about courage and friendship.</span>
    </div>
    <img class="authorPhoto" src="https://images.example.com/author.jpg" />
    <dl>
      <dt>Born</dt><dd>January 2, 1970</dd>
      <dt>Died</dt><dd>March 4, 2020</dd>
      <dt>Website</dt><dd><a href="https://janedoe.example.com">janedoe.example.com</a></dd>
      <dt>Influences</dt>
      <dd>
        <a href="/author/show/1.Author_One">Author One</a>,
        <a href="/author/show/2.Author_Two">Author Two</a>
      </dd>
    </dl>
    <div>1,234 followers</div>
    <div>321 distinct works</div>
    <a href="/genres/fantasy">Fantasy</a>
    <a href="/genres/young-adult">Young Adult</a>
  </body>
</html>
`

	author := parseAuthorPage(body)
	if author.Name != "Jane Doe" {
		t.Fatalf("expected author name, got %q", author.Name)
	}
	if author.Website != "https://janedoe.example.com" {
		t.Fatalf("expected website, got %q", author.Website)
	}
	if author.BornDate != "January 2, 1970" {
		t.Fatalf("expected born date, got %q", author.BornDate)
	}
	if author.DiedDate != "March 4, 2020" {
		t.Fatalf("expected died date, got %q", author.DiedDate)
	}
	if author.WorksCount != 321 {
		t.Fatalf("expected works count 321, got %d", author.WorksCount)
	}
	if author.Followers != 1234 {
		t.Fatalf("expected followers 1234, got %d", author.Followers)
	}
	if author.Genres != "Fantasy, Young Adult" {
		t.Fatalf("expected genres, got %q", author.Genres)
	}
	if author.Influences != "Author One, Author Two" {
		t.Fatalf("expected influences, got %q", author.Influences)
	}
	if author.Bio == "" {
		t.Fatal("expected bio to be populated")
	}
}
