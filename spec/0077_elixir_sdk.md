# RFC 0076: Elixir SDK Generator

## Summary

Add Elixir SDK code generation to the Mizu contract system, enabling production-ready, idiomatic Elixir clients with excellent developer experience for Phoenix, LiveView, and standalone Elixir/OTP applications.

## Motivation

Elixir is a functional language on the BEAM VM, known for fault-tolerance, concurrency, and excellent developer experience. A native Elixir SDK provides:

1. **Idiomatic Elixir**: Pattern matching, pipelines, structs, typespecs
2. **OTP Integration**: Supervision trees, GenServer-based clients, fault tolerance
3. **Streaming Excellence**: Native Stream support for lazy SSE consumption
4. **Type Safety**: Comprehensive typespecs and Dialyzer compatibility
5. **Phoenix Integration**: Works seamlessly in Phoenix/LiveView applications
6. **Production-Ready**: Req-based HTTP with retries, telemetry, and connection pooling

## Design Goals

### Developer Experience (DX)

- **Idiomatic Elixir**: Pattern matching, pipe operator, with statements
- **Tagged tuples**: `{:ok, result}` and `{:error, reason}` return values
- **Structs with typespecs**: Full Dialyzer/Credo compatibility
- **Stream-based SSE**: Lazy enumeration with backpressure
- **ExDoc documentation**: Rich inline documentation with examples
- **Zero configuration**: Sensible defaults, minimal setup required

### Production Readiness

