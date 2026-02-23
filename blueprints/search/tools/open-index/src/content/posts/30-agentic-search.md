---
slug: agentic-search
title: "Agentic Search — When One Query Isn't Enough"
date: 2026-03-20
summary: "Some questions can't be answered with a single search. An agent decomposes them, plans retrieval across multiple backends, and synthesizes an answer. This is search beyond keywords."
tags: [ai, search]
---

Try this question: "Compare the AI research output of universities in California vs. Massachusetts, focusing on papers published by faculty who previously worked at major tech companies."

Type that into Google. Or Bing. Or any search engine. You'll get ten blue links about university rankings. None of them answer the question. Because it isn't a search query. It's a research task -- one that requires identifying universities in both states, finding affiliated researchers, checking their employment history, finding their published papers, and aggregating the results by state.

No single query handles this. An agent does -- by decomposing it into a plan and executing each step against the right backend.

## Decomposition: breaking questions into sub-queries

The agent's first job is figuring out that the question isn't atomic. It reads the natural language input and produces a plan -- a sequence of concrete queries, each feeding results into the next.

Here's how the California-vs-Massachusetts question breaks down:

**Step 1: Find universities in both states**

<pre><code><span style="color:#888">-- GRAPH query: educational organizations in CA and MA</span>
<span style="color:#60a5fa">GRAPH</span> <span style="color:#e0e0e0">Organization</span>
<span style="color:#60a5fa">WHERE</span> <span style="color:#e0e0e0">type</span> = <span style="color:#4ade80">'educational'</span>
  <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">location</span> <span style="color:#60a5fa">IN</span> (<span style="color:#4ade80">'California'</span>, <span style="color:#4ade80">'Massachusetts'</span>)

<span style="color:#888">-- Under the hood, DuckDB SQL:</span>
<span style="color:#60a5fa">SELECT DISTINCT</span> t1.subject <span style="color:#60a5fa">AS</span> university, t2.object <span style="color:#60a5fa">AS</span> state
<span style="color:#60a5fa">FROM</span> triples t1
<span style="color:#60a5fa">JOIN</span> triples t2 <span style="color:#60a5fa">ON</span> t2.subject = t1.subject
<span style="color:#60a5fa">WHERE</span> t1.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> t1.object = <span style="color:#4ade80">'oi:EducationalOrganization'</span>
  <span style="color:#60a5fa">AND</span> t2.predicate = <span style="color:#4ade80">'oi:locatedIn'</span>
  <span style="color:#60a5fa">AND</span> t2.object <span style="color:#60a5fa">IN</span> (<span style="color:#4ade80">'entity/california'</span>, <span style="color:#4ade80">'entity/massachusetts'</span>)</code></pre>

Result: 47 universities (31 CA, 16 MA).

**Step 2: Find affiliated researchers**

<pre><code><span style="color:#888">-- GRAPH query: people affiliated with those universities</span>
<span style="color:#60a5fa">GRAPH</span> <span style="color:#e0e0e0">Person</span> <span style="color:#60a5fa">LINKED_TO</span> <span style="color:#e0e0e0">Organization</span>
<span style="color:#60a5fa">WHERE</span> <span style="color:#e0e0e0">Organization.id</span> <span style="color:#60a5fa">IN</span> (<span style="color:#888">... step 1 results ...</span>)
  <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">predicate</span> = <span style="color:#4ade80">'oi:affiliatedWith'</span></code></pre>

Result: 1,340 researchers.

**Step 3: Filter to those with tech company history**

