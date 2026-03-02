---
slug: llm-search-ranking
title: "LLMs as Search Refiners"
date: 2026-03-08
summary: "BM25 gets you the first 100 results. Cross-encoders pick the best 10. LLMs explain why they matter."
tags: [ai, search]
---

BM25 runs in 2 milliseconds. It counts term frequency, applies inverse document frequency, and hands you 100 results ranked by how well they match the query string. It's fast. It's deterministic. It's been the backbone of search for three decades.

It's also wrong about 30% of the time. The 8th result should be 2nd. The 1st result is barely relevant -- it just happens to repeat the query terms more often. Language models can fix this. But they're 100-1000x slower. The question isn't whether to use them. It's where in the pipeline they earn their latency.

---

## Four stages, three chances for an LLM

The full pipeline looks like this:

<pre><code>  <span style="color:#60a5fa">User Query</span>
       |
       v
  <span style="color:#fbbf24">(1) Query Expansion</span>        <span style="color:#888">+100ms   small LM</span>
       |
       v
  <span style="color:#4ade80">(2) Retrieval</span>              <span style="color:#888">+2ms     BM25 + Vector</span>
       |  top-100 candidates
       v
  <span style="color:#fbbf24">(3) Cross-Encoder Rerank</span>  <span style="color:#888">+200ms   ms-marco-MiniLM</span>
       |  top-10 reranked
       v
  <span style="color:#fbbf24">(4) Answer Synthesis</span>      <span style="color:#888">+2000ms  LLM (RAG)</span>
       |
       v
  <span style="color:#60a5fa">Final Answer</span></code></pre>

Stages 1, 3, and 4 use language models. Stage 2 is classical retrieval -- Tantivy for BM25, Vald for vectors. Each LLM layer adds latency. Each adds quality. The user decides how many layers to run.

## Query expansion: making short queries less terrible

Someone types "JS frameworks." They mean JavaScript frameworks -- React, Vue, Angular, Svelte, Next.js. BM25 sees two tokens and retrieves pages containing "JS" and "frameworks." It misses every page that says "JavaScript" instead.

A small language model (even something as cheap as a prompted GPT-3.5) rewrites the query with related terms. The expanded query hits more relevant documents. Recall goes up.

<table>
  <thead>
    <tr>
      <th>Query</th>
      <th>BM25 Only (top-10 relevant)</th>
      <th>With Expansion (top-10 relevant)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>JS frameworks</code></td>
      <td>4/10</td>
      <td style="color:#4ade80"><strong>7/10</strong></td>
    </tr>
    <tr>
      <td><code>ML deployment</code></td>
      <td>5/10</td>
      <td style="color:#4ade80"><strong>8/10</strong></td>
    </tr>
    <tr>
      <td><code>container networking</code></td>
      <td>6/10</td>
      <td style="color:#4ade80"><strong>8/10</strong></td>
    </tr>
  </tbody>
</table>

Cost: ~100ms. The expansion model doesn't need to be large -- it just needs to know that "JS" means "JavaScript" and that React is a framework. Short, ambiguous queries benefit the most. Longer, specific queries ("React useEffect cleanup function memory leak") already have enough signal for BM25 to work with.

## Cross-encoder reranking: the biggest bang for the latency

BM25 retrieves 100 candidates. A cross-encoder scores each (query, document) pair. This is where the real quality jump happens.

The model -- `ms-marco-MiniLM-L-12-v2` -- sees the full text of both query and document *simultaneously*. It doesn't count term overlap. It reads both passages together and produces a relevance score. This captures things BM25 can't: paraphrasing, negation, context-dependent meaning.

Results on standard benchmarks: **10-30% improvement in nDCG@10**. That's the difference between a mediocre results page and a good one.

<table>
  <thead>
    <tr>
      <th>Retrieval Method</th>
      <th>nDCG@10</th>
      <th>Latency</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>BM25 only</td>
      <td>0.42</td>
      <td>2ms</td>
      <td>Baseline</td>
    </tr>
    <tr>
      <td>BM25 + Vector (hybrid)</td>
      <td>0.49</td>
      <td>~10ms</td>
      <td>Score fusion helps</td>
    </tr>
    <tr>
      <td>BM25 + Cross-encoder rerank</td>
      <td style="color:#4ade80"><strong>0.56</strong></td>
      <td>~200ms</td>
      <td>Biggest quality jump</td>
    </tr>
    <tr>
      <td>Hybrid + Cross-encoder rerank</td>
      <td style="color:#4ade80"><strong>0.59</strong></td>
      <td>~210ms</td>
      <td>Best of everything</td>
    </tr>
  </tbody>
</table>

200ms for a 33% quality improvement over raw BM25. That's the best trade in the entire pipeline.

