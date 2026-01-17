# Spec 0389: Realtime WebSocket API (Supabase Realtime Compatible) Testing Plan

## Document Info

| Field | Value |
|-------|-------|
| Spec ID | 0389 |
| Version | 1.0 |
| Date | 2025-01-17 |
| Status | **In Progress** |
| Priority | Critical |
| Estimated Tests | 150+ |
| Supabase Realtime Version | Latest (Phoenix Protocol 1.0.0/2.0.0) |
| Supabase Local Port | 54021 |
| Localbase Port | 54321 |

## Overview

This document outlines a comprehensive testing plan for the Localbase Realtime WebSocket API to achieve 100% compatibility with Supabase's Realtime implementation. The Realtime API enables three primary features:

1. **Broadcast** - Ephemeral client-to-client messaging with low latency
2. **Presence** - Track and synchronize shared state between clients
3. **Postgres Changes** - Listen to database changes and stream to authorized clients

### Testing Philosophy

- **No mocks**: All tests run against real WebSocket servers
- **Side-by-side comparison**: Every message format verified against Supabase
- **Comprehensive coverage**: Every message type, event, and error condition
- **Protocol accuracy**: Phoenix protocol compliance (versions 1.0.0 and 2.0.0)
- **Response matching**: Message structures and error codes must match exactly

### Compatibility Target

| Aspect | Target |
|--------|--------|
| WebSocket Protocol | 100% Phoenix Protocol Compatible |
| Message Format | 100% match |
| Event Types | 100% match |
| Error Codes | 100% match |
| Rate Limiting Behavior | 100% match |

## Reference Documentation

