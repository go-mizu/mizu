# Clojure SDK Generator

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-01-01

## Overview

This document specifies the design for a production-ready Clojure SDK generator that produces idiomatic, well-documented Clojure code from Mizu service contracts.

## Design Goals

1. **Idiomatic Clojure** - Follow Clojure conventions (kebab-case, data-first, immutable)
2. **Best-in-class DX** - Simple API, excellent error messages, REPL-friendly
3. **Production Ready** - Retries, timeouts, connection pooling, proper resource management
4. **Modern Tooling** - deps.edn (not Leiningen), latest Clojure ecosystem
5. **Minimal Dependencies** - Only essential, well-maintained libraries

## Technology Choices

### Build System: deps.edn (tools.deps)

Modern Clojure projects use `deps.edn` over `project.clj`:
- Simpler, declarative configuration
- Built into Clojure CLI
- Better reproducibility with `:deps` aliases
- Native support in all major editors

### HTTP Client: clj-http

The de facto standard HTTP client for Clojure:
- Connection pooling built-in
- Async support via `clj-http.client/with-async-connection-pool`
- Middleware system for extensibility
- Mature, well-tested (10+ years)

### JSON: Cheshire

Fast, idiomatic JSON encoding/decoding:
- Automatic keyword conversion
- Custom encoders for dates
- Streaming support
- Jackson-based performance

### Streaming: core.async

Clojure's CSP-style concurrency library:
- Channels for SSE event streams
- Composable with transducers
- Backpressure support
- Well-integrated with Clojure ecosystem

### Validation: clojure.spec.alpha (Optional)

Runtime validation and documentation:
- Auto-generated specs for types
- Excellent error messages via expound
- Instrumentation for development
- Optional - doesn't affect runtime if not used

## File Structure

Generated SDK produces the following structure:

```
my-api-sdk/
├── deps.edn                     # Project configuration
├── src/
│   └── my_api/
│       ├── core.clj             # Client factory and config
│       ├── types.clj            # Record definitions and coercion
│       ├── resources.clj        # Resource method implementations
│       ├── streaming.clj        # SSE/streaming support
│       ├── errors.clj           # Exception hierarchy
│       └── spec.clj             # Optional: clojure.spec definitions
└── README.md                    # Usage documentation
```

## API Design

### Client Creation

```clojure
(ns example
  (:require [my-api.core :as api]))

;; Simple creation with API key
(def client (api/create-client {:api-key "sk-xxx"}))

;; Full configuration
(def client
  (api/create-client
    {:api-key "sk-xxx"
     :base-url "https://api.example.com"
     :timeout-ms 30000
     :max-retries 3
     :default-headers {"X-Custom" "value"}}))

;; Modify client (returns new client)
(def dev-client (api/with-base-url client "http://localhost:8080"))
```

### Making Requests

```clojure
;; Synchronous call (returns data or throws)
(api/messages-create client {:model "gpt-4" :content "Hello"})
;; => {:id "msg-123" :content [{:type "text" :text "Hi!"}]}

;; With options (timeout override, etc.)
(api/messages-create client
  {:model "gpt-4" :content "Hello"}
  {:timeout-ms 60000})
```

### Streaming

```clojure
(require '[clojure.core.async :as async])

;; Returns a channel of events
(let [events-ch (api/messages-stream client {:model "gpt-4" :content "Hello"})]
  (async/go-loop []
    (when-let [event (async/<! events-ch)]
      (case (:type event)
        :data (println "Received:" (:data event))
        :error (println "Error:" (:error event))
        :done (println "Stream complete"))
      (recur))))

;; Or collect all events (blocking)
(api/messages-stream-collect client {:model "gpt-4" :content "Hello"})
;; => [{:type "text" :text "Hi"} {:type "text" :text "!"}]
```

### Error Handling

```clojure
(try
  (api/messages-create client {:model "invalid"})
  (catch clojure.lang.ExceptionInfo e
    (let [{:keys [type status body]} (ex-data e)]
      (case type
        :api-error (println "API returned error:" status body)
        :timeout (println "Request timed out")
        :connection-error (println "Connection failed")))))
```

### Types

Types are represented as plain Clojure maps with optional coercion:

```clojure
;; Raw map - always works
{:id "msg-123" :role "user" :content "Hello"}

;; With coercion (validates and converts)
(types/->message {:id "msg-123" :role "user" :content "Hello"})

;; Records for performance-critical paths
(types/map->Message {:id "msg-123" :role "user" :content "Hello"})
```

## Type Mapping

