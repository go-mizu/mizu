package store

import (
	"context"
	"time"
)

// Store defines the interface for all storage operations.
type Store interface {
	// Schema management
	Ensure(ctx context.Context) error
	Close() error

	// Zones
	Zones() ZoneStore
	// DNS
	DNS() DNSStore
	// SSL
	SSL() SSLStore
	// Firewall
	Firewall() FirewallStore
	// Cache
	Cache() CacheStore
	// Workers
	Workers() WorkerStore
	// KV
	KV() KVStore
	// R2
	R2() R2Store
	// D1
	D1() D1Store
	// Load Balancer
	LoadBalancer() LoadBalancerStore
	// Analytics
	Analytics() AnalyticsStore
	// Rules
	Rules() RulesStore
	// Users
	Users() UserStore
	// Durable Objects
	DurableObjects() DurableObjectStore
	// Queues
	Queues() QueueStore
	// Vectorize
	Vectorize() VectorizeStore
	// Analytics Engine
	AnalyticsEngine() AnalyticsEngineStore
	// AI
	AI() AIStore
	// AI Gateway
	AIGateway() AIGatewayStore
	// Hyperdrive
	Hyperdrive() HyperdriveStore
	// Cron Triggers
	Cron() CronStore
}

// Zone represents a DNS zone/domain.
type Zone struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"` // active, pending, paused
	Plan        string    `json:"plan"`   // free, pro, business, enterprise
	NameServers []string  `json:"name_servers"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DNSRecord represents a DNS record.
type DNSRecord struct {
	ID        string    `json:"id"`
	ZoneID    string    `json:"zone_id"`
	Type      string    `json:"type"` // A, AAAA, CNAME, MX, TXT, NS, SRV, CAA, PTR
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	TTL       int       `json:"ttl"`
	Priority  int       `json:"priority,omitempty"`
	Proxied   bool      `json:"proxied"`
	Comment   string    `json:"comment,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Certificate represents an SSL certificate.
type Certificate struct {
	ID          string    `json:"id"`
	ZoneID      string    `json:"zone_id"`
	Type        string    `json:"type"` // edge, origin, client
	Hosts       []string  `json:"hosts"`
	Issuer      string    `json:"issuer"`
	SerialNum   string    `json:"serial_number"`
	Signature   string    `json:"signature"`
	Status      string    `json:"status"` // active, pending, expired
	ExpiresAt   time.Time `json:"expires_at"`
	Certificate string    `json:"certificate"`
	PrivateKey  string    `json:"private_key"`
	CreatedAt   time.Time `json:"created_at"`
}

// SSLSettings represents zone SSL settings.
type SSLSettings struct {
	ZoneID                   string `json:"zone_id"`
	Mode                     string `json:"mode"` // off, flexible, full, strict
	AlwaysHTTPS              bool   `json:"always_https"`
	MinTLSVersion            string `json:"min_tls_version"`
	OpportunisticEncryption  bool   `json:"opportunistic_encryption"`
	TLS13                    bool   `json:"tls_1_3"`
	AutomaticHTTPSRewrites   bool   `json:"automatic_https_rewrites"`
}

