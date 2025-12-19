# RFC 0079: Haskell SDK Generator

## Summary

Add Haskell SDK code generation to the Mizu contract system, enabling production-ready, idiomatic Haskell clients with excellent developer experience leveraging Haskell's powerful type system, algebraic data types, and functional programming paradigms.

## Motivation

Haskell is a purely functional programming language known for its strong static type system, type inference, and expressive type-level programming. A native Haskell SDK provides:

1. **Type-Safe by Design**: Leverage Haskell's type system to catch errors at compile time
2. **Algebraic Data Types**: Perfect representation for discriminated unions and enums
3. **Composable Error Handling**: Either monad for explicit, composable error handling
4. **Streaming Excellence**: Conduit-based streaming with backpressure and resource safety
5. **Production Proven**: Used in fintech, blockchain, and mission-critical systems
6. **Immutability by Default**: Pure functions and immutable data structures

## Design Goals

### Developer Experience (DX)

- **Idiomatic Haskell**: ADTs, pattern matching, type classes, do-notation
- **Type-Safe**: Full type coverage with no partial functions
- **Either-Based Errors**: `Either SDKError a` for explicit error handling
- **Generic Deriving**: Automatic JSON instances via DeriveGeneric and Aeson
- **Haddock Documentation**: Rich inline documentation with examples
- **GHCi Friendly**: Easy to explore in the REPL
- **Minimal Dependencies**: Only well-maintained, battle-tested libraries

### Production Readiness

- **http-conduit**: Battle-tested HTTP client with connection pooling
- **conduit**: Resource-safe streaming with backpressure
- **aeson**: Industry-standard JSON serialization
- **Retry Logic**: Built-in exponential backoff for transient errors
- **Connection Pooling**: Via http-conduit's Manager
- **Resource Safety**: Bracket patterns and ResourceT for cleanup
- **Configurable**: Flexible configuration via record types

## Architecture

### Project Structure

```
{package-name}/
├── {package-name}.cabal        # Cabal package configuration
├── package.yaml                # Optional hpack configuration
├── src/
│   └── {ModuleName}/
│       ├── Client.hs           # HTTP client implementation
│       ├── Config.hs           # Configuration types
│       ├── Types.hs            # Generated type definitions
│       ├── Resources/
│       │   └── {Resource}.hs   # Resource modules
│       ├── Streaming.hs        # SSE streaming support
│       └── Errors.hs           # Error types and handling
│   └── {ModuleName}.hs         # Main module (re-exports)
└── test/
    └── Spec.hs                 # Test suite
```

### Core Components

#### 1. Main Module (`src/{ModuleName}.hs`)

The top-level module re-exporting all public APIs:

```haskell
{-# LANGUAGE OverloadedStrings #-}

-- | {Service description}
--
-- = Quick Start
--
-- @
-- import qualified {ModuleName}
-- import qualified {ModuleName}.Resources.Messages as Messages
--
-- main :: IO ()
-- main = do
--     client <- {ModuleName}.newClient "{ModuleName}.defaultConfig"
--     result <- Messages.create client $ Messages.CreateParams
--         { model = "model-name"
--         , messages = [Message "user" "Hello"]
--         }
--     case result of
--         Left err -> print err
--         Right response -> print response
-- @
--
-- = Configuration
--
-- Configure via environment or explicit options:
--
-- @
-- config <- {ModuleName}.configFromEnv
-- -- or
-- let config = {ModuleName}.defaultConfig
--         { apiKey = Just "your-api-key"
--         , baseUrl = "https://api.example.com"
--         }
-- @
module {ModuleName}
    ( -- * Client
      Client
    , newClient
    , newClientWith

      -- * Configuration
    , Config(..)
    , defaultConfig
    , configFromEnv

      -- * Re-exports
    , module {ModuleName}.Types
    , module {ModuleName}.Errors
    ) where

import {ModuleName}.Client (Client, newClient, newClientWith)
import {ModuleName}.Config (Config(..), defaultConfig, configFromEnv)
import {ModuleName}.Types
import {ModuleName}.Errors
```

#### 2. Config (`src/{ModuleName}/Config.hs`)

Configuration types with sensible defaults:

```haskell
{-# LANGUAGE OverloadedStrings #-}
{-# LANGUAGE RecordWildCards #-}

-- | Configuration for the {ServiceName} client.
module {ModuleName}.Config
    ( Config(..)
    , AuthMode(..)
    , defaultConfig
    , configFromEnv
    ) where

import Data.Text (Text)
import qualified Data.Text as T
import System.Environment (lookupEnv)

-- | Authentication mode for API requests.
data AuthMode
    = BearerAuth     -- ^ Bearer token in Authorization header
    | BasicAuth      -- ^ Basic authentication
    | HeaderAuth     -- ^ API key in X-Api-Key header
    | NoAuth         -- ^ No authentication
    deriving (Show, Eq)

-- | Client configuration.
data Config = Config
    { apiKey      :: !(Maybe Text)
      -- ^ API key for authentication
    , baseUrl     :: !Text
      -- ^ Base URL for API requests
    , timeout     :: !Int
      -- ^ Request timeout in seconds (default: 60)
    , maxRetries  :: !Int
      -- ^ Maximum retry attempts for transient errors (default: 2)
    , authMode    :: !AuthMode
      -- ^ Authentication mode (default: BearerAuth)
    , headers     :: ![(Text, Text)]
      -- ^ Additional headers to include in all requests
    } deriving (Show, Eq)

-- | Default configuration.
--
-- @
-- defaultConfig = Config
--     { apiKey = Nothing
--     , baseUrl = "{default_base_url}"
--     , timeout = 60
--     , maxRetries = 2
--     , authMode = BearerAuth
--     , headers = []
--     }
-- @
defaultConfig :: Config
defaultConfig = Config
    { apiKey = Nothing
    , baseUrl = "{default_base_url}"
    , timeout = 60
    , maxRetries = 2
    , authMode = BearerAuth
    , headers = [{default_headers}]
    }

-- | Load configuration from environment variables.
--
-- Reads:
--
-- * @{ENV_PREFIX}_API_KEY@ - API key
-- * @{ENV_PREFIX}_BASE_URL@ - Base URL (optional)
-- * @{ENV_PREFIX}_TIMEOUT@ - Timeout in seconds (optional)
configFromEnv :: IO Config
configFromEnv = do
    mApiKey <- fmap T.pack <$> lookupEnv "{ENV_PREFIX}_API_KEY"
    mBaseUrl <- fmap T.pack <$> lookupEnv "{ENV_PREFIX}_BASE_URL"
    mTimeout <- fmap read <$> lookupEnv "{ENV_PREFIX}_TIMEOUT"
    pure $ defaultConfig
        { apiKey = mApiKey
        , baseUrl = maybe (baseUrl defaultConfig) id mBaseUrl
        , timeout = maybe (timeout defaultConfig) id mTimeout
        }
```

#### 3. Client (`src/{ModuleName}/Client.hs`)

The HTTP client implementation:

```haskell
{-# LANGUAGE OverloadedStrings #-}
{-# LANGUAGE RecordWildCards #-}
{-# LANGUAGE ScopedTypeVariables #-}

-- | HTTP client for {ServiceName} API.
module {ModuleName}.Client
    ( Client(..)
    , newClient
    , newClientWith
    , request
    , requestJSON
    , stream
    ) where

import Control.Exception (try, SomeException)
import Control.Monad.IO.Class (MonadIO, liftIO)
import Data.Aeson (FromJSON, ToJSON, eitherDecode, encode)
import qualified Data.ByteString.Lazy as LBS
import Data.Text (Text)
import qualified Data.Text as T
import qualified Data.Text.Encoding as TE
import Network.HTTP.Client
    ( Manager, newManager, Request(..), Response(..)
    , httpLbs, parseRequest_, setQueryString
    , RequestBody(..), responseStatus, responseBody
    )
import Network.HTTP.Client.TLS (tlsManagerSettings)
import Network.HTTP.Types (Status(..), hContentType, hAccept, hAuthorization)
import qualified Network.HTTP.Types as HTTP

import {ModuleName}.Config
import {ModuleName}.Errors

-- | API client handle.
data Client = Client
    { clientConfig  :: !Config
    , clientManager :: !Manager
    }

-- | Create a new client with default configuration.
--
-- @
-- client <- newClient
-- @
newClient :: IO Client
newClient = newClientWith defaultConfig

-- | Create a new client with custom configuration.
--
-- @
-- client <- newClientWith $ defaultConfig { apiKey = Just "sk-..." }
-- @
newClientWith :: Config -> IO Client
newClientWith config = do
    manager <- newManager tlsManagerSettings
    pure $ Client config manager

-- | Perform an HTTP request.
request
    :: Client
    -> HTTP.Method           -- ^ HTTP method
    -> Text                  -- ^ Path
    -> Maybe LBS.ByteString  -- ^ Request body
    -> [(Text, Text)]        -- ^ Extra headers
    -> IO (Either SDKError LBS.ByteString)
request Client{..} method path mBody extraHeaders = do
    let Config{..} = clientConfig
        url = T.unpack baseUrl <> T.unpack path
        baseReq = parseRequest_ url
        authHeaders = buildAuthHeaders clientConfig
        allHeaders =
            [ (hContentType, "application/json")
            , (hAccept, "application/json")
            ]
            ++ authHeaders
            ++ [(TE.encodeUtf8 k, TE.encodeUtf8 v) | (k, v) <- headers]
            ++ [(TE.encodeUtf8 k, TE.encodeUtf8 v) | (k, v) <- extraHeaders]
        req = baseReq
            { method = method
            , requestHeaders = allHeaders
            , requestBody = maybe mempty RequestBodyLBS mBody
            }

    result <- try $ httpLbs req clientManager
    case result of
        Left (e :: SomeException) ->
            pure $ Left $ ConnectionError $ T.pack $ show e
        Right response ->
            let status = responseStatus response
                body = responseBody response
            in if statusCode status >= 200 && statusCode status < 300
                then pure $ Right body
                else pure $ Left $ fromResponse status body

-- | Perform an HTTP request and decode JSON response.
requestJSON
    :: (ToJSON req, FromJSON resp)
    => Client
    -> HTTP.Method
    -> Text
    -> Maybe req
    -> IO (Either SDKError resp)
requestJSON client method path mBody = do
    let body = encode <$> mBody
    result <- request client method path body []
    pure $ result >>= \bs ->
        case eitherDecode bs of
            Left err -> Left $ ParseError $ T.pack err
            Right v  -> Right v

-- | Perform a streaming SSE request.
--
-- Returns a conduit source of parsed events.
stream
    :: (ToJSON req)
    => Client
    -> HTTP.Method
    -> Text
    -> req
    -> IO (Either SDKError (ConduitT () Event IO ()))
stream client method path body = do
    -- Implementation uses http-conduit for streaming
    -- See Streaming.hs for full implementation
    undefined

-- | Build authentication headers.
buildAuthHeaders :: Config -> [(HTTP.HeaderName, BS.ByteString)]
buildAuthHeaders Config{..} = case (authMode, apiKey) of
    (BearerAuth, Just key) ->
        [(hAuthorization, "Bearer " <> TE.encodeUtf8 key)]
    (BasicAuth, Just key) ->
        [(hAuthorization, "Basic " <> B64.encode (TE.encodeUtf8 key))]
    (HeaderAuth, Just key) ->
        [("X-Api-Key", TE.encodeUtf8 key)]
    _ -> []
```

