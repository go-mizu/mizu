package web

import "time"

// Config holds web server configuration.
type Config struct {
	Host         string
	Port         int
	DatabasePath string

	// Site configuration
	SiteName        string
	SiteDescription string

	// Limits
	MaxUploadSize   int64
	PostsPerPage    int
	CommentsPerPage int

	// Registration
	RegistrationEnabled bool
	RequireEmail        bool
	MinUsernameLength   int
	MaxUsernameLength   int

	// Moderation
	MinKarmaToPost  int
	MinAccountAge   time.Duration
	AutoApprove     bool

	// Rate limits
	ThreadsPerHour  int
	PostsPerHour    int
	VotesPerHour    int

	// Session
	SessionSecret string
	SessionMaxAge time.Duration
}

// Default returns default configuration.
func Default() Config {
	return Config{
		Host:         "localhost",
		Port:         8080,
		DatabasePath: "forum.db",

		SiteName:        "Forum",
		SiteDescription: "A community forum",

		MaxUploadSize:   10 * 1024 * 1024, // 10 MB
		PostsPerPage:    25,
		CommentsPerPage: 50,

		RegistrationEnabled: true,
		RequireEmail:        true,
		MinUsernameLength:   3,
		MaxUsernameLength:   20,

		MinKarmaToPost: 0,
		MinAccountAge:  0,
		AutoApprove:    true,

		ThreadsPerHour: 10,
		PostsPerHour:   50,
		VotesPerHour:   200,

		SessionSecret: "change-me-in-production",
		SessionMaxAge: 30 * 24 * time.Hour, // 30 days
	}
}