<pre><code><span style="color:#888">-- GRAPH query: which researchers were previously at tech companies?</span>
<span style="color:#60a5fa">SELECT DISTINCT</span> t1.subject <span style="color:#60a5fa">AS</span> researcher, t2.object <span style="color:#60a5fa">AS</span> tech_company
<span style="color:#60a5fa">FROM</span> triples t1
<span style="color:#60a5fa">JOIN</span> triples t2 <span style="color:#60a5fa">ON</span> t2.subject = t1.subject
<span style="color:#60a5fa">JOIN</span> triples t3 <span style="color:#60a5fa">ON</span> t3.subject = t2.object
<span style="color:#60a5fa">WHERE</span> t1.subject <span style="color:#60a5fa">IN</span> (<span style="color:#888">... step 2 results ...</span>)
  <span style="color:#60a5fa">AND</span> t2.predicate = <span style="color:#4ade80">'oi:previouslyAffiliatedWith'</span>
  <span style="color:#60a5fa">AND</span> t3.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> t3.object = <span style="color:#4ade80">'oi:TechCompany'</span></code></pre>

Result: 218 researchers with prior tech company affiliations.

**Step 4: Find their papers**

<pre><code><span style="color:#888">-- SEARCH + GRAPH: papers authored by those researchers</span>
<span style="color:#60a5fa">SEARCH</span> <span style="color:#4ade80">"artificial intelligence"</span>
<span style="color:#60a5fa">GRAPH</span> <span style="color:#e0e0e0">Person</span> <span style="color:#60a5fa">LINKED_TO</span> <span style="color:#e0e0e0">Document</span>
<span style="color:#60a5fa">WHERE</span> <span style="color:#e0e0e0">Person.id</span> <span style="color:#60a5fa">IN</span> (<span style="color:#888">... step 3 results ...</span>)
  <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">predicate</span> = <span style="color:#4ade80">'oi:authorOf'</span></code></pre>

Result: 3,900 papers.

**Step 5: Aggregate by state**

<pre><code><span style="color:#888">-- WHERE clause: aggregate and compare</span>
<span style="color:#60a5fa">SELECT</span> state,
       <span style="color:#60a5fa">COUNT</span>(<span style="color:#60a5fa">DISTINCT</span> researcher) <span style="color:#60a5fa">AS</span> researchers,
       <span style="color:#60a5fa">COUNT</span>(<span style="color:#60a5fa">DISTINCT</span> paper) <span style="color:#60a5fa">AS</span> papers,
       <span style="color:#60a5fa">ROUND</span>(<span style="color:#60a5fa">COUNT</span>(paper) * <span style="color:#fbbf24">1.0</span> / <span style="color:#60a5fa">COUNT</span>(<span style="color:#60a5fa">DISTINCT</span> researcher), <span style="color:#fbbf24">1</span>) <span style="color:#60a5fa">AS</span> papers_per_researcher
<span style="color:#60a5fa">FROM</span> results
<span style="color:#60a5fa">GROUP BY</span> state</code></pre>

Five steps. Three different backend types (GRAPH, SEARCH, WHERE). One answer. The agent decided the order, fed intermediate results forward, and produced a comparison the user could actually read. No search engine does this today.

## Planning strategies

Not every complex question decomposes the same way. Three strategies, each with different trade-offs:

**Sequential** -- each step depends on the previous one. The university example above is sequential: you can't find researchers (step 2) until you have universities (step 1). Simple to implement, easy to reason about, but slow. Every step waits for the previous one to finish.

**Parallel fan-out** -- independent sub-queries run simultaneously. If someone asks "Compare DuckDB vs. PostgreSQL for analytics workloads," the agent can search for DuckDB benchmarks and PostgreSQL benchmarks at the same time. Neither depends on the other. The synthesis step waits for both, but the retrieval runs in parallel.

**Iterative refinement** -- start broad, narrow based on results. Step 2 above returned 1,340 researchers. That's too many to pass directly to the next query. The agent could refine: "Filter to researchers with >5 publications in the last 3 years." This isn't planned ahead of time -- it's a decision the agent makes *after* seeing intermediate results.