#### 4. Types (`src/{ModuleName}/Types.hs`)

Generated type definitions with Aeson instances:

```haskell
{-# LANGUAGE DeriveGeneric #-}
{-# LANGUAGE DuplicateRecordFields #-}
{-# LANGUAGE OverloadedStrings #-}
{-# LANGUAGE RecordWildCards #-}

-- | Type definitions for {ServiceName} API.
--
-- All types derive Generic and have Aeson instances for JSON serialization.
module {ModuleName}.Types
    ( -- * Request/Response Types
      {TypeExports}

      -- * Enums
      {EnumExports}

      -- * Unions
      {UnionExports}
    ) where

import Data.Aeson
    ( FromJSON(..), ToJSON(..)
    , genericParseJSON, genericToJSON
    , defaultOptions, fieldLabelModifier
    , constructorTagModifier, sumEncoding
    , SumEncoding(..), Options(..)
    , object, withObject, (.:), (.:?), (.=)
    )
import Data.Aeson.Types (Parser)
import Data.Text (Text)
import qualified Data.Text as T
import Data.Time (UTCTime)
import GHC.Generics (Generic)

-- | {Type description}
data {TypeName} = {TypeName}
    { {fieldName} :: !{FieldType}
      -- ^ {Field description}
    , {optionalFieldName} :: !(Maybe {FieldType})
      -- ^ {Field description} (optional)
    } deriving (Show, Eq, Generic)

instance FromJSON {TypeName} where
    parseJSON = genericParseJSON jsonOptions

instance ToJSON {TypeName} where
    toJSON = genericToJSON jsonOptions

-- | Enum type for {description}
data {EnumName}
    = {EnumValue1}  -- ^ {Value1 description}
    | {EnumValue2}  -- ^ {Value2 description}
    deriving (Show, Eq, Ord, Bounded, Enum, Generic)

instance FromJSON {EnumName} where
    parseJSON = genericParseJSON enumOptions

instance ToJSON {EnumName} where
    toJSON = genericToJSON enumOptions

-- | Union type for {description}
--
-- Pattern match to access variants:
--
-- @
-- case block of
--     TextBlockVariant tb -> handleText tb
--     ImageBlockVariant ib -> handleImage ib
--     ToolUseBlockVariant tub -> handleToolUse tub
-- @
data {UnionName}
    = {Variant1}Variant !{Variant1Type}
    | {Variant2}Variant !{Variant2Type}
    deriving (Show, Eq, Generic)

instance FromJSON {UnionName} where
    parseJSON = withObject "{UnionName}" $ \o -> do
        tag <- o .: "{tagField}" :: Parser Text
        case tag of
            "{variant1Tag}" -> {Variant1}Variant <$> parseJSON (Object o)
            "{variant2Tag}" -> {Variant2}Variant <$> parseJSON (Object o)
            _ -> fail $ "Unknown {UnionName} type: " <> T.unpack tag

instance ToJSON {UnionName} where
    toJSON ({Variant1}Variant v) = toJSON v
    toJSON ({Variant2}Variant v) = toJSON v

-- JSON options for snake_case field names
jsonOptions :: Options
jsonOptions = defaultOptions
    { fieldLabelModifier = camelToSnake
    , omitNothingFields = True
    }

enumOptions :: Options
enumOptions = defaultOptions
    { constructorTagModifier = camelToSnake
    }

camelToSnake :: String -> String
camelToSnake = concatMap go
  where
    go c
        | c >= 'A' && c <= 'Z' = ['_', toLower c]
        | otherwise = [c]
```

#### 5. Resource Modules (`src/{ModuleName}/Resources/{Resource}.hs`)

Resource operations:

```haskell
{-# LANGUAGE OverloadedStrings #-}
{-# LANGUAGE RecordWildCards #-}

-- | Operations for {resource}.
--
-- {Resource description}
module {ModuleName}.Resources.{Resource}
    ( -- * Operations
      create
    , create_

      -- * Request Types
    , CreateParams(..)
    , defaultCreateParams

      -- * Streaming
    , createStream
    ) where

import Data.Text (Text)
import {ModuleName}.Client (Client, requestJSON, stream)
import {ModuleName}.Errors (SDKError)
import {ModuleName}.Types
import {ModuleName}.Streaming (Event)
import Conduit (ConduitT)

-- | Parameters for creating a {resource}.
data CreateParams = CreateParams
    { model    :: !Text
      -- ^ The model to use
    , messages :: ![Message]
      -- ^ The messages to send
    , maxTokens :: !(Maybe Int)
      -- ^ Maximum tokens to generate
    , temperature :: !(Maybe Double)
      -- ^ Sampling temperature
    } deriving (Show, Eq)

-- | Default parameters (requires model and messages to be set).
defaultCreateParams :: CreateParams
defaultCreateParams = CreateParams
    { model = ""
    , messages = []
    , maxTokens = Nothing
    , temperature = Nothing
    }

-- | Create a new {resource}.
--
-- @
-- result <- {Resource}.create client $ CreateParams
--     { model = "model-name"
--     , messages = [Message "user" "Hello"]
--     , maxTokens = Just 1024
--     , temperature = Nothing
--     }
-- case result of
--     Left err -> handleError err
--     Right response -> handleResponse response
-- @
create :: Client -> CreateParams -> IO (Either SDKError {OutputType})
create client params =
    requestJSON client "POST" "/v1/{resource}" (Just params)

-- | Create a new {resource}, throwing on error.
--
-- @
-- response <- {Resource}.create_ client params
-- @
create_ :: Client -> CreateParams -> IO {OutputType}
create_ client params = do
    result <- create client params
    case result of
        Left err -> throwIO $ toException err
        Right v  -> pure v

-- | Stream a {resource} response.
--
-- @
-- result <- {Resource}.createStream client params
-- case result of
--     Left err -> handleError err
--     Right source -> runConduit $ source .| mapM_C handleEvent
-- @
createStream
    :: Client
    -> CreateParams
    -> IO (Either SDKError (ConduitT () {StreamEventType} IO ()))
createStream client params =
    stream client "POST" "/v1/{resource}" (params { stream = True })
```

#### 6. Streaming (`src/{ModuleName}/Streaming.hs`)

SSE streaming implementation:

```haskell
{-# LANGUAGE OverloadedStrings #-}
{-# LANGUAGE ScopedTypeVariables #-}

-- | Server-Sent Events (SSE) streaming support.
module {ModuleName}.Streaming
    ( -- * Types
      Event(..)
    , StreamState(..)

      -- * Parsing
    , parseSSE
    , sseConduit

      -- * Utilities
    , collectText
    ) where

import Conduit
import Control.Monad (void)
import Data.Aeson (FromJSON, eitherDecode)
import qualified Data.ByteString as BS
import qualified Data.ByteString.Lazy as LBS
import Data.Text (Text)
import qualified Data.Text as T
import qualified Data.Text.Encoding as TE

-- | A parsed SSE event.
data Event a
    = DataEvent !a           -- ^ A data event with parsed payload
    | DoneEvent              -- ^ Stream completion marker
    | HeartbeatEvent         -- ^ Keep-alive ping
    deriving (Show, Eq, Functor)

-- | Internal stream parsing state.
data StreamState = StreamState
    { buffer   :: !BS.ByteString
    , eventId  :: !(Maybe Text)
    , retry    :: !(Maybe Int)
    } deriving (Show, Eq)

initialState :: StreamState
initialState = StreamState BS.empty Nothing Nothing

-- | Conduit that parses raw bytes into SSE events.
--
-- @
-- runConduit $ sourceByteString rawData
--     .| sseConduit
--     .| mapM_C handleEvent
-- @
sseConduit :: (MonadIO m, FromJSON a) => ConduitT BS.ByteString (Event a) m ()
sseConduit = loop initialState
  where
    loop state = do
        mChunk <- await
        case mChunk of
            Nothing -> pure ()
            Just chunk -> do
                let (events, newState) = parseChunk (buffer state <> chunk)
                mapM_ yield events
                loop newState { buffer = buffer newState }

-- | Parse accumulated buffer into events.
parseChunk :: FromJSON a => BS.ByteString -> ([Event a], StreamState)
parseChunk buf = go buf []
  where
    go bs acc =
        case BS.breakSubstring "\n\n" bs of
            (_, rest) | BS.null rest -> (reverse acc, StreamState bs Nothing Nothing)
            (event, rest) ->
                let parsed = parseSSE event
                    remaining = BS.drop 2 rest  -- Drop "\n\n"
                in go remaining (parsed : acc)

-- | Parse a single SSE event.
--
-- @
-- parseSSE "data: {\"text\": \"hello\"}"
-- -- => DataEvent (Object ...)
--
-- parseSSE "data: [DONE]"
-- -- => DoneEvent
-- @
parseSSE :: FromJSON a => BS.ByteString -> Event a
parseSSE bs =
    let lines' = BS.split (fromIntegral (fromEnum '\n')) bs
        dataLines = [d | line <- lines'
                       , Just d <- [BS.stripPrefix "data:" line]
                       , not (BS.null d)]
        dataContent = BS.intercalate "\n" $ map BS.strip dataLines
    in case dataContent of
        "" -> HeartbeatEvent
        "[DONE]" -> DoneEvent
        _ -> case eitherDecode (LBS.fromStrict dataContent) of
            Left _err -> HeartbeatEvent  -- Skip malformed data
            Right v -> DataEvent v

-- | Collect all text deltas from a stream.
--
-- @
-- texts <- runConduit $ eventSource .| collectText
-- let fullText = T.concat texts
-- @
collectText :: Monad m => ConduitT (Event {StreamEventType}) Text m [Text]
collectText = loop []
  where
    loop acc = do
        mEvent <- await
        case mEvent of
            Nothing -> pure (reverse acc)
            Just DoneEvent -> pure (reverse acc)
            Just HeartbeatEvent -> loop acc
            Just (DataEvent e) ->
                case extractDeltaText e of
                    Nothing -> loop acc
                    Just t -> loop (t : acc)

    extractDeltaText :: {StreamEventType} -> Maybe Text
    extractDeltaText e = delta e >>= deltaText
```

