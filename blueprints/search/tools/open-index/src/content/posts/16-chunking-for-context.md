---
slug: chunking-for-context
title: "Chopping Web Pages into Context"
date: 2026-03-04
summary: "Web pages don't fit in context windows. How to cut them up without losing meaning."
tags: [ai, search]
---

A typical web page is 8,000 tokens. An embedding model accepts 512. An LLM might take 128K, but stuffing 16 raw pages into a single prompt gives terrible results -- the model drowns in noise, loses track of what's relevant, and hallucinates connections between unrelated paragraphs.

The quality of your chunks determines the quality of your search results. Bad chunks mean bad retrieval. Bad retrieval means bad answers. This is the most underrated problem in the entire RAG pipeline, and most teams spend weeks tuning their retrieval model while feeding it garbage boundaries.

## Three ways to cut text

There are three mainstream strategies for turning a web page into model-sized pieces. Each trades off simplicity, speed, and quality differently.

<table>
  <thead>
    <tr>
      <th>Strategy</th>
      <th>How it splits</th>
      <th>Speed</th>
      <th>Quality</th>
      <th>Best for</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Fixed-size</strong></td>
      <td>Every N tokens, with overlap</td>
      <td style="color:#4ade80">Fast</td>
      <td style="color:#fbbf24">Low</td>
      <td>Uniform prose, bulk pipelines</td>
    </tr>
    <tr>
      <td><strong>Paragraph-based</strong></td>
      <td>HTML structure (<code>&lt;p&gt;</code>, <code>&lt;h2&gt;</code>, <code>&lt;li&gt;</code>)</td>
      <td style="color:#4ade80">Fast</td>
      <td style="color:#4ade80">Good</td>
      <td>Web content (most pages)</td>
    </tr>
    <tr>
      <td><strong>Semantic</strong></td>
      <td>Embedding similarity between sentences</td>
      <td style="color:#fbbf24">Slow</td>
      <td style="color:#4ade80">Best</td>
      <td>High-value documents</td>
    </tr>
  </tbody>
</table>

## Fixed-size: simple, fast, dumb

Split every 256 or 512 tokens. Add 50 tokens of overlap between chunks. Done.

This works fine for uniform prose -- long articles, books, research papers where every paragraph is roughly the same density. It breaks badly on structured content. A chunk might start mid-sentence, split a code block in half, or cut a table between the header row and the data rows.

<pre><code><span style="color:#888">// What fixed-size chunking does to a code example:</span>

