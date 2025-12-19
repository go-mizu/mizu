# RFC 0076: PHP SDK Generator

## Summary

Add PHP SDK code generation to the Mizu contract system, enabling production-ready, idiomatic PHP clients with excellent developer experience for Laravel, Symfony, and standalone PHP applications.

## Motivation

PHP powers a significant portion of web applications, including major frameworks like Laravel and Symfony. A native PHP SDK provides:

1. **Modern PHP 8.1+**: Enums, readonly properties, named arguments, constructor promotion, attributes
2. **PSR Compliance**: PSR-4 autoloading, PSR-7 HTTP messages, PSR-18 HTTP client compatibility
3. **Framework Integration**: Works seamlessly with Laravel, Symfony, and standalone applications
4. **Excellent DX**: PHPDoc annotations, IDE autocompletion, static analysis support (PHPStan/Psalm)
5. **Production-ready**: Guzzle-based HTTP with retries, timeouts, and connection pooling

## Design Goals

### Developer Experience (DX)

- **Modern PHP idioms**: Strict types, readonly properties, named arguments, constructor promotion
- **IDE-friendly**: Full PHPDoc annotations with @param, @return, @throws for excellent autocompletion
- **Fluent API**: Method chaining for client configuration
- **Minimal dependencies**: Only Guzzle (HTTP) and PSR packages
- **Static analysis**: PHPStan/Psalm level 8+ compatible out of the box

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff via Guzzle middleware
- **Timeout handling**: Per-request and global timeout configuration
- **Connection pooling**: Persistent connections via Guzzle's cURL handler
- **Error handling**: Exception hierarchy with rich error information
- **Logging**: PSR-3 logger integration

## Architecture

### Package Structure

```
{package_name}/
├── composer.json                    # Composer package definition
├── src/
│   ├── Client.php                   # Main client class
│   ├── ClientOptions.php            # Configuration options
│   ├── Types/                       # Generated model types
│   │   ├── {TypeName}.php           # One file per type
│   │   └── ...
│   ├── Resources/                   # Resource classes
│   │   ├── {Resource}Resource.php   # One file per resource
│   │   └── ...
│   ├── Streaming/                   # SSE streaming support
│   │   ├── SSEParser.php            # SSE event parser
│   │   └── StreamIterator.php       # Iterator for streaming
│   └── Exceptions/                  # Exception classes
│       ├── SDKException.php         # Base exception
│       ├── ApiException.php         # HTTP API errors
│       ├── ConnectionException.php  # Network errors
│       └── ...
```

### Core Components

#### 1. Client (`Client.php`)

The main entry point for API interactions:

```php
<?php

declare(strict_types=1);

namespace {Namespace};

use {Namespace}\Resources\{Resource}Resource;
use GuzzleHttp\Client as HttpClient;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Middleware;
use GuzzleHttp\Psr7\Request;
use Psr\Http\Message\RequestInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Log\LoggerInterface;

/**
 * {ServiceDescription}
 *
 * @property-read {Resource}Resource ${resource}
 */
final class Client
{
    private HttpClient $http;
    private ClientOptions $options;

    /** @var array<string, object> */
    private array $resources = [];

    /**
     * Creates a new SDK client.
     *
     * @param string|null $apiKey API key for authentication
     * @param ClientOptions|null $options Configuration options
     */
    public function __construct(
        ?string $apiKey = null,
        ?ClientOptions $options = null,
    ) {
        $this->options = $options ?? new ClientOptions();

        if ($apiKey !== null) {
            $this->options = $this->options->withApiKey($apiKey);
        }

        $this->http = $this->createHttpClient();
    }

    /**
     * Creates a new client with modified configuration.
     */
    public function withOptions(ClientOptions $options): self
    {
        return new self(null, $options);
    }

    /**
     * Magic getter for resource access.
     *
     * @param string $name Resource name
     * @return object Resource instance
     * @throws \InvalidArgumentException If resource doesn't exist
     */
    public function __get(string $name): object
    {
        if (!isset($this->resources[$name])) {
            $this->resources[$name] = match ($name) {
                '{resource}' => new {Resource}Resource($this),
                default => throw new \InvalidArgumentException("Unknown resource: $name"),
            };
        }

        return $this->resources[$name];
    }

    /**
     * @internal
     * @template T
     * @param class-string<T> $responseType
     * @return T
     */
    public function request(
        string $method,
        string $path,
        ?object $body = null,
        string $responseType = 'array',
    ): mixed {
        $options = [
            'headers' => [
                'Content-Type' => 'application/json',
                'Accept' => 'application/json',
            ],
        ];

        if ($body !== null) {
            $options['json'] = $body;
        }

        $response = $this->http->request($method, $path, $options);

        $statusCode = $response->getStatusCode();
        if ($statusCode >= 400) {
            $this->throwApiException($response);
        }

        if ($statusCode === 204) {
            return null;
        }

        $data = json_decode($response->getBody()->getContents(), true);

        if ($responseType === 'array') {
            return $data;
        }

        return $responseType::fromArray($data);
    }

    /**
     * @internal
     * @template T
     * @param class-string<T> $itemType
     * @return \Generator<int, T>
     */
    public function stream(
        string $method,
        string $path,
        ?object $body = null,
        string $itemType = 'array',
    ): \Generator {
        $options = [
            'headers' => [
                'Content-Type' => 'application/json',
                'Accept' => 'text/event-stream',
                'Cache-Control' => 'no-cache',
            ],
            'stream' => true,
        ];

        if ($body !== null) {
            $options['json'] = $body;
        }

        $response = $this->http->request($method, $path, $options);

        if ($response->getStatusCode() >= 400) {
            $this->throwApiException($response);
        }

        $stream = $response->getBody();
        $buffer = '';

        while (!$stream->eof()) {
            $chunk = $stream->read(8192);
            $buffer .= $chunk;

            while (($pos = strpos($buffer, "\n\n")) !== false) {
                $eventText = substr($buffer, 0, $pos);
                $buffer = substr($buffer, $pos + 2);

                $data = $this->parseSSEData($eventText);
                if ($data === null || $data === '[DONE]') {
                    continue;
                }

                $parsed = json_decode($data, true);
                if ($parsed === null) {
                    continue;
                }

                if ($itemType === 'array') {
                    yield $parsed;
                } else {
                    yield $itemType::fromArray($parsed);
                }
            }
        }
    }

    private function parseSSEData(string $eventText): ?string
    {
        $lines = explode("\n", $eventText);
        $dataLines = [];

        foreach ($lines as $line) {
            if (str_starts_with($line, 'data:')) {
                $dataLines[] = ltrim(substr($line, 5));
            }
        }

        if (empty($dataLines)) {
            return null;
        }

        return implode("\n", $dataLines);
    }

    private function createHttpClient(): HttpClient
    {
        $stack = HandlerStack::create();

        // Retry middleware
        $stack->push(Middleware::retry(
            function (int $retries, RequestInterface $request, ?ResponseInterface $response, ?\Throwable $exception): bool {
                if ($retries >= $this->options->maxRetries) {
                    return false;
                }

                if ($exception !== null) {
                    return true;
                }

                if ($response === null) {
                    return false;
                }

                $status = $response->getStatusCode();
                return $status === 429 || $status >= 500;
            },
            function (int $retries): int {
                return (int) (500 * pow(2, $retries)); // Exponential backoff
            }
        ));

        // Auth middleware
        $stack->push(Middleware::mapRequest(function (RequestInterface $request): RequestInterface {
            $apiKey = $this->options->apiKey;
            if ($apiKey === null) {
                return $request;
            }

            return match ($this->options->authMode) {
                AuthMode::Bearer => $request->withHeader('Authorization', "Bearer $apiKey"),
                AuthMode::Basic => $request->withHeader('Authorization', "Basic $apiKey"),
                AuthMode::Header => $request->withHeader('X-API-Key', $apiKey),
                AuthMode::None => $request,
            };
        }));

        // Default headers middleware
        $stack->push(Middleware::mapRequest(function (RequestInterface $request): RequestInterface {
            foreach ($this->options->defaultHeaders as $name => $value) {
                $request = $request->withHeader($name, $value);
            }
            return $request;
        }));

        return new HttpClient([
            'handler' => $stack,
            'base_uri' => $this->options->baseUrl,
            'timeout' => $this->options->timeout,
            'connect_timeout' => $this->options->connectTimeout,
            'http_errors' => false,
        ]);
    }

    private function throwApiException(ResponseInterface $response): never
    {
        $statusCode = $response->getStatusCode();
        $body = $response->getBody()->getContents();

        $message = "HTTP $statusCode";
        $decoded = json_decode($body, true);
        if (is_array($decoded)) {
            $message = $decoded['message'] ?? $decoded['error'] ?? $message;
        }

        $headers = [];
        foreach ($response->getHeaders() as $name => $values) {
            $headers[$name] = $values;
        }

        throw match ($statusCode) {
            400 => new Exceptions\BadRequestException($message, $statusCode, $body, $headers),
            401 => new Exceptions\AuthenticationException($message, $statusCode, $body, $headers),
            403 => new Exceptions\PermissionDeniedException($message, $statusCode, $body, $headers),
            404 => new Exceptions\NotFoundException($message, $statusCode, $body, $headers),
            422 => new Exceptions\UnprocessableEntityException($message, $statusCode, $body, $headers),
            429 => new Exceptions\RateLimitException($message, $statusCode, $body, $headers),
            default => $statusCode >= 500
                ? new Exceptions\ServerException($message, $statusCode, $body, $headers)
                : new Exceptions\ApiException($message, $statusCode, $body, $headers),
        };
    }
}
```

