# RFC 0069: Rust SDK Generator

## Summary

Add Rust SDK code generation to the Mizu contract system, enabling production-ready, type-safe, and memory-safe Rust clients with idiomatic async/await patterns, zero-cost abstractions, and excellent developer experience.

## Motivation

Rust has become the language of choice for performance-critical, reliability-focused applications:

1. **Memory safety without GC**: Zero-cost abstractions and compile-time memory safety
2. **Systems programming**: OS kernels, embedded systems, WebAssembly, and high-performance services
3. **Cloud infrastructure**: Growing adoption in cloud-native tooling (linkerd, Firecracker, tikv)
4. **CLI tooling**: Fast startup, small binaries, cross-platform compilation
5. **WebAssembly**: First-class WASM target for browser and edge computing
6. **Security-critical applications**: Memory safety guarantees reduce vulnerability surface

## Design Goals

### Developer Experience (DX)

- **Idiomatic Rust**: Follow Rust conventions (snake_case, Result<T, E>, Option<T>, iterators)
- **Async-first**: Native async/await with tokio runtime (runtime-agnostic design)
- **Builder pattern**: Ergonomic request construction with type-state builders
- **Zero-cost abstractions**: No runtime overhead for type safety
- **Serde integration**: Seamless JSON serialization with derive macros
- **Comprehensive documentation**: rustdoc with examples for all public APIs
- **Strong typing**: Leverage Rust's type system for compile-time correctness
- **Error handling**: Ergonomic error types with thiserror, compatible with `?` operator

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff via tower middleware
- **Timeout handling**: Per-request and global timeout configuration
- **Connection pooling**: Automatic connection reuse via reqwest/hyper
- **Streaming support**: Async iterators (Stream) for SSE with backpressure
- **Tracing integration**: Optional tracing spans for observability
- **Feature flags**: Granular feature selection for minimal binary size
- **no_std support**: Optional no_std compatibility for embedded targets
- **WASM support**: WebAssembly target compatibility

## Architecture

### Package Structure

```
{crate}/
├── Cargo.toml                 # Package manifest with features
├── src/
│   ├── lib.rs                 # Crate root, re-exports
│   ├── client.rs              # Client builder and configuration
│   ├── types.rs               # Generated type definitions
│   ├── resources.rs           # Resource modules with methods
│   ├── streaming.rs           # SSE streaming support
│   └── error.rs               # Error types
└── examples/
    ├── basic.rs               # Basic usage example
    └── streaming.rs           # Streaming example
```

### Core Components

#### 1. Error Handling (`error.rs`)

Error handling uses thiserror for ergonomic error types:

```rust
//! Error types for the {ServiceName} SDK.

use thiserror::Error;

/// Errors that can occur when using the SDK.
#[derive(Debug, Error)]
#[non_exhaustive]
pub enum Error {
    /// HTTP request failed.
    #[error("HTTP error: {status}")]
    Http {
        /// HTTP status code.
        status: u16,
        /// Response body, if available.
        body: Option<String>,
    },

    /// Request timed out.
    #[error("request timed out")]
    Timeout,

    /// Connection failed.
    #[error("connection error: {0}")]
    Connection(#[source] reqwest::Error),

    /// Request was cancelled.
    #[error("request cancelled")]
    Cancelled,

    /// Failed to serialize request body.
    #[error("serialization error: {0}")]
    Serialization(#[source] serde_json::Error),

    /// Failed to deserialize response body.
    #[error("deserialization error: {0}")]
    Deserialization(#[source] serde_json::Error),

    /// Streaming error.
    #[error("streaming error: {0}")]
    Stream(String),

    /// Invalid configuration.
    #[error("invalid configuration: {0}")]
    InvalidConfig(String),
}

/// A specialized Result type for SDK operations.
pub type Result<T> = std::result::Result<T, Error>;

impl Error {
    /// Returns true if this error is potentially retriable.
    #[must_use]
    pub fn is_retriable(&self) -> bool {
        match self {
            Self::Http { status, .. } => {
                *status >= 500 || *status == 429
            }
            Self::Timeout | Self::Connection(_) => true,
            _ => false,
        }
    }

    /// Returns the HTTP status code, if this is an HTTP error.
    #[must_use]
    pub fn status(&self) -> Option<u16> {
        match self {
            Self::Http { status, .. } => Some(*status),
            _ => None,
        }
    }
}
```

#### 2. Client (`client.rs`)

The client with builder pattern configuration:

