# 0713 Gemini Embedding Driver (Gemini Embedding 2 Preview)

## Background

The existing embedding drivers (`llamacpp`, `onnx`) require local infrastructure — running a local model server or bundling ONNX runtime libraries. The `gemini` driver calls the Google Generative Language API directly, requiring zero local infrastructure beyond an API key.

> **Note:** The model name `gemini-embedding-exp-03-07` returns HTTP 404 and is NOT the correct model name for the current API. The correct model is `gemini-embedding-2-preview`.

## Models

| Model | Dim | Notes |
|---|---|---|
| `text-embedding-004` | 768 | Stable, multilingual, free tier 1500 RPM |
| `gemini-embedding-2-preview` | 3072 | Matryoshka — supports `outputDimensionality`; default for `--driver gemini` |
| `gemini-embedding-001` | 3072 | Matryoshka — stable release |

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
| `cfg.Model` | Model name (default: `gemini-embedding-2-preview`); supports `:NNN` suffix for Matryoshka dimensionality |
| `cfg.BatchSize` | Maximum texts per batch API call (default and maximum: 100) |
| `cfg.Dir` | Unused |

## Matryoshka Support

The `gemini-embedding-2-preview` and `gemini-embedding-001` models support reduced output dimensionality via Matryoshka representation learning. To request a specific dimension, append `:NNN` to the model name:

```
gemini-embedding-2-preview:768
```

When a `:NNN` suffix is present, the driver:
1. Strips the suffix to get the bare model name for the API call
2. Adds `"outputDimensionality": NNN` to each request object in the batch

Example request with Matryoshka dimensionality:
```json
{
  "requests": [
    {
      "model": "models/gemini-embedding-2-preview",
      "content": {"parts": [{"text": "..."}]},
      "taskType": "RETRIEVAL_DOCUMENT",
      "outputDimensionality": 768
    }
  ]
}
```

## CLI Usage Examples

```bash
# Embed with default model (gemini-embedding-2-preview, 3072-dim)
search cc fts embed run --input ./docs/ --driver gemini

# Embed with preview model at full 3072 dimensions (explicit)
search cc fts embed run --input ./docs/ --driver gemini --model gemini-embedding-2-preview

# Embed with preview model at reduced 768 dimensions (Matryoshka)
search cc fts embed run --input ./docs/ --driver gemini --model gemini-embedding-2-preview:768

# Embed with stable model
search cc fts embed run --input ./docs/ --driver gemini --model gemini-embedding-001

# List available embedding models
search cc fts embed models
```

## Rate Limits

- `text-embedding-004` free tier: **1500 RPM** (requests per minute)
- `gemini-embedding-2-preview` free tier: **5 RPM / 100 RPD**
- Batch size of 100 texts per call means effective throughput of up to 150,000 texts/minute for `text-embedding-004` at the free tier
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
- `gemini-embedding-2-preview`: embed a short text, verify dimension = 3072
- `gemini-embedding-2-preview:512`: embed a short text, verify dimension = 512

Run with:
```bash
GEMINI_API_KEY=... go test -tags integration ./pkg/embed/gemini/...
```

### CLI Smoke Test

```bash
echo "hello world" | search cc fts embed run --driver gemini --model text-embedding-004
```

Verify output contains a float vector of length 768.
