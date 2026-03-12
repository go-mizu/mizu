package amazon

import (
	"strings"
	"time"
)

const (
	BaseURL        = "https://www.amazon.com"
	DefaultDelay   = 3 * time.Second
	DefaultWorkers = 2
	DefaultTimeout = 30 * time.Second
)

const (
	EntityProduct    = "product"
	EntityBrand      = "brand"
	EntityAuthor     = "author"
	EntityCategory   = "category"
	EntitySearch     = "search"
	EntityBestseller = "bestseller"
	EntityReview     = "review"
	EntityQA         = "qa"
	EntitySeller     = "seller"
)

// Product represents an Amazon product page.
type Product struct {
	ASIN           string            `json:"asin"`
	Title          string            `json:"title"`
	Brand          string            `json:"brand"`
	BrandID        string            `json:"brand_id"`
	Price          float64           `json:"price"`
	Currency       string            `json:"currency"`
	ListPrice      float64           `json:"list_price"`
	Rating         float64           `json:"rating"`
	RatingsCount   int64             `json:"ratings_count"`
	ReviewsCount   int64             `json:"reviews_count"`
	AnsweredQs     int               `json:"answered_qs"`
	Availability   string            `json:"availability"`
	Description    string            `json:"description"`
	BulletPoints   []string          `json:"bullet_points"`
	Specs          map[string]string `json:"specs"`
	Images         []string          `json:"images"`
	CategoryPath   []string          `json:"category_path"`
	BrowseNodeIDs  []string          `json:"browse_node_ids"`
	SellerID       string            `json:"seller_id"`
	SellerName     string            `json:"seller_name"`
	SoldBy         string            `json:"sold_by"`
	FulfilledBy    string            `json:"fulfilled_by"`
	VariantASINs   []string          `json:"variant_asins"`
	ParentASIN     string            `json:"parent_asin"`
	SimilarASINs   []string          `json:"similar_asins"`
	Rank           int               `json:"rank"`
	RankCategory   string            `json:"rank_category"`
	URL            string            `json:"url"`
	FetchedAt      time.Time         `json:"fetched_at"`
}

// Brand represents an Amazon brand/store page.
type Brand struct {
	BrandID       string    `json:"brand_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	LogoURL       string    `json:"logo_url"`
	BannerURL     string    `json:"banner_url"`
	FollowerCount int       `json:"follower_count"`
	URL           string    `json:"url"`
	FeaturedASINs []string  `json:"featured_asins"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// Author represents an Amazon author profile.
type Author struct {
	AuthorID      string    `json:"author_id"`
	Name          string    `json:"name"`
	Bio           string    `json:"bio"`
	PhotoURL      string    `json:"photo_url"`
	Website       string    `json:"website"`
	Twitter       string    `json:"twitter"`
	BookASINs     []string  `json:"book_asins"`
	FollowerCount int       `json:"follower_count"`
	URL           string    `json:"url"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// Category represents an Amazon browse node / category.
type Category struct {
	NodeID       string    `json:"node_id"`
	Name         string    `json:"name"`
	ParentNodeID string    `json:"parent_node_id"`
	Breadcrumb   []string  `json:"breadcrumb"`
	ChildNodeIDs []string  `json:"child_node_ids"`
	TopASINs     []string  `json:"top_asins"`
	URL          string    `json:"url"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// BestsellerList represents one Amazon bestseller list snapshot.
type BestsellerList struct {
	ListID       string    `json:"list_id"`
	ListType     string    `json:"list_type"`
	Category     string    `json:"category"`
	NodeID       string    `json:"node_id"`
	SnapshotDate string    `json:"snapshot_date"`
	URL          string    `json:"url"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// BestsellerEntry is one ranked item within a BestsellerList.
type BestsellerEntry struct {
	ListID       string  `json:"list_id"`
	ASIN         string  `json:"asin"`
	Rank         int     `json:"rank"`
	Title        string  `json:"title"`
	Price        float64 `json:"price"`
	Rating       float64 `json:"rating"`
	RatingsCount int64   `json:"ratings_count"`
}

// Review represents an Amazon product review.
type Review struct {
	ReviewID        string            `json:"review_id"`
	ASIN            string            `json:"asin"`
	ReviewerID      string            `json:"reviewer_id"`
	ReviewerName    string            `json:"reviewer_name"`
	Rating          int               `json:"rating"`
	Title           string            `json:"title"`
	Text            string            `json:"text"`
	DatePosted      time.Time         `json:"date_posted"`
	VerifiedPurchase bool             `json:"verified_purchase"`
	HelpfulVotes    int               `json:"helpful_votes"`
	TotalVotes      int               `json:"total_votes"`
	Images          []string          `json:"images"`
	VariantAttrs    map[string]string `json:"variant_attrs"`
	URL             string            `json:"url"`
	FetchedAt       time.Time         `json:"fetched_at"`
}

// QA represents a question-and-answer pair for an Amazon product.
type QA struct {
	QAID           string    `json:"qa_id"`
	ASIN           string    `json:"asin"`
	Question       string    `json:"question"`
	QuestionBy     string    `json:"question_by"`
	QuestionDate   time.Time `json:"question_date"`
	Answer         string    `json:"answer"`
	AnswerBy       string    `json:"answer_by"`
	AnswerDate     time.Time `json:"answer_date"`
	HelpfulVotes   int       `json:"helpful_votes"`
	IsSellerAnswer bool      `json:"is_seller_answer"`
	FetchedAt      time.Time `json:"fetched_at"`
}

// Seller represents an Amazon third-party seller profile.
type Seller struct {
	SellerID    string    `json:"seller_id"`
	Name        string    `json:"name"`
	Rating      float64   `json:"rating"`
	RatingCount int       `json:"rating_count"`
	PositivePct float64   `json:"positive_pct"`
	NeutralPct  float64   `json:"neutral_pct"`
	NegativePct float64   `json:"negative_pct"`
	URL         string    `json:"url"`
	FetchedAt   time.Time `json:"fetched_at"`
}

// SearchResult represents a single Amazon search results page.
type SearchResult struct {
	SearchID     string    `json:"search_id"`
	Query        string    `json:"query"`
	Page         int       `json:"page"`
	ResultASINs  []string  `json:"result_asins"`
	TotalResults string    `json:"total_results"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// QueueItem is a row from the crawl queue.
type QueueItem struct {
	ID         int64  `json:"id"`
	URL        string `json:"url"`
	EntityType string `json:"entity_type"`
	Priority   int    `json:"priority"`
}

// JobRecord holds a job record for display.
type JobRecord struct {
	JobID       string    `json:"job_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// ExtractASIN extracts an ASIN from an Amazon product URL (/dp/XXXXXXXXXX).
// Returns empty string if not found.
func ExtractASIN(rawURL string) string {
	const marker = "/dp/"
	idx := strings.Index(rawURL, marker)
	if idx == -1 {
		return ""
	}
	rest := rawURL[idx+len(marker):]
	end := strings.IndexAny(rest, "/?")
	if end == -1 {
		end = len(rest)
	}
	if end < 10 {
		return ""
	}
	return rest[:10]
}