```rust
//! HTTP client for the {ServiceName} API.

use std::sync::Arc;
use std::time::Duration;

use reqwest::header::{HeaderMap, HeaderName, HeaderValue};
use reqwest::Client as HttpClient;

use crate::error::{Error, Result};
use crate::resources;

/// Authentication mode for API requests.
#[derive(Debug, Clone, Default)]
pub enum AuthMode {
    /// Bearer token authentication.
    #[default]
    Bearer,
    /// Basic authentication.
    Basic,
    /// API key in header.
    ApiKey,
    /// No authentication.
    None,
}

/// Configuration for the SDK client.
#[derive(Debug, Clone)]
pub struct ClientConfig {
    /// API key for authentication.
    pub api_key: Option<String>,
    /// Base URL for API requests.
    pub base_url: String,
    /// Authentication mode.
    pub auth_mode: AuthMode,
    /// Request timeout.
    pub timeout: Duration,
    /// Maximum retry attempts.
    pub max_retries: u32,
    /// Additional headers to include in all requests.
    pub default_headers: HeaderMap,
}

impl Default for ClientConfig {
    fn default() -> Self {
        Self {
            api_key: None,
            base_url: "{defaults.base_url}".to_string(),
            auth_mode: AuthMode::default(),
            timeout: Duration::from_secs(60),
            max_retries: 2,
            default_headers: HeaderMap::new(),
        }
    }
}

/// Builder for constructing a [`Client`].
#[derive(Debug, Clone, Default)]
pub struct ClientBuilder {
    config: ClientConfig,
}

impl ClientBuilder {
    /// Creates a new client builder with default configuration.
    #[must_use]
    pub fn new() -> Self {
        Self::default()
    }

    /// Sets the API key for authentication.
    #[must_use]
    pub fn api_key(mut self, key: impl Into<String>) -> Self {
        self.config.api_key = Some(key.into());
        self
    }

    /// Sets the base URL for API requests.
    #[must_use]
    pub fn base_url(mut self, url: impl Into<String>) -> Self {
        self.config.base_url = url.into();
        self
    }

    /// Sets the authentication mode.
    #[must_use]
    pub fn auth_mode(mut self, mode: AuthMode) -> Self {
        self.config.auth_mode = mode;
        self
    }

    /// Sets the request timeout.
    #[must_use]
    pub fn timeout(mut self, timeout: Duration) -> Self {
        self.config.timeout = timeout;
        self
    }

    /// Sets the maximum retry attempts.
    #[must_use]
    pub fn max_retries(mut self, retries: u32) -> Self {
        self.config.max_retries = retries;
        self
    }

    /// Adds a default header to include in all requests.
    #[must_use]
    pub fn header(mut self, name: HeaderName, value: HeaderValue) -> Self {
        self.config.default_headers.insert(name, value);
        self
    }

    /// Builds the client.
    ///
    /// # Errors
    ///
    /// Returns an error if the HTTP client cannot be created.
    pub fn build(self) -> Result<Client> {
        let mut headers = self.config.default_headers.clone();
        headers.insert(
            reqwest::header::CONTENT_TYPE,
            HeaderValue::from_static("application/json"),
        );
        headers.insert(
            reqwest::header::ACCEPT,
            HeaderValue::from_static("application/json"),
        );

        // Apply default headers from contract
        {{range $key, $value := .Defaults.Headers}}
        headers.insert(
            HeaderName::from_static("{{$key | lower}}"),
            HeaderValue::from_static("{{$value}}"),
        );
        {{end}}

        // Apply authentication header
        if let Some(ref api_key) = self.config.api_key {
            let auth_value = match self.config.auth_mode {
                AuthMode::Bearer => format!("Bearer {api_key}"),
                AuthMode::Basic => format!("Basic {api_key}"),
                AuthMode::ApiKey => api_key.clone(),
                AuthMode::None => String::new(),
            };
            if !auth_value.is_empty() {
                let header_name = match self.config.auth_mode {
                    AuthMode::ApiKey => "x-api-key",
                    _ => "authorization",
                };
                headers.insert(
                    HeaderName::from_static(header_name),
                    HeaderValue::from_str(&auth_value)
                        .map_err(|_| Error::InvalidConfig("invalid API key".into()))?,
                );
            }
        }

        let http_client = HttpClient::builder()
            .timeout(self.config.timeout)
            .default_headers(headers)
            .build()
            .map_err(Error::Connection)?;

        Ok(Client {
            inner: Arc::new(ClientInner {
                http: http_client,
                config: self.config,
            }),
        })
    }
}

#[derive(Debug)]
struct ClientInner {
    http: HttpClient,
    config: ClientConfig,
}

/// Client for the {ServiceName} API.
///
/// The client is cheaply cloneable and can be shared across tasks.
#[derive(Debug, Clone)]
pub struct Client {
    inner: Arc<ClientInner>,
}

impl Client {
    /// Creates a new client builder.
    #[must_use]
    pub fn builder() -> ClientBuilder {
        ClientBuilder::new()
    }

    /// Creates a new client with default configuration.
    ///
    /// # Errors
    ///
    /// Returns an error if the HTTP client cannot be created.
    pub fn new() -> Result<Self> {
        Self::builder().build()
    }

    /// Returns the configured base URL.
    #[must_use]
    pub fn base_url(&self) -> &str {
        &self.inner.config.base_url
    }

    /// Returns the HTTP client for making requests.
    pub(crate) fn http(&self) -> &HttpClient {
        &self.inner.http
    }

    /// Returns the client configuration.
    pub(crate) fn config(&self) -> &ClientConfig {
        &self.inner.config
    }

    // Resource accessors are generated below
    {{range .Resources}}
    /// Access the {{.Name}} resource.
    #[must_use]
    pub fn {{.Name | to_snake_case}}(&self) -> resources::{{.Name | to_pascal_case}} {
        resources::{{.Name | to_pascal_case}}::new(self.clone())
    }
    {{end}}
}
```

