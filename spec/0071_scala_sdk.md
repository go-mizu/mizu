# RFC 0071: Scala SDK Generator

## Summary

Add Scala SDK code generation to the Mizu contract system, enabling production-ready, type-safe Scala clients with idiomatic functional programming patterns, excellent developer experience, and seamless integration with Scala's rich type system and ecosystem.

## Motivation

Scala is a powerful language that combines object-oriented and functional programming paradigms, widely used in:

1. **Big Data**: Apache Spark, Flink, Kafka ecosystem
2. **Distributed Systems**: Akka, Lagom, high-throughput services
3. **Financial Services**: High-frequency trading, risk systems
4. **API Development**: Play Framework, http4s, ZIO-http
5. **Data Engineering**: ETL pipelines, streaming applications
6. **Enterprise JVM**: Seamless Java interop with enhanced type safety

## Design Goals

### Developer Experience (DX)

- **Idiomatic Scala**: Follow Scala conventions (case classes, sealed traits, Option, Either)
- **Functional-first**: Immutable data structures, pure functions where possible
- **Effect system agnostic**: Core types work with any effect system (Future, ZIO, Cats Effect)
- **Type safety**: Leverage Scala's expressive type system for compile-time correctness
- **ADT for unions**: Algebraic data types with pattern matching
- **Circe JSON**: Industry-standard JSON library with compile-time derivation
- **sttp client**: Modern, cross-platform HTTP client
- **Comprehensive documentation**: ScalaDoc with examples

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff
- **Timeout handling**: Per-request and global timeout configuration
- **Streaming support**: fs2 Streams for SSE with backpressure
- **Error handling**: Typed errors with Either/ADTs
- **Thread safety**: Immutable types and effect-safe design
- **Cross-platform**: Scala 2.13 and Scala 3 compatible
- **JVM and JS targets**: Optional Scala.js support

## Architecture

### Package Structure

```
{artifact}/
├── build.sbt                    # SBT build configuration
├── project/
│   └── build.properties         # SBT version
└── src/main/scala/{package}/
    ├── Client.scala             # Main client class
    ├── Types.scala              # Generated model types
    ├── Resources.scala          # Resource operations
    ├── Streaming.scala          # SSE streaming support
    └── Errors.scala             # Error types
```

### Core Components

#### 1. Error Handling (`Errors.scala`)

Sealed trait hierarchy for typed errors:

```scala
package com.example.sdk

/** Base error type for all SDK errors. */
sealed trait SDKError extends Exception {
  def message: String
  override def getMessage: String = message
}

object SDKError {
  /** Network connection failed. */
  final case class ConnectionError(
    cause: Throwable,
    message: String = "Connection failed"
  ) extends SDKError

  /** Server returned an error status code. */
  final case class ApiError(
    statusCode: Int,
    message: String,
    body: Option[String] = None
  ) extends SDKError

  /** Request timed out. */
  final case class Timeout(
    message: String = "Request timed out"
  ) extends SDKError

  /** Request was cancelled. */
  final case class Cancelled(
    message: String = "Request cancelled"
  ) extends SDKError

  /** Failed to encode request body. */
  final case class EncodingError(
    cause: Throwable,
    message: String = "Failed to encode request"
  ) extends SDKError

  /** Failed to decode response body. */
  final case class DecodingError(
    cause: Throwable,
    message: String = "Failed to decode response"
  ) extends SDKError
}

/** Type alias for SDK results. */
type SDKResult[A] = Either[SDKError, A]
```

#### 2. Client (`Client.scala`)

The main client with immutable configuration:

```scala
package com.example.sdk

import scala.concurrent.{ExecutionContext, Future}
import scala.concurrent.duration._
import sttp.client3._
import sttp.client3.asynchttpclient.future.AsyncHttpClientFutureBackend
import io.circe.syntax._
import io.circe.parser._

/** Authentication mode for API requests. */
sealed trait AuthMode
object AuthMode {
  case object Bearer extends AuthMode
  case object Basic extends AuthMode
  case object ApiKey extends AuthMode
  case object None extends AuthMode
}

/** Configuration options for the SDK client. */
final case class ClientConfig(
  apiKey: Option[String] = None,
  baseUrl: String = "{defaults.base_url}",
  authMode: AuthMode = AuthMode.Bearer,
  timeout: FiniteDuration = 60.seconds,
  maxRetries: Int = 2,
  defaultHeaders: Map[String, String] = Map.empty
)

object ClientConfig {
  /** Default configuration with standard settings. */
  val default: ClientConfig = ClientConfig()
}

/**
 * {ServiceDescription}
 *
 * Thread-safe client for API interactions.
 */
final class {ServiceName}(
  val config: ClientConfig = ClientConfig.default
)(implicit ec: ExecutionContext) {

  private val backend: SttpBackend[Future, Any] = AsyncHttpClientFutureBackend()

  private val baseHeaders: Map[String, String] = {
    val authHeader = config.apiKey.map { key =>
      config.authMode match {
        case AuthMode.Bearer => "Authorization" -> s"Bearer $key"
        case AuthMode.Basic  => "Authorization" -> s"Basic $key"
        case AuthMode.ApiKey => "X-Api-Key" -> key
        case AuthMode.None   => "" -> ""
      }
    }.filter(_._1.nonEmpty).toMap

    config.defaultHeaders ++ authHeader ++ Map(
      "Content-Type" -> "application/json",
      "Accept" -> "application/json"
    )
  }

  /** Access to {resource} operations. */
  lazy val {resource}: {Resource}Resource = new {Resource}Resource(this)

  /** Creates a new client with modified configuration. */
  def withConfig(f: ClientConfig => ClientConfig): {ServiceName} =
    new {ServiceName}(f(config))

  /** Creates a new client with a different API key. */
  def withApiKey(apiKey: String): {ServiceName} =
    withConfig(_.copy(apiKey = Some(apiKey)))

  /** Creates a new client with a different base URL. */
  def withBaseUrl(baseUrl: String): {ServiceName} =
    withConfig(_.copy(baseUrl = baseUrl))

  /** Closes the client and releases resources. */
  def close(): Future[Unit] = backend.close()

  // Internal request methods
  private[sdk] def request[A: io.circe.Decoder](
    method: Method,
    path: String,
    body: Option[String] = None
  ): Future[SDKResult[A]] = {
    val uri = uri"${config.baseUrl}$path"

    val baseRequest = basicRequest
      .method(method, uri)
      .headers(baseHeaders)
      .readTimeout(config.timeout)

    val requestWithBody = body.fold(baseRequest)(b =>
      baseRequest.body(b)
    )

    executeWithRetry(requestWithBody, config.maxRetries)
  }

  private def executeWithRetry[A: io.circe.Decoder](
    request: Request[Either[String, String], Any],
    retriesLeft: Int
  ): Future[SDKResult[A]] = {
    request.send(backend).flatMap { response =>
      response.body match {
        case Right(bodyStr) if response.code.isSuccess =>
          decode[A](bodyStr) match {
            case Right(value) => Future.successful(Right(value))
            case Left(error)  => Future.successful(Left(
              SDKError.DecodingError(error)
            ))
          }
        case Right(bodyStr) =>
          Future.successful(Left(SDKError.ApiError(
            response.code.code,
            s"HTTP ${response.code.code}",
            Some(bodyStr)
          )))
        case Left(errorBody) if retriesLeft > 0 && isRetriable(response.code.code) =>
          val delay = (500 * math.pow(2, config.maxRetries - retriesLeft)).toLong
          akka.pattern.after(delay.millis, using = ???)(
            executeWithRetry(request, retriesLeft - 1)
          )
        case Left(errorBody) =>
          Future.successful(Left(SDKError.ApiError(
            response.code.code,
            errorBody,
            Some(errorBody)
          )))
      }
    }.recover {
      case e: java.net.SocketTimeoutException =>
        Left(SDKError.Timeout())
      case e: java.util.concurrent.CancellationException =>
        Left(SDKError.Cancelled())
      case e: Throwable =>
        Left(SDKError.ConnectionError(e))
    }
  }

  private def isRetriable(code: Int): Boolean =
    code >= 500 || code == 429
}

object {ServiceName} {
  /** Creates a new client with default configuration. */
  def apply()(implicit ec: ExecutionContext): {ServiceName} =
    new {ServiceName}()

  /** Creates a new client with an API key. */
  def apply(apiKey: String)(implicit ec: ExecutionContext): {ServiceName} =
    new {ServiceName}(ClientConfig(apiKey = Some(apiKey)))

  /** Creates a new client with custom configuration. */
  def apply(config: ClientConfig)(implicit ec: ExecutionContext): {ServiceName} =
    new {ServiceName}(config)
}
```

#### 3. Types (`Types.scala`)

