# 0020 Sync + Live Testing Plan

## Overview

This document details the comprehensive testing strategy for the `mizu/sync` package and its integration with `mizu/live`. Tests are organized by component and integration level.

## 1. Sync Package Unit Tests

### 1.1 Mutation Tests (`mutation_test.go`)

```go
func TestMutation_Basic(t *testing.T)
// - Test mutation creation with name, args, scope
// - Test mutation with client ID and sequence

func TestMutatorFunc(t *testing.T)
// - Test MutatorFunc wraps function correctly
// - Test function receives correct parameters

func TestMutatorMap_Register(t *testing.T)
// - Test registering handlers
// - Test multiple handlers for different names

func TestMutatorMap_Apply(t *testing.T)
// - Test dispatching to correct handler
// - Test ErrUnknownMutation for unregistered name
// - Test handler error propagation
```

### 1.2 ChangeLog Tests (`changelog_test.go`)

```go
func TestMemoryChangeLog_Append(t *testing.T)
// - Test appending single change
// - Test cursor increments monotonically
// - Test timestamp is set

func TestMemoryChangeLog_Since(t *testing.T)
// - Test retrieving changes since cursor 0
// - Test retrieving changes since specific cursor
// - Test filtering by scope
// - Test limit parameter
// - Test empty result when cursor is at head

func TestMemoryChangeLog_Cursor(t *testing.T)
// - Test cursor is 0 initially
// - Test cursor after appends

func TestMemoryChangeLog_Trim(t *testing.T)
// - Test trimming old entries
// - Test entries before cursor are removed
// - Test entries at/after cursor remain

func TestMemoryChangeLog_Concurrent(t *testing.T)
// - Test concurrent appends
// - Test concurrent reads
// - Test concurrent append and read
```

### 1.3 Store Tests (`store_test.go`)

```go
func TestMemoryStore_SetGet(t *testing.T)
// - Test setting and getting entity
// - Test scoped isolation
// - Test entity type isolation

func TestMemoryStore_Delete(t *testing.T)
// - Test deleting entity
// - Test deleting non-existent entity
// - Test cleanup of empty maps

func TestMemoryStore_List(t *testing.T)
// - Test listing entities by type
// - Test empty list for unknown scope/entity
// - Test list returns all items

func TestMemoryStore_Snapshot(t *testing.T)
// - Test snapshot returns all data for scope
// - Test snapshot is a copy (no shared references)
// - Test empty snapshot for unknown scope

func TestMemoryStore_NotFound(t *testing.T)
// - Test ErrNotFound for missing entity
// - Test ErrNotFound for missing scope
// - Test ErrNotFound for missing entity type

func TestMemoryStore_Concurrent(t *testing.T)
// - Test concurrent sets
// - Test concurrent gets
// - Test concurrent set and get
```

### 1.4 Broker Tests (`broker_test.go`)

```go
func TestNopBroker(t *testing.T)
// - Test NopBroker.Poke does nothing
// - Test NopBroker can be used as interface

func TestFuncBroker(t *testing.T)
// - Test function is called with correct parameters
// - Test multiple pokes

func TestMultiBroker(t *testing.T)
// - Test pokes fan out to all brokers
// - Test adding brokers
// - Test empty MultiBroker
```

### 1.5 Engine Tests (`sync_test.go`)

```go
func TestEngine_New(t *testing.T)
// - Test creation with required options
// - Test default broker when nil

func TestEngine_Push_SingleMutation(t *testing.T)
// - Test applying single mutation
// - Test changes are logged
// - Test result contains cursor
// - Test broker is poked

func TestEngine_Push_MultipleMutations(t *testing.T)
// - Test applying multiple mutations
// - Test each mutation result
// - Test affected scopes are poked

func TestEngine_Push_Error(t *testing.T)
// - Test mutation error is captured in result
// - Test other mutations continue after error
// - Test error does not affect changelog

func TestEngine_Pull(t *testing.T)
// - Test pulling changes from cursor 0
// - Test pulling changes from specific cursor
// - Test hasMore flag
// - Test limit parameter

func TestEngine_Snapshot(t *testing.T)
// - Test snapshot returns data and cursor
// - Test snapshot after mutations
```

### 1.6 Handler Tests (`handler_test.go`)

```go
func TestPushHandler(t *testing.T)
// - Test valid push request
// - Test invalid JSON
// - Test empty mutations
// - Test mutation errors in response

func TestPullHandler(t *testing.T)
// - Test valid pull request
// - Test pull with cursor
// - Test pull with limit
// - Test pull returns hasMore

func TestSnapshotHandler(t *testing.T)
// - Test valid snapshot request
// - Test missing scope error
// - Test snapshot returns data and cursor
```

## 2. Live Package Sync Integration Tests

### 2.1 Protocol Tests (`protocol_test.go` additions)

```go
func TestPokePayload_Encode(t *testing.T)
// - Test encoding poke payload
// - Test MsgTypePoke constant

func TestSubscribePayload_Encode(t *testing.T)
// - Test encoding subscribe payload
// - Test MsgTypeSubscribe constant

func TestUnsubscribePayload_Encode(t *testing.T)
// - Test encoding unsubscribe payload
// - Test MsgTypeUnsubscribe constant

func TestJoinPayload_WithScopes(t *testing.T)
// - Test join payload with scopes field
// - Test scopes are serialized correctly
```

### 2.2 Sync Bridge Tests (`sync_bridge_test.go`)

```go
func TestSyncPokeBroker_Poke(t *testing.T)
// - Test poke publishes to correct topic
// - Test poke message format
// - Test multiple pokes

func TestSyncPokeBroker_Integration(t *testing.T)
// - Test with InmemPubSub
// - Test subscriber receives poke
// - Test multiple subscribers
```