- **Req HTTP client**: Modern, composable HTTP with built-in retry and telemetry
- **Supervision ready**: Clients can be supervised in OTP applications
- **Connection pooling**: Via Finch (Req's default adapter)
- **Telemetry integration**: Request timing and metrics via :telemetry
- **Graceful degradation**: Proper error handling and circuit breaker patterns
- **Hot code reload**: Compatible with OTP releases and hot upgrades

## Architecture

### Project Structure

```
{app_name}/
├── mix.exs                           # Mix project configuration
├── lib/
│   └── {app_name}/
│       ├── client.ex                 # Main client module
│       ├── config.ex                 # Configuration struct
│       ├── types.ex                  # Generated type structs
│       ├── resources/
│       │   └── {resource}.ex         # Resource modules
│       ├── streaming.ex              # SSE streaming support
│       └── errors.ex                 # Error types
│   └── {app_name}.ex                 # Main module (API facade)
```

### Core Components

#### 1. Main Module (`lib/{app_name}.ex`)

The top-level API facade with convenience functions:

```elixir
defmodule {ServiceName} do
  @moduledoc """
  {Service description}

  ## Quick Start

      # Create a client
      client = {ServiceName}.client(api_key: "your-api-key")

      # Make a request
      {:ok, response} = {ServiceName}.{resource}.create(client, params)

  ## Configuration

  Configure via application environment:

      config :{app_name},
        api_key: System.get_env("API_KEY"),
        base_url: "https://api.example.com"

  Or pass options directly:

      client = {ServiceName}.client(
        api_key: "your-api-key",
        base_url: "https://custom.api.com",
        timeout: 120_000
      )
  """

  alias {ServiceName}.Client
  alias {ServiceName}.Config

  @doc """
  Creates a new API client.

  ## Options

    * `:api_key` - API key for authentication (required)
    * `:base_url` - Override the default base URL
    * `:timeout` - Request timeout in milliseconds (default: 60_000)
    * `:max_retries` - Maximum retry attempts (default: 2)
    * `:headers` - Additional headers to include in all requests

  ## Examples

      iex> client = {ServiceName}.client(api_key: "sk-...")
      %{ServiceName}.Client{...}

      iex> client = {ServiceName}.client(
      ...>   api_key: System.get_env("API_KEY"),
      ...>   timeout: 120_000
      ...> )
      %{ServiceName}.Client{...}
  """
  @spec client(keyword()) :: Client.t()
  def client(opts \\ []) do
    Client.new(opts)
  end

  # Resource accessors
  defdelegate {resource}(), to: {ServiceName}.Resources.{Resource}
end
```

#### 2. Client (`lib/{app_name}/client.ex`)

The client struct and request handling:

```elixir
defmodule {ServiceName}.Client do
  @moduledoc """
  HTTP client for {ServiceName} API.

  This module handles all HTTP communication including authentication,
  retries, and error handling.
  """

  alias {ServiceName}.Config
  alias {ServiceName}.Errors

  @type t :: %__MODULE__{
          config: Config.t(),
          req: Req.Request.t()
        }

  defstruct [:config, :req]

  @doc """
  Creates a new client with the given options.

  ## Options

  See `{ServiceName}.client/1` for available options.
  """
  @spec new(keyword()) :: t()
  def new(opts \\ []) do
    config = Config.new(opts)
    req = build_req(config)
    %__MODULE__{config: config, req: req}
  end

  @doc """
  Performs an HTTP request.

  Returns `{:ok, response}` on success or `{:error, exception}` on failure.
  """
  @spec request(t(), keyword()) :: {:ok, map()} | {:error, Exception.t()}
  def request(%__MODULE__{req: req}, opts) do
    method = Keyword.fetch!(opts, :method)
    path = Keyword.fetch!(opts, :path)
    body = Keyword.get(opts, :body)
    headers = Keyword.get(opts, :headers, [])

    request_opts =
      [method: method, url: path]
      |> maybe_add_body(body)
      |> Keyword.merge(headers: headers)

    case Req.request(req, request_opts) do
      {:ok, %Req.Response{status: status, body: body}} when status in 200..299 ->
        {:ok, body}

      {:ok, %Req.Response{status: 204}} ->
        {:ok, nil}

      {:ok, %Req.Response{status: status, body: body}} ->
        {:error, Errors.from_response(status, body)}

      {:error, exception} ->
        {:error, Errors.ConnectionError.exception(message: Exception.message(exception))}
    end
  end

  @doc """
  Performs an HTTP request, raising on error.
  """
  @spec request!(t(), keyword()) :: map() | nil
  def request!(client, opts) do
    case request(client, opts) do
      {:ok, result} -> result
      {:error, exception} -> raise exception
    end
  end

  @doc """
  Performs a streaming SSE request.

  Returns a `Stream` that yields parsed SSE events.
  """
  @spec stream(t(), keyword()) :: Enumerable.t()
  def stream(%__MODULE__{req: req, config: config}, opts) do
    method = Keyword.fetch!(opts, :method)
    path = Keyword.fetch!(opts, :path)
    body = Keyword.get(opts, :body)

    Stream.resource(
      fn -> start_stream(req, method, path, body) end,
      &next_event/1,
      &close_stream/1
    )
  end

  # Private functions

  defp build_req(config) do
    Req.new(
      base_url: config.base_url,
      headers: build_headers(config),
      receive_timeout: config.timeout,
      retry: :transient,
      max_retries: config.max_retries
    )
  end

  defp build_headers(config) do
    headers = [
      {"content-type", "application/json"},
      {"accept", "application/json"}
    ]

    headers = headers ++ Map.to_list(config.headers)
    apply_auth(headers, config)
  end

  defp apply_auth(headers, %{api_key: nil}), do: headers

  defp apply_auth(headers, %{api_key: key, auth_mode: :bearer}) do
    [{"authorization", "Bearer #{key}"} | headers]
  end

  defp apply_auth(headers, %{api_key: key, auth_mode: :basic}) do
    encoded = Base.encode64(key)
    [{"authorization", "Basic #{encoded}"} | headers]
  end

  defp apply_auth(headers, %{api_key: key, auth_mode: :header}) do
    [{"x-api-key", key} | headers]
  end

  defp maybe_add_body(opts, nil), do: opts
  defp maybe_add_body(opts, body), do: Keyword.put(opts, :json, body)

  defp start_stream(req, method, path, body) do
    # Implementation for SSE streaming
    request_opts =
      [method: method, url: path, into: :self]
      |> maybe_add_body(body)
      |> Keyword.merge(headers: [{"accept", "text/event-stream"}])

    case Req.request(req, request_opts) do
      {:ok, response} -> {:streaming, response, ""}
      {:error, reason} -> {:error, reason}
    end
  end

  defp next_event({:error, reason}), do: {:halt, {:error, reason}}

  defp next_event({:streaming, response, buffer}) do
    receive do
      {ref, {:data, chunk}} when ref == response.ref ->
        process_sse_chunk(response, buffer <> chunk)

      {ref, :done} when ref == response.ref ->
        {:halt, :done}

      {ref, {:error, reason}} when ref == response.ref ->
        {:halt, {:error, reason}}
    after
      60_000 ->
        {:halt, {:error, :timeout}}
    end
  end

  defp process_sse_chunk(response, buffer) do
    case String.split(buffer, "\n\n", parts: 2) do
      [event, rest] ->
        case parse_sse_event(event) do
          {:ok, :done} ->
            {:halt, :done}

          {:ok, data} ->
            {[data], {:streaming, response, rest}}

          :skip ->
            next_event({:streaming, response, rest})
        end

      [incomplete] ->
        next_event({:streaming, response, incomplete})
    end
  end

  defp parse_sse_event(event) do
    lines = String.split(event, "\n")

    data =
      lines
      |> Enum.filter(&String.starts_with?(&1, "data:"))
      |> Enum.map(&String.trim_leading(&1, "data:"))
      |> Enum.map(&String.trim/1)
      |> Enum.join("\n")

    cond do
      data == "" -> :skip
      data == "[DONE]" -> {:ok, :done}
      true -> {:ok, Jason.decode!(data)}
    end
  end

  defp close_stream(:done), do: :ok
  defp close_stream({:error, _}), do: :ok
  defp close_stream({:streaming, _response, _buffer}), do: :ok
end
```

#### 3. Config (`lib/{app_name}/config.ex`)

Configuration struct:

```elixir
defmodule {ServiceName}.Config do
  @moduledoc """
  Configuration for the {ServiceName} client.
  """

  @type auth_mode :: :bearer | :basic | :header | :none

  @type t :: %__MODULE__{
          api_key: String.t() | nil,
          base_url: String.t(),
          timeout: pos_integer(),
          max_retries: non_neg_integer(),
          auth_mode: auth_mode(),
          headers: map()
        }

  @enforce_keys []
  defstruct api_key: nil,
            base_url: "{default_base_url}",
            timeout: 60_000,
            max_retries: 2,
            auth_mode: :bearer,
            headers: %{}

  @doc """
  Creates a new configuration.

  Options are merged with application config and defaults.

  ## Examples

      iex> Config.new(api_key: "sk-...")
      %Config{api_key: "sk-...", ...}
  """
  @spec new(keyword()) :: t()
  def new(opts \\ []) do
    # Merge: defaults < app config < opts
    app_config = Application.get_all_env(:{app_name}) |> Keyword.take([:api_key, :base_url, :timeout, :max_retries, :auth_mode, :headers])

    merged =
      %__MODULE__{}
      |> Map.from_struct()
      |> Map.merge(Map.new(app_config))
      |> Map.merge(Map.new(opts))

    struct!(__MODULE__, merged)
  end
end
```

#### 4. Types (`lib/{app_name}/types.ex`)

Generated struct types with typespecs:

```elixir
defmodule {ServiceName}.Types do
  @moduledoc """
  Type definitions for {ServiceName} API.

  All types are defined as structs with full typespecs for Dialyzer compatibility.
  """

  # Helper for struct creation from maps
  defmodule Helpers do
    @moduledoc false

    @spec from_map(module(), map()) :: struct()
    def from_map(module, map) when is_map(map) do
      # Convert string keys to atoms
      map =
        map
        |> Enum.map(fn
          {k, v} when is_binary(k) -> {String.to_existing_atom(k), v}
          kv -> kv
        end)
        |> Map.new()

      struct(module, map)
    end

    def from_map(_module, nil), do: nil

    @spec to_map(struct()) :: map()
    def to_map(%{__struct__: _} = struct) do
      struct
      |> Map.from_struct()
      |> Enum.reject(fn {_k, v} -> is_nil(v) end)
      |> Enum.map(fn {k, v} -> {k, serialize_value(v)} end)
      |> Map.new()
    end

    defp serialize_value(%{__struct__: _} = struct), do: to_map(struct)
    defp serialize_value(list) when is_list(list), do: Enum.map(list, &serialize_value/1)
    defp serialize_value(%DateTime{} = dt), do: DateTime.to_iso8601(dt)
    defp serialize_value(value), do: value
  end

  # Struct type with fields
  defmodule {TypeName} do
    @moduledoc """
    {Type description}
    """

    @type t :: %__MODULE__{
            field_name: String.t(),
            optional_field: String.t() | nil
          }

    @derive Jason.Encoder
    defstruct [:field_name, :optional_field]

    @doc """
    Creates a new {TypeName} from a map.
    """
    @spec from_map(map()) :: t()
    def from_map(map), do: {ServiceName}.Types.Helpers.from_map(__MODULE__, map)

    @doc """
    Converts to a map for JSON encoding.
    """
    @spec to_map(t()) :: map()
    def to_map(struct), do: {ServiceName}.Types.Helpers.to_map(struct)
  end

  # Enum as module constants
  defmodule Role do
    @moduledoc """
    Valid role values.
    """

    @type t :: :user | :assistant | :system

    @user "user"
    @assistant "assistant"
    @system "system"

    def user, do: @user
    def assistant, do: @assistant
    def system, do: @system

    @spec all() :: [String.t()]
    def all, do: [@user, @assistant, @system]

    @spec valid?(String.t()) :: boolean()
    def valid?(value), do: value in all()
  end

  # Discriminated union via pattern matching
  defmodule ContentBlock do
    @moduledoc """
    Union type for content blocks.

    Dispatches to the appropriate variant based on the `type` field.
    """

    alias {ServiceName}.Types.{TextBlock, ImageBlock, ToolUseBlock}

    @type t :: TextBlock.t() | ImageBlock.t() | ToolUseBlock.t()

    @doc """
    Creates the appropriate variant from a map.

    ## Examples

        iex> ContentBlock.from_map(%{"type" => "text", "text" => "Hello"})
        %TextBlock{type: "text", text: "Hello"}
    """
    @spec from_map(map()) :: t()
    def from_map(%{"type" => "text"} = map), do: TextBlock.from_map(map)
    def from_map(%{type: "text"} = map), do: TextBlock.from_map(map)
    def from_map(%{"type" => "image"} = map), do: ImageBlock.from_map(map)
    def from_map(%{type: "image"} = map), do: ImageBlock.from_map(map)
    def from_map(%{"type" => "tool_use"} = map), do: ToolUseBlock.from_map(map)
    def from_map(%{type: "tool_use"} = map), do: ToolUseBlock.from_map(map)
    def from_map(map), do: raise(ArgumentError, "Unknown content block type: #{inspect(map)}")
  end

  defmodule TextBlock do
    @moduledoc "Text content block"

    @type t :: %__MODULE__{
            type: String.t(),
            text: String.t()
          }

    @derive Jason.Encoder
    defstruct type: "text", text: nil

    def from_map(map), do: {ServiceName}.Types.Helpers.from_map(__MODULE__, map)
    def to_map(struct), do: {ServiceName}.Types.Helpers.to_map(struct)
  end
end
```

#### 5. Resources (`lib/{app_name}/resources/{resource}.ex`)

Resource modules with operation functions:

```elixir
defmodule {ServiceName}.Resources.{Resource} do
  @moduledoc """
  Operations for {resource}.

  {Resource description}
  """

  alias {ServiceName}.Client
  alias {ServiceName}.Types.{InputType, OutputType, StreamItem}

  @doc """
  {Method description}

  ## Parameters

    * `client` - The API client
    * `params` - Request parameters:
      * `:model` - The model to use (required)
      * `:messages` - List of messages (required)
      * `:max_tokens` - Maximum tokens to generate

  ## Returns

    * `{:ok, %OutputType{}}` - Success
    * `{:error, exception}` - Failure

  ## Examples

      iex> {:ok, response} = {Resource}.create(client, model: "model", messages: [...])
      {:ok, %OutputType{...}}
  """
  @spec create(Client.t(), keyword()) :: {:ok, OutputType.t()} | {:error, Exception.t()}
  def create(client, params) do
    body = build_request_body(params)

    case Client.request(client, method: :post, path: "/v1/{resource}", body: body) do
      {:ok, response} -> {:ok, OutputType.from_map(response)}
      {:error, _} = error -> error
    end
  end

  @doc """
  Same as `create/2` but raises on error.
  """
  @spec create!(Client.t(), keyword()) :: OutputType.t()
  def create!(client, params) do
    case create(client, params) do
      {:ok, result} -> result
      {:error, exception} -> raise exception
    end
  end

  @doc """
  Streaming version of create.

  Returns a `Stream` of events.

  ## Examples

      # With Enum
      client
      |> {Resource}.stream(model: "model", messages: [...])
      |> Enum.each(fn event ->
        IO.write(event.delta.text)
      end)

      # With Stream (lazy)
      client
      |> {Resource}.stream(model: "model", messages: [...])
      |> Stream.take_while(fn event -> event.type != "message_stop" end)
      |> Enum.to_list()
  """
  @spec stream(Client.t(), keyword()) :: Enumerable.t()
  def stream(client, params) do
    body = build_request_body(params) |> Map.put(:stream, true)

    client
    |> Client.stream(method: :post, path: "/v1/{resource}", body: body)
    |> Stream.map(&StreamItem.from_map/1)
  end

  # Private helpers

  defp build_request_body(params) do
    params
    |> Keyword.take([:model, :messages, :max_tokens, :temperature])
    |> Map.new()
  end
end
```

#### 6. Streaming (`lib/{app_name}/streaming.ex`)

SSE streaming utilities:

```elixir
defmodule {ServiceName}.Streaming do
  @moduledoc """
  Server-Sent Events (SSE) streaming support.

  This module provides utilities for parsing and handling SSE streams.
  """

  @type event :: %{
          optional(:id) => String.t(),
          optional(:event) => String.t(),
          :data => term()
        }

  @doc """
  Parses an SSE event string into a map.

  ## Examples

      iex> Streaming.parse_event("data: {\\"text\\": \\"hello\\"}")
      {:ok, %{"text" => "hello"}}

      iex> Streaming.parse_event("data: [DONE]")
      {:ok, :done}
  """
  @spec parse_event(String.t()) :: {:ok, term()} | {:error, term()} | :skip
  def parse_event(event_string) do
    lines = String.split(event_string, "\n")

    data =
      lines
      |> Enum.filter(&String.starts_with?(&1, "data:"))
      |> Enum.map(fn line ->
        line
        |> String.trim_leading("data:")
        |> String.trim()
      end)
      |> Enum.join("\n")

    cond do
      data == "" -> :skip
      data == "[DONE]" -> {:ok, :done}
      true -> parse_json(data)
    end
  end

  defp parse_json(data) do
    case Jason.decode(data) do
      {:ok, parsed} -> {:ok, parsed}
      {:error, reason} -> {:error, {:json_parse_error, reason, data}}
    end
  end

  @doc """
  Transforms a stream of raw chunks into parsed SSE events.

  ## Examples

      raw_stream
      |> Streaming.parse_stream()
      |> Enum.each(&process_event/1)
  """
  @spec parse_stream(Enumerable.t()) :: Enumerable.t()
  def parse_stream(chunk_stream) do
    chunk_stream
    |> Stream.transform("", &accumulate_and_parse/2)
    |> Stream.filter(fn
      {:ok, _} -> true
      _ -> false
    end)
    |> Stream.map(fn {:ok, event} -> event end)
    |> Stream.take_while(fn event -> event != :done end)
  end

  defp accumulate_and_parse(chunk, buffer) do
    buffer = buffer <> chunk
    {events, remaining} = extract_events(buffer, [])
    {events, remaining}
  end

  defp extract_events(buffer, events) do
    case String.split(buffer, "\n\n", parts: 2) do
      [event, rest] ->
        parsed = parse_event(event)
        extract_events(rest, events ++ [parsed])

      [incomplete] ->
        {events, incomplete}
    end
  end
end
```

#### 7. Errors (`lib/{app_name}/errors.ex`)

Exception types:

```elixir
defmodule {ServiceName}.Errors do
  @moduledoc """
  Error types for {ServiceName} API.

  All errors are Elixir exceptions that can be raised or matched.
  """

  defmodule SDKError do
    @moduledoc "Base error for all SDK errors"
    defexception [:message]

    @type t :: %__MODULE__{message: String.t()}
  end

  defmodule APIError do
    @moduledoc "Base error for API errors (HTTP 4xx/5xx)"
    defexception [:message, :status, :body]

    @type t :: %__MODULE__{
            message: String.t(),
            status: integer(),
            body: term()
          }

    @impl true
    def message(%{message: msg, status: status}) do
      "API Error (#{status}): #{msg}"
    end
  end

  defmodule BadRequestError do
    @moduledoc "HTTP 400 Bad Request"
    defexception [:message, :status, :body]
    @type t :: %__MODULE__{message: String.t(), status: integer(), body: term()}
  end

  defmodule AuthenticationError do
    @moduledoc "HTTP 401 Unauthorized"
    defexception [:message, :status, :body]
    @type t :: %__MODULE__{message: String.t(), status: integer(), body: term()}
  end

  defmodule PermissionDeniedError do
    @moduledoc "HTTP 403 Forbidden"
    defexception [:message, :status, :body]
    @type t :: %__MODULE__{message: String.t(), status: integer(), body: term()}
  end

  defmodule NotFoundError do
    @moduledoc "HTTP 404 Not Found"
    defexception [:message, :status, :body]
    @type t :: %__MODULE__{message: String.t(), status: integer(), body: term()}
  end

  defmodule UnprocessableEntityError do
    @moduledoc "HTTP 422 Unprocessable Entity"
    defexception [:message, :status, :body]
    @type t :: %__MODULE__{message: String.t(), status: integer(), body: term()}
  end

  defmodule RateLimitError do
    @moduledoc "HTTP 429 Too Many Requests"
    defexception [:message, :status, :body, :retry_after]

    @type t :: %__MODULE__{
            message: String.t(),
            status: integer(),
            body: term(),
            retry_after: integer() | nil
          }
  end

  defmodule InternalServerError do
    @moduledoc "HTTP 5xx Server Error"
    defexception [:message, :status, :body]
    @type t :: %__MODULE__{message: String.t(), status: integer(), body: term()}
  end

  defmodule ConnectionError do
    @moduledoc "Network/connection error"
    defexception [:message]
    @type t :: %__MODULE__{message: String.t()}
  end

  defmodule TimeoutError do
    @moduledoc "Request timeout"
    defexception [:message]
    @type t :: %__MODULE__{message: String.t()}
  end

  defmodule StreamError do
    @moduledoc "Error parsing SSE stream"
    defexception [:message, :data]
    @type t :: %__MODULE__{message: String.t(), data: term()}
  end

  @doc """
  Creates an appropriate error from an HTTP response.
  """
  @spec from_response(integer(), term()) :: Exception.t()
  def from_response(status, body) do
    message = extract_message(body)

    case status do
      400 -> %BadRequestError{message: message, status: status, body: body}
      401 -> %AuthenticationError{message: message, status: status, body: body}
      403 -> %PermissionDeniedError{message: message, status: status, body: body}
      404 -> %NotFoundError{message: message, status: status, body: body}
      422 -> %UnprocessableEntityError{message: message, status: status, body: body}
      429 -> %RateLimitError{message: message, status: status, body: body, retry_after: nil}
      s when s in 500..599 -> %InternalServerError{message: message, status: status, body: body}
      _ -> %APIError{message: message, status: status, body: body}
    end
  end

  defp extract_message(%{"message" => msg}), do: msg
  defp extract_message(%{"error" => %{"message" => msg}}), do: msg
  defp extract_message(%{"error" => msg}) when is_binary(msg), do: msg
  defp extract_message(body) when is_binary(body), do: body
  defp extract_message(_), do: "Unknown error"
end
```

## Type Mapping

### Primitive Types

| Contract Type     | Elixir Type              | Typespec              |
|-------------------|--------------------------|----------------------|
| `string`          | `String.t()`             | `String.t()`         |
| `bool`, `boolean` | `boolean()`              | `boolean()`          |
| `int`             | `integer()`              | `integer()`          |
| `int8`            | `integer()`              | `integer()`          |
| `int16`           | `integer()`              | `integer()`          |
| `int32`           | `integer()`              | `integer()`          |
| `int64`           | `integer()`              | `integer()`          |
| `uint`            | `non_neg_integer()`      | `non_neg_integer()`  |
| `uint8`           | `non_neg_integer()`      | `non_neg_integer()`  |
| `uint16`          | `non_neg_integer()`      | `non_neg_integer()`  |
| `uint32`          | `non_neg_integer()`      | `non_neg_integer()`  |
| `uint64`          | `non_neg_integer()`      | `non_neg_integer()`  |
| `float32`         | `float()`                | `float()`            |
| `float64`         | `float()`                | `float()`            |
| `time.Time`       | `DateTime.t()`           | `DateTime.t()`       |
| `json.RawMessage` | `map()`                  | `map()`              |
| `any`             | `term()`                 | `term()`             |

### Collection Types

| Contract Type      | Elixir Type              | Typespec               |
|--------------------|--------------------------|------------------------|
| `[]T`              | `list(T)`                | `[T.t()]`              |
| `map[string]T`     | `map()`                  | `%{String.t() => T.t()}` |

### Optional/Nullable

| Contract         | Elixir               | Typespec              |
|------------------|----------------------|----------------------|
| `optional: T`    | `T \| nil`           | `T.t() \| nil`       |
| `nullable: T`    | `T \| nil`           | `T.t() \| nil`       |

### Struct Fields

Fields with `optional: true` or `nullable: true` have `nil` default:

```elixir
defmodule Request do
  @type t :: %__MODULE__{
          required: String.t(),
          optional_field: String.t() | nil
        }

  defstruct [:required, :optional_field]
end
```

### Enum/Const Values

Fields with `enum` constraint generate constant modules:

```elixir
defmodule Role do
  @type t :: String.t()

  @user "user"
  @assistant "assistant"
  @system "system"

  def user, do: @user
  def assistant, do: @assistant
  def system, do: @system

  @spec all() :: [t()]
  def all, do: [@user, @assistant, @system]

  @spec valid?(t()) :: boolean()
  def valid?(value), do: value in all()
end
```

### Discriminated Unions

Union types use pattern matching on the tag field:

```elixir
defmodule ContentBlock do
  @type t :: TextBlock.t() | ImageBlock.t() | ToolUseBlock.t()

  @spec from_map(map()) :: t()
  def from_map(%{"type" => "text"} = map), do: TextBlock.from_map(map)
  def from_map(%{type: "text"} = map), do: TextBlock.from_map(map)
  def from_map(%{"type" => "image"} = map), do: ImageBlock.from_map(map)
  def from_map(%{type: "image"} = map), do: ImageBlock.from_map(map)
  def from_map(%{"type" => "tool_use"} = map), do: ToolUseBlock.from_map(map)
  def from_map(%{type: "tool_use"} = map), do: ToolUseBlock.from_map(map)
  def from_map(map), do: raise(ArgumentError, "Unknown type: #{inspect(map)}")
end
```

## HTTP Client Implementation

### Req-Based HTTP

The SDK uses [Req](https://hexdocs.pm/req) for HTTP, providing:

- Built-in retry with exponential backoff
- Telemetry integration
- Connection pooling via Finch
- Composable request/response pipeline

```elixir
defp build_req(config) do
  Req.new(
    base_url: config.base_url,
    headers: build_headers(config),
    receive_timeout: config.timeout,
    retry: :transient,
    max_retries: config.max_retries,
    # Telemetry events
    plug: {Req.Steps, [:telemetry]}
  )
end
```

### SSE Streaming

Streaming uses Req's `:into` option for chunked responses:

```elixir
def stream(client, opts) do
  body = Keyword.fetch!(opts, :body)
  path = Keyword.fetch!(opts, :path)

  client.req
  |> Req.merge(
    method: :post,
    url: path,
    json: body,
    headers: [{"accept", "text/event-stream"}],
    into: :self
  )
  |> Req.request!()
  |> receive_stream()
end

defp receive_stream(response) do
  Stream.resource(
    fn -> {response.ref, ""} end,
    fn {ref, buffer} ->
      receive do
        {^ref, {:data, chunk}} ->
          {events, remaining} = parse_sse_buffer(buffer <> chunk)
          {events, {ref, remaining}}

        {^ref, :done} ->
          {:halt, :done}
      after
        60_000 ->
          {:halt, {:error, :timeout}}
      end
    end,
    fn _ -> :ok end
  )
end
```

## Configuration

### Application Config

```elixir
# config/config.exs
config :my_service,
  api_key: System.get_env("API_KEY"),
  base_url: "https://api.example.com",
  timeout: 60_000,
  max_retries: 2
```

### Runtime Config

```elixir
# config/runtime.exs
config :my_service,
  api_key: System.fetch_env!("API_KEY")
```

### Direct Options

```elixir
client = MyService.client(
  api_key: "sk-...",
  base_url: "https://custom.api.com",
  timeout: 120_000
)
```

## Naming Conventions

### Elixir Naming

| Contract       | Elixir                  |
|----------------|-------------------------|
| `user-id`      | `:user_id`              |
| `user_name`    | `:user_name`            |
| `UserData`     | `UserData` (module)     |
| `create`       | `create/2`              |
| `get-user`     | `get_user/2`            |
| `maxTokens`    | `:max_tokens`           |

Functions:
- `to_snake(s)`: Converts to snake_case (for atoms/functions)
- `to_pascal(s)`: Converts to PascalCase (for modules)
- `sanitize_ident(s)`: Removes invalid characters

Reserved words handled by appending underscore:
- `def` -> `def_`
- `end` -> `end_`
- `do` -> `do_`

## Code Generation

### Generator Structure

```go
package sdkelixir

type Config struct {
    // AppName is the OTP application name.
    // Default: sanitized lowercase service name with underscores.
    AppName string

    // ModuleName is the root Elixir module name.
    // Default: PascalCase service name.
    ModuleName string

    // Version is the package version for mix.exs.
    Version string
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── mix.exs.tmpl              # Mix project config
├── main.ex.tmpl              # Main module
├── client.ex.tmpl            # Client module
├── config.ex.tmpl            # Config struct
├── types.ex.tmpl             # Type definitions
├── resources.ex.tmpl         # Resource modules
├── streaming.ex.tmpl         # SSE streaming
└── errors.ex.tmpl            # Error types
```

### Generated Files

| File                              | Purpose                      |
|-----------------------------------|------------------------------|
| `mix.exs`                         | Mix project configuration    |
| `lib/{app}.ex`                    | Main module (API facade)     |
| `lib/{app}/client.ex`             | HTTP client                  |
| `lib/{app}/config.ex`             | Configuration struct         |
| `lib/{app}/types.ex`              | All type definitions         |
| `lib/{app}/resources/{name}.ex`   | Resource modules             |
| `lib/{app}/streaming.ex`          | SSE streaming utilities      |
| `lib/{app}/errors.ex`             | Error types                  |

## Usage Examples

### Basic Usage

```elixir
# Add to mix.exs
defp deps do
  [{:my_service, "~> 1.0"}]
end

# Create client
client = MyService.client(api_key: "your-api-key")

# Make a request
{:ok, response} = MyService.Resources.Messages.create(client,
  model: "model-name",
  messages: [%{role: "user", content: "Hello"}]
)

IO.puts(response.content)
```

### Streaming with Enum

```elixir
client
|> MyService.Resources.Messages.stream(
  model: "model-name",
  messages: [%{role: "user", content: "Hello"}]
)
|> Enum.each(fn event ->
  if event.delta && event.delta.text do
    IO.write(event.delta.text)
  end
end)
```

### Streaming with Stream (Lazy)

```elixir
client
|> MyService.Resources.Messages.stream(model: "model-name", messages: [...])
|> Stream.take_while(fn event -> event.type != "message_stop" end)
|> Stream.map(& &1.delta.text)
|> Stream.filter(& &1)
|> Enum.join()
```

### Error Handling

```elixir
case MyService.Resources.Messages.create(client, params) do
  {:ok, response} ->
    IO.puts(response.content)

  {:error, %MyService.Errors.RateLimitError{retry_after: retry}} ->
    Process.sleep(retry * 1000)
    # retry...

  {:error, %MyService.Errors.AuthenticationError{}} ->
    IO.puts("Invalid API key")

  {:error, %MyService.Errors.APIError{status: status, message: msg}} ->
    IO.puts("API Error #{status}: #{msg}")

  {:error, %MyService.Errors.ConnectionError{message: msg}} ->
    IO.puts("Network error: #{msg}")
end
```

### With Statement

```elixir
with {:ok, client} <- get_client(),
     {:ok, response} <- MyService.Resources.Messages.create(client, params),
     {:ok, parsed} <- parse_response(response) do
  {:ok, parsed}
else
  {:error, %MyService.Errors.RateLimitError{}} ->
    {:error, :rate_limited}

  {:error, reason} ->
    {:error, reason}
end
```

### Phoenix Integration

```elixir
# lib/my_app/services/ai_service.ex
defmodule MyApp.AIService do
  alias MyService.Client
  alias MyService.Resources.Messages

  def client do
    MyService.client(
      api_key: Application.fetch_env!(:my_app, :api_key)
    )
  end

  def generate_response(prompt) do
    client()
    |> Messages.create(
      model: "model-name",
      messages: [%{role: "user", content: prompt}]
    )
  end

  def stream_response(prompt) do
    client()
    |> Messages.stream(
      model: "model-name",
      messages: [%{role: "user", content: prompt}]
    )
  end
end

# In LiveView
defmodule MyAppWeb.ChatLive do
  use MyAppWeb, :live_view

  def handle_event("send", %{"message" => message}, socket) do
    task = Task.async(fn ->
      MyApp.AIService.client()
      |> MyService.Resources.Messages.stream(
        model: "model-name",
        messages: [%{role: "user", content: message}]
      )
      |> Enum.each(fn event ->
        send(self(), {:stream_event, event})
      end)
    end)

    {:noreply, assign(socket, task: task)}
  end

  def handle_info({:stream_event, event}, socket) do
    # Update UI with streaming event
    {:noreply, update(socket, :response, &(&1 <> (event.delta.text || "")))}
  end
end
```

### Supervised Client

```elixir
# For applications needing a shared client
defmodule MyApp.APIClient do
  use GenServer

  def start_link(opts) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  def init(opts) do
    client = MyService.client(opts)
    {:ok, %{client: client}}
  end

  def get_client do
    GenServer.call(__MODULE__, :get_client)
  end

  def handle_call(:get_client, _from, state) do
    {:reply, state.client, state}
  end
end
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidElixir_Syntax(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_NamingConventions(t *testing.T)
```

### Generated SDK Tests

```
test/
├── {app}_test.exs
├── client_test.exs
├── types_test.exs
└── resources_test.exs
```

## Platform Support

### Dependencies

**Runtime Dependencies:**
- `req` (~> 0.4) - HTTP client
- `jason` (~> 1.4) - JSON encoding/decoding

**Development Dependencies:**
- `dialyxir` (~> 1.0) - Dialyzer for typespecs
- `credo` (~> 1.7) - Static analysis
- `ex_doc` (~> 0.30) - Documentation
- `mox` (~> 1.0) - Mocking for tests

### Minimum Versions

| Platform | Minimum Version | Rationale                        |
|----------|-----------------|----------------------------------|
| Elixir   | 1.14            | Dbg, improved diagnostics        |
| OTP      | 25              | Modern BEAM features             |
| Req      | 0.4             | Stable streaming support         |

## Future Enhancements

1. **Telemetry events**: Emit telemetry for request timing and errors
2. **Circuit breaker**: Built-in circuit breaker via `:fuse`
3. **Connection pooling config**: Expose Finch pool options
4. **Batch requests**: Support for batched API calls
5. **WebSocket streaming**: Support for WebSocket-based streaming
6. **Nx integration**: Tensor support for ML APIs
7. **Broadway integration**: For high-volume streaming

## References

- [Elixir Style Guide](https://github.com/christopheradams/elixir_style_guide)
- [Req Documentation](https://hexdocs.pm/req)
- [Typespecs Reference](https://hexdocs.pm/elixir/typespecs.html)
- [Mix Tasks](https://hexdocs.pm/mix/Mix.html)
- [OTP Design Principles](https://www.erlang.org/doc/design_principles/des_princ.html)