Case classes and sealed traits with Circe derivation:

```scala
package com.example.sdk

import io.circe._
import io.circe.generic.semiauto._
import io.circe.syntax._

// --- Struct Types ---

/**
 * {Description}
 */
final case class {TypeName}(
  /** {Field description} */
  fieldName: FieldType,

  /** Optional field */
  optionalField: Option[String] = None
)

object {TypeName} {
  implicit val encoder: Encoder[{TypeName}] = deriveEncoder[{TypeName}]
  implicit val decoder: Decoder[{TypeName}] = deriveDecoder[{TypeName}]
}

// --- Enum Types ---

/**
 * {Description}
 */
sealed trait {EnumName} {
  def value: String
}

object {EnumName} {
  case object Value1 extends {EnumName} { val value = "value1" }
  case object Value2 extends {EnumName} { val value = "value2" }

  val values: List[{EnumName}] = List(Value1, Value2)

  def fromString(s: String): Option[{EnumName}] = values.find(_.value == s)

  def unsafeFromString(s: String): {EnumName} =
    fromString(s).getOrElse(throw new IllegalArgumentException(s"Unknown {EnumName}: $s"))

  implicit val encoder: Encoder[{EnumName}] = Encoder.encodeString.contramap(_.value)
  implicit val decoder: Decoder[{EnumName}] = Decoder.decodeString.emap { s =>
    fromString(s).toRight(s"Unknown {EnumName}: $s")
  }
}

// --- Discriminated Unions ---

/**
 * {Description}
 *
 * Discriminated union (tag: "{tag}").
 */
sealed trait {UnionName}

object {UnionName} {
  final case class Variant1(/* fields */) extends {UnionName}
  final case class Variant2(/* fields */) extends {UnionName}

  private val discriminator = "{tag}"

  implicit val encoder: Encoder[{UnionName}] = Encoder.instance {
    case v: Variant1 => v.asJson.deepMerge(Json.obj(discriminator -> Json.fromString("variant1")))
    case v: Variant2 => v.asJson.deepMerge(Json.obj(discriminator -> Json.fromString("variant2")))
  }

  implicit val decoder: Decoder[{UnionName}] = Decoder.instance { cursor =>
    cursor.downField(discriminator).as[String].flatMap {
      case "variant1" => cursor.as[Variant1]
      case "variant2" => cursor.as[Variant2]
      case other      => Left(DecodingFailure(s"Unknown $discriminator: $other", cursor.history))
    }
  }

  // Pattern matching helpers
  implicit class Ops(private val self: {UnionName}) extends AnyVal {
    def isVariant1: Boolean = self.isInstanceOf[Variant1]
    def isVariant2: Boolean = self.isInstanceOf[Variant2]

    def asVariant1: Option[Variant1] = self match {
      case v: Variant1 => Some(v)
      case _ => None
    }

    def asVariant2: Option[Variant2] = self match {
      case v: Variant2 => Some(v)
      case _ => None
    }
  }
}
```

#### 4. Resources (`Resources.scala`)

Resource classes with method operations:

```scala
package com.example.sdk

import scala.concurrent.{ExecutionContext, Future}
import io.circe.syntax._
import sttp.client3._

/**
 * Operations for {resource}.
 *
 * {Description}
 */
final class {Resource}Resource(client: {ServiceName})(implicit ec: ExecutionContext) {

  /**
   * {Method description}
   *
   * @param request The request payload
   * @return The response or an error
   */
  def methodName(request: RequestType): Future[SDKResult[ResponseType]] =
    client.request[ResponseType](
      Method.POST,
      "{path}",
      Some(request.asJson.noSpaces)
    )

  /**
   * {Streaming method description}
   *
   * @param request The request payload
   * @return A stream of events
   */
  def streamMethod(request: RequestType): fs2.Stream[Future, ItemType] =
    client.stream[ItemType](
      Method.POST,
      "{path}",
      Some(request.asJson.noSpaces)
    )
}
```

#### 5. Streaming (`Streaming.scala`)

SSE streaming support with fs2:

```scala
package com.example.sdk

import scala.concurrent.{ExecutionContext, Future}
import fs2._
import io.circe._
import io.circe.parser._

/**
 * Server-Sent Events (SSE) streaming support.
 */
object Streaming {

  /** Represents a parsed SSE event. */
  final case class SSEEvent(
    event: Option[String] = None,
    data: Option[String] = None,
    id: Option[String] = None,
    retry: Option[Long] = None
  )

  /**
   * Parses a stream of bytes into SSE events.
   */
  def parseSSE[F[_]]: Pipe[F, Byte, SSEEvent] = { input =>
    input
      .through(text.utf8.decode)
      .through(text.lines)
      .scan((SSEEvent(), false)) { case ((event, emit), line) =>
        if (line.isEmpty) {
          (SSEEvent(), true)
        } else if (line.startsWith(":")) {
          // Comment, ignore
          (event, false)
        } else if (line.startsWith("event:")) {
          (event.copy(event = Some(line.stripPrefix("event:").trim)), false)
        } else if (line.startsWith("data:")) {
          val newData = line.stripPrefix("data:").trim
          val data = event.data.fold(newData)(_ + "\n" + newData)
          (event.copy(data = Some(data)), false)
        } else if (line.startsWith("id:")) {
          (event.copy(id = Some(line.stripPrefix("id:").trim)), false)
        } else if (line.startsWith("retry:")) {
          val retry = line.stripPrefix("retry:").trim.toLongOption
          (event.copy(retry = retry), false)
        } else {
          (event, false)
        }
      }
      .collect { case (event, true) if event.data.isDefined => event }
  }

  /**
   * Decodes SSE events into typed values.
   */
  def decodeSSE[F[_], A: Decoder]: Pipe[F, SSEEvent, Either[DecodingFailure, A]] = { input =>
    input
      .filter(_.data.exists(_ != "[DONE]"))
      .map { event =>
        event.data.fold[Either[DecodingFailure, A]](
          Left(DecodingFailure("No data in SSE event", Nil))
        ) { data =>
          decode[A](data).left.map(e => DecodingFailure(e.getMessage, Nil))
        }
      }
  }

  /**
   * Combines SSE parsing and decoding.
   */
  def parseAndDecode[F[_], A: Decoder]: Pipe[F, Byte, Either[DecodingFailure, A]] =
    parseSSE.andThen(decodeSSE[F, A])
}

/**
 * Extension methods for streaming operations.
 */
object StreamingOps {
  implicit class StreamOps[F[_], A](private val stream: Stream[F, Either[SDKError, A]]) extends AnyVal {

    /** Collects all successful items into a list. */
    def toListResult(implicit F: cats.effect.Concurrent[F]): F[SDKResult[List[A]]] =
      stream.compile.toList.map { results =>
        val (errors, successes) = results.partitionMap(identity)
        errors.headOption match {
          case Some(error) => Left(error)
          case None        => Right(successes)
        }
      }

    /** Maps over successful items. */
    def mapSuccess[B](f: A => B): Stream[F, Either[SDKError, B]] =
      stream.map(_.map(f))

    /** Filters to only successful items, discarding errors. */
    def collectSuccess: Stream[F, A] =
      stream.collect { case Right(a) => a }
  }
}
```

## Type Mapping

### Primitive Types

| Contract Type     | Scala Type           | Notes                           |
|-------------------|----------------------|--------------------------------|
| `string`          | `String`             |                                |
| `bool`, `boolean` | `Boolean`            |                                |
| `int`             | `Int`                |                                |
| `int8`            | `Byte`               |                                |
| `int16`           | `Short`              |                                |
| `int32`           | `Int`                |                                |
| `int64`           | `Long`               |                                |
| `uint`            | `Int`                | Scala lacks unsigned types     |
| `uint8`           | `Short`              | Widened to avoid overflow      |
| `uint16`          | `Int`                | Widened to avoid overflow      |
| `uint32`          | `Long`               | Widened to avoid overflow      |
| `uint64`          | `BigInt`             | For full 64-bit unsigned range |
| `float32`         | `Float`              |                                |
| `float64`         | `Double`             |                                |
| `time.Time`       | `java.time.Instant`  | With ISO 8601 codec            |
| `json.RawMessage` | `io.circe.Json`      | Raw JSON value                 |
| `any`             | `io.circe.Json`      | Dynamic JSON                   |

### Collection Types

| Contract Type      | Scala Type                 |
|--------------------|----------------------------|
| `[]T`              | `List[ScalaType]`          |
| `map[string]T`     | `Map[String, ScalaType]`   |

### Optional/Nullable

| Contract Pattern   | Scala Type           |
|--------------------|----------------------|
| Optional field     | `Option[T]`          |
| Nullable type      | `Option[T]`          |

### Struct Fields

Fields use Circe annotations for JSON mapping:

```scala
import io.circe._
import io.circe.generic.semiauto._

final case class CreateMessageRequest(
  /** The model to use for generation. */
  model: String,

  /** The messages in the conversation. */
  messages: List[Message],

  /** Maximum tokens to generate. */
  maxTokens: Int,

  /** Temperature for sampling. */
  temperature: Option[Double] = None,

  /** Whether to stream the response. */
  stream: Option[Boolean] = None
)

object CreateMessageRequest {
  implicit val encoder: Encoder[CreateMessageRequest] =
    deriveEncoder[CreateMessageRequest].mapJsonObject { obj =>
      // Convert camelCase to snake_case for JSON
      JsonObject.fromIterable(obj.toIterable.map { case (k, v) =>
        camelToSnake(k) -> v
      })
    }

  implicit val decoder: Decoder[CreateMessageRequest] =
    deriveDecoder[CreateMessageRequest].prepare { cursor =>
      // Convert snake_case to camelCase for parsing
      cursor.withFocus(_.mapObject { obj =>
        JsonObject.fromIterable(obj.toIterable.map { case (k, v) =>
          snakeToCamel(k) -> v
        })
      })
    }
}
```

### Enum/Const Values

String-backed sealed traits:

```scala
sealed trait Role {
  def value: String
}

object Role {
  case object User extends Role { val value = "user" }
  case object Assistant extends Role { val value = "assistant" }
  case object System extends Role { val value = "system" }

  val values: List[Role] = List(User, Assistant, System)

  def fromString(s: String): Option[Role] = values.find(_.value == s)

  def unsafeFromString(s: String): Role =
    fromString(s).getOrElse(throw new IllegalArgumentException(s"Unknown Role: $s"))

  implicit val encoder: Encoder[Role] = Encoder.encodeString.contramap(_.value)
  implicit val decoder: Decoder[Role] = Decoder.decodeString.emap { s =>
    fromString(s).toRight(s"Unknown Role: $s")
  }
}
```

### Discriminated Unions

Tagged sealed traits with pattern matching:

```scala
sealed trait ContentBlock

object ContentBlock {
  final case class Text(
    `type`: String = "text",
    text: String
  ) extends ContentBlock

  final case class Image(
    `type`: String = "image",
    url: String
  ) extends ContentBlock

  final case class ToolUse(
    `type`: String = "tool_use",
    id: String,
    name: String,
    input: Json
  ) extends ContentBlock

  object Text {
    implicit val encoder: Encoder[Text] = deriveEncoder
    implicit val decoder: Decoder[Text] = deriveDecoder
  }

  object Image {
    implicit val encoder: Encoder[Image] = deriveEncoder
    implicit val decoder: Decoder[Image] = deriveDecoder
  }

  object ToolUse {
    implicit val encoder: Encoder[ToolUse] = deriveEncoder
    implicit val decoder: Decoder[ToolUse] = deriveDecoder
  }

  private val discriminator = "type"

  implicit val encoder: Encoder[ContentBlock] = Encoder.instance {
    case v: Text    => v.asJson
    case v: Image   => v.asJson
    case v: ToolUse => v.asJson
  }

  implicit val decoder: Decoder[ContentBlock] = Decoder.instance { cursor =>
    cursor.downField(discriminator).as[String].flatMap {
      case "text"     => cursor.as[Text]
      case "image"    => cursor.as[Image]
      case "tool_use" => cursor.as[ToolUse]
      case other      => Left(DecodingFailure(s"Unknown type: $other", cursor.history))
    }
  }

  // Pattern matching extensions
  implicit class ContentBlockOps(private val self: ContentBlock) extends AnyVal {
    def isText: Boolean = self.isInstanceOf[Text]
    def isImage: Boolean = self.isInstanceOf[Image]
    def isToolUse: Boolean = self.isInstanceOf[ToolUse]

    def asText: Option[Text] = self match {
      case v: Text => Some(v)
      case _       => None
    }

    def asImage: Option[Image] = self match {
      case v: Image => Some(v)
      case _        => None
    }

    def asToolUse: Option[ToolUse] = self match {
      case v: ToolUse => Some(v)
      case _          => None
    }

    def fold[B](
      onText: Text => B,
      onImage: Image => B,
      onToolUse: ToolUse => B
    ): B = self match {
      case v: Text    => onText(v)
      case v: Image   => onImage(v)
      case v: ToolUse => onToolUse(v)
    }
  }
}
```

