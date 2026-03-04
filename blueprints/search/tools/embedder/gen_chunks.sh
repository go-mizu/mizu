#!/bin/bash
# Generate test chunks from real Go source and markdown files in the repo
# Chunks ~500 chars with 200-char overlap, targeting 10K chunks

set -e
REPO_ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
OUTPUT="$(dirname "$0")/testdata/chunks.jsonl"
mkdir -p "$(dirname "$OUTPUT")"

python3 -c "
import json, os, glob

repo = '$REPO_ROOT'
max_chars = 500
overlap = 200
target = 10000

# Collect .go and .md files
files = []
for ext in ('**/*.go', '**/*.md'):
    files.extend(glob.glob(os.path.join(repo, ext), recursive=True))

# Skip vendor, node_modules, .git
files = [f for f in files if '/vendor/' not in f and '/node_modules/' not in f and '/.git/' not in f]
files.sort()

def chunk_text(text, max_chars, overlap):
    chunks = []
    start = 0
    while start < len(text):
        end = start + max_chars
        chunk = text[start:end].strip()
        if chunk:
            chunks.append(chunk)
        start += max_chars - overlap
    return chunks

all_chunks = []
for fpath in files:
    try:
        with open(fpath, 'r', errors='replace') as f:
            text = f.read()
        if len(text) < 50:
            continue
        chunks = chunk_text(text, max_chars, overlap)
        all_chunks.extend(chunks)
        if len(all_chunks) >= target:
            break
    except Exception:
        continue

all_chunks = all_chunks[:target]

with open('$OUTPUT', 'w') as f:
    for chunk in all_chunks:
        f.write(json.dumps({'text': chunk}) + '\n')

print(f'Generated {len(all_chunks)} chunks from {len(files)} files to $OUTPUT')
avg_len = sum(len(c) for c in all_chunks) / len(all_chunks) if all_chunks else 0
print(f'Average chunk length: {avg_len:.0f} chars')
"