<table>
  <thead>
    <tr>
      <th>Strategy</th>
      <th>Latency</th>
      <th>Complexity</th>
      <th>Best For</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Sequential</strong></td>
      <td>Sum of all steps</td>
      <td>Low</td>
      <td>Dependency chains (find X, then filter by Y)</td>
    </tr>
    <tr>
      <td><strong>Parallel fan-out</strong></td>
      <td>Max of parallel branches</td>
      <td>Medium</td>
      <td>Comparisons, multi-faceted queries</td>
    </tr>
    <tr>
      <td><strong>Iterative refinement</strong></td>
      <td>Unpredictable</td>
      <td>High</td>
      <td>Exploratory questions, large result sets</td>
    </tr>
  </tbody>
</table>

In practice, most complex questions use a mix. The university example is mostly sequential but could parallelize steps 2 and 3 if the agent is clever -- find researchers and check tech company affiliations at the same time, then intersect the results.

## Multi-backend orchestration

This is where having OQL's four backends pays off. For a single complex question, the agent picks the right tool for each sub-query:

<pre><code><span style="color:#e0e0e0">Question: "Find recent articles about quantum computing</span>
<span style="color:#e0e0e0">          by researchers connected to IBM, and related</span>
<span style="color:#e0e0e0">          work they might have missed."</span>

<span style="color:#60a5fa">Step 1 — GRAPH (knowledge graph)</span>
  Find Person entities affiliated with IBM
  <span style="color:#888">→ 340 researchers</span>

<span style="color:#60a5fa">Step 2 — SEARCH (Tantivy, BM25)</span>
  <span style="color:#60a5fa">SEARCH</span> <span style="color:#4ade80">"quantum computing"</span>
  <span style="color:#60a5fa">WHERE</span> author <span style="color:#60a5fa">IN</span> (<span style="color:#888">... step 1 ...</span>)
    <span style="color:#60a5fa">AND</span> date > <span style="color:#4ade80">'2025-06-01'</span>
  <span style="color:#888">→ 45 articles with exact keyword matches</span>

<span style="color:#60a5fa">Step 3 — SIMILAR TO (Vald, vectors)</span>
  <span style="color:#60a5fa">SIMILAR TO</span> <span style="color:#4ade80">"quantum error correction topological codes"</span>
  <span style="color:#60a5fa">WHERE</span> date > <span style="color:#4ade80">'2025-06-01'</span>
  <span style="color:#888">→ 30 semantically related articles (some never say "quantum computing")</span>

<span style="color:#60a5fa">Step 4 — WHERE (DuckDB, metadata filter)</span>
  Remove duplicates, rank by combined BM25 + vector score
  Filter to content_type = <span style="color:#4ade80">'text/html'</span>, http_status = <span style="color:#fbbf24">200</span>
  <span style="color:#888">→ 58 unique articles, ranked</span></code></pre>

Four backends, one answer. SEARCH found the obvious keyword matches. SIMILAR TO found the papers about "topological qubits" and "surface code decoders" that never use the phrase "quantum computing" but are deeply relevant. GRAPH gave us the IBM connection. WHERE cleaned up the metadata.

No single backend could answer this question alone. The agent decides which backend handles which part -- not through rules, but by reasoning about what each sub-query needs.

<div class="note">
  <strong>The backend selection isn't hardcoded.</strong> The agent doesn't follow a script that says "always use GRAPH for entity lookups." It reads the sub-query, considers what kind of retrieval is needed (keyword match? semantic similarity? relationship traversal? metadata filter?), and picks the right tool. A different question might use GRAPH for three steps and never touch SIMILAR TO.
</div>

## Intermediate state management

By step 3 of the university example, the agent has accumulated a list of 218 researchers. By step 4, it has 3,900 papers. That state has to live somewhere so later steps can reference it. Three options:

**Option A: In-context.** Stuff intermediate results directly into the LLM prompt. The simplest approach -- the model sees everything it's produced so far.

Problem: token consumption. 218 researcher IDs at ~15 tokens each is 3,270 tokens. 3,900 paper records would be 40,000+ tokens. The context window fills fast, and the model starts forgetting the original question.

**Option B: DuckDB temp tables.** Write intermediate results to temporary tables and query them in later steps.

