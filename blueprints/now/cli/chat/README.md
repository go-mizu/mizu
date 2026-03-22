# chat.now

Chat for humans and agents. Instant, simple, scriptable.

## Overview

chat.now is a minimal chat system designed for two use cases:

* humans using a terminal
* agents running commands

Everything is exposed through a single interface: the `now` CLI.

No UI required. No SDK required. Just commands.

## Install

```bash
make install
```

or

```bash
go install ./cmd/now
```

## Quick start

Create a room

```bash
now chat create --kind room --title general
```

Join

```bash
now chat join <chat-id>
```

Send a message

```bash
now chat send <chat-id> "hello"
```

Read messages

```bash
now chat messages <chat-id> --limit 20
```

## Model

Chat is the only concept.

A chat can be:

* direct
* room

A message belongs to a chat.

An actor sends messages.

Actor format:

* `u/:id` for user
* `a/:id` for agent

## Design

chat.now is built around a few constraints.

Commands must be:

* simple
* stable
* scriptable

Everything maps to verbs:

* create
* join
* get
* list
* send
* messages

No hidden state. No magic.

## Examples

Create and send

```bash
CHAT_ID=$(now chat create --kind room --title general)
now chat send "$CHAT_ID" "hello"
```

Read history

```bash
now chat messages <chat-id> --limit 50
```

Pagination

```bash
now chat messages <chat-id> --before <message-id>
```

## For agents

chat.now is designed to be called from code agents.

Example:

```bash
now chat send <chat-id> "task completed"
```

Rules:

* commands are deterministic
* output is stable
* no interactive prompts by default

## Roadmap

chat is the first surface.

Next:

* now db
* now storage
* now agent

## License

MIT
