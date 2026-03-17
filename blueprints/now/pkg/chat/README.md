# pkg/chat

Minimal chat domain for direct and room conversations. The design focuses on a small set of primitives, consistent verbs, and actor based identity.

## Concepts

A chat is a conversation. It can be direct for one to one or room for multiple members. A message is a unit of text sent by an actor into a chat. An actor is an identity string, using the form u/:id for users and a/:id for agents or bots.

## Kinds

Two kinds are supported. The direct kind represents one to one conversations. The room kind represents group conversations.

## Data model

Chat contains id, kind, title, creator, and time.
Message contains id, chat, actor, text, and time.

## API

Create creates a chat, either direct or room. Join allows an actor to join a chat, optionally with a token. Get returns a chat by id. List returns chats with optional filtering. Send creates a message in a chat. Messages returns messages in a chat with pagination via before.

## Examples

Create room

```go
chat, _ := api.Create(ctx, chat.CreateInput{
	Kind:  chat.KindRoom,
	Title: "general",
})
```

Join

```go
_ = api.Join(ctx, chat.JoinInput{
	Chat: chat.ID,
})
```

Send message

```go
msg, _ := api.Send(ctx, chat.SendInput{
	Chat: chat.ID,
	Text: "hello",
})
```

List messages

```go
res, _ := api.Messages(ctx, chat.MessagesInput{
	Chat:  chat.ID,
	Limit: 20,
})
```

## Store

Persistence is split into three small interfaces. ChatStore handles chats with create, get, and list. MemberStore handles membership with join, leave, membership check, and listing members. MessageStore handles messages with create, get, and list. These are grouped under Store.

```go
type Store struct {
	Chat    ChatStore
	Member  MemberStore
	Message MessageStore
}
```

## Notes

Time is used instead of CreatedAt for brevity. IDs are opaque strings, ULID or similar is recommended. Pagination uses before as a cursor based on message id or time. The API is transport agnostic and maps cleanly to HTTP, CLI, or RPC.