#### 7. Errors (`src/{ModuleName}/Errors.hs`)

Error types and handling:

```haskell
{-# LANGUAGE DeriveGeneric #-}
{-# LANGUAGE OverloadedStrings #-}

-- | Error types for {ServiceName} API.
module {ModuleName}.Errors
    ( -- * Error Types
      SDKError(..)
    , APIError(..)
    , ErrorType(..)

      -- * Error Construction
    , fromResponse
    , fromException

      -- * Error Inspection
    , isRetryable
    , errorMessage
    , errorStatus
    ) where

import Control.Exception (Exception, SomeException)
import Data.Aeson (FromJSON(..), withObject, (.:), (.:?))
import qualified Data.ByteString.Lazy as LBS
import Data.Text (Text)
import qualified Data.Text as T
import GHC.Generics (Generic)
import Network.HTTP.Types (Status(..))

-- | Error type classification.
data ErrorType
    = BadRequest           -- ^ HTTP 400
    | Unauthorized         -- ^ HTTP 401
    | Forbidden            -- ^ HTTP 403
    | NotFound             -- ^ HTTP 404
    | UnprocessableEntity  -- ^ HTTP 422
    | RateLimited          -- ^ HTTP 429
    | ServerError          -- ^ HTTP 5xx
    | NetworkError         -- ^ Connection/network errors
    | ParseError'          -- ^ JSON parsing errors
    | TimeoutError         -- ^ Request timeout
    | StreamError          -- ^ SSE parsing errors
    deriving (Show, Eq, Ord, Bounded, Enum, Generic)

-- | API error details from server response.
data APIError = APIError
    { apiErrorType    :: !Text
    , apiErrorMessage :: !Text
    , apiErrorParam   :: !(Maybe Text)
    , apiErrorCode    :: !(Maybe Text)
    } deriving (Show, Eq, Generic)

instance FromJSON APIError where
    parseJSON = withObject "APIError" $ \o -> do
        err <- o .: "error"
        APIError
            <$> err .: "type"
            <*> err .: "message"
            <*> err .:? "param"
            <*> err .:? "code"

-- | SDK error type.
--
-- Pattern match to handle different error cases:
--
-- @
-- case err of
--     APIError' status apiErr ->
--         putStrLn $ "API error: " <> apiErrorMessage apiErr
--     ConnectionError msg ->
--         putStrLn $ "Network error: " <> msg
--     RateLimitError retryAfter _ ->
--         threadDelay (retryAfter * 1000000)
--     _ -> putStrLn $ "Error: " <> errorMessage err
-- @
data SDKError
    = APIError' !Int !APIError
      -- ^ API returned an error response
    | ConnectionError !Text
      -- ^ Network/connection error
    | TimeoutError' !Text
      -- ^ Request timed out
    | ParseError !Text
      -- ^ Failed to parse response
    | RateLimitError !Int !(Maybe APIError)
      -- ^ Rate limited (retry after N seconds)
    | StreamError' !Text
      -- ^ Error in SSE stream
    deriving (Show, Eq, Generic)

instance Exception SDKError

-- | Create an error from an HTTP response.
fromResponse :: Status -> LBS.ByteString -> SDKError
fromResponse status body =
    let code = statusCode status
        apiErr = decodeAPIError body
    in case code of
        400 -> APIError' code (fromMaybe defaultErr apiErr)
        401 -> APIError' code (fromMaybe defaultErr apiErr)
        403 -> APIError' code (fromMaybe defaultErr apiErr)
        404 -> APIError' code (fromMaybe defaultErr apiErr)
        422 -> APIError' code (fromMaybe defaultErr apiErr)
        429 -> RateLimitError (extractRetryAfter body) apiErr
        _ | code >= 500 -> APIError' code (fromMaybe serverErr apiErr)
          | otherwise -> APIError' code (fromMaybe defaultErr apiErr)
  where
    defaultErr = APIError "unknown_error" "Unknown error" Nothing Nothing
    serverErr = APIError "server_error" "Internal server error" Nothing Nothing

    decodeAPIError :: LBS.ByteString -> Maybe APIError
    decodeAPIError = decode

    extractRetryAfter :: LBS.ByteString -> Int
    extractRetryAfter _ = 60  -- Default retry after 60s

-- | Create an error from an exception.
fromException :: SomeException -> SDKError
fromException e = ConnectionError $ T.pack $ show e

-- | Check if an error is retryable.
--
-- @
-- when (isRetryable err) $ do
--     threadDelay 1000000
--     retry
-- @
isRetryable :: SDKError -> Bool
isRetryable (RateLimitError _ _) = True
isRetryable (ConnectionError _) = True
isRetryable (TimeoutError' _) = True
isRetryable (APIError' code _) = code >= 500
isRetryable _ = False

-- | Get human-readable error message.
errorMessage :: SDKError -> Text
errorMessage (APIError' _ err) = apiErrorMessage err
errorMessage (ConnectionError msg) = msg
errorMessage (TimeoutError' msg) = msg
errorMessage (ParseError msg) = msg
errorMessage (RateLimitError _ (Just err)) = apiErrorMessage err
errorMessage (RateLimitError secs Nothing) =
    "Rate limited. Retry after " <> T.pack (show secs) <> " seconds"
errorMessage (StreamError' msg) = msg

-- | Get HTTP status code if applicable.
errorStatus :: SDKError -> Maybe Int
errorStatus (APIError' code _) = Just code
errorStatus (RateLimitError _ _) = Just 429
errorStatus _ = Nothing
```