#### 3. Types (`types.rs`)

Generated type definitions with serde derive macros:

```rust
//! Type definitions for the {ServiceName} API.

use serde::{Deserialize, Serialize};
{{if .HasDate}}
use chrono::{DateTime, Utc};
{{end}}

// --- Struct Types ---

{{range .Types}}
{{if eq .Kind "struct"}}
/// {{.Description}}
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct {{.Name | to_pascal_case}} {
    {{range .Fields}}
    {{if .Description}}/// {{.Description}}{{end}}
    {{if .Optional}}#[serde(skip_serializing_if = "Option::is_none")]{{end}}
    {{if ne .JSONName ""}}#[serde(rename = "{{.JSONName}}")]{{end}}
    pub {{.Name | to_snake_case}}: {{.Type | to_rust_type}},
    {{end}}
}

{{if .HasRequiredFields}}
/// Builder for [`{{.Name | to_pascal_case}}`].
#[derive(Debug, Clone, Default)]
pub struct {{.Name | to_pascal_case}}Builder {
    {{range .Fields}}
    {{.Name | to_snake_case}}: {{if .Optional}}{{.Type | to_rust_type}}{{else}}Option<{{.Type | to_rust_type_inner}}>{{end}},
    {{end}}
}

impl {{.Name | to_pascal_case}}Builder {
    /// Creates a new builder.
    #[must_use]
    pub fn new() -> Self {
        Self::default()
    }

    {{range .Fields}}
    /// Sets the `{{.Name | to_snake_case}}` field.
    #[must_use]
    pub fn {{.Name | to_snake_case}}(mut self, value: impl Into<{{.Type | to_rust_type_inner}}>) -> Self {
        self.{{.Name | to_snake_case}} = {{if .Optional}}Some(value.into()){{else}}Some(value.into()){{end}};
        self
    }
    {{end}}

    /// Builds the [`{{.Name | to_pascal_case}}`].
    ///
    /// # Panics
    ///
    /// Panics if required fields are not set.
    #[must_use]
    pub fn build(self) -> {{.Name | to_pascal_case}} {
        {{.Name | to_pascal_case}} {
            {{range .Fields}}
            {{.Name | to_snake_case}}: self.{{.Name | to_snake_case}}{{if not .Optional}}.expect("{{.Name}} is required"){{end}},
            {{end}}
        }
    }

    /// Tries to build the [`{{.Name | to_pascal_case}}`].
    ///
    /// # Errors
    ///
    /// Returns an error if required fields are not set.
    pub fn try_build(self) -> Result<{{.Name | to_pascal_case}}, &'static str> {
        Ok({{.Name | to_pascal_case}} {
            {{range .Fields}}
            {{.Name | to_snake_case}}: {{if .Optional}}self.{{.Name | to_snake_case}}{{else}}self.{{.Name | to_snake_case}}.ok_or("{{.Name}} is required")?{{end}},
            {{end}}
        })
    }
}

impl {{.Name | to_pascal_case}} {
    /// Creates a new builder for this type.
    #[must_use]
    pub fn builder() -> {{.Name | to_pascal_case}}Builder {
        {{.Name | to_pascal_case}}Builder::new()
    }
}
{{end}}
{{end}}
{{end}}

// --- Enum Types ---

{{range .Types}}
{{if eq .Kind "enum"}}
/// {{.Description}}
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum {{.Name | to_pascal_case}} {
    {{range .Enum}}
    #[serde(rename = "{{.}}")]
    {{. | to_pascal_case}},
    {{end}}
}

impl {{.Name | to_pascal_case}} {
    /// Returns the string representation of this variant.
    #[must_use]
    pub const fn as_str(&self) -> &'static str {
        match self {
            {{range .Enum}}
            Self::{{. | to_pascal_case}} => "{{.}}",
            {{end}}
        }
    }
}

impl std::fmt::Display for {{.Name | to_pascal_case}} {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(self.as_str())
    }
}

impl std::str::FromStr for {{.Name | to_pascal_case}} {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            {{range .Enum}}
            "{{.}}" => Ok(Self::{{. | to_pascal_case}}),
            {{end}}
            _ => Err(format!("unknown variant: {s}")),
        }
    }
}
{{end}}
{{end}}

// --- Union/Variant Types ---

{{range .Types}}
{{if eq .Kind "union"}}
/// {{.Description}}
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(tag = "{{.Tag}}")]
pub enum {{.Name | to_pascal_case}} {
    {{range .Variants}}
    #[serde(rename = "{{.Value}}")]
    {{.Value | to_pascal_case}}({{.Type | to_rust_type_inner}}),
    {{end}}
}

impl {{.Name | to_pascal_case}} {
    {{range .Variants}}
    /// Returns `true` if this is the `{{.Value | to_pascal_case}}` variant.
    #[must_use]
    pub const fn is_{{.Value | to_snake_case}}(&self) -> bool {
        matches!(self, Self::{{.Value | to_pascal_case}}(_))
    }

    /// Returns the inner value if this is the `{{.Value | to_pascal_case}}` variant.
    #[must_use]
    pub fn as_{{.Value | to_snake_case}}(&self) -> Option<&{{.Type | to_rust_type_inner}}> {
        match self {
            Self::{{.Value | to_pascal_case}}(inner) => Some(inner),
            _ => None,
        }
    }

    /// Converts to the inner value if this is the `{{.Value | to_pascal_case}}` variant.
    #[must_use]
    pub fn into_{{.Value | to_snake_case}}(self) -> Option<{{.Type | to_rust_type_inner}}> {
        match self {
            Self::{{.Value | to_pascal_case}}(inner) => Some(inner),
            _ => None,
        }
    }
    {{end}}
}
{{end}}
{{end}}
```

