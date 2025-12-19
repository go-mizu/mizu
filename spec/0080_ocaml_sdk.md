# RFC 0080: OCaml SDK Generator

## Summary

Add OCaml SDK code generation to the Mizu contract system, enabling production-ready, idiomatic OCaml clients with excellent developer experience leveraging OCaml's powerful module system, algebraic data types, and strong static typing.

## Motivation

OCaml is a statically typed functional programming language known for its expressive type system, powerful module system, and excellent performance. A native OCaml SDK provides:

1. **Strong Static Typing**: Catch errors at compile time with OCaml's type inference
2. **Algebraic Data Types**: Natural representation for discriminated unions and variants
3. **Result-Based Error Handling**: Explicit, composable error handling via `result` type
4. **Powerful Module System**: First-class modules for clean encapsulation and abstraction
5. **Production Proven**: Used in finance, compilers, formal verification, and systems programming
6. **Async Excellence**: Lwt-based async I/O with backpressure and resource safety

## Design Goals

### Developer Experience (DX)

- **Idiomatic OCaml**: Modules, pattern matching, variants, labeled arguments
- **Type-Safe**: Full type coverage with no partial functions
- **Result-Based Errors**: `(response, error) result` for explicit error handling
- **PPX Derivation**: Automatic JSON serialization via ppx_yojson_conv
- **Odoc Documentation**: Rich inline documentation with examples
- **utop Friendly**: Easy to explore in the REPL
- **Minimal Dependencies**: Only well-maintained, battle-tested libraries

### Production Readiness

- **cohttp-lwt-unix**: Battle-tested HTTP client with async I/O
- **lwt**: Cooperative threading with resource safety
- **yojson**: Industry-standard JSON handling
- **ppx_yojson_conv**: Type-safe JSON serialization
- **Retry Logic**: Built-in exponential backoff for transient errors
- **Connection Management**: Via cohttp's connection handling
- **Configurable**: Flexible configuration via record types

## Architecture

### Project Structure

```
{package-name}/
├── dune-project                # Dune project configuration
├── {package-name}.opam         # OPAM package definition
├── lib/
│   ├── dune                    # Library build config
│   ├── {package}.ml            # Main module (re-exports)
│   ├── {package}.mli           # Main module interface
│   ├── config.ml               # Configuration types
│   ├── config.mli              # Configuration interface
│   ├── client.ml               # HTTP client implementation
│   ├── client.mli              # Client interface
│   ├── types.ml                # Generated type definitions
│   ├── types.mli               # Types interface
│   ├── resources.ml            # Resource modules
│   ├── resources.mli           # Resources interface
│   ├── streaming.ml            # SSE streaming support
│   ├── streaming.mli           # Streaming interface
│   ├── errors.ml               # Error types
│   └── errors.mli              # Errors interface
└── test/
    ├── dune                    # Test build config
    └── test_{package}.ml       # Test suite
```

### Core Components

#### 1. Main Module (`lib/{package}.ml`)

The top-level module re-exporting all public APIs:

```ocaml
(** {Package} SDK - {Service description}

    {b Quick Start}

    {[
      open {Package}

      let () =
        let open Lwt.Syntax in
        Lwt_main.run @@
        let* client = Client.create ~api_key:"your-api-key" () in
        let* result = Resources.Messages.create client
          ~model:"model-name"
          ~messages:[Types.Message.make ~role:`User ~content:"Hello" ()]
        in
        match result with
        | Ok response -> Lwt_io.printl (Types.Response.show response)
        | Error err -> Lwt_io.printl (Errors.show err)
    ]}

    {b Configuration}

    Configure via environment or explicit options:

    {[
      (* From environment *)
      let config = Config.from_env ()

      (* Explicit configuration *)
      let config = Config.make
        ~api_key:"your-api-key"
        ~base_url:"https://api.example.com"
        ()
    ]}
*)

