package motherduck

// Account represents a MotherDuck account stored locally.
type Account struct {
	ID        string
	Email     string
	Password  string
	Token     string
	IsActive  bool
	CreatedAt string
	DBCount   int
}

// Database represents a MotherDuck cloud database stored locally.
type Database struct {
	ID         string
	AccountID  string
	Name       string
	Alias      string
	Email      string
	Token      string
	IsDefault  bool
	CreatedAt  string
	LastUsedAt string
	QueryCount int
}

// RegisterResult is the JSON output from `motherduck-tool register --json`.
type RegisterResult struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Token    string `json:"token"`
}