#### 2. ClientOptions (`ClientOptions.php`)

Immutable configuration with fluent builders:

```php
<?php

declare(strict_types=1);

namespace {Namespace};

/**
 * Configuration options for the SDK client.
 */
final readonly class ClientOptions
{
    /**
     * @param string|null $apiKey API key for authentication
     * @param string $baseUrl Base URL for API requests
     * @param float $timeout Request timeout in seconds
     * @param float $connectTimeout Connection timeout in seconds
     * @param int $maxRetries Maximum retry attempts
     * @param array<string, string> $defaultHeaders Default headers
     * @param AuthMode $authMode Authentication mode
     */
    public function __construct(
        public ?string $apiKey = null,
        public string $baseUrl = '{default_base_url}',
        public float $timeout = 60.0,
        public float $connectTimeout = 10.0,
        public int $maxRetries = 2,
        public array $defaultHeaders = [],
        public AuthMode $authMode = AuthMode::Bearer,
    ) {}

    public function withApiKey(?string $apiKey): self
    {
        return new self(
            apiKey: $apiKey,
            baseUrl: $this->baseUrl,
            timeout: $this->timeout,
            connectTimeout: $this->connectTimeout,
            maxRetries: $this->maxRetries,
            defaultHeaders: $this->defaultHeaders,
            authMode: $this->authMode,
        );
    }

    public function withBaseUrl(string $baseUrl): self
    {
        return new self(
            apiKey: $this->apiKey,
            baseUrl: $baseUrl,
            timeout: $this->timeout,
            connectTimeout: $this->connectTimeout,
            maxRetries: $this->maxRetries,
            defaultHeaders: $this->defaultHeaders,
            authMode: $this->authMode,
        );
    }

    public function withTimeout(float $timeout): self
    {
        return new self(
            apiKey: $this->apiKey,
            baseUrl: $this->baseUrl,
            timeout: $timeout,
            connectTimeout: $this->connectTimeout,
            maxRetries: $this->maxRetries,
            defaultHeaders: $this->defaultHeaders,
            authMode: $this->authMode,
        );
    }

    public function withMaxRetries(int $maxRetries): self
    {
        return new self(
            apiKey: $this->apiKey,
            baseUrl: $this->baseUrl,
            timeout: $this->timeout,
            connectTimeout: $this->connectTimeout,
            maxRetries: $maxRetries,
            defaultHeaders: $this->defaultHeaders,
            authMode: $this->authMode,
        );
    }

    /**
     * @param array<string, string> $headers
     */
    public function withDefaultHeaders(array $headers): self
    {
        return new self(
            apiKey: $this->apiKey,
            baseUrl: $this->baseUrl,
            timeout: $this->timeout,
            connectTimeout: $this->connectTimeout,
            maxRetries: $this->maxRetries,
            defaultHeaders: $headers,
            authMode: $this->authMode,
        );
    }

    public function withAuthMode(AuthMode $authMode): self
    {
        return new self(
            apiKey: $this->apiKey,
            baseUrl: $this->baseUrl,
            timeout: $this->timeout,
            connectTimeout: $this->connectTimeout,
            maxRetries: $this->maxRetries,
            defaultHeaders: $this->defaultHeaders,
            authMode: $authMode,
        );
    }
}
```

#### 3. AuthMode (`AuthMode.php`)

PHP 8.1 enum for authentication modes:

```php
<?php

declare(strict_types=1);

namespace {Namespace};

/**
 * Authentication mode for API requests.
 */
enum AuthMode: string
{
    /** Bearer token authentication (Authorization: Bearer {token}). */
    case Bearer = 'bearer';

    /** Basic authentication (Authorization: Basic {token}). */
    case Basic = 'basic';

    /** API key header (X-API-Key: {token}). */
    case Header = 'header';

    /** No authentication. */
    case None = 'none';
}
```