## HTTP Client Implementation

### Request Flow

```scala
final class MessagesResource(client: Anthropic)(implicit ec: ExecutionContext) {

  def create(request: CreateMessageRequest): Future[SDKResult[Message]] =
    client.request[Message](
      Method.POST,
      "/v1/messages",
      Some(request.asJson.noSpaces)
    )

  def createWithRetry(
    request: CreateMessageRequest,
    maxRetries: Int = 3
  ): Future[SDKResult[Message]] = {
    def attempt(retriesLeft: Int, lastError: Option[SDKError]): Future[SDKResult[Message]] = {
      if (retriesLeft <= 0) {
        Future.successful(Left(lastError.getOrElse(
          SDKError.ConnectionError(new Exception("Max retries exceeded"))
        )))
      } else {
        create(request).flatMap {
          case Right(result) => Future.successful(Right(result))
          case Left(error) if error.isRetriable =>
            val delay = (500 * math.pow(2, maxRetries - retriesLeft)).toLong
            after(delay.millis)(attempt(retriesLeft - 1, Some(error)))
          case Left(error) => Future.successful(Left(error))
        }
      }
    }
    attempt(maxRetries, None)
  }
}

// Error retriability extension
implicit class SDKErrorOps(private val self: SDKError) extends AnyVal {
  def isRetriable: Boolean = self match {
    case SDKError.ApiError(code, _, _) => code >= 500 || code == 429
    case _: SDKError.Timeout           => true
    case _: SDKError.ConnectionError   => true
    case _                             => false
  }
}
```

### SSE Streaming Implementation

```scala
def stream(request: CreateMessageRequest): fs2.Stream[IO, SDKResult[MessageStreamEvent]] = {
  val uri = Uri.parse(s"${client.config.baseUrl}/v1/messages").toOption.get

  val sttpRequest = basicRequest
    .post(uri)
    .headers(client.baseHeaders + ("Accept" -> "text/event-stream"))
    .body(request.asJson.noSpaces)
    .response(asStreamAlwaysUnsafe(Fs2Streams[IO]))

  fs2.Stream.eval(sttpRequest.send(client.backend)).flatMap { response =>
    if (response.code.isSuccess) {
      response.body
        .through(Streaming.parseSSE[IO])
        .through(Streaming.decodeSSE[IO, MessageStreamEvent])
        .map {
          case Right(event) => Right(event)
          case Left(error)  => Left(SDKError.DecodingError(
            new Exception(error.message)
          ))
        }
    } else {
      fs2.Stream.emit(Left(SDKError.ApiError(
        response.code.code,
        s"HTTP ${response.code.code}",
        None
      )))
    }
  }
}
```

## Configuration

### Default Values

From contract `Defaults`:

```scala
object ClientConfig {
  val default: ClientConfig = ClientConfig(
    baseUrl = "{defaults.base_url}",
    authMode = AuthMode.Bearer,
    timeout = 60.seconds,
    maxRetries = 2,
    defaultHeaders = Map(
      // From defaults.headers
    )
  )
}
```

### Environment Variables

The SDK does NOT automatically read environment variables. Users should handle this explicitly:

```scala
val client = Anthropic(
  ClientConfig(
    apiKey = sys.env.get("ANTHROPIC_API_KEY")
  )
)
```

## Naming Conventions

### Scala Naming

| Contract       | Scala                     |
|----------------|---------------------------|
| `user-id`      | `userId`                  |
| `user_name`    | `userName`                |
| `UserData`     | `UserData`                |
| `create`       | `create`                  |
| `get-user`     | `getUser`                 |
| `type`         | `` `type` ``              |

Functions:
- `toScalaName(s)`: Converts to camelCase for methods/fields
- `toScalaTypeName(s)`: Converts to PascalCase for types
- `sanitizeIdent(s)`: Removes invalid characters

Reserved words: Scala keywords are escaped with backticks:
- `type` → `` `type` ``
- `class` → `` `class` ``
- `object` → `` `object` ``
- `val` → `` `val` ``
- `var` → `` `var` ``
- `def` → `` `def` ``
- `trait` → `` `trait` ``
- `import` → `` `import` ``
- etc.

## Code Generation

### Generator Structure