<div class="note">
  <strong>Why cross-encoders beat bi-encoders for reranking.</strong> A bi-encoder (like multilingual-e5-large) encodes query and document independently into separate vectors, then computes cosine similarity. Fast, but it can't capture fine-grained interactions between query and document tokens. A cross-encoder encodes them together -- the query tokens attend to the document tokens and vice versa. It sees everything at once. You can't pre-compute cross-encoder scores for the full index (too expensive -- you'd need to score every query against every document). But for reranking 100 candidates? Perfect.
</div>

## Answer synthesis: when you want sentences, not links

The final layer. Take the top-5 reranked chunks, feed them to an LLM, generate a natural language answer with citations. This is retrieval-augmented generation. RAG.

It works well for complex questions: "Explain the CAP theorem trade-offs in distributed search." The LLM reads the retrieved chunks, synthesizes the key points, cites sources. The user gets an answer instead of ten blue links.

It doesn't work well for navigational queries: "OpenIndex GitHub repo." The user wants a URL, not a paragraph. Sending that to an LLM wastes 2 seconds to produce a worse experience than just returning the link.

Latency: 1-3 seconds. Worth it selectively.

## The latency budget

Everything comes down to a budget. Each layer has a cost and a payoff:

<table>
  <thead>
    <tr>
      <th>Layer</th>
      <th>Latency Added</th>
      <th>Quality Gain</th>
      <th>When to Use</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>BM25 retrieval</strong></td>
      <td><span style="color:#4ade80">2ms</span></td>
      <td>Baseline</td>
      <td>Always</td>
    </tr>
    <tr>
      <td><strong>Query expansion</strong></td>
      <td><span style="color:#fbbf24">+100ms</span></td>
      <td>+15% recall</td>
      <td>Short/ambiguous queries</td>
    </tr>
    <tr>
      <td><strong>Cross-encoder rerank</strong></td>
      <td><span style="color:#fbbf24">+200ms</span></td>
      <td>+25% precision</td>
      <td>Most queries</td>
    </tr>
    <tr>
      <td><strong>LLM synthesis</strong></td>
      <td><span style="color:#e0e0e0">+2000ms</span></td>
      <td>Natural language answer</td>
      <td>Exploratory/complex queries</td>
    </tr>
  </tbody>
</table>

The system picks how many layers to run based on query type. A navigational query ("twitter.com") skips everything -- just look up the URL. A known-item query ("Python argparse documentation") runs BM25 alone. An exploratory query ("how do distributed databases handle consistency") benefits from the full stack.

## Not every query needs an LLM

Query classification determines which layers fire. Three categories:

**Navigational** -- the user wants a specific site. "github.com", "youtube", "hacker news". BM25 with URL matching. No expansion, no reranking, no synthesis. Sub-5ms.

**Known-item** -- the user knows what they're looking for. "Python argparse documentation", "RFC 7231", "React useEffect hook". BM25 handles this well. Maybe cross-encoder reranking if the top results look ambiguous. Under 200ms.

**Exploratory** -- the user has a question, not a destination. "How do search engines handle multilingual content", "trade-offs between HNSW and IVF for ANN search". This is where the full pipeline pays off. Expansion catches synonym gaps. Reranking surfaces the most relevant passages. Synthesis gives a coherent answer. 2-3 seconds total.

<div class="note">
  <strong>Classification itself can be fast.</strong> A simple heuristic gets you most of the way: if the query looks like a URL, it's navigational. If it's 1-3 specific terms, it's known-item. If it contains question words or is longer than 5 tokens, it's exploratory. No model needed for the classifier -- save the LLM budget for the stages that need it.
</div>

## Where this stands

Honest status:

- **BM25 retrieval**: in design (Tantivy). Architecture defined, schema prototyped, waiting on indexing pipeline.
- **Vector retrieval**: in design (Vald). Embedding model selected, GPU pipeline benchmarked.
- **Cross-encoder reranking**: model selected (`ms-marco-MiniLM-L-12-v2`), inference benchmarked at ~2ms per (query, doc) pair on GPU. 100 candidates = 200ms.
- **Query expansion**: prototyped with prompted GPT-3.5. Works well for short queries. Needs evaluation on a larger query set.
- **LLM synthesis**: depends on the retrieval pipeline. Can't generate answers from chunks that don't exist yet.

The LLM layers sit on top of the search stack. They can't ship until the stack beneath them does. Tantivy first, then Vald, then the reranking and synthesis layers. Each layer is independently useful -- cross-encoder reranking improves results even without synthesis, and query expansion helps even without vector search.

<div class="note">
  <strong>The latency numbers are estimates from benchmarks, not production measurements.</strong> Real latency depends on hardware, batch size, network hops, and how much text the cross-encoder has to read. We'll publish real numbers once the retrieval pipeline is running and there are actual queries to rerank.
</div>

BM25 gets you the first 100 results in 2 milliseconds. A cross-encoder picks the best 10 in 200 more. An LLM explains why they matter in another 2 seconds. Each layer is optional. Each earns its latency on a different kind of query. The trick isn't turning everything on -- it's knowing when each layer pays for itself.
