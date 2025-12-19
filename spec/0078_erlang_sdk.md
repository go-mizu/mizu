# RFC 0078: Erlang SDK Generator

## Summary

Add Erlang SDK code generation to the Mizu contract system, enabling production-ready, idiomatic Erlang clients with excellent developer experience for OTP applications, embedded systems, and high-concurrency environments.

## Motivation

Erlang is the battle-tested foundation of the BEAM VM, renowned for fault-tolerance, massive concurrency, and "let it crash" philosophy. A native Erlang SDK provides:

1. **Idiomatic Erlang**: Records, tagged tuples, pattern matching, binary handling
2. **OTP Native**: gen_server-based clients, supervision tree integration, hot code upgrade support
3. **Production Proven**: Designed for telecom-grade reliability and 24/7 operation
4. **Streaming Excellence**: Binary pattern matching for efficient SSE parsing
5. **Embedded Friendly**: Minimal dependencies, suitable for embedded/IoT deployments
6. **Interop Ready**: Works seamlessly with Elixir, LFE, and other BEAM languages

## Design Goals

### Developer Experience (DX)

- **Idiomatic Erlang**: Records, pattern matching, proper `-spec` declarations
- **Tagged tuples**: `{ok, Result}` and `{error, Reason}` return values
- **Explicit is better**: No hidden magic, clear control flow
- **EDoc documentation**: Rich inline documentation with examples
- **Dialyzer compatible**: Full type specifications for static analysis
- **Shell friendly**: Easy to explore in `erl` shell

### Production Readiness

- **hackney HTTP client**: Battle-tested, connection pooling, streaming support
- **OTP behaviors**: Optional gen_server-based client for connection management
- **Supervision ready**: Clients can be supervised and restarted automatically
- **Hot code upgrade**: Compatible with OTP release upgrades
- **Telemetry integration**: Optional telemetry events for observability
- **Binary efficiency**: Efficient binary handling for streaming

## Architecture

### Project Structure

```
{app_name}/
├── rebar.config                      # Rebar3 build configuration
├── src/
│   ├── {app_name}.app.src           # OTP application resource file
│   ├── {app_name}.erl               # Main API module
│   ├── {app_name}_client.erl        # HTTP client implementation
│   ├── {app_name}_config.erl        # Configuration handling
│   ├── {app_name}_types.erl         # Type definitions and records
│   ├── {app_name}_types.hrl         # Record definitions (header file)
│   ├── {app_name}_{resource}.erl    # Resource modules (one per resource)
│   ├── {app_name}_streaming.erl     # SSE streaming utilities
│   └── {app_name}_errors.erl        # Error types and handling
└── include/
    └── {app_name}.hrl               # Public header with records
```

### Core Components

#### 1. Main Module (`src/{app_name}.erl`)

The top-level API facade:

```erlang
-module({app_name}).

-export([
    client/0,
    client/1,
    {resource}/1
]).

-include("{app_name}.hrl").

%% @doc Creates a new API client with default options.
%%
%% Configuration is read from application environment.
%%
%% == Example ==
%% ```
%% Client = {app_name}:client(),
%% {ok, Response} = {app_name}_messages:create(Client, #{
%%     model => <<"model-name">>,
%%     messages => [#{role => <<"user">>, content => <<"Hello">>}]
%% }).
%% '''
-spec client() -> {app_name}_client:client().
client() ->
    client([]).

%% @doc Creates a new API client with the given options.
%%
%% == Options ==
%% <ul>
%%   <li>`api_key' - API key for authentication (binary)</li>
%%   <li>`base_url' - Override the default base URL</li>
%%   <li>`timeout' - Request timeout in milliseconds (default: 60000)</li>
%%   <li>`max_retries' - Maximum retry attempts (default: 2)</li>
%% </ul>
%%
%% == Example ==
%% ```
%% Client = {app_name}:client([
%%     {api_key, <<"sk-...">>},
%%     {timeout, 120000}
%% ]).
%% '''
-spec client(proplists:proplist()) -> {app_name}_client:client().
client(Opts) ->
    {app_name}_client:new(Opts).
```

#### 2. Client (`src/{app_name}_client.erl`)

The HTTP client implementation:

```erlang
-module({app_name}_client).

-export([
    new/0,
    new/1,
    request/2,
    stream/2
]).

-include("{app_name}.hrl").

-record(client, {
    config :: {app_name}_config:config(),
    pool :: atom()
}).

-opaque client() :: #client{}.
-export_type([client/0]).

%% @doc Creates a new client with default options.
-spec new() -> client().
new() ->
    new([]).

%% @doc Creates a new client with the given options.
-spec new(proplists:proplist()) -> client().
new(Opts) ->
    Config = {app_name}_config:new(Opts),
    Pool = make_ref_pool(),
    ensure_started(),
    #client{config = Config, pool = Pool}.