<pre><code><span style="color:#888">-- Step 3: store filtered researchers</span>
<span style="color:#60a5fa">CREATE TEMP TABLE</span> filtered_researchers <span style="color:#60a5fa">AS</span>
<span style="color:#60a5fa">SELECT DISTINCT</span> t1.subject <span style="color:#60a5fa">AS</span> researcher,
       t2.object <span style="color:#60a5fa">AS</span> tech_company,
       t3.object <span style="color:#60a5fa">AS</span> university,
       t4.object <span style="color:#60a5fa">AS</span> state
<span style="color:#60a5fa">FROM</span> triples t1
<span style="color:#60a5fa">JOIN</span> triples t2 <span style="color:#60a5fa">ON</span> t2.subject = t1.subject
<span style="color:#60a5fa">JOIN</span> triples t3 <span style="color:#60a5fa">ON</span> t3.subject = t1.subject
<span style="color:#60a5fa">JOIN</span> triples t4 <span style="color:#60a5fa">ON</span> t4.subject = t3.object
<span style="color:#60a5fa">WHERE</span> t2.predicate = <span style="color:#4ade80">'oi:previouslyAffiliatedWith'</span>
  <span style="color:#60a5fa">AND</span> t3.predicate = <span style="color:#4ade80">'oi:affiliatedWith'</span>
  <span style="color:#60a5fa">AND</span> t4.predicate = <span style="color:#4ade80">'oi:locatedIn'</span>;

<span style="color:#888">-- Step 4: query the temp table instead of passing IDs in-context</span>
<span style="color:#60a5fa">SELECT</span> fr.state,
       fr.researcher,
       t.object <span style="color:#60a5fa">AS</span> paper_title
<span style="color:#60a5fa">FROM</span> filtered_researchers fr
<span style="color:#60a5fa">JOIN</span> triples t <span style="color:#60a5fa">ON</span> t.subject = fr.researcher
<span style="color:#60a5fa">WHERE</span> t.predicate = <span style="color:#4ade80">'oi:authorOf'</span>;

<span style="color:#888">-- Step 5: aggregate directly from temp table</span>
<span style="color:#60a5fa">SELECT</span> state,
       <span style="color:#60a5fa">COUNT</span>(<span style="color:#60a5fa">DISTINCT</span> researcher) <span style="color:#60a5fa">AS</span> researchers,
       <span style="color:#60a5fa">COUNT</span>(<span style="color:#60a5fa">DISTINCT</span> paper_title) <span style="color:#60a5fa">AS</span> papers
<span style="color:#60a5fa">FROM</span> researcher_papers
<span style="color:#60a5fa">GROUP BY</span> state;</code></pre>

The model only keeps a summary in context: "Stored 218 researchers in `filtered_researchers` temp table." The actual data lives in DuckDB, queryable without burning tokens.

**Option C: Hybrid.** Small result sets (under ~50 rows) stay in-context for fast reference. Anything larger goes to a temp table. The agent decides based on result size.

<table>
  <thead>
    <tr>
      <th>Approach</th>
      <th>Token Cost</th>
      <th>Implementation</th>
      <th>When to Use</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>In-context</strong></td>
      <td>High (scales with data)</td>
      <td>Simple (just append)</td>
      <td>Small result sets (&lt;50 rows)</td>
    </tr>
    <tr>
      <td><strong>Temp tables</strong></td>
      <td>Low (just the summary)</td>
      <td>Moderate (agent writes SQL)</td>
      <td>Large result sets, multi-step chains</td>
    </tr>
    <tr>
      <td><strong>Hybrid</strong></td>
      <td>Adaptive</td>
      <td>Moderate</td>
      <td>Default for most workloads</td>
    </tr>
  </tbody>
</table>

The temp table approach has a nice side effect: the agent's intermediate work becomes inspectable. You can query `filtered_researchers` yourself to verify what the agent found. Debugging a five-step agent pipeline is miserable when everything's trapped in an opaque context window. Temp tables make intermediate state visible.

## When agentic search beats traditional search

Not always. And that's the honest answer. Agentic search adds latency, complexity, and cost. For most queries, it's overkill.

