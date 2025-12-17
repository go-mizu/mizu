# Live Package Review Implementation Plan

This document outlines the implementation plan to address the issues identified in the code review (0039_live_review.md).

## Summary of Issues

The review identified 16 issues across four categories:
- **Critical protocol correctness issues** (1-5): Must fix for RFC 6455 compliance
- **Security and robustness concerns** (6-11): Important for production safety
- **API and DX improvements** (12-15): Nice-to-have enhancements
- **WebSocket implementation choice** (16): Design consideration

## Implementation Plan

### Phase 1: Critical Protocol Fixes (Must Fix)

#### 1.1 WebSocket Handshake Validation

**Issue #1: WebSocket handshake is incomplete**

Location: `live/live.go:handleConn()` and `isWebSocketUpgrade()`

Changes:
- Add `Sec-WebSocket-Version: 13` validation
- Validate `Sec-WebSocket-Key` is valid base64, 16 bytes when decoded
- Return `426 Upgrade Required` with `Sec-WebSocket-Version: 13` header on version mismatch

```go
// Add new error
var ErrInvalidVersion = errors.New("live: unsupported WebSocket version")

// Add validation function
func validateWebSocketKey(key string) bool {
    decoded, err := base64.StdEncoding.DecodeString(key)
    return err == nil && len(decoded) == 16
}
```

#### 1.2 Require Masked Client Frames

**Issue #2: Accepting unmasked client frames**

Location: `live/live.go:readMessage()`

Changes:
- Check if `masked == false` on client frames
- Close connection with protocol error (1002) if unmasked

```go
// Add protocol error codes
const (
    wsCloseNormal    = 1000
    wsCloseProtocol  = 1002
    wsCloseTooLarge  = 1009
)
```

#### 1.3 Fragmentation Handling

**Issue #3: FIN bit and continuation frames ignored**

Location: `live/live.go:readMessage()`

Decision: **Reject fragmented frames** (minimal server approach)

Changes:
- Check FIN bit in first byte
- Reject frames with FIN=0 or opcode=0 (continuation)
- Close with protocol error status

#### 1.4 Fix 64-bit Payload Length Parsing

**Issue #4: Payload length parsing for 127 incorrect**

Location: `live/live.go:readMessage()` lines 744-748

Changes:
- Parse all 8 bytes as uint64 in network byte order
- Reject values > MaxInt or configured maximum
- Use `encoding/binary.BigEndian.Uint64()`

```go
case 127:
    lenBytes := make([]byte, 8)
    if _, err := io.ReadFull(ws.reader, lenBytes); err != nil {
        return 0, nil, err
    }
    length64 := binary.BigEndian.Uint64(lenBytes)
    if length64 > uint64(ws.readLimit) {
        return 0, nil, ErrMessageTooLarge
    }
    length = int(length64)
```

#### 1.5 Control Frame Constraints

**Issue #5: Control frame constraints not enforced**

Location: `live/live.go:readMessage()`

Changes:
- Enforce control frames (8-10) have payload <= 125 bytes
- Enforce control frames are not fragmented (FIN must be set)

### Phase 2: Security and Robustness

#### 2.1 Message Size Limits

**Issue #6: No message size limits**

Location: `live/live.go:Options` and `readMessage()`

Changes:
- Add `ReadLimit` to Options (default 4MB)
- Check payload length before allocation
- Close with 1009 (message too big) if exceeded

```go
type Options struct {
    // ... existing fields ...

    // ReadLimit is the maximum message size in bytes. Default: 4MB.
    // Messages exceeding this limit will cause the connection to be closed.
    ReadLimit int
}

const defaultReadLimit = 4 * 1024 * 1024 // 4MB
```

#### 2.2 Origin Check Enhancement

**Issue #7: Origin check too naive**

Location: `live/live.go:Options` and `handleConn()`

Changes:
- Add `CheckOrigin func(*http.Request) bool` callback to Options
- Keep `Origins []string` for simple cases
- `CheckOrigin` takes precedence if set

