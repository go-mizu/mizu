package goodreads

import "testing"

func TestParseListPageExtractsRankAndScore(t *testing.T) {
	body := `
<html>
  <body>
    <h1>Best Books Ever</h1>
    <div>345,678 voters</div>
    <table>
      <tr class="bookalike review">
        <td class="number">1</td>
        <td>
          <a class="bookTitle" href="/book/show/100.Book_One"><span>Book One</span></a>
          <a class="authorName"><span>Author One</span></a>
          <img src="https://images.example.com/books/one.jpg" />
          <span>4.55 avg rating</span>
          <span>12,345 ratings</span>
          <span>score: 99,999, and 1,234 people voted</span>
        </td>
      </tr>
      <tr class="bookalike review">
        <td class="number">2</td>
        <td>
          <a class="bookTitle" href="/book/show/200.Book_Two"><span>Book Two</span></a>
          <a class="authorName"><span>Author Two</span></a>
          <img src="https://images.example.com/books/two.jpg" />
          <span>4.10 avg rating</span>
          <span>7,654 ratings</span>
          <span>score: 88,888, and 987 people voted</span>
        </td>
      </tr>
    </table>
  </body>
</html>
`

	list := parseListPage(body)
	if list.Title != "Best Books Ever" {
		t.Fatalf("expected title, got %q", list.Title)
	}
	if list.VoterCount != 345678 {
		t.Fatalf("expected voter count 345678, got %d", list.VoterCount)
	}
	if len(list.Books) != 2 {
		t.Fatalf("expected two list items, got %d", len(list.Books))
	}
	first := list.Books[0]
	if first.Position != 1 {
		t.Fatalf("expected first position 1, got %d", first.Position)
	}
	if first.GoodreadsID != "100" {
		t.Fatalf("expected goodreads id 100, got %q", first.GoodreadsID)
	}
	if first.URL != "https://www.goodreads.com/book/show/100.Book_One" {
		t.Fatalf("expected absolute URL, got %q", first.URL)
	}
	if first.Score != 99999 || first.Voters != 1234 {
		t.Fatalf("expected score/voters, got score=%d voters=%d", first.Score, first.Voters)
	}
}

func TestParseListPageModernTableLayout(t *testing.T) {
	body := `
<html>
  <head>
    <title>Best Books Ever (76831 books)</title>
  </head>
  <body>
    <div id="score_explanation"><h1>Score</h1></div>
    <table class="tableList js-dataTooltip">
      <tr itemscope itemtype="http://schema.org/Book">
        <td class="number">1</td>
        <td>
          <a class="bookTitle" href="/book/show/2767052-the-hunger-games">
            <span>The Hunger Games (The Hunger Games, #1)</span>
          </a>
          <a class="authorName" href="https://www.goodreads.com/author/show/153394.Suzanne_Collins"><span>Suzanne Collins</span></a>
          <img src="https://i.gr-assets.com/images/S/compressed.photo.goodreads.com/books/1586722975i/2767052._SX50_.jpg" />
          <span class="minirating">4.35 avg rating - 9,957,163 ratings</span>
          <a href="#">score: 4,367,395</a>, and <a href="#">44,394 people voted</a>
        </td>
      </tr>
    </table>
    <div>285,871 voters</div>
  </body>
</html>
`
	list := parseListPage(body)
	if list.Title != "Best Books Ever" {
		t.Fatalf("expected page title fallback, got %q", list.Title)
	}
	if list.VoterCount != 285871 {
		t.Fatalf("expected list voter count 285871, got %d", list.VoterCount)
	}
	if len(list.Books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(list.Books))
	}
	if list.Books[0].GoodreadsID != "2767052" {
		t.Fatalf("expected Goodreads ID 2767052, got %q", list.Books[0].GoodreadsID)
	}
}

func TestParseListsBrowse(t *testing.T) {
	body := `
<div>
  <a class="listTitle" href="/list/show/1.Best_Books_Ever">Best Books Ever</a>
  <span>100 books — 2,000 voters</span>
</div>
<div>
  <a class="listTitle" href="/list/show/2.Top_Fantasy">Top Fantasy</a>
  <span>50 books — 500 voters</span>
</div>
`
	lists := parseListsBrowse(body)
	if len(lists) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(lists))
	}
	if lists[0].URL != "https://www.goodreads.com/list/show/1.Best_Books_Ever" {
		t.Fatalf("unexpected URL: %q", lists[0].URL)
	}
	if lists[0].GoodreadsID != "1" {
		t.Fatalf("expected id 1, got %q", lists[0].GoodreadsID)
	}
	if lists[0].BookCount != 100 || lists[0].VoterCount != 2000 {
		t.Fatalf("unexpected first list counts: %#v", lists[0])
	}
}

func TestParseListsBrowseExtractsCountsNearTitleAnchor(t *testing.T) {
	body := `
<div>
  <a href="/list/show/1.Best_Books_Ever"><img src="cover.jpg" /></a>
  <a class="listTitle" href="/list/show/1.Best_Books_Ever">Best Books Ever</a><br />
  <div class="listFullDetails">
    76,831 books
    &mdash;
    285,871 voters
  </div>
</div>
`
	lists := parseListsBrowse(body)
	if len(lists) != 1 {
		t.Fatalf("expected one list, got %d", len(lists))
	}
	if lists[0].BookCount != 76831 || lists[0].VoterCount != 285871 {
		t.Fatalf("unexpected list counts: %#v", lists[0])
	}
}

func TestParseListsBrowseReturnsEmptyForNextShell(t *testing.T) {
	body := `<html><head><script id="__NEXT_DATA__" type="application/json">{}</script></head><body><div id="__next"></div></body></html>`
	lists := parseListsBrowse(body)
	if len(lists) != 0 {
		t.Fatalf("expected no lists from next shell, got %d", len(lists))
	}
}
