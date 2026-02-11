package openlibrary

// SearchResponse is the Open Library search API response
type SearchResponse struct {
	NumFound int         `json:"numFound"`
	Start    int         `json:"start"`
	Docs     []SearchDoc `json:"docs"`
}

// SearchDoc is a single document from the search API
type SearchDoc struct {
	Key            string   `json:"key"`             // e.g. /works/OL12345W
	Title          string   `json:"title"`
	AuthorName     []string `json:"author_name"`
	AuthorKey      []string `json:"author_key"`
	FirstPublishYear int   `json:"first_publish_year"`
	NumberOfPages  int      `json:"number_of_pages_median"`
	ISBN           []string `json:"isbn"`
	CoverI         int      `json:"cover_i"`
	Subject        []string `json:"subject"`
	Publisher      []string `json:"publisher"`
	Language       []string `json:"language"`
	RatingsAverage float64  `json:"ratings_average"`
	RatingsCount   int      `json:"ratings_count"`
}

// WorkResponse is the Open Library works API response
type WorkResponse struct {
	Key         string      `json:"key"`
	Title       string      `json:"title"`
	Description interface{} `json:"description"` // Can be string or {type, value}
	Covers      []int       `json:"covers"`
	Subjects    []string    `json:"subjects"`
	Authors     []struct {
		Author struct {
			Key string `json:"key"`
		} `json:"author"`
	} `json:"authors"`
}

// AuthorResponse is the Open Library authors API response
type AuthorResponse struct {
	Key        string      `json:"key"`
	Name       string      `json:"name"`
	Bio        interface{} `json:"bio"` // Can be string or {type, value}
	Photos     []int       `json:"photos"`
	BirthDate  string      `json:"birth_date"`
	DeathDate  string      `json:"death_date"`
}