### Official Supabase Realtime Documentation
- [Supabase Realtime Protocol](https://supabase.com/docs/guides/realtime/protocol)
- [Supabase Realtime Concepts](https://supabase.com/docs/guides/realtime/concepts)
- [Supabase Realtime Broadcast](https://supabase.com/docs/guides/realtime/broadcast)
- [Supabase Realtime Presence](https://supabase.com/docs/guides/realtime/presence)
- [Supabase Realtime Postgres Changes](https://supabase.com/docs/guides/realtime/subscribing-to-database-changes)
- [Supabase Realtime GitHub](https://github.com/supabase/realtime)

### Phoenix Protocol Documentation
- Phoenix Channels Protocol (basis for Supabase Realtime)

## Test Environment Setup

### Supabase Local Configuration
```
WebSocket: ws://127.0.0.1:54021/realtime/v1/websocket?apikey=[API_KEY]&vsn=1.0.0
REST API: http://127.0.0.1:54021/realtime/v1
Database: postgresql://postgres:postgres@127.0.0.1:54322/postgres
API Key: sb_publishable_ACJWlzQHlZjBrEguHvfOxg_3BJgxAaH
```

### Localbase Configuration
```
WebSocket: ws://localhost:54321/realtime/v1/websocket?apikey=[API_KEY]&vsn=1.0.0
REST API: http://localhost:54321/realtime/v1
Database: postgresql://localbase:localbase@localhost:5432/localbase
API Key: sb_publishable_ACJWlzQHlZjBrEguHvfOxg_3BJgxAaH
```

---

## Protocol Specification

### Connection URL Format
```
ws[s]://{host}/realtime/v1/websocket?apikey={api_key}&vsn={version}&log_level={level}
```

**Parameters:**
- `apikey` (required): API key for authentication (anon or service_role)
- `vsn` (optional): Protocol version, "1.0.0" (default, JSON) or "2.0.0" (binary)
- `log_level` (optional): Logging level (info, debug, error)

### Protocol Version 1.0.0 (JSON)
All messages are JSON encoded with the following structure:
```json
{
  "topic": string,
  "event": string,
  "payload": object,
  "ref": string | null,
  "join_ref": string | null
}
```

### Protocol Version 2.0.0 (Binary/Array)
Messages use a JSON array format:
```json
[join_ref, ref, topic, event, payload]
```

---

## Message Types

### 1. Connection Messages

#### 1.1 phx_join - Channel Join
Client sends to subscribe to a channel with configuration.

**Request Format:**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "phx_join",
  "payload": {
    "config": {
      "broadcast": {
        "self": boolean,
        "ack": boolean
      },
      "presence": {
        "key": string
      },
      "postgres_changes": [
        {
          "event": "*" | "INSERT" | "UPDATE" | "DELETE",
          "schema": string,
          "table": string,
          "filter": string
        }
      ],
      "private": boolean
    },
    "access_token": string
  },
  "ref": string
}
```

**Response Format (success):**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "phx_reply",
  "payload": {
    "status": "ok",
    "response": {
      "postgres_changes": [
        {
          "id": number,
          "event": string,
          "schema": string,
          "table": string,
          "filter": string
        }
      ]
    }
  },
  "ref": string
}
```

#### 1.2 phx_leave - Channel Leave
Client sends to unsubscribe from a channel.

**Request Format:**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "phx_leave",
  "payload": {},
  "ref": string
}
```

#### 1.3 heartbeat - Keep-Alive
Client sends every 25 seconds (default) to maintain connection.

**Request Format:**
```json
{
  "topic": "phoenix",
  "event": "heartbeat",
  "payload": {},
  "ref": string
}
```

**Response Format:**
```json
{
  "topic": "phoenix",
  "event": "phx_reply",
  "payload": {
    "status": "ok",
    "response": {}
  },
  "ref": string
}
```

### 2. Broadcast Messages

#### 2.1 broadcast - Send Message
Client sends broadcast message to all subscribers.

**Request Format:**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "broadcast",
  "payload": {
    "type": "broadcast",
    "event": string,
    "payload": object
  },
  "ref": string
}
```

**Received Format (by subscribers):**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "broadcast",
  "payload": {
    "type": "broadcast",
    "event": string,
    "payload": object
  },
  "ref": null
}
```

### 3. Presence Messages

#### 3.1 presence - Track State
Client sends to track presence state.

**Request Format:**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "presence",
  "payload": {
    "type": "presence",
    "event": "track",
    "payload": object
  },
  "ref": string
}
```

#### 3.2 presence_state - Full State
Server sends full presence state after joining.

**Response Format:**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "presence_state",
  "payload": {
    "{presence_key}": {
      "metas": [
        {
          "phx_ref": string,
          ...custom_fields
        }
      ]
    }
  },
  "ref": null
}
```

#### 3.3 presence_diff - State Changes
Server sends when presence state changes.

**Response Format:**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "presence_diff",
  "payload": {
    "joins": {
      "{presence_key}": {
        "metas": [...]
      }
    },
    "leaves": {
      "{presence_key}": {
        "metas": [...]
      }
    }
  },
  "ref": null
}
```

### 4. Postgres Changes Messages

#### 4.1 postgres_changes - Database Event
Server sends when subscribed database changes occur.

**Response Format:**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "postgres_changes",
  "payload": {
    "data": {
      "columns": [
        {"name": string, "type": string}
      ],
      "commit_timestamp": string,
      "errors": null | object,
      "old_record": object | null,
      "record": object,
      "schema": string,
      "table": string,
      "type": "INSERT" | "UPDATE" | "DELETE"
    },
    "ids": [number]
  },
  "ref": null
}
```

### 5. Token Messages

#### 5.1 access_token - Refresh Token
Client sends to refresh access token.

**Request Format:**
```json
{
  "topic": "realtime:{channel_name}",
  "event": "access_token",
  "payload": {
    "access_token": string
  },
  "ref": string
}
```

---

## Test Cases

### 1. Connection Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| CONN-001 | Connect with valid anon key | `ws://...?apikey={anon_key}` | Connection established |
| CONN-002 | Connect with valid service key | `ws://...?apikey={service_key}` | Connection established |
| CONN-003 | Connect without API key | `ws://...` | 401 Unauthorized |
| CONN-004 | Connect with invalid API key | `ws://...?apikey=invalid` | 401 Unauthorized |
| CONN-005 | Connect with expired JWT | `ws://...?apikey={expired_jwt}` | 401 Unauthorized |
| CONN-006 | Connect with protocol v1.0.0 | `ws://...?vsn=1.0.0` | JSON messages |
| CONN-007 | Connect with protocol v2.0.0 | `ws://...?vsn=2.0.0` | Array/binary messages |
| CONN-008 | Connect with default protocol | `ws://...` | v1.0.0 (JSON) |
| CONN-009 | Connect with log_level=info | `ws://...&log_level=info` | Connection with logging |
| CONN-010 | Reconnect after disconnect | Disconnect and reconnect | New connection established |

### 2. Heartbeat Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| HEART-001 | Send heartbeat on phoenix topic | `{"topic":"phoenix","event":"heartbeat","payload":{},"ref":"1"}` | `phx_reply` with `"status":"ok"` |
| HEART-002 | Receive heartbeat response | Send heartbeat | Response within 1s |
| HEART-003 | Connection timeout without heartbeat | Wait >30s without heartbeat | Connection closed |
| HEART-004 | Maintain connection with periodic heartbeats | Heartbeat every 25s | Connection stays open |
| HEART-005 | Multiple rapid heartbeats | 10 heartbeats in 1s | All acknowledged |
| HEART-006 | Heartbeat ref tracking | Send with unique refs | Responses match refs |

### 3. Channel Join Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| JOIN-001 | Join public channel | `phx_join` on `realtime:public-channel` | `phx_reply` with `"status":"ok"` |
| JOIN-002 | Join private channel with valid token | `phx_join` with access_token | `phx_reply` with `"status":"ok"` |
| JOIN-003 | Join private channel without token | `phx_join` without access_token | `phx_reply` with `"status":"error"` |
| JOIN-004 | Join with broadcast config | `config.broadcast.self=true` | Self receives broadcasts |
| JOIN-005 | Join with broadcast ack | `config.broadcast.ack=true` | Receives acknowledgments |
| JOIN-006 | Join with presence config | `config.presence.key="user_1"` | Presence tracked with key |
| JOIN-007 | Join with postgres_changes config | Subscribe to INSERT events | Receives INSERT notifications |
| JOIN-008 | Join with wildcard schema | `schema: "*"` | All schema changes received |
| JOIN-009 | Join with wildcard table | `table: "*"` | All table changes received |
| JOIN-010 | Join with filter | `filter: "user_id=eq.123"` | Only matching rows received |
| JOIN-011 | Join multiple channels | Join 5 different channels | All joins successful |
| JOIN-012 | Join same channel twice | Duplicate join | Returns existing subscription |
| JOIN-013 | Join with invalid topic format | `topic: "invalid"` | Error response |
| JOIN-014 | Join with malformed payload | Invalid JSON | Error response |
| JOIN-015 | Join ref tracking | Unique join_ref | Response matches join_ref |

### 4. Channel Leave Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| LEAVE-001 | Leave joined channel | `phx_leave` after join | `phx_reply` with `"status":"ok"` |
| LEAVE-002 | Leave non-joined channel | `phx_leave` without join | Error or no-op |
| LEAVE-003 | Leave all channels | Leave each joined channel | All leaves successful |
| LEAVE-004 | Rejoin after leave | Leave then rejoin | New subscription created |
| LEAVE-005 | Messages after leave | Send message after leave | Not received |

### 5. Broadcast Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| BCAST-001 | Send broadcast to channel | `type: "broadcast"` message | Delivered to subscribers |
| BCAST-002 | Receive own broadcast (self=true) | Broadcast with `self=true` | Sender receives message |
| BCAST-003 | No self receive (self=false) | Broadcast with `self=false` | Sender does not receive |
| BCAST-004 | Broadcast with acknowledgment | Broadcast with `ack=true` | Receives ack response |
| BCAST-005 | Broadcast to multiple subscribers | 3 clients on same channel | All 3 receive message |
| BCAST-006 | Broadcast with custom event | `event: "custom_event"` | Event name preserved |
| BCAST-007 | Broadcast with complex payload | Nested JSON object | Payload preserved exactly |
| BCAST-008 | Broadcast with array payload | JSON array payload | Array preserved |
| BCAST-009 | Broadcast empty payload | `payload: {}` | Empty payload delivered |
| BCAST-010 | Broadcast null payload | `payload: null` | Null payload delivered |
| BCAST-011 | Rapid broadcast (100 msgs/s) | 100 messages in 1 second | All delivered or rate limited |
| BCAST-012 | Large broadcast payload | 10KB payload | Delivered or size limited |
| BCAST-013 | Broadcast before join | Send without joining | Error response |
| BCAST-014 | Cross-channel isolation | Broadcast on channel A | Channel B does not receive |
| BCAST-015 | Unicode in broadcast | Unicode characters in payload | Preserved correctly |

### 6. Presence Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| PRES-001 | Track presence after join | `presence` with `event: "track"` | `presence_state` received |
| PRES-002 | Receive initial presence_state | Join with presence config | Full state snapshot |
| PRES-003 | Receive presence_diff on join | New client joins | `joins` contains new client |
| PRES-004 | Receive presence_diff on leave | Client leaves | `leaves` contains client |
| PRES-005 | Custom presence key | `presence.key: "user_123"` | Key used in state |
| PRES-006 | Auto-generated presence key | No key specified | UUID generated |
| PRES-007 | Presence with custom metadata | `{ online_at: timestamp }` | Metadata preserved |
| PRES-008 | Untrack presence | `event: "untrack"` | Client removed from state |
| PRES-009 | Multiple presence entries | Same key, multiple tabs | Array of metas |
| PRES-010 | Presence sync event | After presence changes | `sync` event fired |
| PRES-011 | Presence phx_ref tracking | Each meta has phx_ref | Unique refs per meta |
| PRES-012 | Large presence state | 100 tracked clients | All clients in state |
| PRES-013 | Presence cleanup on disconnect | Client disconnects | Auto-removed from state |
| PRES-014 | Cross-channel presence isolation | Track on channel A | Channel B state unaffected |
| PRES-015 | Presence update | Update existing presence | Diff shows update |

### 7. Postgres Changes Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| PG-001 | Subscribe to INSERT events | `event: "INSERT"` | Receive INSERT notifications |
| PG-002 | Subscribe to UPDATE events | `event: "UPDATE"` | Receive UPDATE notifications |
| PG-003 | Subscribe to DELETE events | `event: "DELETE"` | Receive DELETE notifications |
| PG-004 | Subscribe to all events (*) | `event: "*"` | Receive all change types |
| PG-005 | Filter by schema | `schema: "public"` | Only public schema changes |
| PG-006 | Filter by table | `table: "users"` | Only users table changes |
| PG-007 | Filter with eq operator | `filter: "id=eq.1"` | Only matching rows |
| PG-008 | Filter with neq operator | `filter: "status=neq.deleted"` | Exclude matching rows |
| PG-009 | Filter with gt operator | `filter: "age=gt.18"` | Greater than filter |
| PG-010 | Filter with gte operator | `filter: "age=gte.18"` | Greater than or equal |
| PG-011 | Filter with lt operator | `filter: "age=lt.65"` | Less than filter |
| PG-012 | Filter with lte operator | `filter: "age=lte.65"` | Less than or equal |
| PG-013 | Filter with in operator | `filter: "status=in.(active,pending)"` | In list filter |
| PG-014 | Wildcard schema subscription | `schema: "*"` | All schemas |
| PG-015 | Wildcard table subscription | `table: "*"` | All tables in schema |
| PG-016 | Subscription ID in response | Join response | Contains unique ID |
| PG-017 | Column info in payload | INSERT event | Columns with types |
| PG-018 | commit_timestamp in payload | Any change event | Timestamp present |
| PG-019 | old_record in UPDATE | UPDATE event | Old values included |
| PG-020 | old_record null in INSERT | INSERT event | old_record is null |
| PG-021 | record in INSERT/UPDATE | INSERT/UPDATE | New values included |
| PG-022 | record null in DELETE | DELETE event | Only old_record present |
| PG-023 | Multiple subscription IDs | Subscribe to multiple tables | Each has unique ID |
| PG-024 | RLS enforcement | Anon user subscription | Only authorized rows |
| PG-025 | Service role bypass RLS | Service key subscription | All rows visible |
| PG-026 | Subscription after table change | Create table, subscribe | Notifications work |
| PG-027 | Error in changes payload | Database error | errors field populated |
| PG-028 | High-frequency changes | 100 inserts/second | All changes received |
| PG-029 | Transaction batching | Multiple changes in tx | Ordered delivery |
| PG-030 | Binary data in changes | BYTEA column | Properly encoded |

### 8. Access Token Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| TOKEN-001 | Refresh access token | `access_token` event | Token updated |
| TOKEN-002 | Invalid token refresh | Expired/invalid token | Error response |
| TOKEN-003 | Token refresh on private channel | New valid token | Channel stays subscribed |
| TOKEN-004 | Token expiry handling | Token expires while connected | Graceful handling |
| TOKEN-005 | Token with different claims | New user claims | Permissions updated |

### 9. Error Handling Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| ERR-001 | Invalid message format | Malformed JSON | Error response |
| ERR-002 | Unknown event type | `event: "unknown"` | Error or ignored |
| ERR-003 | Missing required fields | No topic in message | Error response |
| ERR-004 | Rate limit exceeded | >100 events/second | Rate limit error |
| ERR-005 | Max channels exceeded | >100 channels | Channel limit error |
| ERR-006 | Max connections per channel | >200 concurrent | Connection limit error |
| ERR-007 | Payload size exceeded | >100KB payload | Size limit error |
| ERR-008 | Invalid topic format | `topic: "///invalid"` | Format error |
| ERR-009 | Database connection error | DB unavailable | Connection error |
| ERR-010 | Authorization error | Unauthorized action | Auth error code |

### 10. Protocol v2.0.0 Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| V2-001 | Array message format | `[join_ref, ref, topic, event, payload]` | Parsed correctly |
| V2-002 | Text frame encoding | JSON array text | Standard handling |
| V2-003 | Binary frame encoding | Binary WebSocket frame | Decoded correctly |
| V2-004 | Mixed frame types | Text and binary | Both handled |
| V2-005 | Backwards compatibility | v1 client, v2 server | Negotiated properly |

### 11. REST API Broadcast Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| REST-001 | HTTP broadcast to channel | POST with broadcast payload | Delivered to WS clients |
| REST-002 | Broadcast without WS connection | HTTP only | Message buffered/delivered |
| REST-003 | REST broadcast authentication | Valid API key | 200 OK |
| REST-004 | REST broadcast unauthorized | Invalid/no API key | 401 Unauthorized |
| REST-005 | REST broadcast to non-existent channel | POST to unknown channel | Handled gracefully |

### 12. Performance & Stress Tests

| Test ID | Description | Input | Expected Output |
|---------|-------------|-------|-----------------|
| PERF-001 | Concurrent connections | 1000 simultaneous | All connect |
| PERF-002 | Message throughput | 10000 msgs/min | All delivered |
| PERF-003 | Large presence state | 1000 tracked users | State synced |
| PERF-004 | Rapid join/leave | 100 joins/leaves per second | All processed |
| PERF-005 | Connection recovery | Kill connections, reconnect | Graceful recovery |
| PERF-006 | Memory stability | 24h sustained load | No memory leaks |
| PERF-007 | CPU under load | Peak message rate | Reasonable CPU usage |

---

## Rate Limits (Supabase Defaults)

| Limit | Default Value |
|-------|---------------|
| Concurrent users per channel | 200 |
| Maximum channels per client | 100 |
| Events per second | 100 |
| Channel joins per second | 100 |
| Bytes per second | 100,000 |

---

## Error Codes Reference

| Code | Description |
|------|-------------|
| `ChannelRateLimitReached` | Channel rate limit exceeded |
| `UnableToConnectToTenantDatabase` | Database connection failed |
| `JwtSignatureError` | Invalid JWT signature |
| `JwtExpired` | JWT token expired |
| `JwtInvalidClaims` | Missing or invalid JWT claims |
| `UnauthorizedChannel` | Not authorized for private channel |
| `MessageSizeLimitExceeded` | Payload too large |
| `MaxChannelsReached` | Client channel limit exceeded |
| `MaxConnectionsReached` | Channel connection limit exceeded |
| `InvalidMessageFormat` | Malformed message structure |

---

## Implementation Notes

### Current Localbase Implementation Status

**Implemented:**
- WebSocket connection handling with gorilla/websocket
- API key authentication (anon, service_role)
- Basic message echo functionality
- Connection tracking with client registry
- Origin validation (SEC-016)
- Token authentication (SEC-017)
- Channel and subscription database schema

**To Be Enhanced:**
- Full Phoenix protocol message handling
- Broadcast routing to channel subscribers
- Presence tracking and state synchronization
- Postgres Changes (CDC) integration
- Rate limiting
- Protocol v2.0.0 support
- REST broadcast API

### Database Schema

```sql
-- Realtime channels
CREATE TABLE realtime.channels (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT UNIQUE NOT NULL,
  inserted_at TIMESTAMPTZ DEFAULT NOW()
);

-- Channel subscriptions
CREATE TABLE realtime.subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  channel_id UUID NOT NULL REFERENCES realtime.channels(id),
  user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
  filters JSONB DEFAULT '{}',
  claims JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Realtime messages table for authorization
CREATE TABLE realtime.messages (
  id BIGINT GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
  topic TEXT NOT NULL,
  extension TEXT NOT NULL CHECK (extension IN ('broadcast', 'presence')),
  inserted_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Test Implementation Guide

### Test File Location
```
test/integration/realtime_test.go
```

### Test Dependencies
```go
import (
    "github.com/gorilla/websocket"
    "testing"
    "encoding/json"
    "time"
    "sync"
)
```

### Test Helper Functions

```go
// ConnectWebSocket establishes a WebSocket connection
func ConnectWebSocket(t *testing.T, apiKey string) *websocket.Conn

// SendMessage sends a JSON message over WebSocket
func SendMessage(conn *websocket.Conn, msg interface{}) error

// ReceiveMessage receives and parses a JSON message
func ReceiveMessage(conn *websocket.Conn) (map[string]interface{}, error)

// JoinChannel joins a realtime channel with config
func JoinChannel(conn *websocket.Conn, topic string, config interface{}) error

// LeaveChannel leaves a realtime channel
func LeaveChannel(conn *websocket.Conn, topic string) error

// SendHeartbeat sends a heartbeat message
func SendHeartbeat(conn *websocket.Conn, ref string) error

// Broadcast sends a broadcast message
func Broadcast(conn *websocket.Conn, topic, event string, payload interface{}) error
```

---

## Verification Checklist

- [ ] All CONN-xxx tests pass
- [ ] All HEART-xxx tests pass
- [ ] All JOIN-xxx tests pass
- [ ] All LEAVE-xxx tests pass
- [ ] All BCAST-xxx tests pass
- [ ] All PRES-xxx tests pass
- [ ] All PG-xxx tests pass
- [ ] All TOKEN-xxx tests pass
- [ ] All ERR-xxx tests pass
- [ ] All V2-xxx tests pass
- [ ] All REST-xxx tests pass
- [ ] All PERF-xxx tests pass
- [ ] Side-by-side comparison with Supabase passes
- [ ] Message format exactly matches Supabase
- [ ] Error codes exactly match Supabase

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-01-17 | Initial specification |
