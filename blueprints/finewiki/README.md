# FineWiki

FineWiki is a self-hosted, offline Wikipedia reader. It downloads a language dataset once, builds a local index, and serves fast article viewing and search from your machine.

## What this blueprint is for

FineWiki demonstrates a practical pattern for offline data products:
- download a large dataset in a portable format
- build a local index
- serve a simple web UI for search and reading

## Product references

FineWiki is similar in user experience to:
- the Wikipedia website, but offline
- offline encyclopedia readers and local knowledge bases

## How it works

At a high level:
- A dataset provides page content for a given language
- A local index is built for fast title lookup
- A web server renders pages and search results

## Key capabilities

- Offline reading after a one-time download
- Fast title search
- Article rendering in a simple web UI
- Separate data per language, so languages can be added or removed independently
- One executable for serving and basic management tasks

## Non-goals

- Full Wikipedia editing features
- Perfect parity with the online MediaWiki rendering pipeline
- Multi-node deployment
- Full-text search as the default for every language and dataset size

## Quick start

```bash
make build
finewiki import vi
finewiki serve vi
````

Open the server URL shown in the console output.

## Commands

FineWiki provides a small CLI surface:

* `finewiki import <lang>` downloads data for one language
* `finewiki serve <lang>` runs the reader for one language
* `finewiki list` shows available and installed languages

Run `finewiki --help` for the full flags and usage.

## Data layout

FineWiki stores data by language under a single base directory. Each language is isolated so you can manage them independently.

## Notes on datasets

FineWiki is designed around public dataset releases. Dataset format, size, and coverage may change over time. This blueprint treats the dataset as an external input and focuses on indexing, serving, and a predictable local workflow.

## Status

Active development. Folder structure aims to stay stable while the indexing approach, dataset details, and rendering evolve.
