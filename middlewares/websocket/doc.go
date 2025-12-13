/*
Package websocket provides WebSocket upgrade middleware for Mizu.

# Overview

The websocket middleware handles WebSocket protocol upgrades according
to RFC 6455. It allows bidirectional communication between client and
server over a single TCP connection.

# Usage

Basic usage:

	app := mizu.New()
	app.Get("/ws", websocket.New(func(c *mizu.Ctx, ws *websocket.Conn) error {
	    for {
	        msgType, data, err := ws.ReadMessage()
	        if err != nil {
	            return err
	        }
	        if err := ws.WriteMessage(msgType, data); err != nil {
	            return err
	        }
	    }
	}))

# Configuration

Options:

  - Origins: List of allowed origins (default: all)
  - Subprotocols: Supported WebSocket subprotocols
  - CheckOrigin: Custom origin validation function

# Message Types

  - TextMessage (1): UTF-8 text data
  - BinaryMessage (2): Binary data
  - CloseMessage (8): Connection close
  - PingMessage (9): Ping frame
  - PongMessage (10): Pong frame

# Connection Methods

The Conn type provides:

  - ReadMessage: Read the next message
  - WriteMessage: Write a message with type
  - WriteText: Write a text message
  - WriteBinary: Write a binary message
  - Ping/Pong: Send control frames
  - Close: Close the connection

# Example

Echo server with origin checking:

	app.Get("/ws", websocket.WithOptions(
	    func(c *mizu.Ctx, ws *websocket.Conn) error {
	        for {
	            msgType, data, err := ws.ReadMessage()
	            if err != nil {
	                return err
	            }
	            ws.WriteMessage(msgType, data)
	        }
	    },
	    websocket.Options{
	        Origins: []string{"https://example.com"},
	    },
	))

# See Also

  - Package sse for Server-Sent Events
  - Package h2c for HTTP/2 cleartext
*/
package websocket
