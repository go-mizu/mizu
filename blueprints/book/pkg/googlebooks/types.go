package googlebooks

// SearchResponse is the Google Books API search response
type SearchResponse struct {
	TotalItems int      `json:"totalItems"`
	Items      []Volume `json:"items"`
}

// Volume is a Google Books volume
type Volume struct {
	ID         string     `json:"id"`
	VolumeInfo VolumeInfo `json:"volumeInfo"`
}

// VolumeInfo contains book metadata
type VolumeInfo struct {
	Title               string               `json:"title"`
	Subtitle            string               `json:"subtitle"`
	Authors             []string             `json:"authors"`
	Publisher           string               `json:"publisher"`
	PublishedDate       string               `json:"publishedDate"`
	Description         string               `json:"description"`
	IndustryIdentifiers []IndustryIdentifier `json:"industryIdentifiers"`
	PageCount           int                  `json:"pageCount"`
	Categories          []string             `json:"categories"`
	AverageRating       float64              `json:"averageRating"`
	RatingsCount        int                  `json:"ratingsCount"`
	ImageLinks          *ImageLinks          `json:"imageLinks"`
	Language            string               `json:"language"`
}

// IndustryIdentifier holds ISBN data
type IndustryIdentifier struct {
	Type       string `json:"type"` // ISBN_10, ISBN_13
	Identifier string `json:"identifier"`
}

// ImageLinks holds cover image URLs
type ImageLinks struct {
	SmallThumbnail string `json:"smallThumbnail"`
	Thumbnail      string `json:"thumbnail"`
}