| Contract Type | Clojure Type | Notes |
|--------------|--------------|-------|
| `string` | `String` | Java interop |
| `bool` | `Boolean` | |
| `int32` | `Integer` | |
| `int64` | `Long` | Default for integers |
| `float32` | `Float` | |
| `float64` | `Double` | |
| `time.Time` | `java.time.Instant` | ISO-8601 format |
| `json.RawMessage` | `any` | Passthrough |
| `[]T` | `vector` / `(vector-of T)` | |
| `map[string]T` | `map` / `(map-of keyword? T)` | Keywords by default |
| Optional | `nil` or value | Clojure idiom |
| Union | Tagged map `{:type "x" ...}` | Discriminated |

## Naming Conventions

| Contract | Clojure |
|----------|---------|
| Service `Anthropic` | Namespace `anthropic.core` |
| Resource `messages` | Functions `messages-*` |
| Method `create` | Function `messages-create` |
| Type `CreateRequest` | Record `CreateRequest`, coercer `->create-request` |
| Field `userId` | Key `:user-id` (kebab-case keyword) |
| Enum `ACTIVE` | Keyword `:active` (lowercase) |

## Generator Configuration

```go
type Config struct {
    // Namespace is the root Clojure namespace.
    // Default: kebab-case service name (e.g., "my-api")
    Namespace string

    // GroupId for deployment (Maven coordinates).
    // Default: "com.example"
    GroupId string

    // ArtifactId for deployment.
    // Default: kebab-case service name
    ArtifactId string

    // Version of the generated SDK.
    // Default: "0.0.0"
    Version string

    // GenerateSpecs enables clojure.spec generation.
    // Default: true
    GenerateSpecs bool
}
```

## Template Details

### deps.edn

```clojure
{:paths ["src"]
 :deps {org.clojure/clojure {:mvn/version "1.11.1"}
        org.clojure/core.async {:mvn/version "1.6.681"}
        clj-http {:mvn/version "3.12.3"}
        cheshire {:mvn/version "5.12.0"}}
 :aliases
 {:dev {:extra-deps {expound/expound {:mvn/version "0.9.0"}}}
  :test {:extra-deps {org.clojure/test.check {:mvn/version "1.1.1"}}
         :extra-paths ["test"]}}}
```

### core.clj (Client)

Key features:
- Immutable client configuration
- Connection pooling via clj-http
- Automatic retry with exponential backoff
- Configurable timeouts
- Header management (auth, content-type)

```clojure
(ns my-api.core
  (:require [clj-http.client :as http]
            [cheshire.core :as json]
            [my-api.errors :as errors]))

(defrecord Client [config http-client])

(defn create-client
  "Creates a new API client."
  [{:keys [api-key base-url timeout-ms max-retries default-headers]
    :or {base-url "https://api.example.com"
         timeout-ms 60000
         max-retries 2
         default-headers {}}}]
  (->Client
    {:api-key api-key
     :base-url base-url
     :timeout-ms timeout-ms
     :max-retries max-retries
     :default-headers (merge default-headers
                              {"Content-Type" "application/json"
                               "Accept" "application/json"})}
    (http/build-http-client {})))

(defn- build-headers
  "Constructs request headers including auth."
  [{:keys [api-key default-headers]}]
  (cond-> default-headers
    api-key (assoc "Authorization" (str "Bearer " api-key))))

(defn request
  "Makes an HTTP request to the API."
  [client method path body & [{:keys [timeout-ms]}]]
  (let [{:keys [config http-client]} client
        {:keys [base-url max-retries]} config
        url (str base-url path)]
    (loop [attempts 0]
      (let [result (try
                     {:success true
                      :response (http/request
                                  {:method method
                                   :url url
                                   :headers (build-headers config)
                                   :body (when body (json/generate-string body))
                                   :socket-timeout (or timeout-ms (:timeout-ms config))
                                   :connection-timeout (or timeout-ms (:timeout-ms config))
                                   :as :json-string-keys
                                   :throw-exceptions false})}
                     (catch Exception e
                       {:success false :error e}))]
        (cond
          ;; Success
          (:success result)
          (let [{:keys [status body]} (:response result)]
            (if (< status 400)
              (json/parse-string body true)
              (throw (errors/api-error status body))))

          ;; Retryable error
          (and (< attempts max-retries)
               (errors/retryable? (:error result)))
          (do (Thread/sleep (* 500 (Math/pow 2 attempts)))
              (recur (inc attempts)))

          ;; Non-retryable error
          :else
          (throw (errors/connection-error (:error result))))))))
```

### types.clj (Type Definitions)