<span style="color:#e0e0e0">--- Chunk 1 (tokens 0-256) ---</span>
<span style="color:#4ade80">The server configuration requires three parameters:</span>
<span style="color:#60a5fa">func NewServer(host string, port int, opts ...Option) {</span>
<span style="color:#60a5fa">    s := &Server{</span>
<span style="color:#60a5fa">        host:    host,</span>

<span style="color:#e0e0e0">--- Chunk 2 (tokens 206-462) ---</span>
<span style="color:#60a5fa">        host:    host,</span>
<span style="color:#60a5fa">        port:    port,</span>
<span style="color:#60a5fa">        timeout: 30 * time.Second,</span>
<span style="color:#60a5fa">    }</span>
<span style="color:#4ade80">The timeout defaults to 30 seconds but can be</span>
<span style="color:#4ade80">overridden with WithTimeout().</span></code></pre>

Chunk 1 has half a function. Chunk 2 has the other half plus the explanation. Neither chunk makes sense alone. An embedding model will produce mediocre vectors for both, because neither captures a complete thought.

## Paragraph-based: let HTML do the work

HTML already encodes document structure. A `<p>` tag says "this is a paragraph." An `<h2>` says "new section starts here." A `<pre><code>` says "this is a code block -- keep it together." A `<table>` says "these values are related."

The algorithm is straightforward:

1. Walk the DOM tree. Collect text from each block-level element (`<p>`, `<h2>`, `<li>`, `<pre>`, `<table>`, `<blockquote>`).
2. If an element's text fits in the chunk budget, keep it as one unit.
3. If it's too large, split at sentence boundaries (period + space).
4. If an element is too small (under 50 tokens), merge it with the next element.
5. Prepend the nearest `<h2>` heading to each chunk as a section label.

Step 5 is the critical trick. A chunk that says "The timeout defaults to 30 seconds" means nothing in isolation. A chunk that says "**Server Configuration** -- The timeout defaults to 30 seconds" tells the embedding model what this text is about. Section prefixing turns fragments back into self-contained passages.

<div class="note">
  <strong>Why block-level elements?</strong> Inline elements (<code>&lt;a&gt;</code>, <code>&lt;strong&gt;</code>, <code>&lt;em&gt;</code>) don't represent logical boundaries. A bold word in the middle of a sentence isn't a split point. Block-level elements are the ones that carry structural meaning in HTML.
</div>

## Semantic chunking: accurate but expensive

Embed each sentence independently. Compute cosine similarity between consecutive sentence embeddings. When the similarity drops below a threshold -- say, 0.65 -- that's a topic shift. Split there.

This produces the best chunks. A paragraph that starts discussing server configuration and pivots to database connections will get split right at the pivot, even if there's no heading or paragraph break in the HTML.

The cost: you need an embedding pass just to decide where to chunk. For a single document, that's fine. For a billion crawled pages, it means running the embedding model twice -- once for boundary detection, once for the actual index embeddings. At 500 embeddings/second on a GPU, that doubles your pipeline from 70 days to 140 days. Semantic chunking is worth it for high-value documents. It's overkill for bulk crawl data.

## How much overlap?

Chunks need overlap or context dies at the boundaries. If a key sentence spans two chunks, the second chunk needs the tail end of the first or it starts with a dangling reference.

Too much overlap wastes storage and produces duplicate search results. Too little loses context. The sweet spot is 10-20% of chunk size.

<table>
  <thead>
    <tr>
      <th>Chunk size</th>
      <th>Overlap</th>
      <th>Overhead</th>
      <th>Trade-off</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>256 tokens</td>
      <td>25 tokens (10%)</td>
      <td>~10% storage</td>
      <td>Minimal context bleed, some boundary loss</td>
    </tr>
    <tr>
      <td>384 tokens</td>
      <td>50 tokens (13%)</td>
      <td>~13% storage</td>
      <td>Good balance for embeddings</td>
    </tr>
    <tr>
      <td>512 tokens</td>
      <td>75 tokens (15%)</td>
      <td>~15% storage</td>
      <td>Max embedding window, solid context</td>
    </tr>
    <tr>
      <td>2048 tokens</td>
      <td>200 tokens (10%)</td>
      <td>~10% storage</td>
      <td>LLM context chunks, low overhead</td>
    </tr>
  </tbody>
</table>

<div class="note">
  <strong>Overlap applies differently per strategy.</strong> Fixed-size chunking always needs overlap because it splits blind. Paragraph-based chunking often doesn't -- if you split at natural boundaries, the context is already self-contained. When paragraphs do get split (large blocks exceeding the budget), overlap at sentence boundaries preserves meaning.
</div>

## Why HTML structure matters more than you think

Raw text throws away critical information. Strip the tags from a web page and you've lost the signal that tells you where sections start, which text is a heading, what's a code example, and which values belong together in a table.

Consider a page with a `<table>` of API parameters. In raw text, that becomes:

```
Name Type Default Description timeout int 30 Request timeout in seconds retries int 3 Number of retry attempts
```

Good luck chunking that. The column headers and row values are indistinguishable from prose. With HTML structure preserved, the chunker knows the entire `<table>` is one semantic unit and keeps it intact.

Same principle applies to `<pre><code>` blocks. A code example split across two chunks produces two meaningless fragments. The HTML tag tells the chunker: don't split this. If the code block exceeds the chunk budget, it gets its own chunk at full size -- slightly over budget is better than two broken halves.

## What we're building

The OpenIndex chunker uses paragraph-based chunking with HTML-aware boundaries. The configuration:

- **Default chunk size for embeddings:** 384 tokens. Leaves headroom below the 512-token model limit for the section prefix and special tokens.
- **Default chunk size for LLM context:** 2,048 tokens. Four chunks fill an 8K window nicely, with room for the system prompt and query.
- **Section prefixing:** The nearest `<h2>` heading gets prepended to every chunk. If there's no heading, the page `<title>` is used instead.
- **Atomic elements:** `<pre>`, `<code>`, `<table>`, and `<blockquote>` are never split internally.
- **Merge threshold:** Elements under 50 tokens get merged with their neighbor rather than becoming tiny standalone chunks.

## Does it actually matter?

Early benchmarks on a 10K-page sample from Common Crawl, measuring retrieval precision at top-10 against manually labeled queries:

<table>
  <thead>
    <tr>
      <th>Strategy</th>
      <th>Precision@10</th>
      <th>Chunking speed</th>
      <th>Index size (relative)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Fixed-size (512)</strong></td>
      <td>0.54</td>
      <td style="color:#4ade80">48K pages/s</td>
      <td>1.0x</td>
    </tr>
    <tr>
      <td><strong>Paragraph-based (384)</strong></td>
      <td><strong>0.71</strong></td>
      <td style="color:#4ade80">42K pages/s</td>
      <td>1.1x</td>
    </tr>
    <tr>
      <td><strong>Semantic (384)</strong></td>
      <td><strong>0.76</strong></td>
      <td style="color:#fbbf24">180 pages/s</td>
      <td>1.15x</td>
    </tr>
  </tbody>
</table>

Paragraph-based chunking gets 71% of queries right in the top 10, vs 54% for fixed-size. That's a 31% relative improvement from changing nothing about the embedding model, the vector index, or the retrieval algorithm. Just better boundaries.

Semantic chunking ekes out another 5 points but runs 230x slower. For a billion-page crawl, that difference in speed matters more than the difference in precision. Paragraph-based wins on the only metric that matters at scale: quality per compute dollar.

<div class="note">
  <strong>The punchline:</strong> chunking isn't glamorous. Nobody writes blog posts about text splitting. But it sits between your crawler and your embedding model, and every downstream component inherits its mistakes. Get the boundaries wrong and no amount of model tuning fixes the retrieval. Get them right -- especially by using the HTML structure that's already there -- and everything downstream improves for free.
</div>