#### 4. Types (`Types/{TypeName}.php`)

Each type generates its own file with full typing:

```php
<?php

declare(strict_types=1);

namespace {Namespace}\Types;

/**
 * {TypeDescription}
 */
final readonly class {TypeName}
{
    /**
     * @param string $fieldName Description
     * @param string|null $optionalField Optional field
     * @param array<Message> $messages Messages array
     */
    public function __construct(
        public string $fieldName,
        public ?string $optionalField = null,
        public array $messages = [],
    ) {}

    /**
     * Creates an instance from an array.
     *
     * @param array<string, mixed> $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            fieldName: $data['field_name'] ?? throw new \InvalidArgumentException('Missing field_name'),
            optionalField: $data['optional_field'] ?? null,
            messages: array_map(
                fn(array $item) => Message::fromArray($item),
                $data['messages'] ?? [],
            ),
        );
    }

    /**
     * Converts to an array for JSON serialization.
     *
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        $result = [
            'field_name' => $this->fieldName,
        ];

        if ($this->optionalField !== null) {
            $result['optional_field'] = $this->optionalField;
        }

        if (!empty($this->messages)) {
            $result['messages'] = array_map(
                fn(Message $m) => $m->toArray(),
                $this->messages,
            );
        }

        return $result;
    }

    /**
     * @return array<string, mixed>
     */
    public function jsonSerialize(): array
    {
        return $this->toArray();
    }
}
```

#### 5. Enums as PHP 8.1 Enums

```php
<?php

declare(strict_types=1);

namespace {Namespace}\Types;

/**
 * Role enum for message roles.
 */
enum Role: string
{
    case User = 'user';
    case Assistant = 'assistant';
    case System = 'system';
}
```

#### 6. Union Types (Discriminated)

```php
<?php

declare(strict_types=1);

namespace {Namespace}\Types;

/**
 * Content block (discriminated union by 'type' field).
 */
abstract readonly class ContentBlock
{
    public function __construct(
        public string $type,
    ) {}

    /**
     * @param array<string, mixed> $data
     */
    public static function fromArray(array $data): self
    {
        $type = $data['type'] ?? throw new \InvalidArgumentException('Missing type field');

        return match ($type) {
            'text' => TextBlock::fromArray($data),
            'image' => ImageBlock::fromArray($data),
            'tool_use' => ToolUseBlock::fromArray($data),
            default => throw new \InvalidArgumentException("Unknown content block type: $type"),
        };
    }

    /**
     * @return array<string, mixed>
     */
    abstract public function toArray(): array;
}

final readonly class TextBlock extends ContentBlock
{
    public function __construct(
        public string $text,
    ) {
        parent::__construct('text');
    }

    public static function fromArray(array $data): self
    {
        return new self(
            text: $data['text'] ?? throw new \InvalidArgumentException('Missing text'),
        );
    }

    public function toArray(): array
    {
        return [
            'type' => $this->type,
            'text' => $this->text,
        ];
    }
}

final readonly class ImageBlock extends ContentBlock
{
    public function __construct(
        public string $url,
    ) {
        parent::__construct('image');
    }

    public static function fromArray(array $data): self
    {
        return new self(
            url: $data['url'] ?? throw new \InvalidArgumentException('Missing url'),
        );
    }

    public function toArray(): array
    {
        return [
            'type' => $this->type,
            'url' => $this->url,
        ];
    }
}
```

#### 7. Resources (`Resources/{Resource}Resource.php`)

```php
<?php

declare(strict_types=1);

namespace {Namespace}\Resources;

use {Namespace}\Client;
use {Namespace}\Types\{InputType};
use {Namespace}\Types\{OutputType};

/**
 * Operations for {resource}.
 */
final readonly class {Resource}Resource
{
    public function __construct(
        private Client $client,
    ) {}

    /**
     * {MethodDescription}
     *
     * @param string $model The model to use
     * @param array<Message> $messages The messages
     * @param int|null $maxTokens Maximum tokens to generate
     * @return {OutputType}
     * @throws \{Namespace}\Exceptions\ApiException
     *
     * @example
     * ```php
     * $response = $client->{resource}->methodName(
     *     model: 'model-name',
     *     messages: [new Message(role: Role::User, content: 'Hello')],
     * );
     * ```
     */
    public function methodName(
        string $model,
        array $messages,
        ?int $maxTokens = null,
    ): {OutputType} {
        $body = new {InputType}(
            model: $model,
            messages: $messages,
            maxTokens: $maxTokens,
        );

        return $this->client->request(
            method: 'POST',
            path: '/v1/messages',
            body: $body,
            responseType: {OutputType}::class,
        );
    }

    /**
     * {StreamMethodDescription}
     *
     * @param string $model The model to use
     * @param array<Message> $messages The messages
     * @return \Generator<int, {ItemType}>
     * @throws \{Namespace}\Exceptions\ApiException
     *
     * @example
     * ```php
     * foreach ($client->{resource}->streamMethod(model: 'x', messages: []) as $event) {
     *     echo $event->delta?->text;
     * }
     * ```
     */
    public function streamMethod(
        string $model,
        array $messages,
    ): \Generator {
        $body = new {InputType}(
            model: $model,
            messages: $messages,
            stream: true,
        );

        yield from $this->client->stream(
            method: 'POST',
            path: '/v1/messages',
            body: $body,
            itemType: {ItemType}::class,
        );
    }
}
```

