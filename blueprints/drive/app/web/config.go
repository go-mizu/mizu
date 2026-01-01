package web

// Config holds server configuration.
type Config struct {
	Addr    string
	DataDir string
	BaseURL string
	Dev     bool
}