```go
package sdkscala

type Config struct {
    // Package is the Scala package name.
    // Default: sanitized lowercase service name.
    Package string

    // Version is the artifact version.
    // Default: "0.0.0".
    Version string

    // Organization is the SBT organization.
    // Default: "com.example".
    Organization string

    // ArtifactId is the SBT artifact ID.
    // Default: kebab-case service name.
    ArtifactId string

    // ScalaVersion is the Scala version.
    // Default: "2.13.12".
    ScalaVersion string

    // Scala3 enables Scala 3 syntax.
    // Default: false.
    Scala3 bool
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── build.sbt.tmpl            # SBT build configuration
├── Client.scala.tmpl         # Main client
├── Types.scala.tmpl          # Model types
├── Resources.scala.tmpl      # Resource classes
├── Streaming.scala.tmpl      # SSE support
└── Errors.scala.tmpl         # Error types
```

### Generated Files

| File                 | Purpose                           |
|----------------------|-----------------------------------|
| `build.sbt`          | SBT build configuration           |
| `project/build.properties` | SBT version                 |
| `Client.scala`       | Main client class                 |
| `Types.scala`        | All model type definitions        |
| `Resources.scala`    | Resource operations               |
| `Streaming.scala`    | SSE streaming support             |
| `Errors.scala`       | Error type definitions            |

### build.sbt

```scala
ThisBuild / organization := "{organization}"
ThisBuild / version      := "{version}"
ThisBuild / scalaVersion := "{scalaVersion}"

lazy val root = (project in file("."))
  .settings(
    name := "{artifactId}",
    libraryDependencies ++= Seq(
      "io.circe"                   %% "circe-core"           % "0.14.6",
      "io.circe"                   %% "circe-generic"        % "0.14.6",
      "io.circe"                   %% "circe-parser"         % "0.14.6",
      "com.softwaremill.sttp.client3" %% "core"              % "3.9.1",
      "com.softwaremill.sttp.client3" %% "async-http-client-backend-future" % "3.9.1",
      "co.fs2"                     %% "fs2-core"             % "3.9.3",
      "co.fs2"                     %% "fs2-io"               % "3.9.3",
      "org.scalatest"              %% "scalatest"            % "3.2.17" % Test
    )
  )
```

## Usage Examples

### Basic Usage

```scala
import com.example.sdk._
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.Await
import scala.concurrent.duration._

// Create client
val client = Anthropic("your-api-key")

// Build request
val request = CreateMessageRequest(
  model = "claude-3-sonnet-20240229",
  maxTokens = 1024,
  messages = List(
    Message(
      role = Role.User,
      content = List(
        ContentBlock.Text(text = "Hello, Claude!")
      )
    )
  )
)

// Make request
val result = Await.result(client.messages.create(request), 60.seconds)

result match {
  case Right(message) =>
    message.content.foreach {
      case ContentBlock.Text(_, text) => println(text)
      case _ => ()
    }
  case Left(error) =>
    println(s"Error: ${error.message}")
}

// Clean up
client.close()
```

### Streaming

```scala
import cats.effect.IO
import cats.effect.unsafe.implicits.global

val stream = client.messages.stream(request)

stream
  .evalMap {
    case Right(event) => IO(print(event.delta.flatMap(_.text).getOrElse("")))
    case Left(error)  => IO(println(s"\nError: ${error.message}"))
  }
  .compile
  .drain
  .unsafeRunSync()
```

### Error Handling

```scala
client.messages.create(request).map {
  case Right(response) =>
    println(s"Success: ${response.usage.totalTokens} tokens used")

  case Left(SDKError.ApiError(429, _, _)) =>
    println("Rate limited, please retry later")

  case Left(SDKError.ApiError(code, msg, _)) if code >= 400 && code < 500 =>
    println(s"Client error $code: $msg")

  case Left(SDKError.ApiError(code, _, _)) if code >= 500 =>
    println(s"Server error $code")

  case Left(SDKError.Timeout(_)) =>
    println("Request timed out")

  case Left(SDKError.ConnectionError(cause, _)) =>
    println(s"Connection failed: ${cause.getMessage}")

  case Left(error) =>
    println(s"Error: ${error.message}")
}
```

### Pattern Matching on Unions

```scala
response.content.foreach { block =>
  block match {
    case ContentBlock.Text(_, text) =>
      println(s"Text: $text")
    case ContentBlock.Image(_, url) =>
      println(s"Image: $url")
    case ContentBlock.ToolUse(_, id, name, input) =>
      println(s"Tool use: $name -> $input")
  }
}

// Or use helper methods
response.content.foreach { block =>
  block.asText.foreach { text =>
    println(text.text)
  }
}

// Or use fold
response.content.foreach { block =>
  block.fold(
    onText = t => println(s"Text: ${t.text}"),
    onImage = i => println(s"Image: ${i.url}"),
    onToolUse = tu => println(s"Tool: ${tu.name}")
  )
}
```