#### 4. Resources (`resources.rs`)

Resource operations with async methods:

```rust
//! Resource operations for the {ServiceName} API.

use crate::client::Client;
use crate::error::{Error, Result};
use crate::types::*;
{{if .HasSSE}}
use crate::streaming::EventStream;
use futures_core::Stream;
use std::pin::Pin;
{{end}}

{{range .Resources}}
/// Operations for the {{.Name}} resource.
///
/// {{.Description}}
#[derive(Debug, Clone)]
pub struct {{.Name | to_pascal_case}} {
    client: Client,
}

impl {{.Name | to_pascal_case}} {
    /// Creates a new resource accessor.
    pub(crate) fn new(client: Client) -> Self {
        Self { client }
    }

    {{range .Methods}}
    /// {{.Description}}
    ///
    /// # Errors
    ///
    /// Returns an error if the request fails.
    {{if .Stream}}
    pub async fn {{.Name | to_snake_case}}(
        &self,
        request: &{{.Input | to_pascal_case}},
    ) -> Result<Pin<Box<dyn Stream<Item = Result<{{.Stream.Item | to_pascal_case}}>> + Send>>> {
        let url = format!("{}{}", self.client.base_url(), "{{.HTTP.Path}}");

        let response = self
            .client
            .http()
            .{{.HTTP.Method | lower}}(&url)
            .json(request)
            .send()
            .await
            .map_err(Error::Connection)?;

        if !response.status().is_success() {
            let status = response.status().as_u16();
            let body = response.text().await.ok();
            return Err(Error::Http { status, body });
        }

        Ok(EventStream::new(response.bytes_stream()))
    }
    {{else}}
    pub async fn {{.Name | to_snake_case}}(
        &self,
        request: &{{.Input | to_pascal_case}},
    ) -> Result<{{.Output | to_pascal_case}}> {
        let url = format!("{}{}", self.client.base_url(), "{{.HTTP.Path}}");

        let response = self
            .client
            .http()
            .{{.HTTP.Method | lower}}(&url)
            .json(request)
            .send()
            .await
            .map_err(Error::Connection)?;

        let status = response.status();
        if !status.is_success() {
            let body = response.text().await.ok();
            return Err(Error::Http {
                status: status.as_u16(),
                body,
            });
        }

        let body = response
            .json()
            .await
            .map_err(Error::Deserialization)?;

        Ok(body)
    }
    {{end}}
    {{end}}
}
{{end}}
```

#### 5. Streaming (`streaming.rs`)

SSE streaming with async iterators:

```rust
//! Server-Sent Events (SSE) streaming support.

use bytes::Bytes;
use futures_core::Stream;
use std::pin::Pin;
use std::task::{Context, Poll};

use crate::error::{Error, Result};

/// An SSE event.
#[derive(Debug, Clone, Default)]
pub struct SseEvent {
    /// Event type.
    pub event: Option<String>,
    /// Event data.
    pub data: Option<String>,
    /// Event ID.
    pub id: Option<String>,
    /// Retry interval in milliseconds.
    pub retry: Option<u64>,
}

/// SSE parser state.
#[derive(Debug, Default)]
struct SseParser {
    buffer: String,
    current_event: SseEvent,
}

impl SseParser {
    fn new() -> Self {
        Self::default()
    }

    /// Feed data to the parser and return complete events.
    fn feed(&mut self, data: &str) -> Vec<SseEvent> {
        self.buffer.push_str(data);
        let mut events = Vec::new();

        while let Some(pos) = self.buffer.find("\n\n") {
            let event_data = self.buffer[..pos].to_string();
            self.buffer = self.buffer[pos + 2..].to_string();

            if let Some(event) = self.parse_event(&event_data) {
                events.push(event);
            }
        }

        events
    }

    fn parse_event(&mut self, data: &str) -> Option<SseEvent> {
        let mut event = SseEvent::default();
        let mut has_data = false;

        for line in data.lines() {
            if line.is_empty() || line.starts_with(':') {
                continue;
            }

            let (field, value) = if let Some(pos) = line.find(':') {
                let field = &line[..pos];
                let value = line[pos + 1..].trim_start();
                (field, value)
            } else {
                (line, "")
            };

            match field {
                "event" => event.event = Some(value.to_string()),
                "data" => {
                    if let Some(ref mut existing) = event.data {
                        existing.push('\n');
                        existing.push_str(value);
                    } else {
                        event.data = Some(value.to_string());
                    }
                    has_data = true;
                }
                "id" => event.id = Some(value.to_string()),
                "retry" => event.retry = value.parse().ok(),
                _ => {}
            }
        }

        if has_data {
            Some(event)
        } else {
            None
        }
    }
}

/// A stream of typed events from an SSE endpoint.
pub struct EventStream<T, S> {
    inner: S,
    parser: SseParser,
    pending: std::collections::VecDeque<Result<T>>,
    _marker: std::marker::PhantomData<T>,
}

impl<T, S> EventStream<T, S>
where
    T: serde::de::DeserializeOwned,
    S: Stream<Item = std::result::Result<Bytes, reqwest::Error>> + Unpin,
{
    /// Creates a new event stream from a byte stream.
    pub fn new(inner: S) -> Pin<Box<dyn Stream<Item = Result<T>> + Send>>
    where
        S: Send + 'static,
        T: Send + 'static,
    {
        Box::pin(Self {
            inner,
            parser: SseParser::new(),
            pending: std::collections::VecDeque::new(),
            _marker: std::marker::PhantomData,
        })
    }
}

impl<T, S> Stream for EventStream<T, S>
where
    T: serde::de::DeserializeOwned + Unpin,
    S: Stream<Item = std::result::Result<Bytes, reqwest::Error>> + Unpin,
{
    type Item = Result<T>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = self.get_mut();

        // Return pending events first
        if let Some(event) = this.pending.pop_front() {
            return Poll::Ready(Some(event));
        }

        // Poll for more data
        loop {
            match Pin::new(&mut this.inner).poll_next(cx) {
                Poll::Ready(Some(Ok(bytes))) => {
                    let data = String::from_utf8_lossy(&bytes);
                    let events = this.parser.feed(&data);

                    for sse_event in events {
                        if let Some(data) = sse_event.data {
                            // Skip [DONE] marker
                            if data == "[DONE]" {
                                continue;
                            }

                            match serde_json::from_str(&data) {
                                Ok(parsed) => this.pending.push_back(Ok(parsed)),
                                Err(e) => {
                                    this.pending.push_back(Err(Error::Deserialization(e)))
                                }
                            }
                        }
                    }

                    if let Some(event) = this.pending.pop_front() {
                        return Poll::Ready(Some(event));
                    }
                }
                Poll::Ready(Some(Err(e))) => {
                    return Poll::Ready(Some(Err(Error::Connection(e))));
                }
                Poll::Ready(None) => {
                    return if let Some(event) = this.pending.pop_front() {
                        Poll::Ready(Some(event))
                    } else {
                        Poll::Ready(None)
                    };
                }
                Poll::Pending => return Poll::Pending,
            }
        }
    }
}
```

## Type Mapping

### Primitive Types

| Contract Type     | Rust Type            | Notes                           |
|-------------------|----------------------|--------------------------------|
| `string`          | `String`             | Owned string                   |
| `bool`, `boolean` | `bool`               |                                |
| `int`             | `i32`                |                                |
| `int8`            | `i8`                 |                                |
| `int16`           | `i16`                |                                |
| `int32`           | `i32`                |                                |
| `int64`           | `i64`                |                                |
| `uint`            | `u32`                |                                |
| `uint8`           | `u8`                 |                                |
| `uint16`          | `u16`                |                                |
| `uint32`          | `u32`                |                                |
| `uint64`          | `u64`                |                                |
| `float32`         | `f32`                |                                |
| `float64`         | `f64`                |                                |
| `time.Time`       | `DateTime<Utc>`      | chrono crate, ISO 8601         |
| `json.RawMessage` | `serde_json::Value`  | Dynamic JSON                   |
| `any`             | `serde_json::Value`  | Dynamic JSON                   |

### Collection Types

| Contract Type      | Rust Type                           |
|--------------------|-------------------------------------|
| `[]T`              | `Vec<RustType>`                     |
| `map[string]T`     | `std::collections::HashMap<String, RustType>` |

### Optional/Nullable

| Contract Pattern   | Rust Type              |
|--------------------|------------------------|
| Optional field     | `Option<T>`            |
| Nullable type      | `Option<T>`            |

### Struct Fields

Fields use serde attributes for JSON mapping:

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateMessageRequest {
    /// The model to use for generation.
    pub model: String,

    /// The messages in the conversation.
    pub messages: Vec<Message>,

    /// Maximum tokens to generate.
    pub max_tokens: i32,

    /// Temperature for sampling.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub temperature: Option<f64>,

    /// Whether to stream the response.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub stream: Option<bool>,
}
```

### Enum/Const Values

String-backed enums with serde rename:

```rust
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum Role {
    #[serde(rename = "user")]
    User,
    #[serde(rename = "assistant")]
    Assistant,
    #[serde(rename = "system")]
    System,
}