## Type Mapping

### Primitive Types

| Contract Type     | Haskell Type         | Notes                          |
|-------------------|----------------------|--------------------------------|
| `string`          | `Text`               | From Data.Text                 |
| `bool`, `boolean` | `Bool`               |                                |
| `int`             | `Int`                |                                |
| `int8`            | `Int8`               | From Data.Int                  |
| `int16`           | `Int16`              | From Data.Int                  |
| `int32`           | `Int32`              | From Data.Int                  |
| `int64`           | `Int64`              | From Data.Int                  |
| `uint`            | `Word`               |                                |
| `uint8`           | `Word8`              | From Data.Word                 |
| `uint16`          | `Word16`             | From Data.Word                 |
| `uint32`          | `Word32`             | From Data.Word                 |
| `uint64`          | `Word64`             | From Data.Word                 |
| `float32`         | `Float`              |                                |
| `float64`         | `Double`             |                                |
| `time.Time`       | `UTCTime`            | From Data.Time                 |
| `json.RawMessage` | `Value`              | From Data.Aeson                |
| `any`             | `Value`              | From Data.Aeson                |

### Collection Types

| Contract Type      | Haskell Type         | Notes                          |
|--------------------|----------------------|--------------------------------|
| `[]T`              | `[T]`                | List                           |
| `map[string]T`     | `Map Text T`         | From Data.Map                  |

### Optional/Nullable

| Contract         | Haskell              | Notes                          |
|------------------|----------------------|--------------------------------|
| `optional: T`    | `Maybe T`            |                                |
| `nullable: T`    | `Maybe T`            |                                |

### Struct to Record

Contract structs map to Haskell record types:

```haskell
-- From contract type:
-- {Name: "Message", Fields: [{role, string}, {content, string}]}

data Message = Message
    { messageRole    :: !Text
    , messageContent :: !Text
    } deriving (Show, Eq, Generic)

instance FromJSON Message where
    parseJSON = withObject "Message" $ \o ->
        Message
            <$> o .: "role"
            <*> o .: "content"

instance ToJSON Message where
    toJSON Message{..} = object
        [ "role" .= messageRole
        , "content" .= messageContent
        ]
```

### Enum Types

Enums map to Haskell sum types:

```haskell
-- Role = "user" | "assistant" | "system"

data Role
    = RoleUser
    | RoleAssistant
    | RoleSystem
    deriving (Show, Eq, Ord, Bounded, Enum, Generic)

instance FromJSON Role where
    parseJSON = withText "Role" $ \case
        "user" -> pure RoleUser
        "assistant" -> pure RoleAssistant
        "system" -> pure RoleSystem
        other -> fail $ "Unknown Role: " <> T.unpack other

instance ToJSON Role where
    toJSON RoleUser = "user"
    toJSON RoleAssistant = "assistant"
    toJSON RoleSystem = "system"

-- Helper functions
roleToText :: Role -> Text
roleToText RoleUser = "user"
roleToText RoleAssistant = "assistant"
roleToText RoleSystem = "system"
```

### Discriminated Unions

Union types use Haskell sum types with smart constructors:

```haskell
-- ContentBlock = TextBlock | ImageBlock | ToolUseBlock

data ContentBlock
    = ContentBlockText !TextBlock
    | ContentBlockImage !ImageBlock
    | ContentBlockToolUse !ToolUseBlock
    deriving (Show, Eq, Generic)

instance FromJSON ContentBlock where
    parseJSON = withObject "ContentBlock" $ \o -> do
        tag <- o .: "type" :: Parser Text
        case tag of
            "text" -> ContentBlockText <$> parseJSON (Object o)
            "image" -> ContentBlockImage <$> parseJSON (Object o)
            "tool_use" -> ContentBlockToolUse <$> parseJSON (Object o)
            other -> fail $ "Unknown ContentBlock type: " <> T.unpack other

instance ToJSON ContentBlock where
    toJSON (ContentBlockText v) = toJSON v
    toJSON (ContentBlockImage v) = toJSON v
    toJSON (ContentBlockToolUse v) = toJSON v

-- Pattern matching usage:
processContent :: ContentBlock -> IO ()
processContent (ContentBlockText tb) = putStrLn $ "Text: " <> textBlockText tb
processContent (ContentBlockImage ib) = putStrLn $ "Image: " <> imageBlockUrl ib
processContent (ContentBlockToolUse tub) = putStrLn $ "Tool: " <> toolUseBlockName tub
```

## Naming Conventions

### Haskell Naming

