# RFC 0074: Ruby SDK Generator

## Summary

Add Ruby SDK code generation to the Mizu contract system, enabling production-ready, idiomatic Ruby clients with excellent developer experience for Rails, Sinatra, and standalone Ruby applications.

## Motivation

Ruby remains a popular language for web development, scripting, and rapid prototyping. A native Ruby SDK provides:

1. **Idiomatic Ruby**: snake_case naming, blocks, duck typing, and Ruby conventions
2. **Modern Ruby features**: Keyword arguments, pattern matching, Ractor-safe design
3. **Rails integration**: Works seamlessly in Rails applications with ActiveSupport
4. **Excellent DX**: YARD documentation, intuitive API, method chaining
5. **Production-ready**: Faraday-based HTTP with retries, timeouts, and connection pooling

## Design Goals

### Developer Experience (DX)

- **Idiomatic Ruby**: Follow Ruby style guide conventions (snake_case, predicate methods, blocks)
- **Keyword arguments**: Named parameters for all methods with sensible defaults
- **Block support**: Streaming via blocks (`each`, `Enumerator`)
- **Method chaining**: Fluent interface for client configuration
- **YARD documentation**: Rich inline documentation with @param, @return, @example
- **Minimal dependencies**: Only Faraday (HTTP) and standard library

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff via Faraday middleware
- **Timeout handling**: Per-request and global timeout configuration
- **Thread safety**: Safe for concurrent access from multiple threads
- **Connection pooling**: Persistent connections via Faraday adapters
- **Error handling**: Exception hierarchy with rich error information
- **Logging**: Configurable logger integration (stdlib Logger, Rails.logger)

## Architecture

### Gem Structure

```
{gem_name}/
├── {gem_name}.gemspec             # Gem specification
├── Gemfile                        # Development dependencies
├── lib/
│   └── {gem_name}/
│       ├── version.rb             # Gem version constant
│       ├── client.rb              # Main client class
│       ├── types.rb               # Generated model types
│       ├── resources.rb           # Resource namespaces
│       ├── streaming.rb           # SSE streaming support
│       └── errors.rb              # Error types
│   └── {gem_name}.rb              # Main entry point
```

### Core Components

#### 1. Client (`client.rb`)

The main entry point for API interactions:

```ruby
# frozen_string_literal: true

module {ServiceName}
  # Configuration options for the SDK client.
  class Configuration
    attr_accessor :api_key, :base_url, :timeout, :max_retries,
                  :default_headers, :auth_mode, :logger

    def initialize
      @api_key = nil
      @base_url = "{default_base_url}"
      @timeout = 60
      @max_retries = 2
      @default_headers = {}
      @auth_mode = :bearer
      @logger = nil
    end

    def to_h
      {
        api_key: @api_key,
        base_url: @base_url,
        timeout: @timeout,
        max_retries: @max_retries,
        default_headers: @default_headers,
        auth_mode: @auth_mode,
        logger: @logger
      }
    end
  end

  # The main SDK client providing access to all API resources.
  class Client
    attr_reader :config, :{resources}

    def initialize(api_key: nil, base_url: nil, timeout: nil, max_retries: nil, **options)
      @config = Configuration.new
      @config.api_key = api_key if api_key
      @config.base_url = base_url if base_url
      @config.timeout = timeout if timeout
      @config.max_retries = max_retries if max_retries

      yield @config if block_given?

      @connection = build_connection
      # Initialize resource accessors
      @{resource} = {Resource}Resource.new(self)
    end

    # Creates a new client with modified configuration.
    def with(**options, &block)
      new_config = @config.to_h.merge(options)
      self.class.new(**new_config, &block)
    end

    # @api private
    def request(method:, path:, body: nil, headers: {})
      response = @connection.run_request(method, path, body&.to_json, headers)

      if response.status >= 400
        raise_api_error(response)
      end

      return nil if response.status == 204

      JSON.parse(response.body, symbolize_names: true) if response.body && !response.body.empty?
    end

    private

    def build_connection
      Faraday.new(url: @config.base_url) do |f|
        f.request :json
        f.request :retry, max: @config.max_retries, interval: 0.5, backoff_factor: 2
        f.response :raise_error, false
        f.options.timeout = @config.timeout
        f.headers = default_headers
        f.adapter Faraday.default_adapter
      end
    end

    def default_headers
      headers = {
        "Content-Type" => "application/json",
        "Accept" => "application/json"
      }
      headers.merge!(@config.default_headers)
      apply_auth(headers)
      headers
    end

    def apply_auth(headers)
      return unless @config.api_key

      case @config.auth_mode
      when :bearer
        headers["Authorization"] = "Bearer #{@config.api_key}"
      when :basic
        headers["Authorization"] = "Basic #{@config.api_key}"
      end
    end

    def raise_api_error(response)
      body = begin
        JSON.parse(response.body, symbolize_names: true)
      rescue
        response.body
      end

      message = body.is_a?(Hash) ? body[:message] || body[:error] : response.body

      case response.status
      when 400
        raise BadRequestError.new(message, response.status, body)
      when 401
        raise AuthenticationError.new(message, response.status, body)
      when 403
        raise PermissionDeniedError.new(message, response.status, body)
      when 404
        raise NotFoundError.new(message, response.status, body)
      when 422
        raise UnprocessableEntityError.new(message, response.status, body)
      when 429
        raise RateLimitError.new(message, response.status, body)
      when 500..599
        raise InternalServerError.new(message, response.status, body)
      else
        raise APIError.new(message, response.status, body)
      end
    end
  end
end
```

#### 2. Types (`types.rb`)

All model types use Ruby classes with keyword arguments:

```ruby
# frozen_string_literal: true

module {ServiceName}
  module Types
    # Base class for all model types
    class Base
      def initialize(**attributes)
        attributes.each do |key, value|
          instance_variable_set("@#{key}", value) if respond_to?(key)
        end
      end

      def to_h
        instance_variables.each_with_object({}) do |var, hash|
          key = var.to_s.delete_prefix("@")
          value = instance_variable_get(var)
          hash[key.to_sym] = serialize_value(value)
        end
      end

      def to_json(*args)
        to_h.to_json(*args)
      end

      private

      def serialize_value(value)
        case value
        when Base then value.to_h
        when Array then value.map { |v| serialize_value(v) }
        when Hash then value.transform_values { |v| serialize_value(v) }
        else value
        end
      end
    end

    # Request/response models
    class {TypeName} < Base
      # @return [FieldType] Description from contract
      attr_accessor :field_name

      # @return [String, nil] Optional field
      attr_accessor :optional_field

      def initialize(field_name:, optional_field: nil)
        @field_name = field_name
        @optional_field = optional_field
      end
    end

    # Enum-like modules with constants
    module Role
      USER = "user"
      ASSISTANT = "assistant"
      SYSTEM = "system"

      def self.all
        [USER, ASSISTANT, SYSTEM]
      end

      def self.valid?(value)
        all.include?(value)
      end
    end
  end
end
```

#### 3. Resources (`resources.rb`)

Resource classes provide namespaced method access:

```ruby
# frozen_string_literal: true

module {ServiceName}
  # Operations for {resource}
  class {Resource}Resource
    # @api private
    def initialize(client)
      @client = client
    end

    # Description from contract
    #
    # @param model [String] The model to use
    # @param messages [Array<Hash>] The messages
    # @param max_tokens [Integer] Max tokens to generate (optional)
    # @return [{ResponseType}] The response
    # @raise [APIError] if the request fails
    #
    # @example
    #   response = client.{resource}.method_name(
    #     model: "model-name",
    #     messages: [{ role: "user", content: "Hello" }]
    #   )
    def method_name(model:, messages:, max_tokens: nil, **options)
      body = { model: model, messages: messages }
      body[:max_tokens] = max_tokens if max_tokens

      response = @client.request(
        method: :post,
        path: "/v1/messages",
        body: body
      )

      Types::{ResponseType}.from_hash(response)
    end

    # Streaming method returning an Enumerator
    #
    # @param model [String] The model to use
    # @param messages [Array<Hash>] The messages
    # @yield [event] Each streaming event
    # @yieldparam event [{ItemType}] A streaming event
    # @return [Enumerator<{ItemType}>] if no block given
    #
    # @example With block
    #   client.{resource}.stream_method(model: "x", messages: []) do |event|
    #     puts event.delta.text
    #   end
    #
    # @example With Enumerator
    #   events = client.{resource}.stream_method(model: "x", messages: [])
    #   events.each { |e| puts e }
    def stream_method(model:, messages:, **options, &block)
      body = { model: model, messages: messages, stream: true }

      if block_given?
        stream_request(method: :post, path: "/v1/messages", body: body, &block)
      else
        Enumerator.new do |yielder|
          stream_request(method: :post, path: "/v1/messages", body: body) do |event|
            yielder << event
          end
        end
      end
    end

    private

    def stream_request(method:, path:, body:, &block)
      @client.stream(method: method, path: path, body: body) do |event|
        block.call(Types::{ItemType}.from_hash(event))
      end
    end
  end
end
```

#### 4. Streaming (`streaming.rb`)

SSE streaming support via Enumerator:

```ruby
# frozen_string_literal: true

module {ServiceName}
  # Server-Sent Events (SSE) stream parser
  class SSEParser
    def initialize(io)
      @io = io
      @buffer = ""
    end

    def each
      return enum_for(:each) unless block_given?

      @io.each_line do |line|
        line = line.chomp

        if line.empty?
          # Empty line = end of event
          data = @buffer.strip
          @buffer = ""

          next if data.empty?
          break if data == "[DONE]"

          yield parse_event(data)
        elsif line.start_with?("data:")
          content = line.delete_prefix("data:").lstrip
          @buffer << content << "\n"
        end
      end
    end

    private

    def parse_event(data)
      JSON.parse(data, symbolize_names: true)
    rescue JSON::ParserError => e
      raise StreamParseError.new("Failed to parse SSE event: #{e.message}", data)
    end
  end

  # Mixin for streaming support in Client
  module Streaming
    # Performs a streaming SSE request
    #
    # @param method [Symbol] HTTP method (:get, :post, etc.)
    # @param path [String] Request path
    # @param body [Hash, nil] Request body
    # @yield [event] Each parsed SSE event
    # @yieldparam event [Hash] Parsed event data
    def stream(method:, path:, body: nil)
      return enum_for(:stream, method: method, path: path, body: body) unless block_given?

      headers = default_headers.merge("Accept" => "text/event-stream")

      @connection.run_request(method, path, body&.to_json, headers) do |req|
        req.options.on_data = proc do |chunk, _overall_received_bytes, env|
          if env.status >= 400
            raise_api_error(OpenStruct.new(status: env.status, body: chunk))
          end

          parse_sse_chunk(chunk) { |event| yield event }
        end
      end
    end

    private

    def parse_sse_chunk(chunk)
      @sse_buffer ||= ""
      @sse_buffer << chunk

      while (idx = @sse_buffer.index("\n\n"))
        event_str = @sse_buffer.slice!(0, idx + 2).strip
        next if event_str.empty?

        data = extract_sse_data(event_str)
        next if data.empty? || data == "[DONE]"

        begin
          yield JSON.parse(data, symbolize_names: true)
        rescue JSON::ParserError => e
          raise StreamParseError.new("Failed to parse SSE: #{e.message}", data)
        end
      end
    end

    def extract_sse_data(event_str)
      lines = event_str.split("\n")
      data_lines = lines.select { |l| l.start_with?("data:") }
      data_lines.map { |l| l.delete_prefix("data:").lstrip }.join("\n")
    end
  end
end
```