// FirewallRule represents a WAF rule.
type FirewallRule struct {
	ID          string    `json:"id"`
	ZoneID      string    `json:"zone_id"`
	Description string    `json:"description"`
	Expression  string    `json:"expression"`
	Action      string    `json:"action"` // block, challenge, allow, log, skip
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IPAccessRule represents an IP access rule.
type IPAccessRule struct {
	ID        string    `json:"id"`
	ZoneID    string    `json:"zone_id"`
	Mode      string    `json:"mode"` // block, challenge, whitelist
	Target    string    `json:"target"` // ip, ip_range, asn, country
	Value     string    `json:"value"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// RateLimitRule represents a rate limiting rule.
type RateLimitRule struct {
	ID            string    `json:"id"`
	ZoneID        string    `json:"zone_id"`
	Description   string    `json:"description"`
	Expression    string    `json:"expression"`
	Threshold     int       `json:"threshold"`
	Period        int       `json:"period"` // seconds
	Action        string    `json:"action"` // block, challenge, log
	ActionTimeout int       `json:"action_timeout"` // seconds
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
}

// CacheSettings represents cache configuration.
type CacheSettings struct {
	ZoneID          string `json:"zone_id"`
	CacheLevel      string `json:"cache_level"` // no_query_string, ignore_query_string, standard
	BrowserTTL      int    `json:"browser_ttl"`
	EdgeTTL         int    `json:"edge_ttl"`
	DevelopmentMode bool   `json:"development_mode"`
	AlwaysOnline    bool   `json:"always_online"`
}

// CacheRule represents a custom cache rule.
type CacheRule struct {
	ID          string    `json:"id"`
	ZoneID      string    `json:"zone_id"`
	Name        string    `json:"name"`
	Expression  string    `json:"expression"`
	CacheLevel  string    `json:"cache_level"`
	EdgeTTL     int       `json:"edge_ttl"`
	BrowserTTL  int       `json:"browser_ttl"`
	BypassCache bool      `json:"bypass_cache"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// Worker represents a Cloudflare Worker.
type Worker struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Script    string            `json:"script"`
	Routes    []string          `json:"routes"`
	Bindings  map[string]string `json:"bindings"` // KV, R2, D1 bindings
	Enabled   bool              `json:"enabled"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// WorkerRoute represents a worker route.
type WorkerRoute struct {
	ID       string `json:"id"`
	ZoneID   string `json:"zone_id"`
	Pattern  string `json:"pattern"`
	WorkerID string `json:"worker_id"`
	Enabled  bool   `json:"enabled"`
}

// KVNamespace represents a KV namespace.
type KVNamespace struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// KVPair represents a key-value pair.
type KVPair struct {
	Key        string            `json:"key"`
	Value      []byte            `json:"value"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Expiration *time.Time        `json:"expiration,omitempty"`
}

// R2Bucket represents an R2 bucket.
type R2Bucket struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	CreatedAt time.Time `json:"created_at"`
}

// R2Object represents an R2 object.
type R2Object struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ETag         string            `json:"etag"`
	ContentType  string            `json:"content_type"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	LastModified time.Time         `json:"last_modified"`
}

// D1Database represents a D1 database.
type D1Database struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	NumTables int       `json:"num_tables"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

// LoadBalancer represents a load balancer.
type LoadBalancer struct {
	ID             string    `json:"id"`
	ZoneID         string    `json:"zone_id"`
	Name           string    `json:"name"`
	Fallback       string    `json:"fallback_pool"`
	DefaultPools   []string  `json:"default_pools"`
	SessionAffinity string   `json:"session_affinity"` // none, cookie, ip_cookie
	SteeringPolicy string    `json:"steering_policy"` // off, geo, dynamic, proximity
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
}

// OriginPool represents an origin pool.
type OriginPool struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Origins       []Origin  `json:"origins"`
	CheckRegions  []string  `json:"check_regions"`
	Description   string    `json:"description"`
	Enabled       bool      `json:"enabled"`
	MinOrigins    int       `json:"minimum_origins"`
	Monitor       string    `json:"monitor"`
	NotifyEmail   string    `json:"notification_email"`
	CreatedAt     time.Time `json:"created_at"`
}

// Origin represents an origin server.
type Origin struct {
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Weight  float64           `json:"weight"`
	Enabled bool              `json:"enabled"`
	Header  map[string]string `json:"header,omitempty"`
}

// HealthCheck represents a health check monitor.
type HealthCheck struct {
	ID              string   `json:"id"`
	Description     string   `json:"description"`
	Type            string   `json:"type"` // http, https, tcp
	Method          string   `json:"method"`
	Path            string   `json:"path"`
	Header          map[string][]string `json:"header,omitempty"`
	Port            int      `json:"port"`
	Timeout         int      `json:"timeout"`
	Retries         int      `json:"retries"`
	Interval        int      `json:"interval"`
	ExpectedBody    string   `json:"expected_body"`
	ExpectedCodes   string   `json:"expected_codes"`
	FollowRedirects bool     `json:"follow_redirects"`
	AllowInsecure   bool     `json:"allow_insecure"`
}