| Contract       | Haskell                  | Notes                          |
|----------------|--------------------------|--------------------------------|
| `user-id`      | `userId`                 | camelCase for fields           |
| `user_name`    | `userName`               | camelCase for fields           |
| `UserData`     | `UserData`               | PascalCase for types           |
| `create`       | `create`                 | camelCase for functions        |
| `get-user`     | `getUser`                | camelCase for functions        |
| `TEXT`         | `TextValue`              | PascalCase for enum values     |

### Field Naming Strategy

To avoid Haskell's record field name conflicts, fields are prefixed:

```haskell
-- For type Message with field "role"
data Message = Message
    { messageRole :: !Text      -- Prefixed with type name
    , messageContent :: !Text
    }

-- JSON uses original field names via custom FromJSON/ToJSON
instance FromJSON Message where
    parseJSON = withObject "Message" $ \o ->
        Message
            <$> o .: "role"      -- Original JSON key
            <*> o .: "content"
```

Alternative: Use DuplicateRecordFields extension:

```haskell
{-# LANGUAGE DuplicateRecordFields #-}

data Message = Message
    { role :: !Text
    , content :: !Text
    }

data User = User
    { role :: !Text  -- Same field name, different type
    , name :: !Text
    }
```

### Reserved Words

Haskell reserved words are escaped by appending underscore:

- `type` -> `type_`
- `data` -> `data_`
- `class` -> `class_`
- `where` -> `where_`
- `let` -> `let_`
- `in` -> `in_`
- `do` -> `do_`
- `case` -> `case_`
- `of` -> `of_`
- `if` -> `if_`
- `then` -> `then_`
- `else` -> `else_`
- `module` -> `module_`
- `import` -> `import_`

## Code Generation

### Generator Structure

```go
package sdkhaskell

type Config struct {
    // PackageName is the Cabal package name.
    // Default: sanitized lowercase service name with hyphens.
    PackageName string

    // ModuleName is the root Haskell module name.
    // Default: PascalCase service name.
    ModuleName string

    // Version is the package version for .cabal file.
    Version string

    // Author is the package author.
    Author string

    // License is the package license (default: BSD-3-Clause).
    License string

    // Synopsis is a one-line package description.
    Synopsis string
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── package.cabal.tmpl        # Cabal package configuration
├── main.hs.tmpl              # Main module (re-exports)
├── client.hs.tmpl            # HTTP client
├── config.hs.tmpl            # Configuration types
├── types.hs.tmpl             # Type definitions
├── resource.hs.tmpl          # Resource modules (per resource)
├── streaming.hs.tmpl         # SSE streaming
└── errors.hs.tmpl            # Error types
```

### Generated Files

| File                              | Purpose                      |
|-----------------------------------|------------------------------|
| `{package}.cabal`                 | Cabal package configuration  |
| `src/{Module}.hs`                 | Main module (re-exports)     |
| `src/{Module}/Client.hs`          | HTTP client                  |
| `src/{Module}/Config.hs`          | Configuration types          |
| `src/{Module}/Types.hs`           | All type definitions         |
| `src/{Module}/Resources/{R}.hs`   | Resource modules             |
| `src/{Module}/Streaming.hs`       | SSE streaming utilities      |
| `src/{Module}/Errors.hs`          | Error types                  |

## Usage Examples

### Basic Usage

```haskell
-- Add to package.yaml or .cabal:
-- dependencies:
--   - my-service

{-# LANGUAGE OverloadedStrings #-}

import qualified MyService
import qualified MyService.Resources.Messages as Messages

main :: IO ()
main = do
    -- Create client
    client <- MyService.newClientWith $ MyService.defaultConfig
        { MyService.apiKey = Just "your-api-key"
        }

    -- Make a request
    result <- Messages.create client $ Messages.CreateParams
        { Messages.model = "model-name"
        , Messages.messages = [MyService.Message "user" "Hello"]
        , Messages.maxTokens = Just 1024
        , Messages.temperature = Nothing
        }

    case result of
        Left err -> putStrLn $ "Error: " <> show err
        Right response -> print response
```

### Streaming

```haskell
import Conduit
import qualified MyService
import qualified MyService.Resources.Messages as Messages
import qualified MyService.Streaming as Streaming

streamExample :: IO ()
streamExample = do
    client <- MyService.newClientWith $ MyService.defaultConfig
        { MyService.apiKey = Just "your-api-key"
        }

    let params = Messages.CreateParams
            { Messages.model = "model-name"
            , Messages.messages = [MyService.Message "user" "Tell me a story"]
            , Messages.maxTokens = Just 2048
            , Messages.temperature = Nothing
            }

    result <- Messages.createStream client params

    case result of
        Left err -> putStrLn $ "Error: " <> show err
        Right source -> runConduit $ source .| mapM_C handleEvent
  where
    handleEvent (Streaming.DataEvent e) =
        case MyService.delta e of
            Just d -> T.putStr (MyService.deltaText d)
            Nothing -> pure ()
    handleEvent Streaming.DoneEvent = putStrLn "\n[Done]"
    handleEvent Streaming.HeartbeatEvent = pure ()
```

### Error Handling