impl Role {
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::User => "user",
            Self::Assistant => "assistant",
            Self::System => "system",
        }
    }
}
```

### Discriminated Unions

Tagged enums with associated data:

```rust
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum ContentBlock {
    #[serde(rename = "text")]
    Text(TextBlock),
    #[serde(rename = "image")]
    Image(ImageBlock),
    #[serde(rename = "tool_use")]
    ToolUse(ToolUseBlock),
}

impl ContentBlock {
    /// Returns `true` if this is a text block.
    pub const fn is_text(&self) -> bool {
        matches!(self, Self::Text(_))
    }

    /// Returns the text block if this is one.
    pub fn as_text(&self) -> Option<&TextBlock> {
        match self {
            Self::Text(block) => Some(block),
            _ => None,
        }
    }

    /// Converts to a text block if this is one.
    pub fn into_text(self) -> Option<TextBlock> {
        match self {
            Self::Text(block) => Some(block),
            _ => None,
        }
    }
}
```

## HTTP Client Implementation

### Request Flow

```rust
impl Messages {
    pub async fn create(
        &self,
        request: &CreateMessageRequest,
    ) -> Result<Message> {
        let url = format!("{}/v1/messages", self.client.base_url());

        let mut last_error = None;

        for attempt in 0..=self.client.config().max_retries {
            if attempt > 0 {
                // Exponential backoff with jitter
                let delay_ms = (500 << (attempt - 1)) + rand::random::<u64>() % 100;
                tokio::time::sleep(std::time::Duration::from_millis(delay_ms)).await;
            }

            match self.do_request(&url, request).await {
                Ok(response) => return Ok(response),
                Err(e) if e.is_retriable() && attempt < self.client.config().max_retries => {
                    last_error = Some(e);
                    continue;
                }
                Err(e) => return Err(e),
            }
        }

        Err(last_error.unwrap_or_else(|| Error::Connection(
            reqwest::Error::from(std::io::Error::new(
                std::io::ErrorKind::Other,
                "max retries exceeded",
            ))
        )))
    }

    async fn do_request(
        &self,
        url: &str,
        request: &CreateMessageRequest,
    ) -> Result<Message> {
        let response = self
            .client
            .http()
            .post(url)
            .json(request)
            .send()
            .await
            .map_err(Error::Connection)?;

        let status = response.status();
        if !status.is_success() {
            let body = response.text().await.ok();
            return Err(Error::Http {
                status: status.as_u16(),
                body,
            });
        }

        response
            .json()
            .await
            .map_err(Error::Deserialization)
    }
}
```

### SSE Streaming Implementation

```rust
pub async fn stream(
    &self,
    request: &CreateMessageRequest,
) -> Result<impl Stream<Item = Result<MessageStreamEvent>>> {
    let url = format!("{}/v1/messages", self.client.base_url());

    let response = self
        .client
        .http()
        .post(&url)
        .json(request)
        .send()
        .await
        .map_err(Error::Connection)?;

    if !response.status().is_success() {
        let status = response.status().as_u16();
        let body = response.text().await.ok();
        return Err(Error::Http { status, body });
    }

    Ok(EventStream::new(response.bytes_stream()))
}
```

## Configuration

### Default Values

From contract `Defaults`:

```rust
impl Default for ClientConfig {
    fn default() -> Self {
        Self {
            api_key: None,
            base_url: "{defaults.base_url}".to_string(),
            auth_mode: AuthMode::Bearer,  // from defaults.auth
            timeout: Duration::from_secs(60),
            max_retries: 2,
            default_headers: HeaderMap::new(),
        }
    }
}
```

### Environment Variables

The SDK does NOT automatically read environment variables. Users should handle this explicitly:

```rust
use std::env;

let client = Client::builder()
    .api_key(env::var("ANTHROPIC_API_KEY").expect("ANTHROPIC_API_KEY must be set"))
    .build()?;
```

## Naming Conventions

### Rust Naming

| Contract       | Rust                       |
|----------------|----------------------------|
| `user-id`      | `user_id`                  |
| `user_name`    | `user_name`                |
| `UserData`     | `UserData`                 |
| `create`       | `create`                   |
| `get-user`     | `get_user`                 |
| `getMessage`   | `get_message`              |

Functions:
- `to_snake_case(s)`: Converts to snake_case for functions, fields, modules
- `to_pascal_case(s)`: Converts to PascalCase for types, variants
- `to_screaming_snake_case(s)`: Converts to SCREAMING_SNAKE_CASE for constants
- `sanitize_ident(s)`: Removes invalid characters

Reserved words: Rust keywords are prefixed with `r#`:
- `type` → `r#type`
- `async` → `r#async`
- `await` → `r#await`
- `match` → `r#match`
- `move` → `r#move`
- etc.

## Code Generation

### Generator Structure

