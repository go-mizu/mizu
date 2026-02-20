package rabbit

import "time"

// Performance tuning constants - all magic numbers consolidated here.
const (
	// Cache sizes
	L1CacheSize   = 64 * 1024 * 1024  // 64MB hot cache
	L2CacheSize   = 256 * 1024 * 1024 // 256MB warm cache
	MetaCacheSize = 16 * 1024 * 1024  // 16MB metadata cache

	// Size thresholds for operation routing
	TinyThreshold   = 8 * 1024          // 8KB - direct memory write
	SmallThreshold  = 128 * 1024        // 128KB - buffered direct write
	MediumThreshold = 1024 * 1024       // 1MB - mmap reads
	LargeThreshold  = 32 * 1024 * 1024  // 32MB - parallel I/O

	// Buffer pool sizes
	TinyBuffer   = 4 * 1024           // 4KB - L1 cache optimal
	SmallBuffer  = 64 * 1024          // 64KB - L2 cache optimal
	MediumBuffer = 256 * 1024         // 256KB
	LargeBuffer  = 2 * 1024 * 1024    // 2MB
	HugeBuffer   = 8 * 1024 * 1024    // 8MB

	// Concurrency
	NumShards        = 256 // Shards for key index
	NumPoolShards    = 32  // Shards for buffer pools
	NumCacheShards   = 64  // Shards for object cache
	HotCacheSlots    = 4096 // Slots in hot cache ring buffer
	LazyLRUThreshold = 8   // Accesses before LRU update

	// I/O
	ChunkSize     = 4 * 1024 * 1024 // 4MB for parallel I/O
	DirectIOAlign = 4096            // 4KB alignment for O_DIRECT
	MaxWorkers    = 8               // Max parallel I/O workers

	// File system
	DirPermissions  = 0750
	FilePermissions = 0600
	TempFilePattern = ".rabbit-tmp-*"

	// Cache TTLs
	DirCacheTTL       = 1 * time.Second
	DirCacheMaxSize   = 10000
	MetaCacheTTL      = 5 * time.Second
	NegativeCacheTTL  = 100 * time.Millisecond

	// Multipart
	MaxPartNumber = 10000
)

// NoFsync can be set to skip fsync calls for maximum write performance.
// WARNING: Trades durability for speed. Data may be lost on crash.
var NoFsync = false