// AnalyticsData represents analytics data point.
type AnalyticsData struct {
	Timestamp     time.Time `json:"timestamp"`
	Requests      int64     `json:"requests"`
	Bandwidth     int64     `json:"bandwidth"`
	Threats       int64     `json:"threats"`
	PageViews     int64     `json:"page_views"`
	UniqueVisits  int64     `json:"unique_visits"`
	CacheHits     int64     `json:"cache_hits"`
	CacheMisses   int64     `json:"cache_misses"`
	StatusCodes   map[int]int64 `json:"status_codes"`
}

// PageRule represents a page rule.
type PageRule struct {
	ID        string            `json:"id"`
	ZoneID    string            `json:"zone_id"`
	Targets   []string          `json:"targets"`
	Actions   map[string]interface{} `json:"actions"`
	Priority  int               `json:"priority"`
	Status    string            `json:"status"` // active, disabled
	CreatedAt time.Time         `json:"created_at"`
}

// TransformRule represents a transform rule.
type TransformRule struct {
	ID          string    `json:"id"`
	ZoneID      string    `json:"zone_id"`
	Type        string    `json:"type"` // rewrite_url, modify_request_header, modify_response_header
	Expression  string    `json:"expression"`
	Action      string    `json:"action"`
	ActionValue string    `json:"action_value"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// User represents a user account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // admin, user
	CreatedAt    time.Time `json:"created_at"`
}

// Session represents a user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Store interfaces

type ZoneStore interface {
	Create(ctx context.Context, zone *Zone) error
	GetByID(ctx context.Context, id string) (*Zone, error)
	GetByName(ctx context.Context, name string) (*Zone, error)
	List(ctx context.Context) ([]*Zone, error)
	Update(ctx context.Context, zone *Zone) error
	Delete(ctx context.Context, id string) error
}

type DNSStore interface {
	Create(ctx context.Context, record *DNSRecord) error
	GetByID(ctx context.Context, id string) (*DNSRecord, error)
	ListByZone(ctx context.Context, zoneID string) ([]*DNSRecord, error)
	Update(ctx context.Context, record *DNSRecord) error
	Delete(ctx context.Context, id string) error
}

type SSLStore interface {
	CreateCertificate(ctx context.Context, cert *Certificate) error
	GetCertificate(ctx context.Context, id string) (*Certificate, error)
	ListCertificates(ctx context.Context, zoneID string) ([]*Certificate, error)
	DeleteCertificate(ctx context.Context, id string) error
	GetSettings(ctx context.Context, zoneID string) (*SSLSettings, error)
	UpdateSettings(ctx context.Context, settings *SSLSettings) error
}

type FirewallStore interface {
	CreateRule(ctx context.Context, rule *FirewallRule) error
	GetRule(ctx context.Context, id string) (*FirewallRule, error)
	ListRules(ctx context.Context, zoneID string) ([]*FirewallRule, error)
	UpdateRule(ctx context.Context, rule *FirewallRule) error
	DeleteRule(ctx context.Context, id string) error
	CreateIPAccessRule(ctx context.Context, rule *IPAccessRule) error
	ListIPAccessRules(ctx context.Context, zoneID string) ([]*IPAccessRule, error)
	DeleteIPAccessRule(ctx context.Context, id string) error
	CreateRateLimitRule(ctx context.Context, rule *RateLimitRule) error
	ListRateLimitRules(ctx context.Context, zoneID string) ([]*RateLimitRule, error)
}

type CacheStore interface {
	GetSettings(ctx context.Context, zoneID string) (*CacheSettings, error)
	UpdateSettings(ctx context.Context, settings *CacheSettings) error
	CreateRule(ctx context.Context, rule *CacheRule) error
	ListRules(ctx context.Context, zoneID string) ([]*CacheRule, error)
	DeleteRule(ctx context.Context, id string) error
	PurgeAll(ctx context.Context, zoneID string) error
	PurgeURLs(ctx context.Context, zoneID string, urls []string) error
}

type WorkerStore interface {
	Create(ctx context.Context, worker *Worker) error
	GetByID(ctx context.Context, id string) (*Worker, error)
	GetByName(ctx context.Context, name string) (*Worker, error)
	List(ctx context.Context) ([]*Worker, error)
	Update(ctx context.Context, worker *Worker) error
	Delete(ctx context.Context, id string) error
	CreateRoute(ctx context.Context, route *WorkerRoute) error
	ListRoutes(ctx context.Context, zoneID string) ([]*WorkerRoute, error)
	DeleteRoute(ctx context.Context, id string) error
}

type KVStore interface {
	CreateNamespace(ctx context.Context, ns *KVNamespace) error
	GetNamespace(ctx context.Context, id string) (*KVNamespace, error)
	ListNamespaces(ctx context.Context) ([]*KVNamespace, error)
	DeleteNamespace(ctx context.Context, id string) error
	Put(ctx context.Context, nsID string, pair *KVPair) error
	Get(ctx context.Context, nsID, key string) (*KVPair, error)
	Delete(ctx context.Context, nsID, key string) error
	List(ctx context.Context, nsID, prefix string, limit int) ([]*KVPair, error)
}

type R2Store interface {
	CreateBucket(ctx context.Context, bucket *R2Bucket) error
	GetBucket(ctx context.Context, id string) (*R2Bucket, error)
	ListBuckets(ctx context.Context) ([]*R2Bucket, error)
	DeleteBucket(ctx context.Context, id string) error
	PutObject(ctx context.Context, bucketID, key string, data []byte, metadata map[string]string) error
	GetObject(ctx context.Context, bucketID, key string) ([]byte, *R2Object, error)
	DeleteObject(ctx context.Context, bucketID, key string) error
	ListObjects(ctx context.Context, bucketID, prefix, delimiter string, limit int) ([]*R2Object, error)
}

type D1Store interface {
	CreateDatabase(ctx context.Context, db *D1Database) error
	GetDatabase(ctx context.Context, id string) (*D1Database, error)
	ListDatabases(ctx context.Context) ([]*D1Database, error)
	DeleteDatabase(ctx context.Context, id string) error
	Query(ctx context.Context, dbID, sql string, params []interface{}) ([]map[string]interface{}, error)
	Exec(ctx context.Context, dbID, sql string, params []interface{}) (int64, error)
}

type LoadBalancerStore interface {
	Create(ctx context.Context, lb *LoadBalancer) error
	GetByID(ctx context.Context, id string) (*LoadBalancer, error)
	List(ctx context.Context, zoneID string) ([]*LoadBalancer, error)
	Update(ctx context.Context, lb *LoadBalancer) error
	Delete(ctx context.Context, id string) error
	CreatePool(ctx context.Context, pool *OriginPool) error
	GetPool(ctx context.Context, id string) (*OriginPool, error)
	ListPools(ctx context.Context) ([]*OriginPool, error)
	UpdatePool(ctx context.Context, pool *OriginPool) error
	DeletePool(ctx context.Context, id string) error
	CreateHealthCheck(ctx context.Context, hc *HealthCheck) error
	ListHealthChecks(ctx context.Context) ([]*HealthCheck, error)
}

type AnalyticsStore interface {
	Record(ctx context.Context, zoneID string, data *AnalyticsData) error
	Query(ctx context.Context, zoneID string, start, end time.Time) ([]*AnalyticsData, error)
	GetSummary(ctx context.Context, zoneID string, period string) (*AnalyticsData, error)
}

type RulesStore interface {
	CreatePageRule(ctx context.Context, rule *PageRule) error
	ListPageRules(ctx context.Context, zoneID string) ([]*PageRule, error)
	DeletePageRule(ctx context.Context, id string) error
	CreateTransformRule(ctx context.Context, rule *TransformRule) error
	ListTransformRules(ctx context.Context, zoneID string) ([]*TransformRule, error)
	DeleteTransformRule(ctx context.Context, id string) error
}

type UserStore interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
}

// ========== Durable Objects ==========

// DurableObjectNamespace represents a DO namespace.
type DurableObjectNamespace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Script    string    `json:"script"`
	ClassName string    `json:"class"`
	CreatedAt time.Time `json:"created_at"`
}

// DurableObjectInstance represents a DO instance.
type DurableObjectInstance struct {
	ID          string    `json:"id"`
	NamespaceID string    `json:"namespace_id"`
	Name        string    `json:"name,omitempty"`
	HasStorage  bool      `json:"has_storage"`
	CreatedAt   time.Time `json:"created_at"`
	LastAccess  time.Time `json:"last_access"`
}

// DurableObjectStorageEntry represents a KV entry in DO storage.
type DurableObjectStorageEntry struct {
	ObjectID string    `json:"object_id"`
	Key      string    `json:"key"`
	Value    []byte    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DurableObjectAlarm represents a scheduled alarm.
type DurableObjectAlarm struct {
	ObjectID      string    `json:"object_id"`
	ScheduledTime time.Time `json:"scheduled_time"`
}

type DurableObjectStore interface {
	// Namespaces
	CreateNamespace(ctx context.Context, ns *DurableObjectNamespace) error
	GetNamespace(ctx context.Context, id string) (*DurableObjectNamespace, error)
	ListNamespaces(ctx context.Context) ([]*DurableObjectNamespace, error)
	DeleteNamespace(ctx context.Context, id string) error

	// Instances
	GetOrCreateInstance(ctx context.Context, nsID, objectID string, name string) (*DurableObjectInstance, error)
	GetInstance(ctx context.Context, id string) (*DurableObjectInstance, error)
	ListInstances(ctx context.Context, nsID string) ([]*DurableObjectInstance, error)
	DeleteInstance(ctx context.Context, id string) error

	// Storage
	Get(ctx context.Context, objectID, key string) ([]byte, error)
	GetMultiple(ctx context.Context, objectID string, keys []string) (map[string][]byte, error)
	Put(ctx context.Context, objectID, key string, value []byte) error
	PutMultiple(ctx context.Context, objectID string, entries map[string][]byte) error
	Delete(ctx context.Context, objectID, key string) error
	DeleteMultiple(ctx context.Context, objectID string, keys []string) error
	DeleteAll(ctx context.Context, objectID string) error
	List(ctx context.Context, objectID string, opts *DOListOptions) (map[string][]byte, error)

	// Alarms
	GetAlarm(ctx context.Context, objectID string) (*time.Time, error)
	SetAlarm(ctx context.Context, objectID string, scheduledTime time.Time) error
	DeleteAlarm(ctx context.Context, objectID string) error
	GetDueAlarms(ctx context.Context, before time.Time) ([]*DurableObjectAlarm, error)

	// SQL Storage (for SQLite-backed DOs)
	ExecSQL(ctx context.Context, objectID, query string, args []interface{}) ([]map[string]interface{}, error)
}

// DOListOptions for listing DO storage entries.
type DOListOptions struct {
	Start       string
	End         string
	Prefix      string
	Reverse     bool
	Limit       int
	AllowConcurrency bool
	NoCache     bool
}

// ========== Queues ==========

// Queue represents a message queue.
type Queue struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Settings  QueueSettings `json:"settings"`
}

// QueueSettings configures queue behavior.
type QueueSettings struct {
	DeliveryDelay   int `json:"delivery_delay"`    // seconds
	MessageTTL      int `json:"message_ttl"`       // seconds
	MaxRetries      int `json:"max_retries"`
	MaxBatchSize    int `json:"max_batch_size"`
	MaxBatchTimeout int `json:"max_batch_timeout"` // seconds
}

// QueueMessage represents a message in the queue.
type QueueMessage struct {
	ID          string    `json:"id"`
	QueueID     string    `json:"queue_id"`
	Body        []byte    `json:"body"`
	ContentType string    `json:"content_type"` // json, text, bytes, v8
	Attempts    int       `json:"attempts"`
	CreatedAt   time.Time `json:"created_at"`
	VisibleAt   time.Time `json:"visible_at"`  // When message becomes visible
	ExpiresAt   time.Time `json:"expires_at"`
}

// QueueConsumer represents a queue consumer.
type QueueConsumer struct {
	ID              string    `json:"id"`
	QueueID         string    `json:"queue_id"`
	ScriptName      string    `json:"script_name"`
	Type            string    `json:"type"` // worker, http_pull
	MaxBatchSize    int       `json:"max_batch_size"`
	MaxBatchTimeout int       `json:"max_batch_timeout"`
	MaxRetries      int       `json:"max_retries"`
	DeadLetterQueue string    `json:"dead_letter_queue,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type QueueStore interface {
	// Queues
	CreateQueue(ctx context.Context, queue *Queue) error
	GetQueue(ctx context.Context, id string) (*Queue, error)
	GetQueueByName(ctx context.Context, name string) (*Queue, error)
	ListQueues(ctx context.Context) ([]*Queue, error)
	DeleteQueue(ctx context.Context, id string) error

	// Messages
	SendMessage(ctx context.Context, queueID string, msg *QueueMessage) error
	SendBatch(ctx context.Context, queueID string, msgs []*QueueMessage) error
	PullMessages(ctx context.Context, queueID string, batchSize int, visibilityTimeout int) ([]*QueueMessage, error)
	AckMessage(ctx context.Context, queueID, msgID string) error
	AckBatch(ctx context.Context, queueID string, msgIDs []string) error
	RetryMessage(ctx context.Context, queueID, msgID string, delaySeconds int) error
	GetQueueStats(ctx context.Context, queueID string) (*QueueStats, error)

	// Consumers
	CreateConsumer(ctx context.Context, consumer *QueueConsumer) error
	GetConsumer(ctx context.Context, id string) (*QueueConsumer, error)
	ListConsumers(ctx context.Context, queueID string) ([]*QueueConsumer, error)
	DeleteConsumer(ctx context.Context, id string) error

	// Dead Letter Queue
	MoveToDeadLetter(ctx context.Context, queueID, msgID string) error
}

