// Package cc provides tools for downloading, indexing, and extracting pages
// from Common Crawl archives. It supports the columnar index (parquet),
// CDXJ index, and WARC file extraction via byte-range requests.
package cc

import "time"

// Crawl represents a Common Crawl dataset.
type Crawl struct {
	ID      string    // CC-MAIN-2026-04
	Name    string    // January 2026 Index
	From    time.Time // Start of crawl window
	To      time.Time // End of crawl window
	CDXAPI  string    // CDX API endpoint
	Gateway string    // Timegate URL
}

// WARCPointer locates a single record in a WARC file.
type WARCPointer struct {
	URL          string
	WARCFilename string // Relative path to WARC file
	RecordOffset int64  // Byte offset of gzip member
	RecordLength int64  // Byte length of gzip member
	ContentType  string // Detected MIME type
	Language     string // Detected language(s)
	FetchStatus  int    // Original HTTP status
	Domain       string // url_host_registered_domain
}

// WARCResponse holds the parsed contents of a single WARC response record.
type WARCResponse struct {
	WARCType   string            // WARC-Type (response, revisit, etc.)
	TargetURI  string            // WARC-Target-URI
	Date       time.Time         // WARC-Date
	RecordID   string            // WARC-Record-ID
	HTTPStatus int               // HTTP status code
	HTTPHeaders map[string]string // HTTP response headers
	Body       []byte            // HTTP response body
}

// PageResult is the extracted page content stored in the result database.
type PageResult struct {
	URL           string
	StatusCode    int
	ContentType   string
	ContentLength int64
	Body          string
	Title         string
	Description   string
	Language      string
	Domain        string
	WARCFilename  string
	FetchTimeMs   int64     // Time to fetch WARC record via byte-range
	CrawledAt     time.Time // Original crawl time from CC
	Error         string
}

// IndexFilter defines query criteria for the columnar index.
type IndexFilter struct {
	Languages      []string // content_languages LIKE (e.g. ["eng", "deu"])
	TLDs           []string // url_host_tld IN (e.g. ["com", "org"])
	Domains        []string // url_host_registered_domain IN
	MimeTypes      []string // content_mime_detected IN (e.g. ["text/html"])
	StatusCodes    []int    // fetch_status IN (e.g. [200])
	ExcludeDomains []string // Domains to exclude
	Limit          int      // Max results (0 = unlimited)
	Offset         int      // Offset for pagination
}

// IndexSummary holds aggregate statistics about the imported index.
type IndexSummary struct {
	TotalRecords   int64
	UniqueHosts    int64
	UniqueDomains  int64
	StatusDist     map[int]int64    // fetch_status → count
	MimeDist       map[string]int64 // content_mime_detected → count
	TLDDist        map[string]int64 // url_host_tld → count
	LangDist       map[string]int64 // content_languages → count
}

// DownloadProgress reports progress of file downloads.
type DownloadProgress struct {
	File          string
	FileIndex     int
	TotalFiles    int
	BytesReceived int64
	TotalBytes    int64
	Done          bool
	Error         error
}

// ProgressFn is a callback for reporting download progress.
type ProgressFn func(DownloadProgress)

// CDXJEntry represents a single CDXJ index entry.
type CDXJEntry struct {
	SURTKey    string
	Timestamp  string
	URL        string
	Mime       string
	Status     string
	Digest     string
	Length     string
	Offset     string
	Filename   string
	Languages  string
	Encoding   string
}
