package web

// Config holds server configuration.
type Config struct {
	Addr        string
	DataDir     string
	StorageRoot string // Root directory for local file browsing (e.g., $HOME/Downloads)
	Dev         bool
	LocalMode   bool // Skip authentication in local mode
}
