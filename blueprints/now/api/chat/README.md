# api/chat

HTTP API for chat. This layer maps HTTP requests to `pkg/chat.API`. It stays thin and does not contain business logic.

## Overview

Routes are mounted under `/api/chat`. Each handler parses input from path, query, or body, calls the chat API, and returns JSON.

## Routes

Create chat

```
POST /api/chat
```

Body:

```json
{
  "kind": "room",
  "title": "general",
  "visibility": "public"
}
```

Response:

```json
{
  "id": "c_123",
  "kind": "room",
  "title": "general",
  "creator": "u_1",
  "created_at": "2026-01-01T00:00:00Z"
}
```

Join chat

```
POST /api/chat/{id}/join
```

Body:

```json
{
  "token": "optional"
}
```

Response: `204 No Content`

Get chat

```
GET /api/chat/{id}
```

List chats

```
GET /api/chat?kind=room&limit=20
```

Send message

```
POST /api/chat/{id}/messages
```

Body:

```json
{
  "text": "hello"
}
```

Response:

```json
{
  "id": "m_1",
  "chat": "c_123",
  "actor": "u_1",
  "text": "hello",
  "created_at": "2026-01-01T00:00:00Z"
}
```

List messages

```
GET /api/chat/{id}/messages?limit=20&before=m_10
```

Response:

```json
{
  "items": []
}
```

## Mapping

HTTP layer maps directly to the domain API.

Create → `API.Create`
Join → `API.Join`
Get → `API.Get`
List → `API.List`
Send → `API.Send`
Messages → `API.Messages`

## Notes

IDs are opaque strings.
Time fields use RFC3339 format.
Pagination uses `before` as a cursor.
Errors are returned as JSON with an `error` field.