### Custom Configuration

```scala
val client = Anthropic(
  ClientConfig(
    apiKey = Some("your-api-key"),
    baseUrl = "https://custom.api.com",
    timeout = 120.seconds,
    maxRetries = 3,
    defaultHeaders = Map(
      "X-Custom-Header" -> "custom-value"
    )
  )
)
```

### Functional Composition

```scala
import cats.implicits._

// Chain multiple requests
val program: Future[SDKResult[(Message, Message)]] = for {
  result1 <- client.messages.create(request1)
  result2 <- client.messages.create(request2)
} yield (result1, result2).tupled

// With error recovery
val withFallback = client.messages.create(request).recoverWith {
  case Left(SDKError.ApiError(503, _, _)) =>
    // Fallback to different model
    client.messages.create(request.copy(model = "claude-3-haiku"))
}
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidScala_Compiles(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_UnionTypes(t *testing.T)
func TestGenerate_EnumTypes(t *testing.T)
func TestGenerate_CaseClasses(t *testing.T)
```

### Generated SDK Tests

```scala
import org.scalatest.flatspec.AnyFlatSpec
import org.scalatest.matchers.should.Matchers

class TypesSpec extends AnyFlatSpec with Matchers {

  "Role" should "encode to JSON string" in {
    import io.circe.syntax._
    Role.User.asJson.noSpaces shouldBe "\"user\""
  }

  it should "decode from JSON string" in {
    import io.circe.parser._
    decode[Role]("\"assistant\"") shouldBe Right(Role.Assistant)
  }

  "ContentBlock" should "decode text variant" in {
    import io.circe.parser._
    val json = """{"type":"text","text":"hello"}"""
    val result = decode[ContentBlock](json)
    result shouldBe a[Right[_, _]]
    result.toOption.get shouldBe a[ContentBlock.Text]
  }

  "CreateMessageRequest" should "encode with snake_case keys" in {
    import io.circe.syntax._
    val request = CreateMessageRequest(
      model = "test",
      maxTokens = 100,
      messages = Nil
    )
    val json = request.asJson
    json.hcursor.downField("max_tokens").as[Int] shouldBe Right(100)
  }
}
```

## Platform Support

### Dependencies

**Required:**
- `io.circe:circe-core` (0.14+) - JSON encoding
- `io.circe:circe-generic` (0.14+) - Automatic derivation
- `io.circe:circe-parser` (0.14+) - JSON parsing
- `com.softwaremill.sttp.client3:core` (3.9+) - HTTP client
- `com.softwaremill.sttp.client3:async-http-client-backend-future` (3.9+)
- `co.fs2:fs2-core` (3.9+) - Streaming

**Optional:**
- `org.typelevel:cats-effect` (3.5+) - IO effect type
- `dev.zio:zio` (2.0+) - ZIO effect type

### Scala Versions

| Scala Version | Status    | Notes                           |
|---------------|-----------|--------------------------------|
| 2.13.x        | ✅        | Primary target                 |
| 3.3.x         | ✅        | Full support                   |
| 2.12.x        | ⚠️        | Requires minor adjustments     |

### JVM Requirements

JDK 11+ recommended for optimal performance

## Future Enhancements

1. **ZIO integration**: ZIO-native client with ZLayer
2. **Cats Effect integration**: IO-based client with Resource
3. **Scala.js support**: Browser and Node.js targets
4. **Scala Native**: Native compilation support
5. **Request interceptors**: Middleware for logging, metrics
6. **Response caching**: Built-in caching with TTL
7. **Circuit breaker**: Resilience4j integration
8. **Metrics**: Micrometer/Prometheus integration
9. **Refined types**: Compile-time validation
10. **Optics**: Monocle lenses for types

## References

- [Scala Style Guide](https://docs.scala-lang.org/style/)
- [Circe Documentation](https://circe.github.io/circe/)
- [sttp Documentation](https://sttp.softwaremill.com/en/stable/)
- [fs2 Documentation](https://fs2.io/)
- [Cats Effect](https://typelevel.org/cats-effect/)
- [ZIO](https://zio.dev/)
- [Scala with Cats](https://www.scalawithcats.com/)