#### 8. Exceptions (`Exceptions/*.php`)

```php
<?php

declare(strict_types=1);

namespace {Namespace}\Exceptions;

/**
 * Base exception for all SDK errors.
 */
class SDKException extends \Exception
{
}

/**
 * Exception for API errors (HTTP 4xx/5xx).
 */
class ApiException extends SDKException
{
    /**
     * @param array<string, array<string>> $headers
     */
    public function __construct(
        string $message,
        public readonly int $statusCode,
        public readonly ?string $body = null,
        public readonly array $headers = [],
        ?\Throwable $previous = null,
    ) {
        parent::__construct($message, $statusCode, $previous);
    }

    /**
     * Checks if this is a client error (4xx).
     */
    public function isClientError(): bool
    {
        return $this->statusCode >= 400 && $this->statusCode < 500;
    }

    /**
     * Checks if this is a server error (5xx).
     */
    public function isServerError(): bool
    {
        return $this->statusCode >= 500;
    }

    /**
     * Checks if the request can be retried.
     */
    public function isRetryable(): bool
    {
        return $this->statusCode === 429 || $this->statusCode >= 500;
    }

    /**
     * Gets the retry-after header value in seconds.
     */
    public function getRetryAfter(): ?int
    {
        $values = $this->headers['Retry-After'] ?? $this->headers['retry-after'] ?? null;
        if ($values === null || empty($values)) {
            return null;
        }

        $value = $values[0];
        if (is_numeric($value)) {
            return (int) $value;
        }

        // Parse HTTP date
        $timestamp = strtotime($value);
        if ($timestamp === false) {
            return null;
        }

        return max(0, $timestamp - time());
    }

    /**
     * Decodes the response body as the given type.
     *
     * @template T
     * @param class-string<T> $type
     * @return T|null
     */
    public function decodeAs(string $type): ?object
    {
        if ($this->body === null) {
            return null;
        }

        $data = json_decode($this->body, true);
        if (!is_array($data)) {
            return null;
        }

        return $type::fromArray($data);
    }
}

/**
 * HTTP 400 Bad Request.
 */
class BadRequestException extends ApiException {}

/**
 * HTTP 401 Unauthorized.
 */
class AuthenticationException extends ApiException {}

/**
 * HTTP 403 Forbidden.
 */
class PermissionDeniedException extends ApiException {}

/**
 * HTTP 404 Not Found.
 */
class NotFoundException extends ApiException {}

/**
 * HTTP 422 Unprocessable Entity.
 */
class UnprocessableEntityException extends ApiException {}

/**
 * HTTP 429 Too Many Requests.
 */
class RateLimitException extends ApiException {}

/**
 * HTTP 5xx Server Error.
 */
class ServerException extends ApiException {}

/**
 * Network/connection error.
 */
class ConnectionException extends SDKException
{
    public function __construct(
        string $message,
        public readonly ?\Throwable $cause = null,
    ) {
        parent::__construct($message, 0, $cause);
    }
}

/**
 * Request timeout.
 */
class TimeoutException extends SDKException {}

/**
 * SSE stream parsing error.
 */
class StreamParseException extends SDKException
{
    public function __construct(
        string $message,
        public readonly ?string $data = null,
    ) {
        parent::__construct($message);
    }
}
```

## Type Mapping

### Primitive Types