// QueueStats contains queue statistics.
type QueueStats struct {
	Messages       int64 `json:"messages"`
	MessagesReady  int64 `json:"messages_ready"`
	MessagesDelayed int64 `json:"messages_delayed"`
}

// ========== Vectorize ==========

// VectorIndex represents a vector index.
type VectorIndex struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Dimensions  int       `json:"dimensions"`
	Metric      string    `json:"metric"` // cosine, euclidean, dot-product
	CreatedAt   time.Time `json:"created_at"`
	VectorCount int64     `json:"vector_count"`
}

// Vector represents a vector with metadata.
type Vector struct {
	ID        string                 `json:"id"`
	Values    []float32              `json:"values"`
	Namespace string                 `json:"namespace,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// VectorMatch represents a query match.
type VectorMatch struct {
	ID       string                 `json:"id"`
	Score    float32                `json:"score"`
	Values   []float32              `json:"values,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// VectorQueryOptions for vector queries.
type VectorQueryOptions struct {
	TopK           int                    `json:"topK"`
	Namespace      string                 `json:"namespace,omitempty"`
	ReturnValues   bool                   `json:"returnValues"`
	ReturnMetadata string                 `json:"returnMetadata"` // none, indexed, all
	Filter         map[string]interface{} `json:"filter,omitempty"`
}

type VectorizeStore interface {
	// Indexes
	CreateIndex(ctx context.Context, index *VectorIndex) error
	GetIndex(ctx context.Context, name string) (*VectorIndex, error)
	ListIndexes(ctx context.Context) ([]*VectorIndex, error)
	DeleteIndex(ctx context.Context, name string) error

	// Vectors
	Insert(ctx context.Context, indexName string, vectors []*Vector) error
	Upsert(ctx context.Context, indexName string, vectors []*Vector) error
	Query(ctx context.Context, indexName string, vector []float32, opts *VectorQueryOptions) ([]*VectorMatch, error)
	GetByIDs(ctx context.Context, indexName string, ids []string) ([]*Vector, error)
	DeleteByIDs(ctx context.Context, indexName string, ids []string) error
	DeleteByNamespace(ctx context.Context, indexName, namespace string) error
}

// ========== Analytics Engine ==========

// AnalyticsEngineDataset represents a dataset.
type AnalyticsEngineDataset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// AnalyticsEngineDataPoint represents a data point.
type AnalyticsEngineDataPoint struct {
	Dataset   string    `json:"dataset"`
	Timestamp time.Time `json:"timestamp"`
	Indexes   []string  `json:"indexes,omitempty"`   // up to 20
	Doubles   []float64 `json:"doubles,omitempty"`   // up to 20
	Blobs     [][]byte  `json:"blobs,omitempty"`     // up to 20
}

type AnalyticsEngineStore interface {
	// Datasets
	CreateDataset(ctx context.Context, dataset *AnalyticsEngineDataset) error
	GetDataset(ctx context.Context, name string) (*AnalyticsEngineDataset, error)
	ListDatasets(ctx context.Context) ([]*AnalyticsEngineDataset, error)
	DeleteDataset(ctx context.Context, name string) error

	// Data points
	WriteDataPoint(ctx context.Context, point *AnalyticsEngineDataPoint) error
	WriteBatch(ctx context.Context, points []*AnalyticsEngineDataPoint) error

	// SQL queries
	Query(ctx context.Context, sql string) ([]map[string]interface{}, error)
}

// ========== Workers AI ==========

// AIModel represents an AI model.
type AIModel struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Task        string            `json:"task"` // text-generation, text-embeddings, image-classification, etc.
	Properties  map[string]interface{} `json:"properties"`
}