(** Re-export all modules *)
module Config = Config
module Client = Client
module Types = Types
module Resources = Resources
module Errors = Errors
module Streaming = Streaming
```

#### 2. Config (`lib/config.ml`)

Configuration types with sensible defaults:

```ocaml
(** Configuration for the {Service} client. *)

(** Authentication mode for API requests. *)
type auth_mode =
  | Bearer     (** Bearer token in Authorization header *)
  | Basic      (** Basic authentication *)
  | Api_key    (** API key in X-Api-Key header *)
  | None       (** No authentication *)
[@@deriving show, eq]

(** Client configuration. *)
type t = {
  api_key : string option;
  (** API key for authentication *)

  base_url : string;
  (** Base URL for API requests *)

  timeout : float;
  (** Request timeout in seconds (default: 60.0) *)

  max_retries : int;
  (** Maximum retry attempts for transient errors (default: 2) *)

  auth_mode : auth_mode;
  (** Authentication mode (default: Bearer) *)

  headers : (string * string) list;
  (** Additional headers to include in all requests *)
}
[@@deriving show, eq]

(** Default configuration.

    {[
      let default = {
        api_key = None;
        base_url = "{default_base_url}";
        timeout = 60.0;
        max_retries = 2;
        auth_mode = Bearer;
        headers = [];
      }
    ]}
*)
val default : t

(** Create a configuration with optional overrides.

    {[
      let config = make ~api_key:"sk-..." ~timeout:30.0 ()
    ]}
*)
val make :
  ?api_key:string ->
  ?base_url:string ->
  ?timeout:float ->
  ?max_retries:int ->
  ?auth_mode:auth_mode ->
  ?headers:(string * string) list ->
  unit ->
  t

(** Load configuration from environment variables.

    Reads:
    - [{ENV_PREFIX}_API_KEY] - API key
    - [{ENV_PREFIX}_BASE_URL] - Base URL (optional)
    - [{ENV_PREFIX}_TIMEOUT] - Timeout in seconds (optional)
*)
val from_env : unit -> t
```

#### 3. Client (`lib/client.ml`)

The HTTP client implementation:

```ocaml
(** HTTP client for {Service} API. *)

open Lwt.Syntax

(** API client handle. *)
type t = {
  config : Config.t;
  ctx : Cohttp_lwt_unix.Client.ctx;
}

(** Create a new client with default configuration.

    {[
      let* client = create () in
      (* use client *)
    ]}
*)
val create : unit -> t Lwt.t

(** Create a new client with custom configuration.

    {[
      let config = Config.make ~api_key:"sk-..." () in
      let* client = create_with config in
      (* use client *)
    ]}
*)
val create_with : Config.t -> t Lwt.t

(** Perform an HTTP request.

    @param method_ HTTP method
    @param path Request path
    @param body Optional request body
    @param headers Extra headers
    @return Response body or error
*)
val request :
  t ->
  meth:Cohttp.Code.meth ->
  path:string ->
  ?body:string ->
  ?headers:(string * string) list ->
  unit ->
  (string, Errors.t) result Lwt.t