| Contract Type     | PHP Type       |
|-------------------|----------------|
| `string`          | `string`       |
| `bool`, `boolean` | `bool`         |
| `int`             | `int`          |
| `int8`            | `int`          |
| `int16`           | `int`          |
| `int32`           | `int`          |
| `int64`           | `int`          |
| `uint`            | `int`          |
| `uint8`           | `int`          |
| `uint16`          | `int`          |
| `uint32`          | `int`          |
| `uint64`          | `int`          |
| `float32`         | `float`        |
| `float64`         | `float`        |
| `time.Time`       | `\DateTimeImmutable` |
| `json.RawMessage` | `array<string, mixed>` |
| `any`             | `mixed`        |

### Collection Types

| Contract Type      | PHP Type                    |
|--------------------|-----------------------------|
| `[]T`              | `array<PHPType>`            |
| `map[string]T`     | `array<string, PHPType>`    |

### Optional/Nullable

| Contract      | PHP Type            |
|---------------|---------------------|
| `optional: T` | `?T` with `= null`  |
| `nullable: T` | `?T`                |

### Struct Fields

Fields with `optional: true` or `nullable: true` use nullable types with default null:

```php
final readonly class Request
{
    public function __construct(
        public string $required,
        public ?string $optionalField = null,
    ) {}
}
```

### Enum/Const Values

Fields with `enum` constraint generate PHP 8.1 enums:

```php
enum Role: string
{
    case User = 'user';
    case Assistant = 'assistant';
    case System = 'system';

    /**
     * @return array<self>
     */
    public static function all(): array
    {
        return self::cases();
    }
}
```

### Discriminated Unions

Union types use an abstract base class with factory method:

```php
abstract readonly class ContentBlock
{
    public function __construct(
        public string $type,
    ) {}

    public static function fromArray(array $data): self
    {
        return match ($data['type'] ?? null) {
            'text' => TextBlock::fromArray($data),
            'image' => ImageBlock::fromArray($data),
            default => throw new \InvalidArgumentException("Unknown type: " . ($data['type'] ?? 'null')),
        };
    }

    abstract public function toArray(): array;
}
```

## HTTP Client Implementation

### Request Flow

```php
private function createHttpClient(): HttpClient
{
    $stack = HandlerStack::create();

    // Retry middleware with exponential backoff
    $stack->push(Middleware::retry(
        function (int $retries, RequestInterface $request, ?ResponseInterface $response, ?\Throwable $exception): bool {
            if ($retries >= $this->options->maxRetries) {
                return false;
            }

            // Retry on connection errors
            if ($exception !== null) {
                return true;
            }

            if ($response === null) {
                return false;
            }

            // Retry on 429 (rate limit) and 5xx (server errors)
            $status = $response->getStatusCode();
            return $status === 429 || $status >= 500;
        },
        function (int $retries): int {
            // Exponential backoff: 500ms * 2^retries
            return (int) (500 * pow(2, $retries));
        }
    ));

    // Auth middleware
    $stack->push(Middleware::mapRequest(fn(RequestInterface $request) => $this->applyAuth($request)));

    // Default headers middleware
    $stack->push(Middleware::mapRequest(fn(RequestInterface $request) => $this->applyHeaders($request)));

    return new HttpClient([
        'handler' => $stack,
        'base_uri' => $this->options->baseUrl,
        'timeout' => $this->options->timeout,
        'connect_timeout' => $this->options->connectTimeout,
        'http_errors' => false, // We handle errors ourselves
    ]);
}
```

### Authentication

```php
private function applyAuth(RequestInterface $request): RequestInterface
{
    $apiKey = $this->options->apiKey;
    if ($apiKey === null) {
        return $request;
    }

    return match ($this->options->authMode) {
        AuthMode::Bearer => $request->withHeader('Authorization', "Bearer $apiKey"),
        AuthMode::Basic => $request->withHeader('Authorization', "Basic " . base64_encode($apiKey)),
        AuthMode::Header => $request->withHeader('X-API-Key', $apiKey),
        AuthMode::None => $request,
    };
}
```

### SSE Streaming