```go
type Options struct {
    // ... existing fields ...

    // CheckOrigin validates the Origin header. Return true to allow.
    // If nil and Origins is empty, all origins are allowed.
    // If nil and Origins is set, exact string matching is used.
    CheckOrigin func(r *http.Request) bool
}
```

#### 2.3 Goroutine Lifetime

**Issue #10: Goroutine lifetime issues**

Location: `live/live.go:writeLoop()` and `Session`

Changes:
- Store net.Conn reference in Session
- On write failure in writeLoop, close the underlying conn to unblock readLoop
- On server-initiated close (queue full), close the underlying conn

```go
type Session struct {
    // ... existing fields ...
    conn net.Conn // Store conn reference for cleanup
}

func (s *Session) closeWithError(err error) error {
    if !s.closed.CompareAndSwap(false, true) {
        return nil
    }
    close(s.doneCh)
    if s.conn != nil {
        s.conn.Close() // Unblock readLoop
    }
    return nil
}
```

#### 2.4 Atomic Session Counter

**Issue #11: SessionCount is O(n)**

Location: `live/live.go:Server`

Changes:
- Add atomic counter for session count
- Increment on addSession, decrement on removeSession

```go
type Server struct {
    // ... existing fields ...
    sessionCount atomic.Int64
}

func (srv *Server) SessionCount() int {
    return int(srv.sessionCount.Load())
}
```

### Phase 3: API Improvements (Deferred)

These items are noted but deferred as they are non-critical:

- **Issue #8**: Auth/origin order - Current order is acceptable
- **Issue #9**: Close reason storage - Optional, not required
- **Issue #12**: Body as []byte - Breaking change, defer to v2
- **Issue #13**: Clarify Type field - Doc improvement only
- **Issue #14**: Subscription tracking - Current approach is correct
- **Issue #15**: Helper methods - Can add incrementally

### Phase 4: Testing

Update tests to verify:
- WebSocket version validation rejects non-13 versions
- Invalid WebSocket key rejected
- Unmasked client frames rejected
- Fragmented frames rejected with protocol error
- 64-bit payload lengths parsed correctly
- Control frames > 125 bytes rejected
- Messages > ReadLimit rejected
- CheckOrigin callback used when provided
- Session count is accurate
- Write failures close underlying connection

## File Changes Summary

### live/live.go

1. Add imports: `encoding/binary`
2. Add new error variables: `ErrInvalidVersion`, `ErrMessageTooLarge`, `ErrProtocolError`
3. Add close code constants
4. Modify `Options`: add `ReadLimit`, `CheckOrigin`
5. Modify `Server`: add `sessionCount atomic.Int64`
6. Modify `Session`: add `conn net.Conn`
7. Modify `handleConn()`: add version and key validation
8. Modify `readMessage()`: add FIN check, masked requirement, size limits, control frame validation
9. Modify `writeLoop()`: close conn on write error
10. Modify `closeWithError()`: close underlying conn
11. Modify `addSession()`/`removeSession()`: update atomic counter
12. Modify `SessionCount()`: use atomic counter

### live/live_test.go

1. Add tests for version validation
2. Add tests for key validation
3. Add tests for unmasked frame rejection
4. Add tests for fragmented frame rejection
5. Add tests for 64-bit payload length
6. Add tests for control frame constraints
7. Add tests for read limits
8. Add tests for CheckOrigin callback
9. Update existing tests that use unmasked frames

### middlewares/websocket/websocket.go

Apply same protocol fixes:
1. Add version validation
2. Add key validation
3. Add read limits
4. Fix 64-bit length parsing
5. Add masked frame requirement

## Implementation Order

1. **Critical protocol fixes** - Required for RFC compliance
2. **Security fixes** - Required for production safety
3. **Test updates** - Verify all changes work correctly
4. **Documentation** - Update package docs to reflect new options

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing clients | High | Clients must mask frames per RFC - non-compliant clients should be fixed |
| Performance impact of validation | Low | Validation is O(1) per message |
| Memory for read limits | Low | Limit is checked before allocation |