```clojure
(ns my-api.types
  (:require [cheshire.core :as json]
            [cheshire.generate :as json-gen])
  (:import [java.time Instant]))

;; Custom JSON encoder for Instant
(json-gen/add-encoder Instant
  (fn [d jsonGenerator]
    (.writeString jsonGenerator (.toString d))))

;; Records for types
(defrecord Message [id type role content model stop-reason usage])

;; Coercion functions
(defn ->message
  "Coerces a map to a Message, converting keys to keywords."
  [m]
  (map->Message
    (-> m
        (update :created-at #(when % (Instant/parse %))))))

;; Union type handling
(defmulti ->content-block :type)

(defmethod ->content-block "text" [m]
  (select-keys m [:type :text]))

(defmethod ->content-block "image" [m]
  (select-keys m [:type :source]))
```

### resources.clj (API Methods)

```clojure
(ns my-api.resources
  (:require [my-api.core :as core]
            [my-api.types :as types]))

;; messages resource
(defn messages-create
  "Creates a new message."
  ([client request]
   (messages-create client request {}))
  ([client request opts]
   (-> (core/request client :post "/v1/messages" request opts)
       types/->message)))

(defn messages-list
  "Lists messages."
  ([client]
   (messages-list client {} {}))
  ([client params]
   (messages-list client params {}))
  ([client params opts]
   (-> (core/request client :get "/v1/messages" nil opts)
       (update :data #(mapv types/->message %)))))
```

### streaming.clj (SSE Support)

```clojure
(ns my-api.streaming
  (:require [clojure.core.async :as async]
            [clojure.string :as str]
            [clj-http.client :as http]
            [cheshire.core :as json]
            [my-api.core :as core]))

(defn- parse-sse-line
  "Parses a single SSE line."
  [line]
  (cond
    (str/blank? line) {:type :dispatch}
    (str/starts-with? line ":") nil ; comment
    (str/starts-with? line "data:")
    {:type :data :value (str/trim (subs line 5))}
    (str/starts-with? line "event:")
    {:type :event :value (str/trim (subs line 6))}
    (str/starts-with? line "id:")
    {:type :id :value (str/trim (subs line 3))}
    :else nil))

(defn- sse-events
  "Returns a channel of parsed SSE events from an input stream."
  [input-stream]
  (let [out-ch (async/chan 100)]
    (async/thread
      (with-open [reader (clojure.java.io/reader input-stream)]
        (let [current-event (atom {})]
          (doseq [line (line-seq reader)]
            (when-let [parsed (parse-sse-line line)]
              (case (:type parsed)
                :dispatch
                (when (seq @current-event)
                  (async/>!! out-ch @current-event)
                  (reset! current-event {}))

                :data
                (let [data-str (:value parsed)]
                  (when (not= data-str "[DONE]")
                    (swap! current-event assoc
                           :data (json/parse-string data-str true))))

                :event
                (swap! current-event assoc :event (:value parsed))

                :id
                (swap! current-event assoc :id (:value parsed))

                nil)))))
        (async/close! out-ch)))
    out-ch))

(defn stream-request
  "Makes a streaming request, returns a channel of events."
  [client method path body]
  (let [{:keys [config]} client
        {:keys [base-url api-key default-headers timeout-ms]} config
        url (str base-url path)
        headers (merge default-headers
                       {"Accept" "text/event-stream"}
                       (when api-key
                         {"Authorization" (str "Bearer " api-key)}))]
    (let [response (http/request
                     {:method method
                      :url url
                      :headers headers
                      :body (when body (json/generate-string body))
                      :as :stream
                      :throw-exceptions false})]
      (if (< (:status response) 400)
        (sse-events (:body response))
        (let [ch (async/chan 1)]
          (async/>!! ch {:error {:status (:status response)
                                 :body (slurp (:body response))}})
          (async/close! ch)
          ch)))))

(defn collect-stream
  "Collects all events from a stream into a vector."
  [events-ch]
  (async/<!! (async/into [] events-ch)))
```

### errors.clj (Error Handling)

```clojure
(ns my-api.errors)

(defn api-error
  "Creates an API error exception."
  [status body]
  (ex-info (str "API error: " status)
           {:type :api-error
            :status status
            :body body}))

(defn connection-error
  "Creates a connection error exception."
  [cause]
  (ex-info "Connection error"
           {:type :connection-error
            :cause cause}
           cause))

(defn timeout-error
  "Creates a timeout error exception."
  []
  (ex-info "Request timed out"
           {:type :timeout}))

(defn decoding-error
  "Creates a JSON decoding error exception."
  [cause raw]
  (ex-info "Failed to decode response"
           {:type :decoding-error
            :raw raw
            :cause cause}
           cause))

(defn retryable?
  "Returns true if the error should be retried."
  [e]
  (or (instance? java.net.SocketTimeoutException e)
      (instance? java.net.ConnectException e)
      (and (instance? clojure.lang.ExceptionInfo e)
           (let [{:keys [type status]} (ex-data e)]
             (or (= type :timeout)
                 (and (= type :api-error)
                      (or (>= status 500)
                          (= status 429))))))))
```

