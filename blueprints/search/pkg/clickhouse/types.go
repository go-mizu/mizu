package clickhouse

// Account represents a ClickHouse Cloud account stored locally.
type Account struct {
	ID           string
	Email        string
	Password     string
	OrgID        string
	APIKeyID     string
	APIKeySecret string
	IsActive     bool
	CreatedAt    string
	SvcCount     int
}

// Service represents a ClickHouse Cloud service stored locally.
type Service struct {
	ID         string
	AccountID  string
	CloudID    string
	Name       string
	Alias      string
	Host       string
	Port       int
	DBUser     string
	DBPassword string
	Provider   string
	Region     string
	IsDefault  bool
	CreatedAt  string
	LastUsedAt string
	QueryCount int
	Email      string // joined from accounts
}

// RegisterResult is the JSON output from `clickhouse-tool register --json`.
type RegisterResult struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	OrgID        string `json:"org_id"`
	APIKeyID     string `json:"api_key_id"`
	APIKeySecret string `json:"api_key_secret"`
	ServiceID    string `json:"service_id"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	DBPassword   string `json:"db_password"`
}