```haskell
import Control.Exception (catch)
import qualified MyService
import qualified MyService.Errors as Errors
import qualified MyService.Resources.Messages as Messages

handleErrors :: IO ()
handleErrors = do
    client <- MyService.newClient

    result <- Messages.create client params

    case result of
        Right response ->
            print response

        Left (Errors.RateLimitError retryAfter _) -> do
            putStrLn $ "Rate limited, retrying in " <> show retryAfter <> "s"
            threadDelay (retryAfter * 1000000)
            -- Retry...

        Left (Errors.APIError' 401 _) ->
            putStrLn "Invalid API key"

        Left err | Errors.isRetryable err -> do
            putStrLn $ "Retryable error: " <> T.unpack (Errors.errorMessage err)
            -- Retry with backoff...

        Left err ->
            putStrLn $ "Error: " <> T.unpack (Errors.errorMessage err)
```

### Configuration from Environment

```haskell
import qualified MyService

envConfig :: IO ()
envConfig = do
    -- Reads MY_SERVICE_API_KEY, MY_SERVICE_BASE_URL, etc.
    config <- MyService.configFromEnv

    client <- MyService.newClientWith config
    -- Use client...
```

### Custom HTTP Manager

```haskell
import Network.HTTP.Client (newManager, managerSetProxy, proxyEnvironment)
import Network.HTTP.Client.TLS (tlsManagerSettings)
import qualified MyService

customManager :: IO ()
customManager = do
    -- Create manager with proxy support
    manager <- newManager $ managerSetProxy proxyEnvironment tlsManagerSettings

    let client = MyService.Client
            { MyService.clientConfig = MyService.defaultConfig
                { MyService.apiKey = Just "sk-..."
                }
            , MyService.clientManager = manager
            }

    -- Use client...
```

### With Retry

```haskell
import Control.Retry
import qualified MyService.Errors as Errors

withRetry :: IO (Either Errors.SDKError a) -> IO (Either Errors.SDKError a)
withRetry action = retrying policy shouldRetry (const action)
  where
    policy = exponentialBackoff 1000000 <> limitRetries 3

    shouldRetry _ (Left err) = pure $ Errors.isRetryable err
    shouldRetry _ (Right _) = pure False
```

## Platform Support

### Dependencies

**Runtime Dependencies:**
- `aeson` (>= 2.0) - JSON serialization
- `http-conduit` (>= 2.3) - HTTP client with streaming
- `http-types` (>= 0.12) - HTTP types
- `conduit` (>= 1.3) - Streaming with resource safety
- `text` (>= 1.2) - Text handling
- `bytestring` (>= 0.10) - Byte strings
- `containers` (>= 0.6) - Maps and sets
- `time` (>= 1.9) - Date/time handling

**Development Dependencies:**
- `hspec` (>= 2.10) - Testing framework
- `QuickCheck` (>= 2.14) - Property-based testing

### Minimum Versions

| Platform  | Minimum Version | Rationale                        |
|-----------|-----------------|----------------------------------|
| GHC       | 9.2             | Modern extensions, stable ABI    |
| Cabal     | 3.6             | Modern build system              |
| Stack     | 2.9             | Resolver support                 |

### GHC Extensions

Required extensions (enabled in .cabal):

```cabal
default-extensions:
    OverloadedStrings
    DeriveGeneric
    RecordWildCards
    ScopedTypeVariables
    LambdaCase
    TypeApplications
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidHaskell_Syntax(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_NamingConventions(t *testing.T)
func TestGenerate_ReservedWords(t *testing.T)
func TestGenerate_UnionTypes(t *testing.T)
func TestGenerate_EnumTypes(t *testing.T)
```

### Generated SDK Tests

```haskell
-- test/Spec.hs
{-# LANGUAGE OverloadedStrings #-}

import Test.Hspec
import qualified MyService
import qualified MyService.Types as Types

main :: IO ()
main = hspec $ do
    describe "Types" $ do
        it "parses Message from JSON" $ do
            let json = "{\"role\":\"user\",\"content\":\"Hello\"}"
            decode json `shouldBe` Just (Types.Message "user" "Hello")

        it "encodes Message to JSON" $ do
            let msg = Types.Message "user" "Hello"
            encode msg `shouldBe` "{\"role\":\"user\",\"content\":\"Hello\"}"

    describe "Config" $ do
        it "has sensible defaults" $ do
            MyService.baseUrl MyService.defaultConfig `shouldBe` "https://api.example.com"
            MyService.timeout MyService.defaultConfig `shouldBe` 60

    describe "Client" $ do
        it "creates client with config" $ do
            client <- MyService.newClient
            MyService.clientConfig client `shouldSatisfy` \_ -> True
```

## Future Enhancements

1. **Servant Integration**: Generate servant API types for type-safe client/server
2. **Lens Generation**: Generate lenses for record field access
3. **MTL Support**: Transformer-based effect handling
4. **Async Operations**: Support for async/await patterns
5. **WebSocket Streaming**: Support for WebSocket-based streaming
6. **Retry Policies**: Built-in configurable retry strategies
7. **Metrics/Observability**: Integration with prometheus-client
8. **OpenTelemetry**: Tracing support

## References

- [Haskell Style Guide](https://github.com/tibbe/haskell-style-guide)
- [Aeson Documentation](https://hackage.haskell.org/package/aeson)
- [http-conduit Documentation](https://hackage.haskell.org/package/http-conduit)
- [Conduit Documentation](https://hackage.haskell.org/package/conduit)
- [Cabal User Guide](https://cabal.readthedocs.io/en/stable/)
- [Haddock Documentation](https://haskell-haddock.readthedocs.io/)