#### 5. Errors (`errors.rb`)

Exception hierarchy for typed errors:

```ruby
# frozen_string_literal: true

module {ServiceName}
  # Base error class for all SDK errors
  class Error < StandardError
  end

  # Error connecting to the API (network issues)
  class ConnectionError < Error
  end

  # Base class for API errors (HTTP 4xx/5xx)
  class APIError < Error
    # @return [Integer] HTTP status code
    attr_reader :status

    # @return [Hash, String, nil] Response body
    attr_reader :body

    def initialize(message = nil, status = nil, body = nil)
      @status = status
      @body = body
      super(message || "API error")
    end

    def to_s
      "#{self.class.name}: #{message} (status: #{status})"
    end
  end

  # HTTP 400 Bad Request
  class BadRequestError < APIError
  end

  # HTTP 401 Unauthorized
  class AuthenticationError < APIError
  end

  # HTTP 403 Forbidden
  class PermissionDeniedError < APIError
  end

  # HTTP 404 Not Found
  class NotFoundError < APIError
  end

  # HTTP 422 Unprocessable Entity
  class UnprocessableEntityError < APIError
  end

  # HTTP 429 Too Many Requests
  class RateLimitError < APIError
    # @return [Integer, nil] Retry after seconds (from header)
    attr_reader :retry_after

    def initialize(message = nil, status = nil, body = nil, retry_after: nil)
      @retry_after = retry_after
      super(message, status, body)
    end
  end

  # HTTP 5xx Server Error
  class InternalServerError < APIError
  end

  # Error parsing SSE stream
  class StreamParseError < Error
    # @return [String] Raw data that failed to parse
    attr_reader :data

    def initialize(message, data = nil)
      @data = data
      super(message)
    end
  end

  # Request timeout
  class TimeoutError < Error
  end

  # Request was cancelled
  class CancelledError < Error
  end
end
```

## Type Mapping

### Primitive Types

| Contract Type     | Ruby Type      |
|-------------------|----------------|
| `string`          | `String`       |
| `bool`, `boolean` | `TrueClass/FalseClass` |
| `int`             | `Integer`      |
| `int8`            | `Integer`      |
| `int16`           | `Integer`      |
| `int32`           | `Integer`      |
| `int64`           | `Integer`      |
| `uint`            | `Integer`      |
| `uint8`           | `Integer`      |
| `uint16`          | `Integer`      |
| `uint32`          | `Integer`      |
| `uint64`          | `Integer`      |
| `float32`         | `Float`        |
| `float64`         | `Float`        |
| `time.Time`       | `Time`         |
| `json.RawMessage` | `Hash`         |
| `any`             | `Object`       |

### Collection Types

| Contract Type      | Ruby Type        |
|--------------------|------------------|
| `[]T`              | `Array<RubyType>`|
| `map[string]T`     | `Hash{String => RubyType}` |

### Optional/Nullable

| Contract      | Ruby Type          |
|---------------|-------------------|
| `optional: T` | `T` with `nil` default |
| `nullable: T` | `T` or `nil`      |

### Struct Fields

Fields with `optional: true` or `nullable: true` have `nil` as default value:

```ruby
class Request < Base
  attr_accessor :required, :optional_field

  def initialize(required:, optional_field: nil)
    @required = required
    @optional_field = optional_field
  end
end
```

### Enum/Const Values

Fields with `enum` constraint generate module constants:

```ruby
module Role
  USER = "user"
  ASSISTANT = "assistant"
  SYSTEM = "system"

  ALL = [USER, ASSISTANT, SYSTEM].freeze

  def self.valid?(value)
    ALL.include?(value)
  end
end
```

### Discriminated Unions

Union types use a factory method with type-based dispatch:

```ruby
module ContentBlock
  def self.from_hash(hash)
    case hash[:type]
    when "text"
      TextBlock.new(**hash)
    when "image"
      ImageBlock.new(**hash)
    when "tool_use"
      ToolUseBlock.new(**hash)
    else
      raise ArgumentError, "Unknown content block type: #{hash[:type]}"
    end
  end
end

class TextBlock < Types::Base
  attr_accessor :type, :text

  def initialize(type: "text", text:)
    @type = type
    @text = text
  end
end

class ImageBlock < Types::Base
  attr_accessor :type, :url

  def initialize(type: "image", url:)
    @type = type
    @url = url
  end
end

class ToolUseBlock < Types::Base
  attr_accessor :type, :id, :name, :input

  def initialize(type: "tool_use", id:, name:, input:)
    @type = type
    @id = id
    @name = name
    @input = input
  end
end
```

## HTTP Client Implementation

### Request Flow

```ruby
class Client
  include Streaming

  private

  def build_connection
    Faraday.new(url: @config.base_url) do |conn|
      # Request middleware
      conn.request :json  # Encode body as JSON

      # Retry middleware with exponential backoff
      conn.request :retry,
                   max: @config.max_retries,
                   interval: 0.5,
                   interval_randomness: 0.5,
                   backoff_factor: 2,
                   exceptions: [
                     Faraday::ConnectionFailed,
                     Faraday::TimeoutError,
                     "Timeout::Error"
                   ],
                   retry_statuses: [429, 500, 502, 503, 504]

      # Response middleware
      conn.response :logger, @config.logger if @config.logger

      # Timeouts
      conn.options.timeout = @config.timeout
      conn.options.open_timeout = 10

      # Default headers
      conn.headers = default_headers

      # Adapter (pluggable: net_http, patron, typhoeus, etc.)
      conn.adapter Faraday.default_adapter
    end
  end

  def request(method:, path:, body: nil, headers: {})
    merged_headers = @connection.headers.merge(headers)

    response = @connection.run_request(method, path, body&.to_json, merged_headers)

    handle_response(response)
  rescue Faraday::ConnectionFailed => e
    raise ConnectionError, "Connection failed: #{e.message}"
  rescue Faraday::TimeoutError => e
    raise TimeoutError, "Request timed out: #{e.message}"
  end

  def handle_response(response)
    raise_api_error(response) if response.status >= 400
    return nil if response.status == 204
    return nil if response.body.nil? || response.body.empty?

    JSON.parse(response.body, symbolize_names: true)
  end
end
```

### Authentication

```ruby
def apply_auth(headers)
  return unless @config.api_key

  case @config.auth_mode
  when :bearer
    headers["Authorization"] = "Bearer #{@config.api_key}"
  when :basic
    encoded = Base64.strict_encode64(@config.api_key)
    headers["Authorization"] = "Basic #{encoded}"
  when :header
    headers["X-API-Key"] = @config.api_key
  end
end
```

### SSE Streaming

```ruby
def stream(method:, path:, body: nil)
  return enum_for(:stream, method: method, path: path, body: body) unless block_given?

  headers = default_headers.merge(
    "Accept" => "text/event-stream",
    "Cache-Control" => "no-cache"
  )

  buffer = ""

  @connection.run_request(method, path, body&.to_json, headers) do |req|
    req.options.on_data = lambda do |chunk, _size, env|
      # Check for error status on first chunk
      if env.status >= 400
        error_body = buffer + chunk
        raise_api_error(OpenStruct.new(status: env.status, body: error_body))
      end

      buffer << chunk

      # Process complete events (double newline terminated)
      while (idx = buffer.index("\n\n"))
        event_text = buffer.slice!(0, idx + 2)
        event = parse_sse_event(event_text)
        yield event if event
      end
    end
  end
end

def parse_sse_event(text)
  data_lines = []

  text.each_line do |line|
    line = line.chomp
    if line.start_with?("data:")
      data_lines << line.delete_prefix("data:").lstrip
    end
  end

  return nil if data_lines.empty?

  data = data_lines.join("\n")
  return nil if data.empty? || data == "[DONE]"

  JSON.parse(data, symbolize_names: true)
end
```