<table>
  <thead>
    <tr>
      <th>Question Type</th>
      <th>Traditional Search</th>
      <th>Agentic Search</th>
      <th>Winner</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>"What is DuckDB?"</td>
      <td>Fast, accurate</td>
      <td>Overkill -- 15s for the same answer</td>
      <td><strong>Traditional</strong></td>
    </tr>
    <tr>
      <td>"Compare X vs Y across criteria"</td>
      <td>Two separate searches, manual merge</td>
      <td>Automatic decomposition + comparison</td>
      <td><strong>Agentic</strong></td>
    </tr>
    <tr>
      <td>"Find connections between A and B"</td>
      <td>Can't do this</td>
      <td>Graph traversal + relationship discovery</td>
      <td><strong>Agentic</strong></td>
    </tr>
    <tr>
      <td>"Summarize latest research on X"</td>
      <td>Returns documents, user reads them</td>
      <td>Reads, filters, synthesizes</td>
      <td><strong>Agentic</strong></td>
    </tr>
    <tr>
      <td>Navigation ("github.com")</td>
      <td>Instant redirect</td>
      <td>10 seconds of pointless reasoning</td>
      <td><strong>Traditional</strong></td>
    </tr>
    <tr>
      <td>"What happened last week in AI?"</td>
      <td>Returns news articles</td>
      <td>Aggregates across sources, deduplicates</td>
      <td><strong>Agentic</strong></td>
    </tr>
  </tbody>
</table>

The pattern: if the question can be answered by a single retrieval step, traditional search wins. If it requires multiple retrievals, filtering, cross-referencing, or synthesis, the agent earns its latency.

## The latency tax

Let's be honest about the cost. A BM25 keyword search returns in 2ms. An agentic search with five sub-queries takes 15-25 seconds. Here's where the time goes:

<pre><code><span style="color:#888">// Realistic latency breakdown for a 5-step agentic query</span>

<span style="color:#e0e0e0">Query decomposition (LLM plans the steps)</span>    <span style="color:#fbbf24">2-3s</span>
<span style="color:#e0e0e0">Sub-query 1: GRAPH lookup</span>                     <span style="color:#fbbf24">0.5s</span>  <span style="color:#888">DuckDB</span>
<span style="color:#e0e0e0">Sub-query 2: GRAPH lookup</span>                     <span style="color:#fbbf24">0.8s</span>  <span style="color:#888">DuckDB + LLM reasoning</span>
<span style="color:#e0e0e0">Sub-query 3: GRAPH + filter</span>                   <span style="color:#fbbf24">1.2s</span>  <span style="color:#888">DuckDB + LLM reasoning</span>
<span style="color:#e0e0e0">Sub-query 4: SEARCH (Tantivy)</span>                 <span style="color:#fbbf24">0.8s</span>  <span style="color:#888">Tantivy + LLM reasoning</span>
<span style="color:#e0e0e0">Sub-query 5: WHERE (aggregate)</span>                <span style="color:#fbbf24">0.3s</span>  <span style="color:#888">DuckDB</span>
<span style="color:#e0e0e0">Answer synthesis (LLM reads results, writes)</span>  <span style="color:#fbbf24">2-4s</span>
<span style="color:#e0e0e0">────────────────────────────────────────────────────</span>
<span style="color:#60a5fa">Total</span>                                          <span style="color:#fbbf24">8-11s</span> <span style="color:#888">(sequential)</span>
<span style="color:#60a5fa">Total with parallelism</span>                         <span style="color:#fbbf24">6-8s</span>  <span style="color:#888">(fan-out where possible)</span></code></pre>

The database queries themselves are fast -- DuckDB and Tantivy respond in single-digit milliseconds. The LLM reasoning between steps is where the time goes. Each step requires the model to read the previous result, decide what to do next, and generate the next query. That's 1-3 seconds per turn, and it adds up.

