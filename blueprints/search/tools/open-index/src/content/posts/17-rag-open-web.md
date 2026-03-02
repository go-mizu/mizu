---
slug: rag-open-web
title: "RAG Against the Open Web"
date: 2026-03-05
summary: "Retrieval-augmented generation backed by an open web index. Grounded answers with citations, not hallucinations."
tags: [ai, search]
---

Ask GPT-4 "What percentage of .edu domains in Common Crawl return a 200 status?" and it'll confidently give you a number. That number will be wrong. Not because the model is broken, but because it doesn't have the data. It was never trained on your specific crawl run. It has no access to your index. It's guessing with conviction.

RAG fixes this. Retrieve the actual data first, then generate an answer grounded in evidence. The LLM becomes a synthesis engine, not an oracle.

## The pipeline in five steps

User question comes in. The system parses it into a search query, retrieves relevant chunks from the crawl corpus, injects those chunks into the LLM prompt, and generates an answer with citations back to source URLs.

<pre><code>  <span style="color:#60a5fa">User Question</span>
       |
       v
  <span style="color:#e0e0e0">┌──────────────────────────────────────────┐</span>
  <span style="color:#e0e0e0">│</span>  <span style="color:#fbbf24">Query Parser</span>                            <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Natural language → search query          <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">└──────────────┬───────────────────────────┘</span>
               |
               v
  <span style="color:#e0e0e0">┌──────────────────────────────────────────┐</span>
  <span style="color:#e0e0e0">│</span>  <span style="color:#4ade80">Hybrid Retrieval</span>                        <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Tantivy (BM25) + Vald (vector) → top-100<span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">└──────────────┬───────────────────────────┘</span>
               |
               v
  <span style="color:#e0e0e0">┌──────────────────────────────────────────┐</span>
  <span style="color:#e0e0e0">│</span>  <span style="color:#fbbf24">Cross-Encoder Reranking</span>                 <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Score each (query, chunk) → top-5       <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">└──────────────┬───────────────────────────┘</span>
               |
               v
  <span style="color:#e0e0e0">┌──────────────────────────────────────────┐</span>
  <span style="color:#e0e0e0">│</span>  <span style="color:#60a5fa">Prompt Construction</span>                     <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  System prompt + chunks + source URLs     <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">└──────────────┬───────────────────────────┘</span>
               |
               v
  <span style="color:#e0e0e0">┌──────────────────────────────────────────┐</span>
  <span style="color:#e0e0e0">│</span>  <span style="color:#4ade80">LLM Generation</span>                          <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Answer with inline citations [1][2]...  <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">└──────────────────────────────────────────┘</span></code></pre>

Every step is auditable. You can inspect the retrieved chunks, the reranking scores, and the final prompt. If the answer is wrong, you know exactly where the chain failed.

## Why a web corpus, not a knowledge base

Most RAG tutorials show the same setup: embed your company docs, query them, pipe results into GPT. That works for internal knowledge bases. It doesn't work for questions about the actual web.

"How many .edu sites still serve HTTP/1.0?" Nobody's internal knowledge base has that answer. A crawl index does. "What's the average page weight of news sites in 2026?" Same thing. The data exists in the crawled corpus -- millions of pages with HTTP headers, content types, response sizes, all sitting in DuckDB.

RAG against a web-scale index means the corpus is public, diverse, constantly changing, and much messier than a curated doc set. But it also means the system can answer questions that no static knowledge base covers.

## Hybrid retrieval catches what keyword search misses

Keyword search and vector search fail in complementary ways. BM25 finds exact terms but misses synonyms. Vector search finds semantic matches but fumbles on specific identifiers. The best retrieval uses both.

Example: "JavaScript date formatting libraries."

- **BM25 (Tantivy)** finds pages containing those exact words -- npm package docs, Stack Overflow answers, blog posts that literally say "date formatting."
- **Vector search (Vald)** finds pages about "handling timestamps in JS" and "Intl.DateTimeFormat API" that never use the phrase "date formatting."

Combine them and you get higher recall without sacrificing precision. RRF merges the two ranked lists by position, ignoring incompatible score scales.

<div class="note">
  <strong>This is the same hybrid retrieval that OQL will use.</strong> The <code>SEARCH</code> + <code>SIMILAR TO</code> clauses already target Tantivy and Vald respectively. RAG just consumes the combined result set instead of displaying it.
</div>

## Reranking: the 200ms that matters most

Hybrid retrieval returns ~100 candidates. Most aren't relevant enough to feed to an LLM. A cross-encoder reranker scores each (query, chunk) pair and keeps the top 5.

The cross-encoder differs from the bi-encoder used in vector search. A bi-encoder embeds query and document independently, then compares. A cross-encoder processes both together -- it sees the full interaction between query terms and document terms. More accurate, but too expensive to run on the whole corpus. That's why it only runs on 100 candidates.

<table>
  <thead>
    <tr>
      <th>Stage</th>
      <th>Candidates</th>
      <th>Latency</th>
      <th>Model</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Hybrid retrieval</strong></td>
      <td>Entire index &rarr; 100</td>
      <td><span style="color:#4ade80">20-50ms</span></td>
      <td>BM25 + ANN</td>
    </tr>
    <tr>
      <td><strong>Cross-encoder reranking</strong></td>
      <td>100 &rarr; 5</td>
      <td><span style="color:#fbbf24">100-300ms</span></td>
      <td>ms-marco-MiniLM-L-6</td>
    </tr>
    <tr>
      <td><strong>LLM generation</strong></td>
      <td>5 chunks &rarr; answer</td>
      <td><span style="color:#e0e0e0">1-3s</span></td>
      <td>Any instruction-tuned LLM</td>
    </tr>
  </tbody>