### 2.3 Handler Tests (additions to `handler_test.go`)

```go
func TestSessionHandler_ScopeSubscription(t *testing.T)
// - Test join with scopes subscribes to pubsub
// - Test subscribe message adds subscription
// - Test unsubscribe message removes subscription

func TestSessionHandler_PokeMessage(t *testing.T)
// - Test poke from pubsub is forwarded to client
// - Test poke message format on wire
```

## 3. Integration Tests

### 3.1 Full Sync Flow Test (`sync_integration_test.go`)

```go
func TestSyncFlow_PushPullCycle(t *testing.T)
// Full push/pull cycle:
// 1. Create engine with todo mutator
// 2. Push create mutation
// 3. Pull from cursor 0
// 4. Verify changes match

func TestSyncFlow_MultipleClients(t *testing.T)
// Multiple client simulation:
// 1. Client A pushes mutation
// 2. Client B pulls changes
// 3. Verify B sees A's changes

func TestSyncFlow_OfflineQueue(t *testing.T)
// Offline queue simulation:
// 1. Queue multiple mutations
// 2. Push all at once
// 3. Verify all applied in order
```

### 3.2 Sync + Live Integration Test (`sync_live_integration_test.go`)

```go
func TestSyncLive_PokeAfterPush(t *testing.T)
// 1. Create live with pubsub
// 2. Create sync with SyncPokeBroker
// 3. Subscribe to scope via pubsub
// 4. Push mutation
// 5. Verify poke received

func TestSyncLive_WebSocketPoke(t *testing.T)
// 1. Set up live server
// 2. Connect WebSocket with scope subscription
// 3. Push mutation via HTTP
// 4. Verify poke message received on WebSocket

func TestSyncLive_MultipleScopes(t *testing.T)
// 1. Subscribe to multiple scopes
// 2. Push to different scopes
// 3. Verify correct pokes received
```

### 3.3 HTTP Handler Integration Test (`handler_integration_test.go`)

```go
func TestHTTPHandlers_Mount(t *testing.T)
// 1. Create Mizu app
// 2. Mount sync engine
// 3. Make HTTP requests to endpoints
// 4. Verify responses

func TestHTTPHandlers_PushPull(t *testing.T)
// End-to-end HTTP test:
// 1. POST /_sync/push with mutation
// 2. POST /_sync/pull with cursor 0
// 3. Verify changes in response

func TestHTTPHandlers_Snapshot(t *testing.T)
// 1. Push several mutations
// 2. POST /_sync/snapshot
// 3. Verify full data returned
```

## 4. Performance Tests

### 4.1 Benchmark Tests (`benchmark_test.go`)

```go
func BenchmarkMemoryChangeLog_Append(b *testing.B)
// Benchmark append operations

func BenchmarkMemoryChangeLog_Since(b *testing.B)
// Benchmark since queries

func BenchmarkMemoryStore_SetGet(b *testing.B)
// Benchmark store operations

func BenchmarkEngine_Push(b *testing.B)
// Benchmark push with mutations

func BenchmarkEngine_Pull(b *testing.B)
// Benchmark pull operations

func BenchmarkPokeBroker_Fanout(b *testing.B)
// Benchmark poke fanout to many subscribers
```

## 5. Test Utilities

### 5.1 Test Helpers

```go
// todoMutator creates a test mutator for todo operations.
func todoMutator() sync.Mutator

// createTestEngine creates an engine with in-memory stores.
func createTestEngine() *sync.Engine

// createTestEngineWithBroker creates an engine with a custom broker.
func createTestEngineWithBroker(broker sync.PokeBroker) *sync.Engine

// pushAndVerify pushes a mutation and verifies the result.
func pushAndVerify(t *testing.T, e *sync.Engine, mutation sync.Mutation, expectOK bool)

// pullAndVerify pulls changes and verifies expected changes.
func pullAndVerify(t *testing.T, e *sync.Engine, scope string, cursor uint64, expectedCount int)
```

### 5.2 Mock Implementations

```go
// MockBroker records poke calls for verification.
type MockBroker struct {
    Pokes []Poke
    mu    sync.Mutex
}

// MockMutator returns predefined changes.
type MockMutator struct {
    Changes []sync.Change
    Err     error
}
```

## 6. Test Coverage Goals

| Component | Target Coverage |
|-----------|-----------------|
| mutation.go | 95% |
| changelog.go | 95% |
| store.go | 95% |
| broker.go | 100% |
| sync.go | 90% |
| handler.go | 85% |
| sync_bridge.go | 100% |
| protocol.go (new) | 100% |

## 7. Test Execution

```bash
# Run all sync tests
go test ./sync/... -v

# Run with coverage
go test ./sync/... -coverprofile=coverage.out

# Run live sync integration tests
go test ./view/live/... -run Sync -v

# Run benchmarks
go test ./sync/... -bench=. -benchmem

# Run race detector
go test ./sync/... -race
```

## 8. Edge Cases to Test

1. **Empty states**
   - Empty mutation list
   - Empty change log
   - Empty store
   - Zero cursor

2. **Boundary conditions**
   - Very large mutations
   - Many concurrent operations
   - Cursor overflow (uint64 max)

3. **Error conditions**
   - Invalid mutation
   - Unknown mutation type
   - Store errors
   - Changelog errors

4. **Concurrency**
   - Parallel push requests
   - Parallel pull requests
   - Push while pulling
   - Multiple subscribers receiving pokes

5. **Cleanup**
   - Changelog trimming
   - Session unsubscribe
   - Connection close during poke