### spec.clj (Optional Specs)

```clojure
(ns my-api.spec
  (:require [clojure.spec.alpha :as s]))

;; Example generated specs
(s/def ::id string?)
(s/def ::role #{"user" "assistant" "system"})
(s/def ::content string?)
(s/def ::model string?)

(s/def ::message
  (s/keys :req-un [::id ::role ::content]
          :opt-un [::model]))

(s/def ::create-message-request
  (s/keys :req-un [::model ::content]
          :opt-un [::max-tokens ::temperature]))

;; Instrumentation helper
(defn instrument!
  "Enables spec instrumentation for development."
  []
  (require 'clojure.spec.test.alpha)
  ((resolve 'clojure.spec.test.alpha/instrument)))
```

## Reserved Words

Clojure reserved words and special forms that must be escaped:

```
def if do let quote var fn loop recur throw try catch finally
monitor-enter monitor-exit new set! . & ns import
```

Additionally, clojure.core functions that may shadow:
```
name type count first rest map filter reduce get assoc
update keys vals merge into conj cons list vector set
str print println pr prn format read load require use
apply partial comp identity constantly true? false? nil?
some? any? every? not and or cond case when if-let when-let
```

Strategy: Prefix with the namespace or use backtick escaping in edge cases.

## Production Features

### Connection Pooling

clj-http uses Apache HttpClient with connection pooling by default.

### Retry Logic

Exponential backoff for:
- HTTP 5xx errors
- HTTP 429 (rate limit)
- Connection timeouts
- Network errors

### Graceful Shutdown

```clojure
(defn close-client
  "Closes the client and releases resources."
  [client]
  (when-let [http-client (:http-client client)]
    (.close http-client)))
```

### Metrics/Logging Hooks

```clojure
(defn create-client
  [{:keys [on-request on-response on-error] :as opts}]
  ;; Callbacks for observability
  ...)
```

## Testing Support

Generated SDKs include test helpers:

```clojure
(ns my-api.test-helpers
  (:require [my-api.core :as core]))

(defn mock-client
  "Creates a mock client for testing."
  [responses]
  (let [call-count (atom 0)
        responses (vec responses)]
    (reify core/IClient
      (request [_ method path body opts]
        (let [idx (swap! call-count inc)]
          (get responses (dec idx) {:error "No more mocked responses"}))))))
```

## Implementation Plan

1. **Phase 1: Generator Core**
   - Config struct and buildModel
   - Naming utilities (kebab-case, etc.)
   - Type mapping

2. **Phase 2: Templates**
   - deps.edn
   - core.clj (client)
   - types.clj
   - errors.clj

3. **Phase 3: Resources & Streaming**
   - resources.clj
   - streaming.clj

4. **Phase 4: Optional Features**
   - spec.clj
   - README.md generation

## Example Generated Output

For a simple API contract:

```go
&contract.Service{
    Name: "MyAPI",
    Defaults: &contract.Defaults{
        BaseURL: "https://api.myapi.com",
        Auth:    "bearer",
    },
    Resources: []*contract.Resource{{
        Name: "messages",
        Methods: []*contract.Method{{
            Name:   "create",
            Input:  "CreateMessageRequest",
            Output: "Message",
            HTTP:   &contract.MethodHTTP{Method: "POST", Path: "/v1/messages"},
        }},
    }},
    Types: []*contract.Type{{
        Name: "Message",
        Kind: contract.KindStruct,
        Fields: []contract.Field{
            {Name: "id", Type: "string"},
            {Name: "content", Type: "string"},
        },
    }},
}
```

Generates:

```clojure
;; my_api/core.clj
(ns my-api.core
  (:require [my-api.resources :as resources]))

(defn create-client [opts]
  ...)

(def messages-create resources/messages-create)

;; my_api/types.clj
(ns my-api.types)

(defrecord Message [id content])

(defn ->message [m] ...)

;; my_api/resources.clj
(ns my-api.resources
  (:require [my-api.core :as core]
            [my-api.types :as types]))

(defn messages-create
  [client request & [opts]]
  (-> (core/request client :post "/v1/messages" request opts)
      types/->message))
```

## References

- [Clojure Style Guide](https://github.com/bbatsov/clojure-style-guide)
- [clj-http Documentation](https://github.com/dakrone/clj-http)
- [Cheshire Documentation](https://github.com/dakrone/cheshire)
- [core.async Guide](https://clojure.org/guides/async)
- [clojure.spec Guide](https://clojure.org/guides/spec)