```go
package sdkrust

type Config struct {
    // Crate is the Rust crate name.
    // Default: sanitized kebab-case service name.
    Crate string

    // Version is the crate version.
    // Default: "0.1.0".
    Version string

    // Authors is the list of crate authors.
    Authors []string

    // Repository is the crate repository URL.
    Repository string

    // Documentation is the docs.rs URL.
    Documentation string

    // Edition is the Rust edition (2018, 2021).
    // Default: "2021".
    Edition string

    // Features configures optional features.
    Features FeatureConfig
}

type FeatureConfig struct {
    // Tracing enables tracing integration.
    Tracing bool

    // Rustls uses rustls instead of native-tls.
    Rustls bool
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── Cargo.toml.tmpl          # Package manifest
├── lib.rs.tmpl              # Crate root
├── client.rs.tmpl           # Client implementation
├── types.rs.tmpl            # Type definitions
├── resources.rs.tmpl        # Resource operations
├── streaming.rs.tmpl        # SSE streaming
├── error.rs.tmpl            # Error types
└── examples/
    ├── basic.rs.tmpl        # Basic usage example
    └── streaming.rs.tmpl    # Streaming example
```

### Generated Files

| File                     | Purpose                           |
|--------------------------|-----------------------------------|
| `Cargo.toml`             | Package manifest                  |
| `src/lib.rs`             | Crate root with re-exports        |
| `src/client.rs`          | Client and configuration          |
| `src/types.rs`           | Type definitions                  |
| `src/resources.rs`       | Resource operations               |
| `src/streaming.rs`       | SSE streaming support             |
| `src/error.rs`           | Error types                       |
| `examples/basic.rs`      | Basic usage example               |
| `examples/streaming.rs`  | Streaming example                 |

### Cargo.toml

```toml
[package]
name = "{crate}"
version = "{version}"
edition = "2021"
authors = [{authors}]
description = "{ServiceDescription}"
repository = "{repository}"
documentation = "{documentation}"
license = "MIT OR Apache-2.0"
keywords = ["api", "sdk", "{service}"]
categories = ["api-bindings", "web-programming::http-client"]

[features]
default = ["native-tls"]
native-tls = ["reqwest/native-tls"]
rustls-tls = ["reqwest/rustls-tls"]
tracing = ["dep:tracing"]

[dependencies]
bytes = "1"
futures-core = "0.3"
reqwest = { version = "0.12", default-features = false, features = ["json", "stream"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
thiserror = "1"
tokio = { version = "1", features = ["time"] }

# Optional
chrono = { version = "0.4", features = ["serde"], optional = true }
tracing = { version = "0.1", optional = true }

[dev-dependencies]
tokio = { version = "1", features = ["full"] }
tokio-test = "0.4"
```

## Usage Examples

### Basic Usage

```rust
use anthropic::{Client, types::*};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Create client
    let client = Client::builder()
        .api_key(std::env::var("ANTHROPIC_API_KEY")?)
        .build()?;

    // Build request
    let request = CreateMessageRequest::builder()
        .model("claude-3-sonnet-20240229")
        .max_tokens(1024)
        .messages(vec![
            Message::builder()
                .role(Role::User)
                .content(vec![
                    ContentBlock::Text(TextBlock {
                        text: "Hello, Claude!".to_string(),
                    }),
                ])
                .build(),
        ])
        .build();

    // Make request
    let response = client.messages().create(&request).await?;

    // Print response
    for block in &response.content {
        if let ContentBlock::Text(text) = block {
            println!("{}", text.text);
        }
    }

    Ok(())
}
```

### Streaming

```rust
use anthropic::{Client, types::*};
use futures_util::StreamExt;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = Client::builder()
        .api_key(std::env::var("ANTHROPIC_API_KEY")?)
        .build()?;

    let request = CreateMessageRequest::builder()
        .model("claude-3-sonnet-20240229")
        .max_tokens(1024)
        .stream(true)
        .messages(vec![
            Message::builder()
                .role(Role::User)
                .content(vec![
                    ContentBlock::Text(TextBlock {
                        text: "Tell me a story about a robot.".to_string(),
                    }),
                ])
                .build(),
        ])
        .build();

    // Stream response
    let mut stream = client.messages().stream(&request).await?;

    while let Some(event) = stream.next().await {
        match event? {
            MessageStreamEvent::ContentBlockDelta { delta, .. } => {
                if let Some(text) = delta.text {
                    print!("{text}");
                    std::io::Write::flush(&mut std::io::stdout())?;
                }
            }
            MessageStreamEvent::MessageStop => {
                println!();
            }
            _ => {}
        }
    }

    Ok(())
}
```

### Error Handling

```rust
use anthropic::{Client, error::Error, types::*};

async fn example(client: &Client, request: &CreateMessageRequest) {
    match client.messages().create(request).await {
        Ok(response) => {
            println!("Success: {} tokens used", response.usage.total_tokens);
        }
        Err(Error::Http { status: 429, .. }) => {
            eprintln!("Rate limited, please retry later");
        }
        Err(Error::Http { status, body }) if status >= 400 && status < 500 => {
            eprintln!("Client error {status}");
            if let Some(body) = body {
                eprintln!("Response: {body}");
            }
        }
        Err(Error::Http { status, .. }) if status >= 500 => {
            eprintln!("Server error {status}");
        }
        Err(Error::Timeout) => {
            eprintln!("Request timed out");
        }
        Err(Error::Connection(e)) => {
            eprintln!("Connection failed: {e}");
        }
        Err(e) => {
            eprintln!("Error: {e}");
        }
    }
}
```