%% @doc Performs an HTTP request.
%%
%% Returns `{ok, Response}' on success or `{error, Reason}' on failure.
%%
%% == Options ==
%% <ul>
%%   <li>`method' - HTTP method (required): get, post, put, patch, delete</li>
%%   <li>`path' - Request path (required)</li>
%%   <li>`body' - Request body as map (optional)</li>
%%   <li>`headers' - Additional headers (optional)</li>
%% </ul>
-spec request(client(), proplists:proplist()) ->
    {ok, map() | undefined} | {error, term()}.
request(#client{config = Config}, Opts) ->
    Method = proplists:get_value(method, Opts),
    Path = proplists:get_value(path, Opts),
    Body = proplists:get_value(body, Opts, undefined),
    ExtraHeaders = proplists:get_value(headers, Opts, []),

    URL = build_url(Config, Path),
    Headers = build_headers(Config, ExtraHeaders),
    ReqBody = encode_body(Body),
    HTTPOpts = build_http_opts(Config),

    case hackney:request(Method, URL, Headers, ReqBody, HTTPOpts) of
        {ok, Status, _RespHeaders, ClientRef} when Status >= 200, Status < 300 ->
            {ok, RespBody} = hackney:body(ClientRef),
            case RespBody of
                <<>> -> {ok, undefined};
                _ -> {ok, jsx:decode(RespBody, [return_maps])}
            end;
        {ok, 204, _RespHeaders, ClientRef} ->
            hackney:skip_body(ClientRef),
            {ok, undefined};
        {ok, Status, _RespHeaders, ClientRef} ->
            {ok, RespBody} = hackney:body(ClientRef),
            {error, {app_name}_errors:from_response(Status, RespBody)};
        {error, Reason} ->
            {error, {app_name}_errors:connection_error(Reason)}
    end.

%% @doc Performs a streaming SSE request.
%%
%% Returns a function that can be called repeatedly to get events.
%% Each call returns `{ok, Event, Continuation}' or `{done, Continuation}'
%% or `{error, Reason}'.
%%
%% == Example ==
%% ```
%% {ok, Stream} = {app_name}_client:stream(Client, [
%%     {method, post},
%%     {path, <<"/v1/messages">>},
%%     {body, Body}
%% ]),
%% consume_stream(Stream).
%%
%% consume_stream({done, _}) -> ok;
%% consume_stream({error, Reason}) -> {error, Reason};
%% consume_stream(Stream) ->
%%     case {app_name}_streaming:next(Stream) of
%%         {ok, Event, NextStream} ->
%%             io:format("Event: ~p~n", [Event]),
%%             consume_stream(NextStream);
%%         {done, _} ->
%%             ok;
%%         {error, Reason} ->
%%             {error, Reason}
%%     end.
%% '''
-spec stream(client(), proplists:proplist()) ->
    {ok, {app_name}_streaming:stream()} | {error, term()}.
stream(#client{config = Config}, Opts) ->
    Method = proplists:get_value(method, Opts),
    Path = proplists:get_value(path, Opts),
    Body = proplists:get_value(body, Opts, undefined),

    URL = build_url(Config, Path),
    Headers = build_headers(Config, [{<<"accept">>, <<"text/event-stream">>}]),
    ReqBody = encode_body(Body),
    HTTPOpts = [{async, once}, {stream_to, self()}] ++ build_http_opts(Config),

    case hackney:request(Method, URL, Headers, ReqBody, HTTPOpts) of
        {ok, ClientRef} ->
            {ok, {app_name}_streaming:new(ClientRef)};
        {error, Reason} ->
            {error, {app_name}_errors:connection_error(Reason)}
    end.

%% Private functions

ensure_started() ->
    case application:ensure_all_started(hackney) of
        {ok, _} -> ok;
        {error, _} = Err -> Err
    end.

make_ref_pool() ->
    list_to_atom("pool_" ++ integer_to_list(erlang:unique_integer([positive]))).

build_url(Config, Path) ->
    BaseURL = {app_name}_config:base_url(Config),
    <<BaseURL/binary, Path/binary>>.

build_headers(Config, Extra) ->
    Base = [
        {<<"content-type">>, <<"application/json">>},
        {<<"accept">>, <<"application/json">>}
    ],
    Auth = apply_auth(Config),
    Custom = {app_name}_config:headers(Config),
    lists:usort(Auth ++ Base ++ Custom ++ Extra).

apply_auth(Config) ->
    case {app_name}_config:api_key(Config) of
        undefined -> [];
        Key ->
            case {app_name}_config:auth_mode(Config) of
                bearer -> [{<<"authorization">>, <<"Bearer ", Key/binary>>}];
                basic -> [{<<"authorization">>, <<"Basic ", (base64:encode(Key))/binary>>}];
                header -> [{<<"x-api-key">>, Key}];
                none -> []
            end
    end.

encode_body(undefined) -> <<>>;
encode_body(Body) when is_map(Body) -> jsx:encode(Body);
encode_body(Body) when is_binary(Body) -> Body.

build_http_opts(Config) ->
    Timeout = {app_name}_config:timeout(Config),
    [
        {connect_timeout, Timeout},
        {recv_timeout, Timeout},
        with_body
    ].
```

#### 3. Config (`src/{app_name}_config.erl`)

Configuration handling:

```erlang
-module({app_name}_config).

-export([
    new/0,
    new/1,
    api_key/1,
    base_url/1,
    timeout/1,
    max_retries/1,
    auth_mode/1,
    headers/1
]).

-record(config, {
    api_key :: binary() | undefined,
    base_url :: binary(),
    timeout :: pos_integer(),
    max_retries :: non_neg_integer(),
    auth_mode :: bearer | basic | header | none,
    headers :: [{binary(), binary()}]
}).

-opaque config() :: #config{}.
-export_type([config/0]).

-define(DEFAULT_BASE_URL, <<"{default_base_url}">>).
-define(DEFAULT_TIMEOUT, 60000).
-define(DEFAULT_MAX_RETRIES, 2).
-define(DEFAULT_AUTH_MODE, bearer).

%% @doc Creates a new configuration with defaults.
-spec new() -> config().
new() ->
    new([]).

%% @doc Creates a new configuration from options.
%%
%% Options override application environment which overrides defaults.
-spec new(proplists:proplist()) -> config().
new(Opts) ->
    ApiKey = get_opt(api_key, Opts, undefined),
    BaseURL = get_opt(base_url, Opts, ?DEFAULT_BASE_URL),
    Timeout = get_opt(timeout, Opts, ?DEFAULT_TIMEOUT),
    MaxRetries = get_opt(max_retries, Opts, ?DEFAULT_MAX_RETRIES),
    AuthMode = get_opt(auth_mode, Opts, ?DEFAULT_AUTH_MODE),
    Headers = get_opt(headers, Opts, []),

    #config{
        api_key = ensure_binary(ApiKey),
        base_url = ensure_binary(BaseURL),
        timeout = Timeout,
        max_retries = MaxRetries,
        auth_mode = AuthMode,
        headers = normalize_headers(Headers)
    }.

%% Accessors
-spec api_key(config()) -> binary() | undefined.
api_key(#config{api_key = V}) -> V.

-spec base_url(config()) -> binary().
base_url(#config{base_url = V}) -> V.

-spec timeout(config()) -> pos_integer().
timeout(#config{timeout = V}) -> V.

-spec max_retries(config()) -> non_neg_integer().
max_retries(#config{max_retries = V}) -> V.

-spec auth_mode(config()) -> bearer | basic | header | none.
auth_mode(#config{auth_mode = V}) -> V.

-spec headers(config()) -> [{binary(), binary()}].
headers(#config{headers = V}) -> V.

%% Private

get_opt(Key, Opts, Default) ->
    case proplists:get_value(Key, Opts) of
        undefined -> get_app_env(Key, Default);
        Value -> Value
    end.

get_app_env(Key, Default) ->
    application:get_env({app_name}, Key, Default).

ensure_binary(undefined) -> undefined;
ensure_binary(B) when is_binary(B) -> B;
ensure_binary(L) when is_list(L) -> list_to_binary(L);
ensure_binary(A) when is_atom(A) -> atom_to_binary(A, utf8).

normalize_headers(Headers) ->
    [{ensure_binary(K), ensure_binary(V)} || {K, V} <- Headers].
```

#### 4. Types Header (`include/{app_name}.hrl`)

Record definitions for public use:

```erlang
%% {app_name}.hrl - Generated record definitions
%% DO NOT EDIT - Generated by sdkerlang

-ifndef({APP_NAME}_HRL).
-define({APP_NAME}_HRL, true).

%% {TypeName} record
%% {Type description}
-record({type_name}, {
    field_name :: binary(),
    optional_field :: binary() | undefined
}).

-type {type_name}() :: #{type_name}{}.

%% More records...

-endif.
```

#### 5. Types Module (`src/{app_name}_types.erl`)

Type conversion and utilities:

```erlang
-module({app_name}_types).

-export([
    from_map/2,
    to_map/1,
    parse_datetime/1
]).

-include("{app_name}.hrl").

%% @doc Creates a record from a map.
%%
%% == Example ==
%% ```
%% Map = #{<<"id">> => <<"123">>, <<"name">> => <<"test">>},
%% Record = {app_name}_types:from_map(my_record, Map).
%% '''
-spec from_map(atom(), map()) -> tuple().
from_map(RecordType, Map) when is_map(Map) ->
    Fields = record_fields(RecordType),
    Values = [get_field_value(F, Map) || F <- Fields],
    list_to_tuple([RecordType | Values]).

%% @doc Converts a record to a map.
-spec to_map(tuple()) -> map().
to_map(Record) when is_tuple(Record) ->
    RecordType = element(1, Record),
    Fields = record_fields(RecordType),
    Values = tl(tuple_to_list(Record)),
    Pairs = lists:zip(Fields, Values),
    maps:from_list([{atom_to_binary(K, utf8), serialize_value(V)}
                    || {K, V} <- Pairs, V =/= undefined]).

%% @doc Parses an ISO8601 datetime string.
-spec parse_datetime(binary() | undefined) -> calendar:datetime() | undefined.
parse_datetime(undefined) -> undefined;
parse_datetime(<<Y:4/binary, "-", M:2/binary, "-", D:2/binary, "T",
                 H:2/binary, ":", Mi:2/binary, ":", S:2/binary, _/binary>>) ->
    {{binary_to_integer(Y), binary_to_integer(M), binary_to_integer(D)},
     {binary_to_integer(H), binary_to_integer(Mi), binary_to_integer(S)}};
parse_datetime(Bin) when is_binary(Bin) -> Bin.

%% Private

get_field_value(Field, Map) ->
    FieldBin = atom_to_binary(Field, utf8),
    case maps:find(FieldBin, Map) of
        {ok, Value} -> Value;
        error ->
            %% Try snake_case to camelCase conversion
            FieldCamel = to_camel_case(FieldBin),
            maps:get(FieldCamel, Map, undefined)
    end.

serialize_value(undefined) -> undefined;
serialize_value(V) when is_tuple(V), tuple_size(V) > 0 ->
    case is_record(V) of
        true -> to_map(V);
        false -> V
    end;
serialize_value(L) when is_list(L) ->
    [serialize_value(E) || E <- L];
serialize_value(V) -> V.

is_record(T) when is_tuple(T), tuple_size(T) > 0 ->
    is_atom(element(1, T));
is_record(_) -> false.

to_camel_case(Bin) ->
    Parts = binary:split(Bin, <<"_">>, [global]),
    case Parts of
        [First | Rest] ->
            Capitalized = [capitalize(P) || P <- Rest],
            iolist_to_binary([First | Capitalized]);
        _ -> Bin
    end.

capitalize(<<>>) -> <<>>;
capitalize(<<C, Rest/binary>>) when C >= $a, C =< $z ->
    <<(C - 32), Rest/binary>>;
capitalize(Bin) -> Bin.

%% Record field definitions (generated per type)
record_fields({type_name}) -> [field1, field2, field3];
record_fields(_) -> [].
```

#### 6. Resource Module (`src/{app_name}_{resource}.erl`)

Resource operations:

```erlang
-module({app_name}_{resource}).

-export([
    create/2,
    create/3,
    stream/2,
    stream/3
]).

-include("{app_name}.hrl").

%% @doc Creates a new {resource}.
%%
%% == Parameters ==
%% <ul>
%%   <li>`Client' - The API client</li>
%%   <li>`Params' - Request parameters as a map</li>
%% </ul>
%%
%% == Example ==
%% ```
%% {ok, Response} = {app_name}_{resource}:create(Client, #{
%%     model => <<"model-name">>,
%%     messages => [#{role => <<"user">>, content => <<"Hello">>}]
%% }).
%% '''
-spec create({app_name}_client:client(), map()) ->
    {ok, #{type_name}()} | {error, term()}.
create(Client, Params) ->
    create(Client, Params, []).

-spec create({app_name}_client:client(), map(), proplists:proplist()) ->
    {ok, #{type_name}()} | {error, term()}.
create(Client, Params, Opts) ->
    case {app_name}_client:request(Client, [
        {method, post},
        {path, <<"/v1/{resource}">>},
        {body, Params}
    ] ++ Opts) of
        {ok, Response} ->
            {ok, {app_name}_types:from_map({type_name}, Response)};
        {error, _} = Err ->
            Err
    end.

%% @doc Creates a streaming request.
%%
%% Returns a stream that yields events.
%%
%% == Example ==
%% ```
%% {ok, Stream} = {app_name}_{resource}:stream(Client, #{
%%     model => <<"model-name">>,
%%     messages => [#{role => <<"user">>, content => <<"Hello">>}]
%% }),
%% print_stream(Stream).
%% '''
-spec stream({app_name}_client:client(), map()) ->
    {ok, {app_name}_streaming:stream()} | {error, term()}.
stream(Client, Params) ->
    stream(Client, Params, []).

-spec stream({app_name}_client:client(), map(), proplists:proplist()) ->
    {ok, {app_name}_streaming:stream()} | {error, term()}.
stream(Client, Params, _Opts) ->
    Body = maps:put(stream, true, Params),
    {app_name}_client:stream(Client, [
        {method, post},
        {path, <<"/v1/{resource}">>},
        {body, Body}
    ]).
```

#### 7. Streaming (`src/{app_name}_streaming.erl`)

SSE streaming implementation:

```erlang
-module({app_name}_streaming).

-export([
    new/1,
    next/1,
    close/1,
    fold/3,
    foreach/2
]).

-record(stream, {
    client_ref :: reference(),
    buffer :: binary(),
    done :: boolean()
}).

-opaque stream() :: #stream{}.
-export_type([stream/0]).

%% @doc Creates a new stream from a hackney client reference.
-spec new(reference()) -> stream().
new(ClientRef) ->
    #stream{client_ref = ClientRef, buffer = <<>>, done = false}.

%% @doc Gets the next event from the stream.
%%
%% Returns:
%% <ul>
%%   <li>`{ok, Event, Stream}' - An event was received</li>
%%   <li>`{done, Stream}' - Stream completed normally</li>
%%   <li>`{error, Reason}' - An error occurred</li>
%% </ul>
-spec next(stream()) ->
    {ok, map(), stream()} | {done, stream()} | {error, term()}.
next(#stream{done = true} = Stream) ->
    {done, Stream};
next(#stream{client_ref = Ref, buffer = Buffer} = Stream) ->
    ok = hackney:stream_next(Ref),
    receive
        {hackney_response, Ref, {status, Status, _Reason}} when Status >= 200, Status < 300 ->
            next(Stream);
        {hackney_response, Ref, {status, Status, _Reason}} ->
            {error, {http_error, Status}};
        {hackney_response, Ref, {headers, _Headers}} ->
            next(Stream);
        {hackney_response, Ref, done} ->
            {done, Stream#stream{done = true}};
        {hackney_response, Ref, {error, Reason}} ->
            {error, Reason};
        {hackney_response, Ref, Chunk} when is_binary(Chunk) ->
            process_chunk(Stream, <<Buffer/binary, Chunk/binary>>)
    after 60000 ->
        {error, timeout}
    end.

%% @doc Closes the stream.
-spec close(stream()) -> ok.
close(#stream{client_ref = Ref}) ->
    hackney:close(Ref),
    ok.

%% @doc Folds over all events in the stream.
%%
%% == Example ==
%% ```
%% {ok, Result} = {app_name}_streaming:fold(Stream, [], fun(Event, Acc) ->
%%     [Event | Acc]
%% end).
%% '''
-spec fold(stream(), Acc, fun((map(), Acc) -> Acc)) ->
    {ok, Acc} | {error, term()} when Acc :: term().
fold(Stream, Acc, Fun) ->
    case next(Stream) of
        {ok, Event, NextStream} ->
            fold(NextStream, Fun(Event, Acc), Fun);
        {done, _} ->
            {ok, Acc};
        {error, Reason} ->
            {error, Reason}
    end.

%% @doc Iterates over all events in the stream.
%%
%% == Example ==
%% ```
%% ok = {app_name}_streaming:foreach(Stream, fun(Event) ->
%%     io:format("~p~n", [Event])
%% end).
%% '''
-spec foreach(stream(), fun((map()) -> term())) -> ok | {error, term()}.
foreach(Stream, Fun) ->
    case next(Stream) of
        {ok, Event, NextStream} ->
            Fun(Event),
            foreach(NextStream, Fun);
        {done, _} ->
            ok;
        {error, Reason} ->
            {error, Reason}
    end.

%% Private

process_chunk(Stream, Buffer) ->
    case binary:split(Buffer, <<"\n\n">>) of
        [Event, Rest] ->
            case parse_sse_event(Event) of
                done ->
                    {done, Stream#stream{buffer = Rest, done = true}};
                skip ->
                    next(Stream#stream{buffer = Rest});
                {ok, Data} ->
                    {ok, Data, Stream#stream{buffer = Rest}}
            end;
        [Incomplete] ->
            next(Stream#stream{buffer = Incomplete})
    end.

parse_sse_event(Event) ->
    Lines = binary:split(Event, <<"\n">>, [global]),
    DataLines = [extract_data(L) || L <- Lines, is_data_line(L)],
    Data = iolist_to_binary(lists:join(<<"\n">>, DataLines)),
    case Data of
        <<>> -> skip;
        <<"[DONE]">> -> done;
        _ ->
            try
                {ok, jsx:decode(Data, [return_maps])}
            catch
                _:_ -> skip
            end
    end.

is_data_line(<<"data:", _/binary>>) -> true;
is_data_line(_) -> false.

extract_data(<<"data:", Rest/binary>>) ->
    string:trim(Rest);
extract_data(_) ->
    <<>>.
```

#### 8. Errors (`src/{app_name}_errors.erl`)

Error types and handling:

```erlang
-module({app_name}_errors).

-export([
    from_response/2,
    connection_error/1,
    format_error/1
]).

-type error_type() ::
    bad_request |
    unauthorized |
    forbidden |
    not_found |
    unprocessable_entity |
    rate_limited |
    server_error |
    connection_error |
    timeout_error |
    stream_error.

-type sdk_error() :: {error_type(), #{
    message := binary(),
    status => integer(),
    body => term(),
    retry_after => integer()
}}.

-export_type([error_type/0, sdk_error/0]).

%% @doc Creates an error from an HTTP response.
-spec from_response(integer(), binary() | map()) -> sdk_error().
from_response(Status, Body) when is_binary(Body) ->
    ParsedBody = try jsx:decode(Body, [return_maps]) catch _:_ -> Body end,
    from_response(Status, ParsedBody);
from_response(400, Body) ->
    {bad_request, #{message => extract_message(Body), status => 400, body => Body}};
from_response(401, Body) ->
    {unauthorized, #{message => extract_message(Body), status => 401, body => Body}};
from_response(403, Body) ->
    {forbidden, #{message => extract_message(Body), status => 403, body => Body}};
from_response(404, Body) ->
    {not_found, #{message => extract_message(Body), status => 404, body => Body}};
from_response(422, Body) ->
    {unprocessable_entity, #{message => extract_message(Body), status => 422, body => Body}};
from_response(429, Body) ->
    {rate_limited, #{
        message => extract_message(Body),
        status => 429,
        body => Body,
        retry_after => extract_retry_after(Body)
    }};
from_response(Status, Body) when Status >= 500 ->
    {server_error, #{message => extract_message(Body), status => Status, body => Body}};
from_response(Status, Body) ->
    {server_error, #{message => extract_message(Body), status => Status, body => Body}}.

%% @doc Creates a connection error.
-spec connection_error(term()) -> sdk_error().
connection_error(timeout) ->
    {timeout_error, #{message => <<"Request timed out">>}};
connection_error(Reason) ->
    {connection_error, #{message => format_reason(Reason)}}.

%% @doc Formats an error for display.
-spec format_error(sdk_error()) -> binary().
format_error({Type, #{message := Msg} = Info}) ->
    Status = maps:get(status, Info, undefined),
    case Status of
        undefined ->
            iolist_to_binary(io_lib:format("~s: ~s", [Type, Msg]));
        _ ->
            iolist_to_binary(io_lib:format("~s (~p): ~s", [Type, Status, Msg]))
    end.

%% Private

extract_message(#{<<"message">> := Msg}) -> Msg;
extract_message(#{<<"error">> := #{<<"message">> := Msg}}) -> Msg;
extract_message(#{<<"error">> := Msg}) when is_binary(Msg) -> Msg;
extract_message(Msg) when is_binary(Msg) -> Msg;
extract_message(_) -> <<"Unknown error">>.

extract_retry_after(#{<<"retry_after">> := V}) when is_integer(V) -> V;
extract_retry_after(_) -> undefined.

format_reason(Reason) when is_atom(Reason) ->
    atom_to_binary(Reason, utf8);
format_reason(Reason) when is_binary(Reason) ->
    Reason;
format_reason(Reason) ->
    iolist_to_binary(io_lib:format("~p", [Reason])).
```

## Type Mapping

### Primitive Types

| Contract Type     | Erlang Type              | Type Spec              |
|-------------------|--------------------------|------------------------|
| `string`          | `binary()`               | `binary()`             |
| `bool`, `boolean` | `boolean()`              | `boolean()`            |
| `int`             | `integer()`              | `integer()`            |
| `int8`            | `integer()`              | `-128..127`            |
| `int16`           | `integer()`              | `-32768..32767`        |
| `int32`           | `integer()`              | `integer()`            |
| `int64`           | `integer()`              | `integer()`            |
| `uint`            | `non_neg_integer()`      | `non_neg_integer()`    |
| `uint8`           | `non_neg_integer()`      | `0..255`               |
| `uint16`          | `non_neg_integer()`      | `0..65535`             |
| `uint32`          | `non_neg_integer()`      | `non_neg_integer()`    |
| `uint64`          | `non_neg_integer()`      | `non_neg_integer()`    |
| `float32`         | `float()`                | `float()`              |
| `float64`         | `float()`                | `float()`              |
| `time.Time`       | `calendar:datetime()`    | `calendar:datetime()`  |
| `json.RawMessage` | `map()`                  | `map()`                |
| `any`             | `term()`                 | `term()`               |

### Collection Types

| Contract Type      | Erlang Type              | Type Spec              |
|--------------------|--------------------------|------------------------|
| `[]T`              | `[T]`                    | `[T]`                  |
| `map[string]T`     | `#{binary() => T}`       | `#{binary() => T}`     |

### Optional/Nullable

| Contract         | Erlang               | Type Spec             |
|------------------|----------------------|----------------------|
| `optional: T`    | `T \| undefined`     | `T \| undefined`     |
| `nullable: T`    | `T \| undefined`     | `T \| undefined`     |

### Struct to Record

Contract structs map to Erlang records:

```erlang
%% From contract type:
%% {Name: "Message", Fields: [{role, string}, {content, string}]}

-record(message, {
    role :: binary(),
    content :: binary()
}).

-type message() :: #message{}.
```

### Discriminated Unions

Unions use tagged tuples:

```erlang
%% ContentBlock = TextBlock | ImageBlock | ToolUseBlock

-type content_block() ::
    {text_block, #text_block{}} |
    {image_block, #image_block{}} |
    {tool_use_block, #tool_use_block{}}.

%% Pattern matching:
handle_content({text_block, #text_block{text = Text}}) ->
    io:format("Text: ~s~n", [Text]);
handle_content({image_block, #image_block{url = URL}}) ->
    io:format("Image: ~s~n", [URL]).
```

## Naming Conventions

### Erlang Naming

| Contract       | Erlang                  |
|----------------|-------------------------|
| `user-id`      | `user_id`               |
| `userName`     | `user_name`             |
| `UserData`     | `user_data` (record)    |
| `create`       | `create/2`              |
| `getMessage`   | `get_message/2`         |
| `maxTokens`    | `max_tokens`            |

### Module Naming

- Application name: `{service}_sdk` (snake_case)
- Main module: `{service}`
- Client: `{service}_client`
- Resources: `{service}_{resource}`
- Types: `{service}_types`

### Reserved Words

Erlang reserved words are handled by appending underscore:
- `after` -> `after_`
- `receive` -> `receive_`
- `case` -> `case_`
- `if` -> `if_`
- `end` -> `end_`
- `fun` -> `fun_`

## Code Generation

### Generator Structure

```go
package sdkerlang

type Config struct {
    // AppName is the OTP application name.
    // Default: sanitized lowercase service name with underscores.
    AppName string

    // Version is the package version.
    Version string

    // Description is the package description.
    Description string
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── rebar.config.tmpl         # Rebar3 configuration
├── app.src.tmpl              # Application resource file
├── main.erl.tmpl             # Main module
├── client.erl.tmpl           # HTTP client
├── config.erl.tmpl           # Configuration
├── types.erl.tmpl            # Type conversion utilities
├── types.hrl.tmpl            # Record definitions
├── include.hrl.tmpl          # Public header
├── resource.erl.tmpl         # Resource modules
├── streaming.erl.tmpl        # SSE streaming
└── errors.erl.tmpl           # Error handling
```

### Generated Files

| File                              | Purpose                      |
|-----------------------------------|------------------------------|
| `rebar.config`                    | Rebar3 build configuration   |
| `src/{app}.app.src`               | OTP application resource     |
| `src/{app}.erl`                   | Main API module              |
| `src/{app}_client.erl`            | HTTP client                  |
| `src/{app}_config.erl`            | Configuration handling       |
| `src/{app}_types.erl`             | Type utilities               |
| `src/{app}_{resource}.erl`        | Resource modules             |
| `src/{app}_streaming.erl`         | SSE streaming                |
| `src/{app}_errors.erl`            | Error types                  |
| `include/{app}.hrl`               | Public header with records   |

## Usage Examples

### Basic Usage

```erlang
%% Add to rebar.config
{deps, [
    {my_service, "1.0.0"}
]}.

%% Create client
Client = my_service:client([
    {api_key, <<"your-api-key">>}
]),

%% Make a request
{ok, Response} = my_service_messages:create(Client, #{
    model => <<"model-name">>,
    messages => [#{role => <<"user">>, content => <<"Hello">>}]
}),

io:format("Response: ~p~n", [Response]).
```

### Streaming

```erlang
%% Start streaming
{ok, Stream} = my_service_messages:stream(Client, #{
    model => <<"model-name">>,
    messages => [#{role => <<"user">>, content => <<"Hello">>}]
}),

%% Process events with foreach
ok = my_service_streaming:foreach(Stream, fun(Event) ->
    case maps:get(<<"delta">>, Event, undefined) of
        undefined -> ok;
        Delta ->
            case maps:get(<<"text">>, Delta, undefined) of
                undefined -> ok;
                Text -> io:format("~s", [Text])
            end
    end
end),
io:format("~n").
```

### Streaming with Fold

```erlang
%% Collect all text chunks
{ok, Texts} = my_service_streaming:fold(Stream, [], fun(Event, Acc) ->
    case maps:get(<<"delta">>, Event, undefined) of
        #{<<"text">> := Text} -> [Text | Acc];
        _ -> Acc
    end
end),

FullText = iolist_to_binary(lists:reverse(Texts)),
io:format("Full response: ~s~n", [FullText]).
```

### Error Handling

```erlang
case my_service_messages:create(Client, Params) of
    {ok, Response} ->
        io:format("Success: ~p~n", [Response]);
    {error, {rate_limited, #{retry_after := Retry}}} ->
        timer:sleep(Retry * 1000),
        %% Retry...
        ok;
    {error, {unauthorized, _}} ->
        io:format("Invalid API key~n");
    {error, {Type, #{message := Msg, status := Status}}} ->
        io:format("Error ~p (~p): ~s~n", [Type, Status, Msg]);
    {error, {connection_error, #{message := Msg}}} ->
        io:format("Network error: ~s~n", [Msg])
end.
```

### OTP Application Configuration

```erlang
%% sys.config
[
    {my_service, [
        {api_key, <<"your-api-key">>},
        {base_url, <<"https://api.example.com">>},
        {timeout, 60000},
        {max_retries, 2}
    ]}
].

%% In code
Client = my_service:client().  %% Uses app env
```

### With Supervision Tree

```erlang
%% my_service_pool.erl - Optional GenServer wrapper
-module(my_service_pool).
-behaviour(gen_server).

-export([start_link/1, request/2]).
-export([init/1, handle_call/3, handle_cast/2]).

start_link(Opts) ->
    gen_server:start_link({local, ?MODULE}, ?MODULE, Opts, []).

init(Opts) ->
    Client = my_service:client(Opts),
    {ok, #{client => Client}}.

request(Resource, Params) ->
    gen_server:call(?MODULE, {request, Resource, Params}).

handle_call({request, messages, Params}, _From, #{client := Client} = State) ->
    Result = my_service_messages:create(Client, Params),
    {reply, Result, State}.

%% In supervisor
init([]) ->
    Children = [
        #{id => my_service_pool,
          start => {my_service_pool, start_link, [[{api_key, get_api_key()}]]},
          restart => permanent}
    ],
    {ok, {{one_for_one, 10, 60}, Children}}.
```

## Platform Support

### Dependencies

**Runtime Dependencies:**
- `hackney` (~> 1.18) - HTTP client with connection pooling
- `jsx` (~> 3.1) - JSON encoding/decoding

**Development Dependencies:**
- `rebar3_hex` - Hex.pm publishing
- `rebar3_dialyzer` - Dialyzer analysis

### Minimum Versions

| Platform | Minimum Version | Rationale                        |
|----------|-----------------|----------------------------------|
| Erlang   | OTP 24          | Modern features, maps syntax     |
| rebar3   | 3.18            | Hex plugin, deps management      |
| hackney  | 1.18            | Stable streaming support         |

### OTP Compatibility

- OTP 24, 25, 26, 27 supported
- Fully compatible with releases and hot code upgrade
- Works with all BEAM languages (Elixir, LFE, Gleam)

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidErlang_Syntax(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_NamingConventions(t *testing.T)
func TestGenerate_ReservedWords(t *testing.T)
func TestGenerate_Records(t *testing.T)
```

### Generated SDK Tests

```erlang
%% test/{app}_SUITE.erl - Common Test suite
-module({app}_SUITE).
-include_lib("common_test/include/ct.hrl").

all() -> [
    client_creation,
    request_handling,
    streaming,
    error_handling
].

client_creation(_Config) ->
    Client = my_service:client([{api_key, <<"test">>}]),
    true = is_tuple(Client).

%% More tests...
```

## Future Enhancements

1. **Telemetry events**: Emit telemetry for request timing and errors
2. **Connection pooling**: Built-in hackney pool management
3. **Circuit breaker**: Integration with `fuse` library
4. **Batch requests**: Support for batched API calls
5. **WebSocket streaming**: Support for WebSocket-based streaming
6. **Property-based testing**: PropEr/QuickCheck test generation
7. **Metrics integration**: Prometheus/Folsom integration

## References

- [Erlang Programming Rules](http://www.erlang.se/doc/programming_rules.shtml)
- [OTP Design Principles](https://www.erlang.org/doc/design_principles/des_princ.html)
- [hackney Documentation](https://github.com/benoitc/hackney)
- [jsx Documentation](https://github.com/talentdeficit/jsx)
- [Rebar3 Documentation](https://rebar3.org/docs/)
- [EDoc User Guide](https://www.erlang.org/doc/apps/edoc/chapter.html)
- [Dialyzer User Guide](https://www.erlang.org/doc/apps/dialyzer/dialyzer_chapter.html)
