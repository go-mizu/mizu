#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = ["datasets", "huggingface_hub"]
# ///

from datasets import load_dataset

# Test 1: Stream the full history without downloading everything first
print("Test 1: streaming...")
ds = load_dataset("open-index/hacker-news", split="train", streaming=True)
for i, item in enumerate(ds):
    if i == 0:
        print(f"  first item: id={item['id']}, type={item['type']}, title={repr(item['title'][:50])}")
    if i >= 2:
        break
print("  streaming OK")

# Test 2: Load a specific year into memory (use a small/fast year)
print("\nTest 2: load by data_files glob...")
ds2 = load_dataset(
    "open-index/hacker-news",
    data_files="data/2006/*.parquet",
    split="train",
)
print(f"  {len(ds2):,} items in 2006")

# Test 3: Load today's live blocks
print("\nTest 3: today config streaming...")
ds3 = load_dataset(
    "open-index/hacker-news",
    name="today",
    split="train",
    streaming=True,
)
for i, item in enumerate(ds3):
    if i == 0:
        print(f"  first today item: id={item['id']}")
    if i >= 0:
        break
print("  today OK")