```php
/**
 * @template T
 * @param class-string<T> $itemType
 * @return \Generator<int, T>
 */
public function stream(
    string $method,
    string $path,
    ?object $body = null,
    string $itemType = 'array',
): \Generator {
    $options = [
        'headers' => [
            'Content-Type' => 'application/json',
            'Accept' => 'text/event-stream',
            'Cache-Control' => 'no-cache',
        ],
        'stream' => true,
    ];

    if ($body !== null) {
        $options['json'] = $body;
    }

    $response = $this->http->request($method, $path, $options);

    if ($response->getStatusCode() >= 400) {
        $this->throwApiException($response);
    }

    $stream = $response->getBody();
    $buffer = '';

    while (!$stream->eof()) {
        $chunk = $stream->read(8192);
        $buffer .= $chunk;

        // Process complete events (double newline terminated)
        while (($pos = strpos($buffer, "\n\n")) !== false) {
            $eventText = substr($buffer, 0, $pos);
            $buffer = substr($buffer, $pos + 2);

            $data = $this->parseSSEData($eventText);
            if ($data === null || $data === '[DONE]') {
                continue;
            }

            $parsed = json_decode($data, true);
            if ($parsed === null) {
                continue;
            }

            if ($itemType === 'array') {
                yield $parsed;
            } else {
                yield $itemType::fromArray($parsed);
            }
        }
    }
}

private function parseSSEData(string $eventText): ?string
{
    $lines = explode("\n", $eventText);
    $dataLines = [];

    foreach ($lines as $line) {
        if (str_starts_with($line, 'data:')) {
            $dataLines[] = ltrim(substr($line, 5));
        }
    }

    return empty($dataLines) ? null : implode("\n", $dataLines);
}
```

## Configuration

### Default Values

From contract `Client`:

```php
final readonly class ClientOptions
{
    public function __construct(
        public ?string $apiKey = null,
        public string $baseUrl = '{client.baseURL}',
        public float $timeout = 60.0,
        public float $connectTimeout = 10.0,
        public int $maxRetries = 2,
        public array $defaultHeaders = [
            // From client.headers
        ],
        public AuthMode $authMode = AuthMode::Bearer,
    ) {}
}
```

### Environment Variables

The SDK does NOT automatically read environment variables. Users should explicitly pass them:

```php
$client = new Client(
    apiKey: getenv('SERVICE_API_KEY') ?: null,
);
```

### Fluent Configuration

```php
$options = (new ClientOptions())
    ->withApiKey('your-api-key')
    ->withBaseUrl('https://custom.api.com')
    ->withTimeout(120.0)
    ->withMaxRetries(3)
    ->withDefaultHeaders(['X-Custom' => 'value']);

$client = new Client(options: $options);
```

## Naming Conventions

### PHP Naming

| Contract       | PHP                    |
|----------------|------------------------|
| `user-id`      | `$userId`              |
| `user_name`    | `$userName`            |
| `UserData`     | `UserData` (class)     |
| `create`       | `create()`             |
| `get-user`     | `getUser()`            |
| `maxTokens`    | `$maxTokens`           |

Functions:
- `toCamel(s)`: Converts to camelCase (for methods/properties)
- `toPascal(s)`: Converts to PascalCase (for classes)
- `toSnake(s)`: Converts to snake_case (for JSON field names)

Special handling:
- Reserved words: Prefixed with underscore (`$_class`, `$_interface`)

## Code Generation

### Generator Structure

```go
package sdkphp

type Config struct {
    // Namespace is the PHP namespace.
    // Default: PascalCase service name.
    Namespace string

    // PackageName is the Composer package name.
    // Default: "vendor/service-sdk".
    PackageName string

    // Version is the package version for composer.json.
    Version string

    // Author is the package author.
    Author string

    // License is the package license.
    License string
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── composer.json.tmpl           # Composer package definition
├── Client.php.tmpl              # Main client class
├── ClientOptions.php.tmpl       # Configuration options
├── AuthMode.php.tmpl            # Auth enum
├── Types.php.tmpl               # Model types (one per type)
├── Resources.php.tmpl           # Resource classes
├── Streaming.php.tmpl           # SSE support (conditional)
└── Exceptions.php.tmpl          # Error types
```

### Generated Files

| File                                | Purpose                        |
|-------------------------------------|--------------------------------|
| `composer.json`                     | Composer package definition    |
| `src/Client.php`                    | Main client class              |
| `src/ClientOptions.php`             | Configuration options          |
| `src/AuthMode.php`                  | Authentication enum            |
| `src/Types/{Type}.php`              | Type definitions (one per type)|
| `src/Resources/{Resource}Resource.php` | Resource classes            |
| `src/Exceptions/*.php`              | Exception classes              |

## Usage Examples

### Basic Usage

```php
<?php

require 'vendor/autoload.php';

use ServiceSDK\Client;
use ServiceSDK\Types\Message;
use ServiceSDK\Types\Role;

// Create client
$client = new Client(apiKey: 'your-api-key');

// Make a request
$response = $client->completions->create(
    model: 'model-name',
    messages: [
        new Message(role: Role::User, content: 'Hello'),
    ],
);

echo $response->content;
```

### Streaming

```php
foreach ($client->completions->createStream(
    model: 'model-name',
    messages: [new Message(role: Role::User, content: 'Hello')],
) as $event) {
    echo $event->delta?->text;
}
```