(** Perform an HTTP request and decode JSON response.

    @param method_ HTTP method
    @param path Request path
    @param body Optional request body (will be JSON encoded)
    @return Decoded response or error
*)
val request_json :
  t ->
  meth:Cohttp.Code.meth ->
  path:string ->
  ?body:'a ->
  encode:('a -> Yojson.Safe.t) ->
  decode:(Yojson.Safe.t -> 'b) ->
  unit ->
  ('b, Errors.t) result Lwt.t

(** Perform a streaming SSE request.

    @param method_ HTTP method
    @param path Request path
    @param body Request body
    @return Stream of parsed events or error
*)
val stream :
  t ->
  meth:Cohttp.Code.meth ->
  path:string ->
  body:'a ->
  encode:('a -> Yojson.Safe.t) ->
  decode:(Yojson.Safe.t -> 'b) ->
  unit ->
  ('b Streaming.event Lwt_stream.t, Errors.t) result Lwt.t
```

#### 4. Types (`lib/types.ml`)

Generated type definitions with JSON derivation:

```ocaml
(** Type definitions for {Service} API.

    All types use ppx_yojson_conv for JSON serialization.
*)

(** {Type description} *)
type {type_name} = {
  {field_name} : {field_type};
  (** {Field description} *)

  {optional_field_name} : {field_type} option; [@yojson.option]
  (** {Field description} (optional) *)
}
[@@deriving show, eq, yojson]

(** Enum type for {description} *)
type {enum_name} =
  | {EnumValue1} [@name "{value1}"]  (** {Value1 description} *)
  | {EnumValue2} [@name "{value2}"]  (** {Value2 description} *)
[@@deriving show, eq, yojson]

(** Union type for {description}

    Pattern match to access variants:

    {[
      match block with
      | `Text tb -> handle_text tb
      | `Image ib -> handle_image ib
      | `Tool_use tub -> handle_tool_use tub
    ]}
*)
type {union_name} = [
  | `{Variant1} of {variant1_type}
  | `{Variant2} of {variant2_type}
]
[@@deriving show, eq]

val {union_name}_of_yojson : Yojson.Safe.t -> {union_name}
val yojson_of_{union_name} : {union_name} -> Yojson.Safe.t

(** Smart constructor with defaults *)
module {TypeName} : sig
  val make :
    {required_field}:{field_type} ->
    ?{optional_field}:{field_type} ->
    unit ->
    {type_name}
end
```

#### 5. Resources (`lib/resources.ml`)

Resource operations:

```ocaml
(** Operations for {resource}.

    {Resource description}
*)

open Lwt.Syntax

module {Resource} : sig
  (** Create a new {resource}.

      {[
        let* result = {Resource}.create client
          ~model:"model-name"
          ~messages:[message]
          ~max_tokens:1024
          ()
        in
        match result with
        | Ok response -> (* handle success *)
        | Error err -> (* handle error *)
      ]}
  *)
  val create :
    Client.t ->
    ~model:string ->
    ~messages:Types.Message.t list ->
    ?max_tokens:int ->
    ?temperature:float ->
    unit ->
    (Types.{OutputType}.t, Errors.t) result Lwt.t

  (** Create a new {resource}, raising on error.

      {[
        let* response = {Resource}.create_exn client ~model ~messages () in
        (* use response *)
      ]}
  *)
  val create_exn :
    Client.t ->
    ~model:string ->
    ~messages:Types.Message.t list ->
    ?max_tokens:int ->
    ?temperature:float ->
    unit ->
    Types.{OutputType}.t Lwt.t

  (** Stream a {resource} response.

      {[
        let* result = {Resource}.create_stream client ~model ~messages () in
        match result with
        | Ok stream ->
          Lwt_stream.iter_s (fun event ->
            match event with
            | Streaming.Data e -> handle_event e
            | Streaming.Done -> Lwt.return_unit
            | Streaming.Heartbeat -> Lwt.return_unit
          ) stream
        | Error err -> (* handle error *)
      ]}
  *)
  val create_stream :
    Client.t ->
    ~model:string ->
    ~messages:Types.Message.t list ->
    ?max_tokens:int ->
    ?temperature:float ->
    unit ->
    (Types.{StreamEventType}.t Streaming.event Lwt_stream.t, Errors.t) result Lwt.t
end
```

#### 6. Streaming (`lib/streaming.ml`)

SSE streaming implementation:

```ocaml
(** Server-Sent Events (SSE) streaming support. *)

(** A parsed SSE event. *)
type 'a event =
  | Data of 'a        (** A data event with parsed payload *)
  | Done              (** Stream completion marker *)
  | Heartbeat         (** Keep-alive ping *)
[@@deriving show, eq]

(** Internal stream parsing state. *)
type state = {
  buffer : Buffer.t;
  mutable event_id : string option;
  mutable retry : int option;
}

(** Create initial parsing state. *)
val initial_state : unit -> state

(** Parse a chunk of SSE data.

    @param state Current parsing state
    @param chunk New data chunk
    @param decode JSON decoder for events
    @return List of parsed events and updated state
*)
val parse_chunk :
  state:state ->
  chunk:string ->
  decode:(Yojson.Safe.t -> 'a) ->
  'a event list

(** Create an SSE stream from a byte stream.

    {[
      let event_stream = sse_stream ~decode:Event.of_yojson byte_stream in
      Lwt_stream.iter_s handle_event event_stream
    ]}
*)
val sse_stream :
  decode:(Yojson.Safe.t -> 'a) ->
  string Lwt_stream.t ->
  'a event Lwt_stream.t

(** Collect all data events from a stream.

    {[
      let* events = collect_data event_stream in
      (* events : 'a list *)
    ]}
*)
val collect_data : 'a event Lwt_stream.t -> 'a list Lwt.t

(** Collect all text deltas from a stream.

    {[
      let* texts = collect_text ~extract_text event_stream in
      let full_text = String.concat "" texts
    ]}
*)
val collect_text :
  extract_text:('a -> string option) ->
  'a event Lwt_stream.t ->
  string list Lwt.t
```

#### 7. Errors (`lib/errors.ml`)

Error types and handling:

```ocaml
(** Error types for {Service} API. *)

(** Error type classification. *)
type error_type =
  | Bad_request           (** HTTP 400 *)
  | Unauthorized          (** HTTP 401 *)
  | Forbidden             (** HTTP 403 *)
  | Not_found             (** HTTP 404 *)
  | Unprocessable_entity  (** HTTP 422 *)
  | Rate_limited          (** HTTP 429 *)
  | Server_error          (** HTTP 5xx *)
  | Network_error         (** Connection/network errors *)
  | Parse_error           (** JSON parsing errors *)
  | Timeout_error         (** Request timeout *)
  | Stream_error          (** SSE parsing errors *)
[@@deriving show, eq]

(** API error details from server response. *)
type api_error = {
  error_type : string;
  message : string;
  param : string option;
  code : string option;
}
[@@deriving show, eq, yojson]

(** SDK error type.

    Pattern match to handle different error cases:

    {[
      match err with
      | Api_error { status; error; _ } ->
          Printf.printf "API error: %s\n" error.message
      | Connection_error msg ->
          Printf.printf "Network error: %s\n" msg
      | Rate_limit_error { retry_after; _ } ->
          Unix.sleepf (Float.of_int retry_after);
          (* retry *)
      | _ -> Printf.printf "Error: %s\n" (show err)
    ]}
*)
type t =
  | Api_error of { status : int; error : api_error }
      (** API returned an error response *)
  | Connection_error of string
      (** Network/connection error *)
  | Timeout_error of string
      (** Request timed out *)
  | Parse_error of string
      (** Failed to parse response *)
  | Rate_limit_error of { retry_after : int; error : api_error option }
      (** Rate limited (retry after N seconds) *)
  | Stream_error of string
      (** Error in SSE stream *)
[@@deriving show, eq]

(** Create an error from an HTTP response.

    @param status HTTP status code
    @param body Response body
    @return Appropriate error type
*)
val of_response : status:int -> body:string -> t

(** Create an error from an exception.

    @param exn The exception
    @return Connection_error with exception message
*)
val of_exn : exn -> t

(** Check if an error is retryable.

    {[
      if is_retryable err then
        (* retry with backoff *)
      else
        (* fail *)
    ]}
*)
val is_retryable : t -> bool

(** Get human-readable error message. *)
val message : t -> string

(** Get HTTP status code if applicable. *)
val status : t -> int option

(** Get error classification. *)
val classify : t -> error_type
```

## Type Mapping

### Primitive Types

| Contract Type     | OCaml Type           | Notes                          |
|-------------------|----------------------|--------------------------------|
| `string`          | `string`             |                                |
| `bool`, `boolean` | `bool`               |                                |
| `int`             | `int`                |                                |
| `int8`            | `int`                | OCaml uses native int          |
| `int16`           | `int`                |                                |
| `int32`           | `int32`              | From Int32 module              |
| `int64`           | `int64`              | From Int64 module              |
| `uint`            | `int`                | OCaml int is signed            |
| `uint8`           | `int`                |                                |
| `uint16`          | `int`                |                                |
| `uint32`          | `int32`              |                                |
| `uint64`          | `int64`              |                                |
| `float32`         | `float`              | OCaml uses double precision    |
| `float64`         | `float`              |                                |
| `time.Time`       | `Ptime.t`            | From ptime library             |
| `json.RawMessage` | `Yojson.Safe.t`      | From yojson                    |
| `any`             | `Yojson.Safe.t`      |                                |

### Collection Types

| Contract Type      | OCaml Type           | Notes                          |
|--------------------|----------------------|--------------------------------|
| `[]T`              | `T list`             | List type                      |
| `map[string]T`     | `(string * T) list`  | Association list               |

### Optional/Nullable

| Contract         | OCaml                | Notes                          |
|------------------|----------------------|--------------------------------|
| `optional: T`    | `T option`           | With [@yojson.option]          |
| `nullable: T`    | `T option`           | With [@yojson.option]          |

### Struct to Record

Contract structs map to OCaml record types:

```ocaml
(* From contract type:
   {Name: "Message", Fields: [{role, string}, {content, string}]} *)

type message = {
  role : string;
  content : string;
}
[@@deriving show, eq, yojson]

module Message = struct
  type t = message
  [@@deriving show, eq, yojson]

  let make ~role ~content () = { role; content }
end
```

### Enum Types

Enums map to OCaml polymorphic variants:

```ocaml
(* Role = "user" | "assistant" | "system" *)

type role = [
  | `User [@name "user"]
  | `Assistant [@name "assistant"]
  | `System [@name "system"]
]
[@@deriving show, eq, yojson]

(* Or using regular variants with custom JSON encoding *)
type role =
  | User
  | Assistant
  | System
[@@deriving show, eq]

let role_of_yojson = function
  | `String "user" -> User
  | `String "assistant" -> Assistant
  | `String "system" -> System
  | _ -> failwith "Unknown role"

let yojson_of_role = function
  | User -> `String "user"
  | Assistant -> `String "assistant"
  | System -> `String "system"
```

### Discriminated Unions

Union types use OCaml polymorphic variants:

```ocaml
(* ContentBlock = TextBlock | ImageBlock | ToolUseBlock *)

type content_block = [
  | `Text of text_block
  | `Image of image_block
  | `Tool_use of tool_use_block
]
[@@deriving show, eq]

let content_block_of_yojson json =
  let open Yojson.Safe.Util in
  match json |> member "type" |> to_string with
  | "text" -> `Text (text_block_of_yojson json)
  | "image" -> `Image (image_block_of_yojson json)
  | "tool_use" -> `Tool_use (tool_use_block_of_yojson json)
  | typ -> failwith ("Unknown content_block type: " ^ typ)

let yojson_of_content_block = function
  | `Text v -> yojson_of_text_block v
  | `Image v -> yojson_of_image_block v
  | `Tool_use v -> yojson_of_tool_use_block v

(* Pattern matching usage *)
let process_content = function
  | `Text tb -> Printf.printf "Text: %s\n" tb.text
  | `Image ib -> Printf.printf "Image: %s\n" ib.url
  | `Tool_use tub -> Printf.printf "Tool: %s\n" tub.name
```

## Naming Conventions

### OCaml Naming

| Contract       | OCaml                    | Notes                          |
|----------------|--------------------------|--------------------------------|
| `user-id`      | `user_id`                | snake_case for values          |
| `user_name`    | `user_name`              | snake_case for values          |
| `UserData`     | `User_data` or `user_data`| Module or type name           |
| `create`       | `create`                 | snake_case for functions       |
| `get-user`     | `get_user`               | snake_case for functions       |
| `TEXT`         | `Text`                   | PascalCase for variants        |

### Module Naming Strategy

OCaml modules use PascalCase:

```ocaml
(* For resource "messages" *)
module Messages = struct
  (* ... *)
end

(* For type "CreateRequest" *)
module Create_request = struct
  type t = { ... }
  [@@deriving show, eq, yojson]
end
```

### Reserved Words

OCaml reserved words are escaped by appending underscore:

- `type` -> `type_`
- `and` -> `and_`
- `as` -> `as_`
- `begin` -> `begin_`
- `class` -> `class_`
- `constraint` -> `constraint_`
- `do` -> `do_`
- `done` -> `done_`
- `downto` -> `downto_`
- `else` -> `else_`
- `end` -> `end_`
- `exception` -> `exception_`
- `external` -> `external_`
- `false` -> `false_`
- `for` -> `for_`
- `fun` -> `fun_`
- `function` -> `function_`
- `functor` -> `functor_`
- `if` -> `if_`
- `in` -> `in_`
- `include` -> `include_`
- `inherit` -> `inherit_`
- `initializer` -> `initializer_`
- `lazy` -> `lazy_`
- `let` -> `let_`
- `match` -> `match_`
- `method` -> `method_`
- `module` -> `module_`
- `mutable` -> `mutable_`
- `new` -> `new_`
- `nonrec` -> `nonrec_`
- `object` -> `object_`
- `of` -> `of_`
- `open` -> `open_`
- `or` -> `or_`
- `private` -> `private_`
- `rec` -> `rec_`
- `sig` -> `sig_`
- `struct` -> `struct_`
- `then` -> `then_`
- `to` -> `to_`
- `true` -> `true_`
- `try` -> `try_`
- `val` -> `val_`
- `virtual` -> `virtual_`
- `when` -> `when_`
- `while` -> `while_`
- `with` -> `with_`

## Code Generation

### Generator Structure

```go
package sdkocaml

type Config struct {
    // PackageName is the opam/dune package name.
    // Default: sanitized lowercase service name with underscores.
    PackageName string

    // ModuleName is the root OCaml module name.
    // Default: PascalCase service name.
    ModuleName string

    // Version is the package version for opam.
    Version string

    // Author is the package author.
    Author string

    // License is the package license (default: MIT).
    License string

    // Synopsis is a one-line package description.
    Synopsis string
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── dune-project.tmpl           # Dune project configuration
├── opam.tmpl                   # OPAM package definition
├── lib_dune.tmpl               # Library dune file
├── main.ml.tmpl                # Main module
├── main.mli.tmpl               # Main module interface
├── config.ml.tmpl              # Configuration implementation
├── config.mli.tmpl             # Configuration interface
├── client.ml.tmpl              # HTTP client implementation
├── client.mli.tmpl             # Client interface
├── types.ml.tmpl               # Type definitions
├── types.mli.tmpl              # Types interface
├── resources.ml.tmpl           # Resource implementations
├── resources.mli.tmpl          # Resources interface
├── streaming.ml.tmpl           # SSE streaming
├── streaming.mli.tmpl          # Streaming interface
├── errors.ml.tmpl              # Error types
└── errors.mli.tmpl             # Errors interface
```

### Generated Files

| File                              | Purpose                      |
|-----------------------------------|------------------------------|
| `dune-project`                    | Dune project configuration   |
| `{package}.opam`                  | OPAM package definition      |
| `lib/dune`                        | Library build configuration  |
| `lib/{package}.ml`                | Main module                  |
| `lib/{package}.mli`               | Main module interface        |
| `lib/config.ml`                   | Configuration implementation |
| `lib/config.mli`                  | Configuration interface      |
| `lib/client.ml`                   | HTTP client                  |
| `lib/client.mli`                  | Client interface             |
| `lib/types.ml`                    | Type definitions             |
| `lib/types.mli`                   | Types interface              |
| `lib/resources.ml`                | Resource implementations     |
| `lib/resources.mli`               | Resources interface          |
| `lib/streaming.ml`                | SSE streaming                |
| `lib/streaming.mli`               | Streaming interface          |
| `lib/errors.ml`                   | Error types                  |
| `lib/errors.mli`                  | Errors interface             |

## Usage Examples

### Basic Usage

```ocaml
(* Add to dune:
   (libraries my_service) *)

open My_service

let () =
  let open Lwt.Syntax in
  Lwt_main.run @@
  (* Create client *)
  let config = Config.make ~api_key:"your-api-key" () in
  let* client = Client.create_with config in

  (* Make a request *)
  let* result = Resources.Messages.create client
    ~model:"model-name"
    ~messages:[Types.Message.make ~role:`User ~content:"Hello" ()]
    ~max_tokens:1024
    ()
  in

  match result with
  | Ok response -> Lwt_io.printlf "Response: %s" (Types.Response.show response)
  | Error err -> Lwt_io.printlf "Error: %s" (Errors.show err)
```

### Streaming

```ocaml
open My_service
open Lwt.Syntax

let stream_example () =
  let config = Config.make ~api_key:"your-api-key" () in
  let* client = Client.create_with config in

  let* result = Resources.Messages.create_stream client
    ~model:"model-name"
    ~messages:[Types.Message.make ~role:`User ~content:"Tell me a story" ()]
    ~max_tokens:2048
    ()
  in

  match result with
  | Error err -> Lwt_io.printlf "Error: %s" (Errors.show err)
  | Ok stream ->
    Lwt_stream.iter_s (function
      | Streaming.Data event ->
        (match Types.StreamEvent.delta event with
         | Some delta -> Lwt_io.printf "%s" (Types.Delta.text delta)
         | None -> Lwt.return_unit)
      | Streaming.Done -> Lwt_io.printl "\n[Done]"
      | Streaming.Heartbeat -> Lwt.return_unit
    ) stream
```

### Error Handling

```ocaml
open My_service
open Lwt.Syntax

let handle_errors () =
  let* client = Client.create () in
  let* result = Resources.Messages.create client ~model:"m" ~messages:[] () in

  match result with
  | Ok response ->
    Lwt_io.printl (Types.Response.show response)

  | Error (Errors.Rate_limit_error { retry_after; _ }) ->
    let* () = Lwt_io.printlf "Rate limited, retrying in %d seconds" retry_after in
    Lwt_unix.sleep (Float.of_int retry_after)
    (* Retry... *)

  | Error (Errors.Api_error { status = 401; _ }) ->
    Lwt_io.printl "Invalid API key"

  | Error err when Errors.is_retryable err ->
    let* () = Lwt_io.printlf "Retryable error: %s" (Errors.message err) in
    (* Retry with backoff... *)
    Lwt.return_unit

  | Error err ->
    Lwt_io.printlf "Error: %s" (Errors.message err)
```

### Configuration from Environment

```ocaml
open My_service

let env_config () =
  (* Reads MY_SERVICE_API_KEY, MY_SERVICE_BASE_URL, etc. *)
  let config = Config.from_env () in
  let open Lwt.Syntax in
  let* client = Client.create_with config in
  (* Use client... *)
  Lwt.return_unit
```

### With Custom Context

```ocaml
open My_service
open Lwt.Syntax

let custom_context () =
  let resolver = Resolver_lwt_unix.system in
  let ctx = Cohttp_lwt_unix.Client.custom_ctx ~resolver () in

  let config = Config.make ~api_key:"sk-..." () in
  let* client = Client.create_with_ctx ~ctx config in
  (* Use client... *)
  Lwt.return_unit
```

### With Retry

```ocaml
open My_service
open Lwt.Syntax

let with_retry ?(max_retries = 3) action =
  let rec loop n =
    let* result = action () in
    match result with
    | Ok _ as ok -> Lwt.return ok
    | Error err when n < max_retries && Errors.is_retryable err ->
      let delay = Float.pow 2.0 (Float.of_int n) in
      let* () = Lwt_unix.sleep delay in
      loop (n + 1)
    | Error _ as err -> Lwt.return err
  in
  loop 0
```

## Platform Support

### Dependencies

**Runtime Dependencies:**
- `yojson` (>= 2.0) - JSON handling
- `ppx_yojson_conv` (>= 0.15) - JSON derivation
- `cohttp-lwt-unix` (>= 5.0) - HTTP client
- `lwt` (>= 5.6) - Async I/O
- `ptime` (>= 1.0) - Time handling
- `uri` (>= 4.0) - URI handling
- `logs` (>= 0.7) - Logging

**Development Dependencies:**
- `alcotest` (>= 1.7) - Testing framework
- `alcotest-lwt` (>= 1.7) - Async test support
- `odoc` (>= 2.2) - Documentation generation

### Minimum Versions

| Platform | Minimum Version | Rationale                        |
|----------|-----------------|----------------------------------|
| OCaml    | 4.14            | Modern features, LTS             |
| Dune     | 3.0             | Modern build system              |
| opam     | 2.1             | Current stable                   |

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidOCaml_Syntax(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_NamingConventions(t *testing.T)
func TestGenerate_ReservedWords(t *testing.T)
func TestGenerate_UnionTypes(t *testing.T)
func TestGenerate_EnumTypes(t *testing.T)
```

### Generated SDK Tests

```ocaml
(* test/test_{package}.ml *)
open Alcotest
open My_service

let test_types () =
  let json = `Assoc [("role", `String "user"); ("content", `String "Hello")] in
  let msg = Types.Message.of_yojson json in
  check string "role" "user" msg.role;
  check string "content" "Hello" msg.content

let test_config () =
  let config = Config.default in
  check string "base_url" "https://api.example.com" config.base_url;
  check (float 0.01) "timeout" 60.0 config.timeout

let test_json_roundtrip () =
  let msg = Types.Message.make ~role:`User ~content:"Hello" () in
  let json = Types.Message.yojson_of_t msg in
  let msg' = Types.Message.t_of_yojson json in
  check bool "roundtrip" true (Types.Message.equal msg msg')

let () =
  run "SDK Tests" [
    "Types", [
      test_case "parse message" `Quick test_types;
      test_case "json roundtrip" `Quick test_json_roundtrip;
    ];
    "Config", [
      test_case "defaults" `Quick test_config;
    ];
  ]
```

## Future Enhancements

1. **Eio Support**: Alternative to Lwt using OCaml 5 effects
2. **Async Support**: Jane Street's Async library support
3. **Custom Derive**: Custom ppx for more control over serialization
4. **Multicore**: OCaml 5 multicore-ready patterns
5. **WebSocket Streaming**: Support for WebSocket-based streaming
6. **Metrics**: Integration with prometheus-client-ocaml
7. **OpenTelemetry**: Tracing support via opentelemetry-ocaml
8. **MirageOS**: Unikernel-compatible client generation

## References

- [OCaml Manual](https://ocaml.org/manual/)
- [Real World OCaml](https://dev.realworldocaml.org/)
- [Dune Documentation](https://dune.build/docs)
- [Yojson Documentation](https://ocaml-community.github.io/yojson/)
- [Cohttp Documentation](https://github.com/mirage/ocaml-cohttp)
- [Lwt Manual](https://ocsigen.org/lwt/latest/manual/manual)
- [ppx_yojson_conv](https://github.com/janestreet/ppx_yojson_conv)
