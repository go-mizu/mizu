#!/bin/bash
set -e

MODEL_DIR="${HOME}/data/models"
mkdir -p "$MODEL_DIR"
cd "$MODEL_DIR"

echo "========================================"
echo "  LLM Model Downloader"
echo "========================================"
echo ""
echo "Download directory: $MODEL_DIR"
echo ""

# Function to download with progress
download_model() {
    local url="$1"
    local output="$2"
    local description="$3"

    if [ -f "$output" ]; then
        echo "✓ $description already exists, skipping..."
        return 0
    fi

    echo "→ Downloading $description..."
    echo "  URL: $url"
    echo ""

    # Use curl with progress bar, resume support, and follow redirects
    curl -L --progress-bar -o "${output}.tmp" -C - "$url"
    mv "${output}.tmp" "$output"

    echo ""
    echo "✓ $description downloaded successfully"
    echo ""
}

# Hugging Face base URL
HF_BASE="https://huggingface.co"

# Gemma 3 270M (Quick mode) - ~180MB
download_model \
    "${HF_BASE}/google/gemma-3-270m-it-qat-q4_0-gguf/resolve/main/gemma-3-270m-it-q4_0.gguf" \
    "gemma-3-270m-it.gguf" \
    "Gemma 3 270M (Quick mode, ~180MB)"

# Gemma 3 1B (Deep mode) - ~600MB
download_model \
    "${HF_BASE}/google/gemma-3-1b-it-qat-q4_0-gguf/resolve/main/gemma-3-1b-it-q4_0.gguf" \
    "gemma-3-1b-it.gguf" \
    "Gemma 3 1B (Deep mode, ~600MB)"

# Gemma 3 4B (Research mode) - ~2.5GB
download_model \
    "${HF_BASE}/google/gemma-3-4b-it-qat-q4_0-gguf/resolve/main/gemma-3-4b-it-q4_0.gguf" \
    "gemma-3-4b-it.gguf" \
    "Gemma 3 4B (Research mode, ~2.5GB)"

# GPT-OSS 20B (Large reasoning model) - ~12GB
download_model \
    "${HF_BASE}/ggml-org/gpt-oss-20b-GGUF/resolve/main/gpt-oss-20b-mxfp4.gguf" \
    "gpt-oss-20b.gguf" \
    "GPT-OSS 20B (Reasoning mode, ~12GB)"

echo "========================================"
echo "  Download Complete!"
echo "========================================"
echo ""
echo "Models in $MODEL_DIR:"
echo ""
ls -lh "$MODEL_DIR"/*.gguf 2>/dev/null | awk '{print "  " $9 " (" $5 ")"}'
echo ""
echo "To start the servers:"
echo "  cd docker/llamacpp && docker compose up -d"
echo ""
