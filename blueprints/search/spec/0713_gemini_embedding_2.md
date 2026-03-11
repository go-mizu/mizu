# 0713 Gemini Embedding Driver (v2)

## Background

The existing embedding drivers (`llamacpp`, `onnx`) require local infrastructure — running a local model server or bundling ONNX runtime libraries. The `gemini` driver calls the Google Generative Language API directly, requiring zero local infrastructure beyond an API key.

## Models

| Model | Dim | Notes |
|---|---|---|
| `text-embedding-004` | 768 | Stable, multilingual, free tier 1500 RPM |
| `gemini-embedding-exp-03-07` | 3072 | Matryoshka — supports `outputDimensionality` |

## API Endpoint

```
POST https://generativelanguage.googleapis.com/v1beta/models/{model}:batchEmbedContents?key={apiKey}
```

## Request / Response Format

**Request:**
```json
{
  "requests": [
    {
      "model": "models/text-embedding-004",
      "content": {
        "parts": [{"text": "..."}]
      },
      "taskType": "RETRIEVAL_DOCUMENT"
    }
  ]
}
```

**Response:**
```json
{
  "embeddings": [
    {"values": [0.123, -0.456, ...]}
  ]
}
```

**Batch limit:** 100 requests per API call.

## API Key Loading Priority

The driver resolves the API key in the following order, stopping at the first non-empty value:

1. `cfg.Addr` — direct override (useful for tests and programmatic usage)
2. `os.Getenv("GEMINI_API_KEY")`
3. Parse `$HOME/data/.local.env` for lines matching:
   - `export GEMINI_API_KEY="..."` (with or without quotes)
   - `GEMINI_API_KEY=...`

If no key is found after all three steps, the driver returns an error at construction time.

## Driver Config

`embed.Config` fields are interpreted as follows:

| Field | Usage |
|---|---|
| `cfg.Addr` | API key override (step 1 of key loading) |
| `cfg.Model` | Model name (default: `text-embedding-004`); supports `:NNN` suffix for Matryoshka dimensionality |
| `cfg.BatchSize` | Maximum texts per batch API call (default and maximum: 100) |
| `cfg.Dir` | Unused |

## Matryoshka Support

The `gemini-embedding-exp-03-07` model supports reduced output dimensionality via Matryoshka representation learning. To request a specific dimension, append `:NNN` to the model name:

```
gemini-embedding-exp-03-07:768
```

When a `:NNN` suffix is present, the driver:
1. Strips the suffix to get the bare model name for the API call
2. Adds `"outputDimensionality": NNN` to each request object in the batch

Example request with Matryoshka dimensionality:
```json
{
  "requests": [
    {
      "model": "models/gemini-embedding-exp-03-07",
      "content": {"parts": [{"text": "..."}]},
      "taskType": "RETRIEVAL_DOCUMENT",
      "outputDimensionality": 768
    }
  ]
}
```

## CLI Usage Examples

```bash
# Embed with default model (text-embedding-004)
search cc fts embed run --input ./docs/ --driver gemini

# Embed with experimental model at full 3072 dimensions
search cc fts embed run --input ./docs/ --driver gemini --model gemini-embedding-exp-03-07

# Embed with experimental model at reduced 768 dimensions (Matryoshka)
search cc fts embed run --input ./docs/ --driver gemini --model gemini-embedding-exp-03-07:768

# List available embedding models
search cc fts embed models
```

## Rate Limits

- `text-embedding-004` free tier: **1500 RPM** (requests per minute)
- Batch size of 100 texts per call means effective throughput of up to 150,000 texts/minute at the free tier
- HTTP 429 responses are surfaced directly to the user with the error message from the API
- No automatic retry on 429; the caller is responsible for backoff if needed

## Testing

### Unit Tests

Use `httptest.NewServer` to mock the Google API endpoint. Tests cover:

- Successful single and multi-batch embedding
- Matryoshka suffix parsing and `outputDimensionality` injection
- API key loading from each priority level (env var, `.local.env`, `cfg.Addr`)
- HTTP 429 error propagation
- Response with mismatched embedding count

### Integration Tests

Build tag: `//go:build integration`

Run against the real Gemini API using `GEMINI_API_KEY` from the environment. Tests cover:

- `text-embedding-004`: embed a short text, verify dimension = 768
- `gemini-embedding-exp-03-07`: embed a short text, verify dimension = 3072
- `gemini-embedding-exp-03-07:512`: embed a short text, verify dimension = 512

Run with:
```bash
GEMINI_API_KEY=... go test -tags integration ./pkg/embed/gemini/...
```

### CLI Smoke Test

```bash
echo "hello world" | search cc fts embed run --driver gemini --model text-embedding-004
```

Verify output contains a float vector of length 768.