### Pattern Matching on Unions

```rust
for block in &response.content {
    match block {
        ContentBlock::Text(text) => {
            println!("Text: {}", text.text);
        }
        ContentBlock::Image(image) => {
            println!("Image: {:?}", image.source);
        }
        ContentBlock::ToolUse(tool) => {
            println!("Tool use: {} -> {:?}", tool.name, tool.input);
        }
    }
}

// Or use helper methods
for block in &response.content {
    if let Some(text) = block.as_text() {
        println!("{}", text.text);
    }
}
```

### Custom Configuration

```rust
use std::time::Duration;
use reqwest::header::{HeaderName, HeaderValue};

let client = Client::builder()
    .api_key("your-api-key")
    .base_url("https://custom.api.com")
    .timeout(Duration::from_secs(120))
    .max_retries(3)
    .header(
        HeaderName::from_static("x-custom-header"),
        HeaderValue::from_static("custom-value"),
    )
    .build()?;
```

### Using with Tracing

```rust
use tracing_subscriber;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize tracing subscriber
    tracing_subscriber::fmt::init();

    let client = Client::builder()
        .api_key(std::env::var("ANTHROPIC_API_KEY")?)
        .build()?;

    // Requests will now emit tracing spans
    let response = client.messages().create(&request).await?;

    Ok(())
}
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidRust_Compiles(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_UnionTypes(t *testing.T)
func TestGenerate_BuilderPattern(t *testing.T)
```

### Generated SDK Tests

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_client_builder() {
        let client = Client::builder()
            .api_key("test-key")
            .build()
            .expect("client should build");

        assert_eq!(client.base_url(), "https://api.anthropic.com");
    }

    #[test]
    fn test_request_builder() {
        let request = CreateMessageRequest::builder()
            .model("claude-3")
            .max_tokens(100)
            .messages(vec![])
            .build();

        assert_eq!(request.model, "claude-3");
        assert_eq!(request.max_tokens, 100);
    }

    #[test]
    fn test_enum_serialization() {
        let role = Role::User;
        let json = serde_json::to_string(&role).unwrap();
        assert_eq!(json, r#""user""#);
    }

    #[test]
    fn test_union_serialization() {
        let block = ContentBlock::Text(TextBlock {
            text: "Hello".to_string(),
        });
        let json = serde_json::to_string(&block).unwrap();
        assert!(json.contains(r#""type":"text""#));
    }
}
```

## Platform Support

### Dependencies

**Required:**
- `reqwest` (0.12+) - HTTP client with async support
- `serde` (1.x) - Serialization framework
- `serde_json` (1.x) - JSON support
- `thiserror` (1.x) - Error derive macros
- `tokio` (1.x) - Async runtime
- `bytes` (1.x) - Byte buffer utilities
- `futures-core` (0.3) - Stream trait

**Optional:**
- `chrono` (0.4) - DateTime support
- `tracing` (0.1) - Observability

### Minimum Supported Rust Version (MSRV)

Rust 1.70.0 (for async fn in traits stabilization path)

### Target Support

| Target              | Status    | Notes                           |
|---------------------|-----------|--------------------------------|
| x86_64-unknown-linux-gnu | ✅ | Primary target                 |
| x86_64-apple-darwin | ✅ | macOS Intel                    |
| aarch64-apple-darwin | ✅ | macOS Apple Silicon            |
| x86_64-pc-windows-msvc | ✅ | Windows MSVC                   |
| wasm32-unknown-unknown | ⚠️ | Requires feature flags         |

### Feature Flags

```toml
[features]
default = ["native-tls"]

# TLS backend (mutually exclusive)
native-tls = ["reqwest/native-tls"]
rustls-tls = ["reqwest/rustls-tls"]

# Optional integrations
tracing = ["dep:tracing"]
chrono = ["dep:chrono"]

# Minimal for WASM
wasm = ["reqwest/wasm"]
```

## Future Enhancements

1. **Middleware support**: Tower-based middleware for logging, metrics, auth
2. **Connection pooling tuning**: Expose hyper connection pool settings
3. **Request signing**: AWS SigV4-style request signing
4. **Mock client**: Built-in mock client for testing
5. **Retry policies**: Customizable retry strategies
6. **Rate limiting**: Client-side rate limit handling
7. **Response caching**: Built-in caching with TTL
8. **Batch requests**: Support for batch API endpoints
9. **GraphQL support**: For services with GraphQL endpoints
10. **gRPC support**: For services with gRPC endpoints

## References

- [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/)
- [Rust Error Handling](https://doc.rust-lang.org/book/ch09-00-error-handling.html)
- [Serde Documentation](https://serde.rs/)
- [Reqwest Documentation](https://docs.rs/reqwest/)
- [Tokio Documentation](https://tokio.rs/)
- [Async Book](https://rust-lang.github.io/async-book/)
- [thiserror Documentation](https://docs.rs/thiserror/)