// AIInferenceRequest represents an inference request.
type AIInferenceRequest struct {
	Model   string                 `json:"model"`
	Inputs  map[string]interface{} `json:"inputs"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// AIInferenceResponse represents an inference response.
type AIInferenceResponse struct {
	Result interface{} `json:"result"`
	Usage  *AIUsage    `json:"usage,omitempty"`
}

// AIUsage tracks token usage.
type AIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type AIStore interface {
	// Models
	ListModels(ctx context.Context, task string) ([]*AIModel, error)
	GetModel(ctx context.Context, name string) (*AIModel, error)

	// Inference
	Run(ctx context.Context, req *AIInferenceRequest) (*AIInferenceResponse, error)

	// Embeddings (specialized)
	GenerateEmbeddings(ctx context.Context, model string, texts []string) ([][]float32, error)

	// Text generation (specialized)
	GenerateText(ctx context.Context, model string, prompt string, opts map[string]interface{}) (string, error)
	StreamText(ctx context.Context, model string, prompt string, opts map[string]interface{}) (<-chan string, error)
}

// ========== AI Gateway ==========

// AIGateway represents an AI Gateway configuration.
type AIGateway struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	CollectLogs        bool      `json:"collect_logs"`
	CacheEnabled       bool      `json:"cache_enabled"`
	CacheTTL           int       `json:"cache_ttl"` // seconds
	RateLimitEnabled   bool      `json:"rate_limit_enabled"`
	RateLimitCount     int       `json:"rate_limit_count"`
	RateLimitPeriod    int       `json:"rate_limit_period"` // seconds
	CreatedAt          time.Time `json:"created_at"`
}

// AIGatewayLog represents a gateway request log.
type AIGatewayLog struct {
	ID          string    `json:"id"`
	GatewayID   string    `json:"gateway_id"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	Cached      bool      `json:"cached"`
	Status      int       `json:"status"`
	Duration    int       `json:"duration_ms"`
	Tokens      int       `json:"tokens"`
	Cost        float64   `json:"cost"`
	Request     []byte    `json:"request"`
	Response    []byte    `json:"response"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type AIGatewayStore interface {
	// Gateways
	CreateGateway(ctx context.Context, gw *AIGateway) error
	GetGateway(ctx context.Context, id string) (*AIGateway, error)
	GetGatewayByName(ctx context.Context, name string) (*AIGateway, error)
	ListGateways(ctx context.Context) ([]*AIGateway, error)
	UpdateGateway(ctx context.Context, gw *AIGateway) error
	DeleteGateway(ctx context.Context, id string) error

	// Logs
	LogRequest(ctx context.Context, log *AIGatewayLog) error
	GetLogs(ctx context.Context, gatewayID string, limit, offset int) ([]*AIGatewayLog, error)

	// Cache
	GetCachedResponse(ctx context.Context, gatewayID, cacheKey string) ([]byte, bool, error)
	SetCachedResponse(ctx context.Context, gatewayID, cacheKey string, response []byte, ttl int) error
}

// ========== Hyperdrive ==========

// HyperdriveConfig represents a Hyperdrive configuration.
type HyperdriveConfig struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Origin       HyperdriveOrigin `json:"origin"`
	Caching      HyperdriveCaching `json:"caching"`
	CreatedAt    time.Time `json:"created_at"`
}

// HyperdriveOrigin defines the database connection.
type HyperdriveOrigin struct {
	Database string `json:"database"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Scheme   string `json:"scheme"` // postgres, mysql
	User     string `json:"user"`
	Password string `json:"-"` // never expose
}

