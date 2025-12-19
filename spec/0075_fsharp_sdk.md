# RFC 0075: F# SDK Generator

## Summary

Add F# SDK code generation to the Mizu contract system, enabling idiomatic, type-safe F# clients that leverage the language's functional-first features including discriminated unions, computation expressions, and pattern matching.

## Motivation

F# is a mature functional-first language on .NET with growing adoption in domains requiring correctness and expressiveness. A native F# SDK provides:

1. **Discriminated Unions**: Native sum types perfectly match contract union types with exhaustive pattern matching
2. **Computation Expressions**: Clean async workflows with `task { }` syntax (F# 6+)
3. **Type Inference**: Minimal boilerplate with F#'s powerful type system
4. **Option Type**: Idiomatic null-safety with `option` instead of nullable references
5. **Result Type**: Functional error handling with `Result<'T, 'E>`
6. **Pipeline Operator**: Fluent data transformation with `|>`
7. **Pattern Matching**: First-class exhaustive matching on all types
8. **Immutability**: Records are immutable by default with copy-and-update syntax

## Design Goals

### Developer Experience (DX)

- **Idiomatic F#**: Follow F# conventions and functional patterns
- **Task-based async**: Use `task { }` computation expression (F# 6+)
- **Discriminated unions**: Native DU types for contract unions
- **Option types**: Use `option` for optional fields, not null
- **Result-based errors**: Return `Result<'T, SdkError>` for non-streaming operations
- **IAsyncEnumerable streaming**: SSE via async sequences with cancellation
- **XML documentation**: F#-style `///` documentation comments
- **Minimal dependencies**: System.Net.Http, System.Text.Json, FSharp.Core

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff
- **Timeout handling**: Per-request timeout via CancellationToken
- **Cancellation**: Full CancellationToken support throughout
- **Error handling**: Discriminated union for typed errors
- **Thread safety**: HttpClient best practices
- **Pipeline-friendly**: Results work naturally with `|>` and `Result.map`

## Architecture

### Package Structure

```
{PackageName}/
├── {PackageName}.fsproj          # Project file
└── src/
    ├── Types.fs                  # Type definitions (records, DUs)
    ├── Errors.fs                 # Error types (SdkError DU)
    ├── Http.fs                   # HTTP client implementation
    ├── Streaming.fs              # SSE streaming support
    ├── Resources.fs              # Resource modules
    └── Client.fs                 # Main client module
```

### Core Components

#### 1. Types (`Types.fs`)

All model types use F# records and discriminated unions:

```fsharp
namespace {Namespace}

open System
open System.Text.Json.Serialization

/// Request/response model description.
[<CLIMutable>]
type {TypeName} = {
    /// Field description from contract.
    [<JsonPropertyName("field_name")>]
    FieldName: string

    /// Optional field using F# option type.
    [<JsonPropertyName("optional_field")>]
    OptionalField: string option
}

/// Discriminated union for content blocks.
[<JsonConverter(typeof<ContentBlockConverter>)>]
type ContentBlock =
    /// Text content block.
    | TextBlock of text: string
    /// Image content block.
    | ImageBlock of url: string * mediaType: string option
    /// Tool use content block.
    | ToolUseBlock of id: string * name: string * input: JsonElement

/// Role enum as discriminated union.
type Role =
    | User
    | Assistant
    | System

module Role =
    let toString = function
        | User -> "user"
        | Assistant -> "assistant"
        | System -> "system"

    let fromString = function
        | "user" -> Some User
        | "assistant" -> Some Assistant
        | "system" -> Some System
        | _ -> None
```

#### 2. Errors (`Errors.fs`)

Discriminated union for typed error handling:

```fsharp
namespace {Namespace}

open System

/// SDK error types as a discriminated union.
type SdkError =
    /// Network connection failed.
    | ConnectionError of exn: Exception
    /// Server returned an error status code.
    | ApiError of statusCode: int * message: string * responseBody: string option
    /// Request timed out.
    | TimeoutError of timeout: TimeSpan
    /// Request was cancelled.
    | CancelledError
    /// Failed to serialize request.
    | SerializationError of exn: Exception
    /// Failed to deserialize response.
    | DeserializationError of exn: Exception

module SdkError =
    /// Returns true if this error is potentially retriable.
    let isRetriable = function
        | ConnectionError _ -> true
        | ApiError (status, _, _) -> status >= 500 || status = 429
        | TimeoutError _ -> true
        | CancelledError -> false
        | SerializationError _ -> false
        | DeserializationError _ -> false

    /// Returns the error message.
    let message = function
        | ConnectionError ex -> sprintf "Connection failed: %s" ex.Message
        | ApiError (status, msg, _) -> sprintf "API error %d: %s" status msg
        | TimeoutError ts -> sprintf "Request timed out after %O" ts
        | CancelledError -> "Request was cancelled"
        | SerializationError ex -> sprintf "Serialization failed: %s" ex.Message
        | DeserializationError ex -> sprintf "Deserialization failed: %s" ex.Message
```

#### 3. Client Configuration (`Client.fs`)

```fsharp
namespace {Namespace}

open System
open System.Net.Http

/// Authentication mode for API requests.
type AuthMode =
    | Bearer
    | Basic
    | None

/// Client configuration options.
type ClientOptions = {
    /// API key for authentication.
    ApiKey: string option
    /// Base URL for API requests.
    BaseUrl: string
    /// Request timeout. Default: 60 seconds.
    Timeout: TimeSpan
    /// Maximum retry attempts. Default: 2.
    MaxRetries: int
    /// Default headers.
    DefaultHeaders: Map<string, string>
    /// Authentication mode.
    AuthMode: AuthMode
    /// Optional HttpClient to use.
    HttpClient: HttpClient option
}

module ClientOptions =
    /// Default client options.
    let defaults = {
        ApiKey = None
        BaseUrl = "{default_base_url}"
        Timeout = TimeSpan.FromSeconds(60.0)
        MaxRetries = 2
        DefaultHeaders = Map.empty
        AuthMode = Bearer
        HttpClient = None
    }

    /// Create options with API key.
    let withApiKey apiKey opts = { opts with ApiKey = Some apiKey }

    /// Create options with base URL.
    let withBaseUrl url opts = { opts with BaseUrl = url }

    /// Create options with timeout.
    let withTimeout timeout opts = { opts with Timeout = timeout }

    /// Create options with max retries.
    let withMaxRetries retries opts = { opts with MaxRetries = retries }

    /// Add a default header.
    let withHeader key value opts =
        { opts with DefaultHeaders = opts.DefaultHeaders |> Map.add key value }

/// The main SDK client providing access to all API resources.
type {ServiceName}Client(options: ClientOptions) =
    let httpClient =
        options.HttpClient
        |> Option.defaultWith (fun () ->
            let client = new HttpClient()
            client.Timeout <- options.Timeout
            client)
    let ownsHttpClient = options.HttpClient.IsNone
    let http = HttpClientWrapper(httpClient, options)

    /// Create client with default options.
    new() = {ServiceName}Client(ClientOptions.defaults)

    /// Create client with API key.
    new(apiKey: string) =
        {ServiceName}Client(ClientOptions.defaults |> ClientOptions.withApiKey apiKey)

    /// Access to {resource} operations.
    member _.{Resource} = {Resource}Resource(http)

    interface IDisposable with
        member _.Dispose() =
            if ownsHttpClient then httpClient.Dispose()
```

#### 4. Resources (`Resources.fs`)

Resource modules provide namespaced method access:

```fsharp
namespace {Namespace}

open System
open System.Threading
open System.Threading.Tasks

/// Operations for {resource}.
type {Resource}Resource internal (http: HttpClientWrapper) =

    /// Method description from contract.
    member _.{MethodName}Async
        (request: {InputType}, ?cancellationToken: CancellationToken)
        : Task<Result<{OutputType}, SdkError>> =
        let ct = defaultArg cancellationToken CancellationToken.None
        http.RequestAsync<{InputType}, {OutputType}>(
            HttpMethod.{Method}, "{path}", request, ct)

    /// Streaming method returning async sequence.
    member _.{StreamMethodName}Async
        (request: {InputType}, ?cancellationToken: CancellationToken)
        : IAsyncEnumerable<{ItemType}> =
        let ct = defaultArg cancellationToken CancellationToken.None
        http.StreamAsync<{InputType}, {ItemType}>(
            HttpMethod.{Method}, "{path}", request, ct)
```

#### 5. HTTP Client (`Http.fs`)

```fsharp
namespace {Namespace}

open System
open System.Net.Http
open System.Net.Http.Headers
open System.Net.Http.Json
open System.Text.Json
open System.Threading
open System.Threading.Tasks
open System.Runtime.CompilerServices

type internal HttpClientWrapper(httpClient: HttpClient, options: ClientOptions) =
    let jsonOptions =
        JsonSerializerOptions(
            PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull)

    let random = Random()

    let buildUri path = Uri(options.BaseUrl.TrimEnd('/') + path)

    let getBackoff attempt =
        let baseDelay = 500.0
        let exponential = baseDelay * float (1 <<< attempt)
        let jitter = float (random.Next(0, 100))
        TimeSpan.FromMilliseconds(exponential + jitter)

    let applyHeaders (request: HttpRequestMessage) =
        // Apply default headers
        for kvp in options.DefaultHeaders do
            request.Headers.TryAddWithoutValidation(kvp.Key, kvp.Value) |> ignore

        // Apply authentication
        match options.ApiKey with
        | Some key ->
            match options.AuthMode with
            | Bearer ->
                request.Headers.Authorization <- AuthenticationHeaderValue("Bearer", key)
            | Basic ->
                request.Headers.Authorization <- AuthenticationHeaderValue("Basic", key)
            | None -> ()
        | None -> ()

    member _.RequestAsync<'TReq, 'TResp>
        (method: HttpMethod, path: string, body: 'TReq, ct: CancellationToken)
        : Task<Result<'TResp, SdkError>> = task {
        let mutable lastError = Unchecked.defaultof<SdkError>

        for attempt = 0 to options.MaxRetries do
            try
                use request = new HttpRequestMessage(method, buildUri path)
                applyHeaders request
                request.Content <- JsonContent.Create(body, options = jsonOptions)

                let! response = httpClient.SendAsync(request, ct)

                if not response.IsSuccessStatusCode then
                    let! responseBody = response.Content.ReadAsStringAsync(ct)
                    let error = ApiError(
                        int response.StatusCode,
                        sprintf "API error: %O" response.StatusCode,
                        Some responseBody)

                    if not (SdkError.isRetriable error) || attempt >= options.MaxRetries then
                        return Error error
                    else
                        lastError <- error
                        do! Task.Delay(getBackoff attempt, ct)
                else
                    let! result = response.Content.ReadFromJsonAsync<'TResp>(jsonOptions, ct)
                    return Ok result

            with
            | :? OperationCanceledException when ct.IsCancellationRequested ->
                return Error CancelledError
            | :? TaskCanceledException as ex when not ct.IsCancellationRequested ->
                lastError <- TimeoutError options.Timeout
                if attempt < options.MaxRetries then
                    do! Task.Delay(getBackoff attempt, ct)
            | :? HttpRequestException as ex ->
                lastError <- ConnectionError ex
                if attempt < options.MaxRetries then
                    do! Task.Delay(getBackoff attempt, ct)
            | ex ->
                return Error (DeserializationError ex)

        return Error lastError
    }

    member _.StreamAsync<'TReq, 'TItem>
        (method: HttpMethod, path: string, body: 'TReq, ct: CancellationToken)
        : IAsyncEnumerable<'TItem> =
        SseParser.parseAsync<'TReq, 'TItem>(
            httpClient, options, method, path, body, jsonOptions, ct)
```

#### 6. SSE Streaming (`Streaming.fs`)

```fsharp
namespace {Namespace}

open System
open System.Collections.Generic
open System.IO
open System.Net.Http
open System.Net.Http.Headers
open System.Net.Http.Json
open System.Runtime.CompilerServices
open System.Text
open System.Text.Json
open System.Threading
open System.Threading.Tasks

module SseParser =
    /// Parse SSE events from a stream.
    let parseAsync<'TReq, 'T>
        (httpClient: HttpClient)
        (options: ClientOptions)
        (method: HttpMethod)
        (path: string)
        (body: 'TReq)
        (jsonOptions: JsonSerializerOptions)
        (ct: CancellationToken)
        : IAsyncEnumerable<'T> =
        { new IAsyncEnumerable<'T> with
            member _.GetAsyncEnumerator(ct) =
                let mutable current = Unchecked.defaultof<'T>
                let mutable stream: Stream = null
                let mutable reader: StreamReader = null
                let mutable response: HttpResponseMessage = null
                let dataBuffer = StringBuilder()
                let mutable started = false

                { new IAsyncEnumerator<'T> with
                    member _.Current = current

                    member _.MoveNextAsync() = ValueTask<bool>(task {
                        if not started then
                            started <- true
                            let request = new HttpRequestMessage(method,
                                Uri(options.BaseUrl.TrimEnd('/') + path))

                            for kvp in options.DefaultHeaders do
                                request.Headers.TryAddWithoutValidation(kvp.Key, kvp.Value)
                                |> ignore

                            match options.ApiKey with
                            | Some key ->
                                match options.AuthMode with
                                | Bearer ->
                                    request.Headers.Authorization <-
                                        AuthenticationHeaderValue("Bearer", key)
                                | Basic ->
                                    request.Headers.Authorization <-
                                        AuthenticationHeaderValue("Basic", key)
                                | None -> ()
                            | None -> ()

                            request.Headers.Accept.Add(
                                MediaTypeWithQualityHeaderValue("text/event-stream"))
                            request.Content <- JsonContent.Create(body, options = jsonOptions)

                            response <- httpClient.SendAsync(
                                request,
                                HttpCompletionOption.ResponseHeadersRead,
                                ct).Result

                            if not response.IsSuccessStatusCode then
                                let responseBody = response.Content.ReadAsStringAsync(ct).Result
                                raise (Exception(sprintf "API error %d: %s"
                                    (int response.StatusCode) responseBody))

                            stream <- response.Content.ReadAsStreamAsync(ct).Result
                            reader <- new StreamReader(stream, Encoding.UTF8)

                        while not reader.EndOfStream do
                            ct.ThrowIfCancellationRequested()
                            let! line = reader.ReadLineAsync(ct)

                            if isNull line then
                                return false
                            elif line.Length = 0 then
                                let data = dataBuffer.ToString().Trim()
                                dataBuffer.Clear() |> ignore

                                if data.Length > 0 && data <> "[DONE]" then
                                    current <- JsonSerializer.Deserialize<'T>(data, jsonOptions)
                                    return true
                            elif line.StartsWith("data:", StringComparison.Ordinal) then
                                let content = line.AsSpan(5).TrimStart()
                                if dataBuffer.Length > 0 then
                                    dataBuffer.AppendLine() |> ignore
                                dataBuffer.Append(content) |> ignore

                        return false
                    })

                    member _.DisposeAsync() = ValueTask(task {
                        if not (isNull reader) then reader.Dispose()
                        if not (isNull stream) then stream.Dispose()
                        if not (isNull response) then response.Dispose()
                    })
                }
        }
```

## Type Mapping

### Primitive Types

| Contract Type     | F# Type          |
|-------------------|------------------|
| `string`          | `string`         |
| `bool`, `boolean` | `bool`           |
| `int`             | `int`            |
| `int8`            | `sbyte`          |
| `int16`           | `int16`          |
| `int32`           | `int`            |
| `int64`           | `int64`          |
| `uint`            | `uint32`         |
| `uint8`           | `byte`           |
| `uint16`          | `uint16`         |
| `uint32`          | `uint32`         |
| `uint64`          | `uint64`         |
| `float32`         | `float32`        |
| `float64`         | `float`          |
| `time.Time`       | `DateTimeOffset` |
| `json.RawMessage` | `JsonElement`    |
| `any`             | `JsonElement`    |

### Collection Types

| Contract Type      | F# Type                        |
|--------------------|--------------------------------|
| `[]T`              | `{FSharpType} list`            |
| `map[string]T`     | `Map<string, {FSharpType}>`    |

### Optional/Nullable

| Contract      | F# Type             |
|---------------|---------------------|
| `optional: T` | `{FSharpType} option` |
| `nullable: T` | `{FSharpType} option` |

### Records

Contract structs become F# records:

```fsharp
/// Message model.
[<CLIMutable>]
type Message = {
    /// The message ID.
    [<JsonPropertyName("id")>]
    Id: string

    /// The message role.
    [<JsonPropertyName("role")>]
    Role: Role

    /// Optional content.
    [<JsonPropertyName("content")>]
    Content: ContentBlock list option
}

/// Create with record syntax:
let msg = { Id = "123"; Role = User; Content = Some [TextBlock "Hello"] }

/// Update with copy-and-update:
let updated = { msg with Content = None }
```

### Discriminated Unions

Contract union types become F# discriminated unions:

```fsharp
/// Content block variants.
type ContentBlock =
    | TextBlock of text: string
    | ImageBlock of url: string * mediaType: string option
    | ToolUseBlock of id: string * name: string * input: JsonElement

/// Pattern matching:
let describe block =
    match block with
    | TextBlock text -> sprintf "Text: %s" text
    | ImageBlock (url, _) -> sprintf "Image: %s" url
    | ToolUseBlock (_, name, _) -> sprintf "Tool: %s" name

/// Exhaustive matching is enforced by the compiler.
```

### Enum Types

Contract enum fields become F# discriminated unions with conversion helpers:

```fsharp
type Role =
    | User
    | Assistant
    | System

module Role =
    let toString = function
        | User -> "user"
        | Assistant -> "assistant"
        | System -> "system"

    let fromString = function
        | "user" -> Some User
        | "assistant" -> Some Assistant
        | "system" -> Some System
        | _ -> None

    let parse s =
        match fromString s with
        | Some r -> r
        | None -> failwithf "Invalid role: %s" s
```

## Configuration

### Default Values

From contract `Defaults`:

```fsharp
module ClientOptions =
    let defaults = {
        ApiKey = None
        BaseUrl = "{defaults.baseURL}"
        Timeout = TimeSpan.FromSeconds(60.0)
        MaxRetries = 2
        DefaultHeaders =
            Map.ofList [
                // From defaults.headers
            ]
        AuthMode = Bearer
        HttpClient = None
    }
```

### Fluent Configuration

```fsharp
let options =
    ClientOptions.defaults
    |> ClientOptions.withApiKey "sk-xxx"
    |> ClientOptions.withTimeout (TimeSpan.FromMinutes 2.0)
    |> ClientOptions.withMaxRetries 3
    |> ClientOptions.withHeader "X-Custom" "value"

let client = new AnthropicClient(options)
```

## Naming Conventions

### F# Naming

| Contract       | F#                       |
|----------------|--------------------------|
| `user_id`      | `UserId`                 |
| `user-name`    | `UserName`               |
| `user`         | `User`                   |
| `create`       | `CreateAsync`            |
| `get-user`     | `GetUserAsync`           |

Functions:
- `toFSharpName(s)`: Converts to PascalCase for types/properties
- `toFSharpMethodName(s)`: Converts to PascalCase + "Async" suffix for methods
- `sanitizeIdent(s)`: Removes invalid characters

Reserved words are prefixed with backticks:
```fsharp
type Record = {
    ``type``: string  // F# keyword
    ``module``: string option
}
```

F# reserved words: `abstract`, `and`, `as`, `assert`, `base`, `begin`, `class`, `default`, `delegate`, `do`, `done`, `downcast`, `downto`, `elif`, `else`, `end`, `exception`, `extern`, `false`, `finally`, `fixed`, `for`, `fun`, `function`, `global`, `if`, `in`, `inherit`, `inline`, `interface`, `internal`, `lazy`, `let`, `match`, `member`, `module`, `mutable`, `namespace`, `new`, `not`, `null`, `of`, `open`, `or`, `override`, `private`, `public`, `rec`, `return`, `select`, `static`, `struct`, `then`, `to`, `true`, `try`, `type`, `upcast`, `use`, `val`, `void`, `when`, `while`, `with`, `yield`

## Code Generation

### Generator Structure

```go
package sdkfsharp

type Config struct {
    // Namespace is the F# namespace for generated code.
    Namespace string

    // PackageName is the NuGet package name.
    PackageName string

    // Version is the package version.
    Version string

    // TargetFramework is the target .NET version.
    // Default: "net8.0".
    TargetFramework string
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── project.fsproj.tmpl       # Project file
├── Types.fs.tmpl             # Model types (records, DUs)
├── Errors.fs.tmpl            # Error discriminated union
├── Http.fs.tmpl              # HTTP client wrapper
├── Streaming.fs.tmpl         # SSE streaming support
├── Resources.fs.tmpl         # Resource types
└── Client.fs.tmpl            # Main client module
```

### Generated Files

| File                        | Purpose                           |
|-----------------------------|-----------------------------------|
| `{Package}.fsproj`          | F# project file                   |
| `src/Types.fs`              | Record and DU type definitions    |
| `src/Errors.fs`             | SdkError discriminated union      |
| `src/Http.fs`               | HTTP client implementation        |
| `src/Streaming.fs`          | IAsyncEnumerable SSE parsing      |
| `src/Resources.fs`          | Resource types with methods       |
| `src/Client.fs`             | Main client type                  |

**Note**: F# requires files to be listed in compilation order in .fsproj.

## Usage Examples

### Basic Usage

```fsharp
open Anthropic.Sdk

// Create client with API key
let client = new AnthropicClient("your-api-key")

// Make a request
async {
    let request = {
        Model = "claude-3-opus"
        Messages = [
            { Role = User; Content = Some [TextBlock "Hello"] }
        ]
        MaxTokens = 1024
        Stream = None
    }

    match! client.Messages.CreateAsync(request) |> Async.AwaitTask with
    | Ok response -> printfn "Response: %A" response.Content
    | Error err -> printfn "Error: %s" (SdkError.message err)
}
```

### Streaming

```fsharp
open System.Threading.Tasks

task {
    let request = {
        Model = "claude-3-opus"
        Messages = [{ Role = User; Content = Some [TextBlock "Hello"] }]
        MaxTokens = 1024
        Stream = Some true
    }

    let stream = client.Messages.StreamAsync(request)

    for await chunk in stream do
        match chunk.Delta with
        | Some delta -> printf "%s" delta.Text
        | None -> ()
}
```

### Error Handling with Result

```fsharp
let handleRequest request = async {
    match! client.Messages.CreateAsync(request) |> Async.AwaitTask with
    | Ok response ->
        return Ok response
    | Error (ApiError (429, _, _)) ->
        printfn "Rate limited, retrying..."
        return Error "Rate limited"
    | Error (ApiError (status, msg, _)) when status >= 400 && status < 500 ->
        printfn "Client error %d: %s" status msg
        return Error msg
    | Error (ApiError (status, msg, _)) when status >= 500 ->
        printfn "Server error %d: %s" status msg
        return Error msg
    | Error TimeoutError _ ->
        printfn "Request timed out"
        return Error "Timeout"
    | Error CancelledError ->
        printfn "Request cancelled"
        return Error "Cancelled"
    | Error err ->
        printfn "SDK error: %s" (SdkError.message err)
        return Error (SdkError.message err)
}
```

### Pattern Matching on Unions

```fsharp
// Exhaustive pattern matching
let processBlock block =
    match block with
    | TextBlock text ->
        printfn "Text: %s" text
    | ImageBlock (url, mediaType) ->
        printfn "Image: %s (type: %A)" url mediaType
    | ToolUseBlock (id, name, input) ->
        printfn "Tool %s: %s with %A" id name input

// Active patterns for complex matching
let (|LongText|ShortText|) block =
    match block with
    | TextBlock text when text.Length > 100 -> LongText text
    | TextBlock text -> ShortText text
    | _ -> ShortText ""

let summarize block =
    match block with
    | LongText text -> text.Substring(0, 100) + "..."
    | ShortText text -> text
```

### Pipeline-Friendly Operations

```fsharp
// Chain Result operations with bind
let processMessages messages =
    messages
    |> List.map (fun msg ->
        { Model = "claude-3-opus"
          Messages = [msg]
          MaxTokens = 1024
          Stream = None })
    |> List.map client.Messages.CreateAsync
    |> Task.WhenAll
    |> Async.AwaitTask

// Result combinators
module Result =
    let mapAsync f result = async {
        match result with
        | Ok x ->
            let! y = f x
            return Ok y
        | Error e -> return Error e
    }

// Usage
let! result =
    client.Messages.CreateAsync(request)
    |> Async.AwaitTask
    |> Async.map (Result.map (fun r -> r.Content))
```

### Custom Configuration

```fsharp
let options =
    ClientOptions.defaults
    |> ClientOptions.withApiKey (Environment.GetEnvironmentVariable "ANTHROPIC_API_KEY")
    |> ClientOptions.withBaseUrl "https://custom.api.com"
    |> ClientOptions.withTimeout (TimeSpan.FromMinutes 2.0)
    |> ClientOptions.withMaxRetries 3
    |> ClientOptions.withHeader "X-Custom-Header" "value"

let client = new AnthropicClient(options)
```

### Cancellation

```fsharp
open System.Threading

task {
    use cts = new CancellationTokenSource(TimeSpan.FromSeconds 30.0)

    let! result = client.Messages.CreateAsync(request, cts.Token)

    match result with
    | Ok response -> printfn "Success: %A" response
    | Error CancelledError -> printfn "Request was cancelled"
    | Error err -> printfn "Error: %s" (SdkError.message err)
}
```

### Dependency Injection (ASP.NET Core with Giraffe)

```fsharp
open Giraffe
open Microsoft.Extensions.DependencyInjection

// In Startup/Program
let configureServices (services: IServiceCollection) =
    let client = new AnthropicClient(
        ClientOptions.defaults
        |> ClientOptions.withApiKey (config.["Anthropic:ApiKey"]))
    services.AddSingleton<AnthropicClient>(client) |> ignore

// In handler
let chatHandler : HttpHandler =
    fun next ctx -> task {
        let client = ctx.GetService<AnthropicClient>()
        let! result = client.Messages.CreateAsync(request)

        match result with
        | Ok response -> return! json response next ctx
        | Error err -> return! RequestErrors.BAD_REQUEST (SdkError.message err) next ctx
    }
```

### Computation Expression Style

```fsharp
// Define result computation expression
type ResultBuilder() =
    member _.Bind(m, f) = Result.bind f m
    member _.Return(x) = Ok x
    member _.ReturnFrom(m) = m

let result = ResultBuilder()

// Usage
let processRequest request = result {
    let! response = client.Messages.CreateAsync(request) |> Async.AwaitTask |> Async.RunSynchronously
    let! firstBlock =
        response.Content
        |> List.tryHead
        |> Option.toResult (ApiError(400, "No content", None))
    return firstBlock
}
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidFSharp_Compiles(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_DiscriminatedUnions(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_RecordTypes(t *testing.T)
func TestGenerate_OptionTypes(t *testing.T)
```

### Generated SDK Tests

```fsharp
open Xunit
open FsUnit.Xunit

[<Fact>]
let ``CreateAsync returns response`` () = task {
    // Arrange
    let handler = MockHttpMessageHandler()
    handler.When("*").Respond("application/json", "{}")

    let httpClient = new HttpClient(handler)
    let client = new AnthropicClient({
        ClientOptions.defaults with
            HttpClient = Some httpClient
            ApiKey = Some "test-key"
    })

    // Act
    let! result = client.Messages.CreateAsync(request)

    // Assert
    result |> should be (ofCase <@ Ok @>)
}

[<Fact>]
let ``Pattern matching on unions is exhaustive`` () =
    let block = TextBlock "hello"

    let result =
        match block with
        | TextBlock t -> t
        | ImageBlock (url, _) -> url
        | ToolUseBlock (_, name, _) -> name

    result |> should equal "hello"
```

## Platform Support

### Dependencies

**Core Dependencies:**
- `FSharp.Core` >= 6.0 (for task CE)
- `System.Net.Http` - HTTP client
- `System.Text.Json` - JSON serialization

**No external NuGet packages required** (other than FSharp.Core).

### Target Frameworks

| Framework | Minimum Version | Rationale                          |
|-----------|-----------------|-----------------------------------|
| .NET      | 8.0             | LTS, task CE, latest F# features  |

### Project File

```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <GenerateDocumentationFile>true</GenerateDocumentationFile>

    <PackageId>{PackageName}</PackageId>
    <Version>{Version}</Version>
    <Description>{Description}</Description>
    <Authors>Generated</Authors>
  </PropertyGroup>

  <ItemGroup>
    <!-- Files must be in compilation order -->
    <Compile Include="src/Types.fs" />
    <Compile Include="src/Errors.fs" />
    <Compile Include="src/Http.fs" />
    <Compile Include="src/Streaming.fs" />
    <Compile Include="src/Resources.fs" />
    <Compile Include="src/Client.fs" />
  </ItemGroup>
</Project>
```

## Key F# Advantages

### 1. Native Discriminated Unions

Unlike C# which requires polymorphic serialization attributes, F# DUs are native:

```fsharp
// F# - native, concise
type ContentBlock =
    | TextBlock of text: string
    | ImageBlock of url: string

// Pattern matching is exhaustive by default
let handle block =
    match block with
    | TextBlock t -> t
    | ImageBlock u -> u
    // Compiler warns if cases are missing
```

### 2. Option vs Null

```fsharp
// F# - null-safe by design
type Message = {
    Content: string option  // Explicitly optional
}

// Must handle None case
match msg.Content with
| Some c -> printfn "%s" c
| None -> printfn "No content"
```

### 3. Result for Error Handling

```fsharp
// Compose operations that can fail
let createAndProcess request =
    client.Messages.CreateAsync(request)
    |> Async.AwaitTask
    |> Async.map (Result.bind processResponse)
    |> Async.map (Result.map formatOutput)
```

### 4. Immutability by Default

```fsharp
// Records are immutable
let msg = { Id = "1"; Content = Some "Hello" }

// Create new record with updates
let updated = { msg with Content = Some "Updated" }
```

## Future Enhancements

1. **Type providers**: F# type provider for compile-time contract checking
2. **Computation expressions**: Custom CE for cleaner async/result composition
3. **Fable support**: Compile to JavaScript for browser use
4. **Bolero/WebAssembly**: WebAssembly target support
5. **Elmish integration**: MVU pattern helpers
6. **FsHttp integration**: Alternative HTTP client using FsHttp library

## References

- [F# Language Reference](https://docs.microsoft.com/en-us/dotnet/fsharp/language-reference/)
- [F# Style Guide](https://docs.microsoft.com/en-us/dotnet/fsharp/style-guide/)
- [F# Core Library](https://fsharp.github.io/fsharp-core-docs/)
- [Task Computation Expression](https://docs.microsoft.com/en-us/dotnet/fsharp/language-reference/task-expressions)
- [Discriminated Unions](https://docs.microsoft.com/en-us/dotnet/fsharp/language-reference/discriminated-unions)
- [Option Type](https://docs.microsoft.com/en-us/dotnet/fsharp/language-reference/options)
- [Result Type](https://docs.microsoft.com/en-us/dotnet/fsharp/language-reference/results)
- [Pattern Matching](https://docs.microsoft.com/en-us/dotnet/fsharp/language-reference/pattern-matching)