Compare to Google's "instant" results. The trade-off is depth vs. speed. Agentic search is for questions worth waiting for. Nobody wants to wait 10 seconds for "what's the weather." Everyone would wait 15 seconds for "compare the research output of universities across two states, filtered by faculty employment history."

<div class="note note-warn">
  <strong>The bottleneck isn't the database.</strong> DuckDB answers queries in milliseconds. Tantivy answers in 2ms. Vald answers in 5ms. The latency comes from LLM inference -- the model thinking about what to do next. Faster models directly translate to faster agentic search. As inference gets cheaper and faster (and it's getting cheaper and faster), the tax shrinks.
</div>

## Caching and pre-computation

The latency tax isn't fixed. Several strategies reduce it:

**Cache decomposition plans.** Questions about comparing X vs. Y have a common structure. Cache the plan template: "parallel fan-out on both entities, then merge results." When a similar question arrives, skip the decomposition step entirely and jump to execution. That's 2-3 seconds saved.

**Pre-compute common aggregations.** Entity counts by type, top domains by category, publication counts by organization -- these don't change often. Compute them on a schedule and store them in materialized views. When the agent needs "how many AI papers from Stanford?", it hits a pre-computed table instead of traversing the graph.

**Cache sub-query results with TTL.** "Find universities in California" doesn't change day to day. Cache the result for 24 hours. The next question about California universities skips the GRAPH query entirely.

**Route simple questions directly.** Not every question needs an agent. "What is DuckDB?" maps to a single `SEARCH "DuckDB"` query. Detect these and skip the planning step. A classifier (even a simple regex heuristic) can route 70% of questions to direct OQL execution.

<pre><code><span style="color:#888">// Decision routing</span>
<span style="color:#e0e0e0">if</span> question matches <span style="color:#4ade80">single-entity lookup</span>     → <span style="color:#60a5fa">direct OQL</span>     <span style="color:#888">(2ms)</span>
<span style="color:#e0e0e0">if</span> question matches <span style="color:#4ade80">cached plan template</span>    → <span style="color:#60a5fa">skip decompose</span>  <span style="color:#888">(5-8s)</span>
<span style="color:#e0e0e0">if</span> question matches <span style="color:#4ade80">novel complex query</span>     → <span style="color:#60a5fa">full agent loop</span> <span style="color:#888">(10-25s)</span></code></pre>

## The OQL + agent pipeline

Here's the complete architecture. The agent layer sits between the user and OQL, adding intelligence when needed and getting out of the way when it isn't.

