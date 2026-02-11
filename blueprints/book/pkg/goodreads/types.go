package goodreads

// GoodreadsBook holds all data scraped from a Goodreads book page.
type GoodreadsBook struct {
	GoodreadsID      string            `json:"goodreads_id"`
	Title            string            `json:"title"`
	AuthorName       string            `json:"author_name"`
	AuthorURL        string            `json:"author_url"`
	Description      string            `json:"description"`
	ISBN             string            `json:"isbn"`
	ISBN13           string            `json:"isbn13"`
	ASIN             string            `json:"asin"`
	PageCount        int               `json:"page_count"`
	Format           string            `json:"format"`
	Publisher        string            `json:"publisher"`
	PublishDate      string            `json:"publish_date"`
	FirstPublished   string            `json:"first_published"`
	Language         string            `json:"language"`
	CoverURL         string            `json:"cover_url"`
	Series           string            `json:"series"`
	AverageRating    float64           `json:"average_rating"`
	RatingsCount     int               `json:"ratings_count"`
	ReviewsCount     int               `json:"reviews_count"`
	CurrentlyReading int               `json:"currently_reading"`
	WantToRead       int               `json:"want_to_read"`
	RatingDist       [5]int            `json:"rating_dist"` // [0]=5star .. [4]=1star
	Genres           []string          `json:"genres"`
	Reviews          []GoodreadsReview `json:"reviews"`
	Quotes           []GoodreadsQuote  `json:"quotes"`
}

// GoodreadsReview is a single review from Goodreads.
type GoodreadsReview struct {
	ReviewerName string `json:"reviewer_name"`
	Rating       int    `json:"rating"`
	Date         string `json:"date"`
	Text         string `json:"text"`
	LikesCount   int    `json:"likes_count"`
	Shelves      string `json:"shelves"`
}

// GoodreadsQuote is a quote from a Goodreads book page.
type GoodreadsQuote struct {
	Text       string `json:"text"`
	AuthorName string `json:"author_name"`
	LikesCount int    `json:"likes_count"`
}

// jsonLD is the Schema.org Book JSON-LD embedded in Goodreads pages.
type jsonLD struct {
	Type            string `json:"@type"`
	Name            string `json:"name"`
	Image           string `json:"image"`
	BookFormat      string `json:"bookFormat"`
	NumberOfPages   int    `json:"numberOfPages"`
	InLanguage      string `json:"inLanguage"`
	ISBN            string `json:"isbn"`
	Author          []struct {
		Name string `json:"name"`
	} `json:"author"`
	AggregateRating struct {
		RatingValue float64 `json:"ratingValue"`
		RatingCount int     `json:"ratingCount"`
		ReviewCount int     `json:"reviewCount"`
	} `json:"aggregateRating"`
}