</table>

Total end-to-end: 1.2 to 3.5 seconds. The reranking step is the most impactful per millisecond -- it's what separates "good enough retrieval" from "the right five chunks."

## Building the prompt

The retrieved chunks go into the system message with their source URLs. The LLM gets explicit constraints:

<pre><code><span style="color:#60a5fa">SYSTEM:</span>
<span style="color:#e0e0e0">You are a research assistant. Answer the user's question using
ONLY the provided context. Cite sources using [1], [2], etc.
If the context doesn't contain enough information, say so.</span>

<span style="color:#60a5fa">CONTEXT:</span>
<span style="color:#4ade80">[1]</span> <span style="color:#888">https://httparchive.org/reports/state-of-the-web</span>
<span style="color:#e0e0e0">The median page weight for desktop sites reached 2.4 MB in
January 2026, with images accounting for 48% of total bytes...</span>

<span style="color:#4ade80">[2]</span> <span style="color:#888">https://almanac.httparchive.org/en/2025/page-weight</span>
<span style="color:#e0e0e0">News sites averaged 3.1 MB per page, 30% above the overall
median. JavaScript accounted for 22% of transfer size...</span>

<span style="color:#4ade80">[3]</span> <span style="color:#888">https://web.dev/articles/performance-budgets</span>
<span style="color:#e0e0e0">Performance budgets recommend keeping total page weight under
1.5 MB for mobile users on 3G connections...</span>

<span style="color:#60a5fa">USER:</span>
<span style="color:#e0e0e0">What's the average page weight of news sites?</span></code></pre>

Three rules make this work: (a) answer from context only, (b) cite sources, (c) admit ignorance when the context falls short. The third rule is the one most RAG implementations skip -- and it's the most important. An LLM that says "I don't know" when the evidence isn't there is more useful than one that fabricates a plausible answer.

## Citations are the whole point

Every claim in the generated answer links to a source URL from the index. The user can click through and verify. This is the fundamental difference between RAG and "just ask ChatGPT."

<table>
  <thead>
    <tr>
      <th>Property</th>
      <th>Direct LLM</th>
      <th>RAG + OpenIndex</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Source of truth</strong></td>
      <td>Training data (static, opaque)</td>
      <td>Crawl corpus (live, inspectable)</td>
    </tr>
    <tr>
      <td><strong>Citations</strong></td>
      <td>None or fabricated</td>
      <td>Real URLs from the index</td>
    </tr>
    <tr>
      <td><strong>Freshness</strong></td>
      <td>Months-old training cutoff</td>
      <td>Latest crawl run</td>
    </tr>
    <tr>
      <td><strong>Verifiability</strong></td>
      <td>Trust the model</td>
      <td>Click the link, read the page</td>
    </tr>
    <tr>
      <td><strong>Failure mode</strong></td>
      <td>Confident hallucination</td>
      <td>"Context insufficient" + partial results</td>
    </tr>
  </tbody>
</table>

Provenance turns an AI answer from "probably right" into "here's where we got this." That matters for any use case where being wrong has consequences.

## The latency problem

Users expect search results in under 2 seconds. RAG blows past that budget. Retrieval is fast (20-50ms). Reranking adds 100-300ms. LLM generation takes 1-3 seconds. Total: 1.5-3.5 seconds, and that's optimistic.

Three strategies to keep it usable:

1. **Stream the response.** Show retrieval results immediately. Start streaming the LLM output as tokens arrive. The user reads the first sentence while the model is still generating the third.
2. **Precompute popular queries.** Cache the top 10,000 query patterns with pre-retrieved chunks. Skip retrieval and reranking entirely for cache hits.
3. **Tiered response.** Return the raw search results in 50ms. Show the AI-generated summary when it's ready, 2-3 seconds later. The user has something useful immediately.

<div class="note">
  <strong>Streaming changes the perceived latency.</strong> A 3-second response that appears all at once feels slow. A response that starts appearing after 400ms and fills in over 3 seconds feels fast. Time-to-first-token matters more than total generation time.
</div>

## Where this stands

Honest status:

- **Retrieval pipeline** -- in design. Depends on Tantivy (full-text) and Vald (vector) shipping first. The DuckDB metadata layer already works.
- **Reranking** -- prototyped with ms-marco-MiniLM-L-6 cross-encoder. Works on small candidate sets. Hasn't been benchmarked at production scale.
- **Prompt templates** -- drafted. The system prompt structure, citation format, and refusal behavior are defined.
- **End-to-end RAG** -- post-search deployment. The retrieval backends need to exist before we can build on top of them.

<div class="note">
  <strong>This isn't a product announcement.</strong> RAG is the natural layer above search, but it can't ship until the search layer underneath it works. Tantivy and Vald come first. Once hybrid retrieval returns ranked results reliably, wiring in reranking and an LLM is straightforward engineering, not research.
</div>

The crawl corpus already has the data. The storage layer already holds it. The retrieval backends are in progress. RAG is the layer that turns "here are 50 relevant pages" into "here's the answer, and here's where we found it." Grounded answers with citations. That's the goal.