// HyperdriveCaching configures query caching.
type HyperdriveCaching struct {
	Disabled      bool `json:"disabled"`
	MaxAge        int  `json:"max_age"`        // seconds
	StaleWhileRevalidate int `json:"stale_while_revalidate"`
}

type HyperdriveStore interface {
	// Configs
	CreateConfig(ctx context.Context, cfg *HyperdriveConfig) error
	GetConfig(ctx context.Context, id string) (*HyperdriveConfig, error)
	GetConfigByName(ctx context.Context, name string) (*HyperdriveConfig, error)
	ListConfigs(ctx context.Context) ([]*HyperdriveConfig, error)
	UpdateConfig(ctx context.Context, cfg *HyperdriveConfig) error
	DeleteConfig(ctx context.Context, id string) error

	// Connection pooling stats
	GetStats(ctx context.Context, configID string) (*HyperdriveStats, error)
}

// HyperdriveStats contains connection pool statistics.
type HyperdriveStats struct {
	ActiveConnections int     `json:"active_connections"`
	IdleConnections   int     `json:"idle_connections"`
	TotalConnections  int     `json:"total_connections"`
	QueriesPerSecond  float64 `json:"queries_per_second"`
	CacheHitRate      float64 `json:"cache_hit_rate"`
}

// ========== Cron Triggers ==========