## Configuration

### Default Values

From contract `Client`:

```ruby
class Configuration
  DEFAULTS = {
    base_url: "{client.baseURL}",
    timeout: 60,
    max_retries: 2,
    auth_mode: :bearer
  }.freeze

  def initialize
    DEFAULTS.each do |key, value|
      instance_variable_set("@#{key}", value)
    end
    @api_key = nil
    @default_headers = {
      # From client.headers
    }
    @logger = nil
  end
end
```

### Environment Variables

The SDK does NOT automatically read environment variables for API keys. This is intentional for security and explicitness:

```ruby
client = ServiceName::Client.new(
  api_key: ENV["SERVICE_API_KEY"]
)
```

### Block Configuration

```ruby
client = ServiceName::Client.new do |config|
  config.api_key = "your-api-key"
  config.base_url = "https://custom.api.com"
  config.timeout = 120
  config.max_retries = 3
  config.logger = Logger.new($stdout)
end
```

## Naming Conventions

### Ruby Naming

| Contract       | Ruby                   |
|----------------|------------------------|
| `user-id`      | `user_id`              |
| `user_name`    | `user_name`            |
| `UserData`     | `UserData` (class)     |
| `create`       | `create`               |
| `get-user`     | `get_user`             |
| `maxTokens`    | `max_tokens`           |

Functions:
- `to_snake(s)`: Converts to snake_case (for methods/attributes)
- `to_pascal(s)`: Converts to PascalCase (for classes/modules)
- `sanitize_ident(s)`: Removes invalid characters

Special handling:
- Reserved words: Prefixed with underscore (`_class`, `_module`)
- Predicate methods: Add `?` suffix for boolean returns (`valid?`, `empty?`)

## Code Generation

### Generator Structure

```go
package sdkruby

type Config struct {
    // GemName is the gem name (used in Gemfile, require).
    // Default: sanitized lowercase service name with underscores.
    GemName string

    // ModuleName is the Ruby module name.
    // Default: PascalCase service name.
    ModuleName string

    // Version is the gem version for the gemspec.
    Version string

    // Authors is the list of gem authors.
    Authors []string

    // Homepage is the gem homepage URL.
    Homepage string
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── gemspec.tmpl            # Gem specification
├── gemfile.tmpl            # Development Gemfile
├── lib.rb.tmpl             # Main require file
├── version.rb.tmpl         # Version constant
├── client.rb.tmpl          # Main client class
├── types.rb.tmpl           # Model types
├── resources.rb.tmpl       # Resource classes
├── streaming.rb.tmpl       # SSE support
└── errors.rb.tmpl          # Error types
```

### Generated Files

| File                     | Purpose                         |
|--------------------------|----------------------------------|
| `{gem}.gemspec`          | Gem specification               |
| `Gemfile`                | Development dependencies        |
| `lib/{gem}.rb`           | Main entry point                |
| `lib/{gem}/version.rb`   | Version constant                |
| `lib/{gem}/client.rb`    | Main client class               |
| `lib/{gem}/types.rb`     | All model type definitions      |
| `lib/{gem}/resources.rb` | Resource namespaces and methods |
| `lib/{gem}/streaming.rb` | Enumerator-based SSE            |
| `lib/{gem}/errors.rb`    | Error type definitions          |

## Usage Examples

### Basic Usage

```ruby
require "service_sdk"

# Create client
client = ServiceSDK::Client.new(api_key: "your-api-key")

# Make a request
response = client.completions.create(
  model: "model-name",
  messages: [
    { role: "user", content: "Hello" }
  ]
)

puts response.content
```

### Streaming with Block