<pre><code>
  <span style="color:#e0e0e0">Natural Language Question</span>
              |
              v
  <span style="color:#fbbf24">┌───────────────────────────┐</span>
  <span style="color:#fbbf24">│</span>  <strong>CLASSIFIER</strong>              <span style="color:#fbbf24">│</span>
  <span style="color:#fbbf24">│</span>  Simple or complex?      <span style="color:#fbbf24">│</span>
  <span style="color:#fbbf24">│</span>  <span style="color:#888">regex + small model</span>     <span style="color:#fbbf24">│</span>
  <span style="color:#fbbf24">└─────┬───────────┬─────────┘</span>
        |           |
     <span style="color:#4ade80">simple</span>      <span style="color:#60a5fa">complex</span>
        |           |
        v           v
  <span style="color:#4ade80">┌──────────┐</span>  <span style="color:#60a5fa">┌──────────────────────────┐</span>
  <span style="color:#4ade80">│ Translate│</span>  <span style="color:#60a5fa">│</span>  <strong>DECOMPOSE</strong>               <span style="color:#60a5fa">│</span>
  <span style="color:#4ade80">│ to OQL   │</span>  <span style="color:#60a5fa">│</span>  Break into sub-queries   <span style="color:#60a5fa">│</span>
  <span style="color:#4ade80">│ directly │</span>  <span style="color:#60a5fa">│</span>  Choose planning strategy <span style="color:#60a5fa">│</span>
  <span style="color:#4ade80">└────┬─────┘</span>  <span style="color:#60a5fa">└────────────┬─────────────┘</span>
       |                      |
       v                      v
  <span style="color:#888">╔══════════════════════════════════════════════╗</span>
  <span style="color:#888">║</span>              <strong>OQL EXECUTION LAYER</strong>             <span style="color:#888">║</span>
  <span style="color:#888">╠══════════╤══════════╤══════════╤═════════════╣</span>
  <span style="color:#888">║</span> <span style="color:#4ade80">SEARCH</span>   <span style="color:#888">│</span> <span style="color:#60a5fa">SIMILAR</span>  <span style="color:#888">│</span> <span style="color:#fbbf24">GRAPH</span>    <span style="color:#888">│</span> <span style="color:#e0e0e0">WHERE</span>       <span style="color:#888">║</span>
  <span style="color:#888">║</span> <span style="color:#4ade80">Tantivy</span>  <span style="color:#888">│</span> <span style="color:#60a5fa">Vald</span>     <span style="color:#888">│</span> <span style="color:#fbbf24">DuckDB</span>   <span style="color:#888">│</span> <span style="color:#e0e0e0">DuckDB</span>      <span style="color:#888">║</span>
  <span style="color:#888">║</span> <span style="color:#4ade80">BM25</span>     <span style="color:#888">│</span> <span style="color:#60a5fa">Vectors</span>  <span style="color:#888">│</span> <span style="color:#fbbf24">Triples</span>  <span style="color:#888">│</span> <span style="color:#e0e0e0">Metadata</span>    <span style="color:#888">║</span>
  <span style="color:#888">╚══════════╧══════════╧══════════╧═════════════╝</span>
       |                      |
       v                      v
  <span style="color:#e0e0e0">┌──────────┐</span>  <span style="color:#60a5fa">┌──────────────────────────┐</span>
  <span style="color:#e0e0e0">│ Return   │</span>  <span style="color:#60a5fa">│</span>  <strong>SYNTHESIZE</strong>              <span style="color:#60a5fa">│</span>
  <span style="color:#e0e0e0">│ results  │</span>  <span style="color:#60a5fa">│</span>  Merge partial results    <span style="color:#60a5fa">│</span>
  <span style="color:#e0e0e0">└──────────┘</span>  <span style="color:#60a5fa">│</span>  Generate final answer    <span style="color:#60a5fa">│</span>
               <span style="color:#60a5fa">└──────────────────────────┘</span>
</code></pre>

Simple questions take the fast path: classify, translate to OQL, execute, return. Total latency: under 500ms including the classification step.

Complex questions take the agent path: classify, decompose into sub-queries, execute each against the appropriate backend, accumulate intermediate state, synthesize a final answer. Total latency: 6-25 seconds depending on the number of steps and whether they can run in parallel.

The key insight: the agent layer is optional. It doesn't sit in the critical path for simple queries. When someone types "DuckDB tutorial," the classifier routes it to direct OQL translation and the agent never wakes up. The agent only activates for questions that genuinely require multi-step reasoning.

---

This is how search needs to evolve. Not every query is a research task, and not every search engine needs an agent. But the questions worth asking -- the ones that require decomposition, cross-referencing, and synthesis -- those can't be answered by keyword matching. They need a system that can plan, execute, observe, and refine.

OQL gives us the backends. The knowledge graph gives us entity relationships. Tantivy and Vald give us keyword and semantic search. DuckDB gives us metadata and aggregation. The agent layer ties them together -- deciding which backend to use for each sub-query, managing intermediate state, and synthesizing results into actual answers.

<div class="note">
  <strong>Status: architecture, not shipping code.</strong> OQL is in design (see <a href="/blog/oql-query-language">post 12</a>). The agent layer is further out on the roadmap. The search backends -- Tantivy, Vald, the knowledge graph, DuckDB -- those exist and are being built now. This post describes the architecture we're building toward, not code you can run today. The agent loop itself is straightforward (see <a href="/blog/agent-loop">post 26</a>). The hard part is building the tools it uses well enough that the agent can do useful work. We're building tools first.
</div>
