#!/bin/bash
set -e

MODEL_DIR="${HOME}/data/models"
mkdir -p "$MODEL_DIR"
cd "$MODEL_DIR"

echo "Downloading Gemma 3 models to $MODEL_DIR..."
echo ""

# Check if huggingface-cli is installed
if ! command -v huggingface-cli &> /dev/null; then
    echo "Error: huggingface-cli not found. Install with: pip install huggingface_hub"
    exit 1
fi

# Gemma 3 270M (Quick mode)
if [ ! -f "gemma-3-270m-it.gguf" ]; then
    echo "Downloading Gemma 3 270M..."
    huggingface-cli download google/gemma-3-270m-it-qat-q4_0-gguf gemma-3-270m-it-q4_0.gguf --local-dir . --local-dir-use-symlinks False
    mv gemma-3-270m-it-q4_0.gguf gemma-3-270m-it.gguf
else
    echo "Gemma 3 270M already exists, skipping..."
fi

# Gemma 3 1B (Deep mode)
if [ ! -f "gemma-3-1b-it.gguf" ]; then
    echo "Downloading Gemma 3 1B..."
    huggingface-cli download google/gemma-3-1b-it-qat-q4_0-gguf gemma-3-1b-it-q4_0.gguf --local-dir . --local-dir-use-symlinks False
    mv gemma-3-1b-it-q4_0.gguf gemma-3-1b-it.gguf
else
    echo "Gemma 3 1B already exists, skipping..."
fi

# Gemma 3 4B (Research mode)
if [ ! -f "gemma-3-4b-it.gguf" ]; then
    echo "Downloading Gemma 3 4B..."
    huggingface-cli download google/gemma-3-4b-it-qat-q4_0-gguf gemma-3-4b-it-q4_0.gguf --local-dir . --local-dir-use-symlinks False
    mv gemma-3-4b-it-q4_0.gguf gemma-3-4b-it.gguf
else
    echo "Gemma 3 4B already exists, skipping..."
fi

echo ""
echo "Done! Models saved to $MODEL_DIR"
echo ""
echo "Models:"
ls -lh "$MODEL_DIR"/*.gguf 2>/dev/null || echo "No .gguf files found"