```ruby
client.completions.create_stream(
  model: "model-name",
  messages: [{ role: "user", content: "Hello" }]
) do |event|
  print event.delta&.text
end
```

### Streaming with Enumerator

```ruby
events = client.completions.create_stream(
  model: "model-name",
  messages: [{ role: "user", content: "Hello" }]
)

events.each do |event|
  print event.delta&.text
end

# Or use Enumerator methods
events.lazy.take(10).each { |e| puts e }
```

### Error Handling

```ruby
begin
  response = client.completions.create(
    model: "model-name",
    messages: []
  )
rescue ServiceSDK::RateLimitError => e
  puts "Rate limited! Retry after: #{e.retry_after}s"
  sleep(e.retry_after || 60)
  retry
rescue ServiceSDK::AuthenticationError => e
  puts "Invalid API key: #{e.message}"
rescue ServiceSDK::APIError => e
  puts "API Error #{e.status}: #{e.message}"
rescue ServiceSDK::ConnectionError => e
  puts "Network error: #{e.message}"
rescue ServiceSDK::Error => e
  puts "SDK error: #{e.message}"
end
```

### Custom Configuration

```ruby
client = ServiceSDK::Client.new do |config|
  config.api_key = "your-api-key"
  config.base_url = "https://custom.api.com"
  config.timeout = 120
  config.max_retries = 3
  config.default_headers = {
    "X-Custom-Header" => "value"
  }
  config.logger = Logger.new($stdout)
end
```

### Rails Integration

```ruby
# config/initializers/service_sdk.rb
ServiceSDK.configure do |config|
  config.api_key = Rails.application.credentials.service_api_key
  config.logger = Rails.logger
  config.timeout = 30
end

# In a controller or service
class ChatService
  def initialize
    @client = ServiceSDK::Client.new
  end

  def generate_response(prompt)
    @client.completions.create(
      model: "model-name",
      messages: [{ role: "user", content: prompt }]
    )
  end
end
```

### Thread-Safe Usage

```ruby
# Thread-safe: each thread gets its own client
threads = 10.times.map do |i|
  Thread.new do
    client = ServiceSDK::Client.new(api_key: ENV["API_KEY"])
    response = client.completions.create(
      model: "model-name",
      messages: [{ role: "user", content: "Thread #{i}" }]
    )
    puts response.content
  end
end

threads.each(&:join)
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidRuby_Syntax(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_NamingConventions(t *testing.T)
```

### Generated SDK Tests

The generated SDK includes a test structure for users to add integration tests:

```
spec/
└── {gem}_spec.rb
```

## Platform Support

### Dependencies

**Runtime Dependencies:**
- `faraday` (~> 2.0) - HTTP client

**Development Dependencies:**
- `bundler` (~> 2.0)
- `rake` (~> 13.0)
- `rspec` (~> 3.0)
- `webmock` (~> 3.0)
- `rubocop` (~> 1.0)

### Minimum Versions

| Platform | Minimum Version | Rationale                      |
|----------|-----------------|--------------------------------|
| Ruby     | 3.0             | Pattern matching, Ractor-safe  |
| Faraday  | 2.0             | Modern middleware, streaming   |

## Future Enhancements

1. **Sorbet types**: Optional Sorbet type signatures for static analysis
2. **Async support**: async-http or concurrent-ruby integration
3. **Connection pooling**: Faraday persistent connections
4. **Request middleware**: Custom middleware for request/response transformation
5. **Response caching**: Built-in response caching with configurable TTL
6. **Metrics**: Request timing and success rate tracking
7. **JRuby support**: Ensure compatibility with JRuby

## References

- [Ruby Style Guide](https://rubystyle.guide/)
- [YARD Documentation](https://yardoc.org/)
- [Faraday HTTP Client](https://lostisland.github.io/faraday/)
- [RubyGems Specification](https://guides.rubygems.org/specification-reference/)
- [Semantic Versioning](https://semver.org/)