### Error Handling

```php
use ServiceSDK\Exceptions\{
    ApiException,
    RateLimitException,
    AuthenticationException,
    ConnectionException,
    SDKException,
};

try {
    $response = $client->completions->create(
        model: 'model-name',
        messages: [],
    );
} catch (RateLimitException $e) {
    $retryAfter = $e->getRetryAfter() ?? 60;
    echo "Rate limited! Retry after: {$retryAfter}s\n";
    sleep($retryAfter);
    // retry...
} catch (AuthenticationException $e) {
    echo "Invalid API key: {$e->getMessage()}\n";
} catch (ApiException $e) {
    echo "API Error {$e->statusCode}: {$e->getMessage()}\n";
    if ($e->isRetryable()) {
        // retry...
    }
} catch (ConnectionException $e) {
    echo "Network error: {$e->getMessage()}\n";
} catch (SDKException $e) {
    echo "SDK error: {$e->getMessage()}\n";
}
```

### Custom Configuration

```php
use ServiceSDK\Client;
use ServiceSDK\ClientOptions;
use ServiceSDK\AuthMode;

$options = (new ClientOptions())
    ->withApiKey('your-api-key')
    ->withBaseUrl('https://custom.api.com')
    ->withTimeout(120.0)
    ->withMaxRetries(3)
    ->withDefaultHeaders([
        'X-Custom-Header' => 'value',
    ])
    ->withAuthMode(AuthMode::Bearer);

$client = new Client(options: $options);
```

### Laravel Integration

```php
// config/services.php
return [
    'sdk' => [
        'api_key' => env('SDK_API_KEY'),
        'base_url' => env('SDK_BASE_URL', 'https://api.example.com'),
    ],
];

// app/Providers/AppServiceProvider.php
public function register(): void
{
    $this->app->singleton(Client::class, function ($app) {
        $config = $app['config']['services.sdk'];

        return new Client(
            apiKey: $config['api_key'],
            options: (new ClientOptions())
                ->withBaseUrl($config['base_url']),
        );
    });
}

// In a controller
class ChatController extends Controller
{
    public function __construct(
        private readonly Client $client,
    ) {}

    public function chat(Request $request)
    {
        $response = $this->client->completions->create(
            model: 'model-name',
            messages: [
                new Message(role: Role::User, content: $request->input('message')),
            ],
        );

        return response()->json(['reply' => $response->content]);
    }
}
```

### Symfony Integration

```yaml
# config/services.yaml
services:
    ServiceSDK\Client:
        arguments:
            $apiKey: '%env(SDK_API_KEY)%'
            $options: '@ServiceSDK\ClientOptions'

    ServiceSDK\ClientOptions:
        factory: ['ServiceSDK\ClientOptions', '__construct']
        arguments:
            $baseUrl: '%env(SDK_BASE_URL)%'
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidPHP_Syntax(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_NamingConventions(t *testing.T)
func TestGenerate_UnionTypes(t *testing.T)
func TestGenerate_EnumTypes(t *testing.T)
```

## Platform Support

### Dependencies

**Runtime Dependencies:**
- `guzzlehttp/guzzle` (^7.0) - HTTP client
- `psr/log` (^3.0) - Logging interface

**Development Dependencies:**
- `phpstan/phpstan` (^1.0) - Static analysis
- `phpunit/phpunit` (^10.0) - Testing
- `php-cs-fixer/php-cs-fixer` (^3.0) - Code style

### Minimum Versions

| Platform | Minimum Version | Rationale                           |
|----------|-----------------|-------------------------------------|
| PHP      | 8.1             | Enums, readonly properties, match   |
| Guzzle   | 7.0             | PSR-18, middleware stack            |

## Future Enhancements

1. **PSR-18 abstraction**: Allow any PSR-18 compliant HTTP client
2. **Async support**: ReactPHP/Amp integration for non-blocking I/O
3. **Response caching**: Built-in PSR-6/PSR-16 cache integration
4. **Request middleware**: Custom middleware for request/response transformation
5. **Metrics**: Request timing and success rate tracking via PSR-14 events
6. **OpenTelemetry**: Built-in tracing support

## References

- [PHP-FIG PSR Standards](https://www.php-fig.org/psr/)
- [Guzzle HTTP Client](https://docs.guzzlephp.org/)
- [PHPStan](https://phpstan.org/)
- [Composer](https://getcomposer.org/)
- [PHP 8.1 New Features](https://www.php.net/releases/8.1/en.php)