// CronTrigger represents a scheduled trigger.
type CronTrigger struct {
	ID         string    `json:"id"`
	ScriptName string    `json:"script_name"`
	Cron       string    `json:"cron"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CronExecution represents a cron execution record.
type CronExecution struct {
	ID          string    `json:"id"`
	TriggerID   string    `json:"trigger_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Status      string    `json:"status"` // pending, running, success, failed
	Error       string    `json:"error,omitempty"`
}

type CronStore interface {
	// Triggers
	CreateTrigger(ctx context.Context, trigger *CronTrigger) error
	GetTrigger(ctx context.Context, id string) (*CronTrigger, error)
	ListTriggers(ctx context.Context) ([]*CronTrigger, error)
	ListTriggersByScript(ctx context.Context, scriptName string) ([]*CronTrigger, error)
	UpdateTrigger(ctx context.Context, trigger *CronTrigger) error
	DeleteTrigger(ctx context.Context, id string) error

	// Executions
	RecordExecution(ctx context.Context, exec *CronExecution) error
	UpdateExecution(ctx context.Context, exec *CronExecution) error
	GetRecentExecutions(ctx context.Context, triggerID string, limit int) ([]*CronExecution, error)
	GetDueTriggers(ctx context.Context, before time.Time) ([]*CronTrigger, error)
}
