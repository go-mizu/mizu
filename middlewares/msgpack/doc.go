// Package msgpack provides MessagePack serialization middleware for Mizu.
//
// MessagePack is an efficient binary serialization format that is more compact
// than JSON and faster to parse. This package provides a lightweight implementation
// without external dependencies.
//
// # Overview
//
// The msgpack middleware enables content negotiation and automatic serialization
// of MessagePack data for HTTP requests and responses. It intercepts requests with
// MessagePack content types and provides helper functions for encoding and decoding.
//
// # Installation
//
// Import the package:
//
//	import "github.com/go-mizu/mizu/middlewares/msgpack"
//
// # Basic Usage
//
// Enable the middleware:
//
//	app := mizu.New()
//	app.Use(msgpack.New())
//
//	app.Get("/data", func(c *mizu.Ctx) error {
//	    // Auto-negotiates based on Accept header
//	    return c.Negotiate(200, data)
//	})
//
// # Sending MessagePack Responses
//
// Use the Response function to send MessagePack-encoded responses:
//
//	app.Get("/data", func(c *mizu.Ctx) error {
//	    data := map[string]any{
//	        "name": "John",
//	        "age":  30,
//	    }
//	    return msgpack.Response(c, 200, data)
//	})
//
// # Parsing MessagePack Requests
//
// Use the Bind function to parse MessagePack request bodies:
//
//	app.Post("/data", func(c *mizu.Ctx) error {
//	    data, err := msgpack.Bind(c)
//	    if err != nil {
//	        return err
//	    }
//	    // Process data...
//	    return c.JSON(200, data)
//	})
//
// # Content Negotiation
//
// The middleware supports content negotiation with Mizu's Negotiate method:
//
//	app.Use(msgpack.New())
//
//	app.Get("/data", func(c *mizu.Ctx) error {
//	    // Returns MessagePack if Accept: application/msgpack
//	    // Returns JSON if Accept: application/json
//	    return c.Negotiate(200, data)
//	})
//
// # Custom Options
//
// Configure the middleware with custom content types:
//
//	app.Use(msgpack.WithOptions(msgpack.Options{
//	    ContentTypes: []string{
//	        "application/msgpack",
//	        "application/x-msgpack",
//	        "application/vnd.api+msgpack",
//	    },
//	}))
//
// # Encoding and Decoding
//
// The package provides low-level Marshal and Unmarshal functions:
//
//	// Encode to MessagePack
//	data := map[string]any{"hello": "world"}
//	encoded, err := msgpack.Marshal(data)
//	if err != nil {
//	    // Handle error
//	}
//
//	// Decode from MessagePack
//	decoded, err := msgpack.Unmarshal(encoded)
//	if err != nil {
//	    // Handle error
//	}
//
// # Supported Types
//
// The encoder/decoder supports the following Go types:
//
//   - nil
//   - bool
//   - int, int8, int16, int32, int64
//   - uint, uint8, uint16, uint32, uint64
//   - float32, float64
//   - string
//   - []byte (binary data)
//   - []any (arrays)
//   - map[string]any (maps)
//
// # Error Handling
//
// The package defines three error types:
//
//   - ErrUnsupportedType: Returned when encoding unsupported Go types
//   - ErrInvalidFormat: Returned when decoding invalid MessagePack data
//   - ErrBufferTooSmall: Returned when the buffer is too small for the data
//
// Example error handling:
//
//	data, err := msgpack.Marshal(value)
//	if err != nil {
//	    if errors.Is(err, msgpack.ErrUnsupportedType) {
//	        // Handle unsupported type
//	    }
//	    return err
//	}
//
// # Content Types
//
// The middleware recognizes the following content types by default:
//
//   - application/msgpack (standard)
//   - application/x-msgpack (alternative)
//
// # Best Practices
//
//   - Use MessagePack for internal APIs where both client and server support it
//   - Fall back to JSON for browser-based clients
//   - Consider client library support before adopting MessagePack
//   - Benchmark your specific use case to verify performance benefits
//   - Use the Body() function to access raw request data for debugging
//
// # Implementation Details
//
// The encoder uses optimized MessagePack formats based on value ranges:
//
//   - fixint for small integers (0-127, -32 to -1)
//   - int8/16/32/64 for larger integers
//   - fixstr for short strings (up to 31 bytes)
//   - str8/16/32 for longer strings
//   - fixarray/fixmap for small collections (up to 15 elements)
//   - array16/32 and map16/32 for larger collections
//
// The decoder validates buffer boundaries to prevent out-of-bounds access
// and returns strongly-typed values (int64, uint64, float32/64, string, etc.).
//
// # Context Storage
//
// The middleware stores the raw MessagePack request body in the request context,
// allowing multiple reads of the body and access via the Body() helper function.
// This is useful for logging, debugging, or custom processing.
package msgpack
