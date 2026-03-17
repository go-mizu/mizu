# 0743 — Direct Messages

Add a first-class direct message API to chat.now. Today, `kind: "direct"` is
just a room with a 2-member cap — no deduplication, no auto-invite, not
private by default. This spec fixes all of that.

## Design

### POST /api/chat/dm

Start or resume a direct conversation with another actor.

```
POST /api/chat/dm
Authorization: CHAT-ED25519 ...
Content-Type: application/json

{ "peer": "u/bob" }
```

**Behavior:**

1. Validate `peer` is a valid actor format and is a registered actor
2. Reject if `peer` is the caller (can't DM yourself)
3. Look for an existing direct chat between the two actors
   - Query: find a chat with `kind = 'direct'` where both actors are members
4. If found, return the existing chat (200)
5. If not found, create a new one:
   - `kind: "direct"`, `visibility: "private"`, `title: ""`
   - Auto-add both actors as members
   - Return the new chat (201)

**Response (200 or 201):**
```json
{
  "id": "c_abc123",
  "kind": "direct",
  "title": "",
  "creator": "u/alice",
  "peer": "u/bob",
  "created_at": "2026-03-17T..."
}
```

The `peer` field is always the other actor (not the caller). This makes it
easy for clients to show "DM with Bob" regardless of who created it.

### GET /api/chat/dm

List all direct message conversations for the authenticated actor.

```
GET /api/chat/dm
Authorization: CHAT-ED25519 ...
```

**Response (200):**
```json
{
  "items": [
    {
      "id": "c_abc123",
      "kind": "direct",
      "title": "",
      "creator": "u/alice",
      "peer": "u/bob",
      "created_at": "2026-03-17T..."
    }
  ]
}
```

Returns all direct chats where the caller is a member, sorted by most
recent message (or creation time if no messages). Each item includes the
`peer` field showing the other party.

### Existing endpoints — behavioral changes

- `POST /api/chat` with `kind: "direct"` → returns 400 with error
  "Use POST /api/chat/dm to create direct messages". This prevents the
  old unprotected flow.
- `POST /api/chat/:id/join` on a direct chat → returns 403 "Cannot join
  direct chat" (already enforced by the 2-member limit, but make it explicit)
- Sending and reading messages on DMs works exactly like rooms (no changes)

### Schema

No schema changes. Direct chats use the existing `chats` and `members`
tables. The deduplication query uses an intersection of memberships:

```sql
SELECT c.id FROM chats c
JOIN members m1 ON m1.chat_id = c.id AND m1.actor = ?
JOIN members m2 ON m2.chat_id = c.id AND m2.actor = ?
WHERE c.kind = 'direct'
LIMIT 1
```

### Security

- Both actors must be registered (verified against `actors` table)
- DMs are always private (`visibility = 'private'`) — non-members get 404
- Only the two members can send/read messages (existing membership enforcement)
- The caller must be authenticated (Ed25519 signature auth)

### DX

- Single endpoint to start or resume a DM — no need to create then join
- Peer name validation at creation time (fail fast if peer doesn't exist)
- Response always includes `peer` field for easy UI rendering
- `GET /api/chat/dm` for listing all DMs separately from rooms

## Acceptance Criteria

1. `POST /api/chat/dm` with valid peer creates a new DM, both actors auto-joined
2. Calling again with same peer returns existing DM (idempotent, 200 not 201)
3. Peer sees the DM in their `GET /api/chat/dm` list
4. Both can send and read messages
5. Non-member gets 404 when reading DM messages
6. DM self → 400
7. DM non-existent actor → 404
8. `POST /api/chat` with `kind: "direct"` → 400
9. `POST /api/chat/:id/join` on DM → 403
10. `peer` field in response always shows the other actor
11. Docs updated with DM section and working examples
