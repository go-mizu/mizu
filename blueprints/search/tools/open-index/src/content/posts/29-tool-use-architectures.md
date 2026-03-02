---
slug: tool-use-architectures
title: "Tool-Use Architectures — Teaching Agents to Call Functions"
date: 2026-03-19
summary: "An LLM generates text. A tool turns that text into action. The architecture connecting them — schemas, execution, error recovery — is where agents get interesting."
tags: [ai, data]
---

An LLM can't query a database. It can't fetch a URL. It can't read a file, update a record, or send a message. It produces strings. That's it. Sometimes those strings look like function calls -- perfectly formatted JSON with parameter names and values. But the model has no idea what happens after it emits that JSON. It doesn't have a network socket. It doesn't have a process. It generates text and hopes something on the other side does the right thing.

That something is the agent runtime. The runtime reads the model's output, decides whether it's a tool call, validates the arguments, executes the function, and feeds the result back into the model's context. The tool-use architecture is the bridge between "the model wants to query DuckDB" and "DuckDB actually runs the query." Get the bridge wrong and you get an agent that hallucinates actions it never took. Get it right and the model becomes genuinely useful -- a reasoning engine with hands.

## Anatomy of a tool definition

Every tool starts as a JSON Schema. The model reads this schema to decide when and how to call the tool. Here's a complete definition:

<pre><code>{
  <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"query_knowledge_graph"</span>,
  <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Run a SQL query against the knowledge graph triples table. Returns rows matching the query. Use this for entity lookups, relationship traversal, and graph pattern matching. The triples table has columns: subject, predicate, object, source_url, confidence."</span>,
  <span style="color:#60a5fa">"parameters"</span>: {
    <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"object"</span>,
    <span style="color:#60a5fa">"properties"</span>: {
      <span style="color:#60a5fa">"sql"</span>: {
        <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>,
        <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"A DuckDB SQL query against the triples table. Example: SELECT subject, object FROM triples WHERE predicate = 'oi:affiliatedWith' AND object = 'entity/mit' LIMIT 20"</span>
      }
    },
    <span style="color:#60a5fa">"required"</span>: [<span style="color:#4ade80">"sql"</span>]
  },
  <span style="color:#60a5fa">"returns"</span>: <span style="color:#4ade80">"JSON array of row objects"</span>
}</code></pre>

Three fields matter. `name` is what the model writes when it wants to call the tool. `parameters` constrains what it can send. But `description` is the most important field by far -- it's the only thing the model reads to decide *whether* to use this tool in the first place. A vague description ("queries data") means the model won't know when to reach for it. A specific one tells the model exactly what this tool can do and what the input looks like.

Here are two more tools from the OpenIndex surface:

<pre><code>{
  <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"search_index"</span>,
  <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Full-text keyword search via Tantivy (BM25). Returns ranked documents matching the query. Supports phrase queries, boolean operators (AND, OR, NOT), and field-specific search (title:, body:). Use for finding documents by keyword."</span>,
  <span style="color:#60a5fa">"parameters"</span>: {
    <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"object"</span>,
    <span style="color:#60a5fa">"properties"</span>: {
      <span style="color:#60a5fa">"query"</span>: {
        <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>,
        <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"BM25 search query. Example: \"climate change\" AND policy"</span>
      },
      <span style="color:#60a5fa">"domain_filter"</span>: {
        <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>,
        <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Optional domain glob filter. Example: *.edu"</span>
      },
      <span style="color:#60a5fa">"limit"</span>: {
        <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"integer"</span>,
        <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Max results to return. Default: 20"</span>
      }
    },
    <span style="color:#60a5fa">"required"</span>: [<span style="color:#4ade80">"query"</span>]
  }
}</code></pre>

<pre><code>{
  <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"fetch_url"</span>,
  <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Fetch a URL and return its content as plain text. Use for reading a specific page when you know its URL. Returns the page body with HTML stripped. Times out after 10 seconds."</span>,
  <span style="color:#60a5fa">"parameters"</span>: {
    <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"object"</span>,
    <span style="color:#60a5fa">"properties"</span>: {
      <span style="color:#60a5fa">"url"</span>: {
        <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>,
        <span style="color:#60a5fa">"format"</span>: <span style="color:#4ade80">"uri"</span>,
        <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Full URL to fetch. Must start with https://"</span>
      }
    },
    <span style="color:#60a5fa">"required"</span>: [<span style="color:#4ade80">"url"</span>]
  }
}</code></pre>

Notice the pattern. Each tool has one clear purpose. Parameters are few and constrained. Descriptions include examples. The model doesn't have to guess.

## The execution sandwich

Every tool call follows the same flow. The model generates a request. The runtime processes it. The result goes back to the model. Four layers, strict ordering:

<pre><code>  <span style="color:#60a5fa">LLM generates tool call</span>
       |
       |  <span style="color:#888">{ "name": "query_knowledge_graph",</span>
       |  <span style="color:#888">  "arguments": { "sql": "SELECT ..." } }</span>
       v
  <span style="color:#fbbf24">Runtime validates arguments</span>
       |
       |  <span style="color:#888">Does "sql" exist? Is it a string?</span>
       |  <span style="color:#888">Is the SQL syntactically valid?</span>
       v
  <span style="color:#4ade80">Runtime executes function</span>
       |
       |  <span style="color:#888">DuckDB runs the query</span>
       |  <span style="color:#888">Returns rows or error</span>
       v
  <span style="color:#60a5fa">Result returned to LLM context</span>
       |
       |  <span style="color:#888">{ "rows": [...], "count": 15 }</span>
       |  <span style="color:#888">or { "error": "column 'namee' not found" }</span>
       v
  <span style="color:#60a5fa">LLM generates next action</span></code></pre>

The validation step catches problems before they reach the backend. The model sometimes emits malformed JSON, sends a number where a string is expected, or omits required fields. Without validation, those bad inputs hit the database and produce confusing errors. With validation, the runtime rejects them early and sends a clear message back to the model: "parameter `sql` is required but missing."

This isn't hypothetical. Models mess up tool calls regularly. They forget quotes around strings. They invent parameter names that don't exist in the schema. They send arrays when the schema expects a single value. The validation layer is the difference between an agent that crashes and one that self-corrects.

## Error handling is the whole game

What happens when a tool call fails? This is the question that separates a demo from a useful system. Here's a concrete example -- the model writes bad SQL:

**Step 1: Bad query**

<pre><code><span style="color:#888">// LLM generates:</span>
<span style="color:#60a5fa">query_knowledge_graph</span>({
  <span style="color:#4ade80">"sql"</span>: <span style="color:#4ade80">"SELECT subject, namee FROM triples WHERE predicate = 'rdf:type'"</span>
})</code></pre>

**Step 2: Error with context**

<pre><code><span style="color:#888">// Runtime returns:</span>
{
  <span style="color:#60a5fa">"error"</span>: <span style="color:#4ade80">"Binder Error: column 'namee' not found"</span>,
  <span style="color:#60a5fa">"hint"</span>: <span style="color:#4ade80">"Available columns: subject, predicate, object, source_url, confidence"</span>,
  <span style="color:#60a5fa">"suggestion"</span>: <span style="color:#4ade80">"Did you mean 'predicate' or 'object'?"</span>
}</code></pre>

**Step 3: Corrected query**

<pre><code><span style="color:#888">// LLM generates (second attempt):</span>
<span style="color:#60a5fa">query_knowledge_graph</span>({
  <span style="color:#4ade80">"sql"</span>: <span style="color:#4ade80">"SELECT subject, object FROM triples WHERE predicate = 'rdf:type' LIMIT 20"</span>
})</code></pre>

The model reads the error, sees the available columns, fixes its query, and gets results. Three strategies for error handling, from worst to best:

<table>
  <thead>
    <tr>
      <th>Strategy</th>
      <th>What the runtime returns</th>
      <th>Recovery rate</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Naive</strong></td>
      <td>Raw error string from the backend</td>
      <td>~40%</td>
    </tr>
    <tr>
      <td><strong>Annotated</strong></td>
      <td>Error + hint (column list, valid values)</td>
      <td>~75%</td>
    </tr>
    <tr>
      <td><strong>Structured</strong></td>
      <td>Error + schema info + suggested fix</td>
      <td>~90%</td>
    </tr>
  </tbody>
</table>

The naive approach just passes through whatever DuckDB says. Sometimes the model figures it out. Often it doesn't -- it'll try the same broken query again, or hallucinate a different wrong column. The annotated approach gives the model enough context to self-correct: "here are the actual columns, pick one." The structured approach goes further: pre-validate the SQL, parse the AST, and suggest a specific fix. More work to build, but the agent almost always recovers.

<div class="note">
  <strong>Retry budgets matter.</strong> An agent that retries forever on a bad tool call burns tokens and time. Cap retries at 2-3 attempts per tool call. If the model can't fix the query in 3 tries, return an error to the user. Better to fail fast than spin in a retry loop.
</div>

## Tool composition: the real power

A single tool call isn't very interesting. The agent calls `search_index`, gets results, done. The interesting part is when the agent chains tools together, with each call building on the previous result.

Here's a concrete sequence -- the user asks: *"Find organizations working on climate change and their key people."*

**Call 1: Full-text search**

<pre><code><span style="color:#60a5fa">search_index</span>({
  <span style="color:#4ade80">"query"</span>: <span style="color:#4ade80">"climate change organizations"</span>,
  <span style="color:#4ade80">"limit"</span>: <span style="color:#fbbf24">10</span>
})

<span style="color:#888">→ Returns 10 documents with IDs, titles, URLs, BM25 scores</span></code></pre>

**Call 2: Entity extraction from results**

<pre><code><span style="color:#60a5fa">query_knowledge_graph</span>({
  <span style="color:#4ade80">"sql"</span>: <span style="color:#4ade80">"SELECT DISTINCT subject FROM triples WHERE predicate = 'rdf:type' AND object = 'oi:Organization' AND source_url IN ('https://...', 'https://...') LIMIT 20"</span>
})

<span style="color:#888">→ Returns Organization entities: ipcc, greenpeace, unfccc, wri</span></code></pre>

**Call 3: Find affiliated people**

<pre><code><span style="color:#60a5fa">query_knowledge_graph</span>({
  <span style="color:#4ade80">"sql"</span>: <span style="color:#4ade80">"SELECT subject AS person, object AS organization FROM triples WHERE predicate = 'oi:affiliatedWith' AND object IN ('ipcc', 'greenpeace', 'unfccc', 'wri')"</span>
})

<span style="color:#888">→ Returns person-organization pairs</span></code></pre>

**Call 4: Get details on key people**

<pre><code><span style="color:#60a5fa">query_knowledge_graph</span>({
  <span style="color:#4ade80">"sql"</span>: <span style="color:#4ade80">"SELECT subject, predicate, object FROM triples WHERE subject IN ('hoesung_lee', 'jennifer_morgan') AND predicate IN ('oi:role', 'oi:locatedIn')"</span>
})

<span style="color:#888">→ Returns roles and locations for each person</span></code></pre>

Four calls. Each one takes the output of the previous call and uses it to construct the next query. The model decides what to call next based on what came back. Nobody hardcoded this sequence -- the model figured out that to answer "organizations and their people," it needed to search first, extract entities second, follow relationships third, and gather details fourth.

This is the difference between a tool and an agent. A tool does one thing. An agent plans a sequence of tool calls to answer a question that no single tool can handle.

## Sandboxing and permissions

Not every tool should be available to every agent. A read-only search agent has no business triggering a crawl. A crawl agent shouldn't be able to drop tables. The permission model is straightforward:

<table>
  <thead>
    <tr>
      <th>Category</th>
      <th>Tools</th>
      <th>Permission Level</th>
      <th>Risk</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Read</strong></td>
      <td><code>query_knowledge_graph</code>, <code>search_index</code>, <code>search_vector</code>, <code>query_oql</code></td>
      <td>Always available</td>
      <td>Low -- worst case is a slow query</td>
    </tr>
    <tr>
      <td><strong>Fetch</strong></td>
      <td><code>fetch_url</code></td>
      <td>Available with rate limits</td>
      <td>Medium -- can hit external servers</td>
    </tr>
    <tr>
      <td><strong>Write</strong></td>
      <td><code>crawl_urls</code>, <code>update_entity</code>, <code>insert_triple</code></td>
      <td>Requires explicit grant</td>
      <td>High -- modifies the index</td>
    </tr>
    <tr>
      <td><strong>Admin</strong></td>
      <td><code>delete_entity</code>, <code>drop_index</code>, <code>execute_sql</code></td>
      <td>Restricted to admin agents</td>
      <td>Critical -- destructive operations</td>
    </tr>
  </tbody>
</table>

The runtime enforces this before execution. When the model calls `crawl_urls` and the agent only has read permissions, the runtime returns a clear error: "tool `crawl_urls` is not available to this agent." The model sees that, adjusts, and uses a read tool instead.

Rate limiting is per-tool, not per-agent. `fetch_url` gets 10 calls per minute -- enough to read a few pages, not enough to DDOS anything. `query_knowledge_graph` gets 100 calls per minute because it's hitting a local database, not the open internet. The limits are different because the risks are different.

## Schema design matters more than you think

Bad tool schemas produce bad agent behavior. The model is working from the schema alone -- it doesn't have access to source code, documentation, or examples beyond what you put in the description. Common mistakes:

**Too many parameters.** A tool with 12 optional parameters confuses the model. It doesn't know which ones matter, so it either fills them all (incorrectly) or ignores them all (missing useful filters). Keep it under 5 parameters. Split complex tools into simpler ones.

**Ambiguous types.** A parameter called `query` with type `string` could be SQL, natural language, a regex, or a URL. The model guesses. It often guesses wrong.

Here's a bad schema vs. a good one:

<pre><code><span style="color:#888">// Bad: ambiguous, too flexible</span>
{
  <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"search"</span>,
  <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Search for data"</span>,
  <span style="color:#60a5fa">"parameters"</span>: {
    <span style="color:#60a5fa">"query"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
    <span style="color:#60a5fa">"options"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"object"</span> }
  }
}</code></pre>

<pre><code><span style="color:#888">// Good: specific, constrained, documented</span>
{
  <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"search_fulltext"</span>,
  <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"BM25 keyword search via Tantivy. Returns documents ranked by term frequency. Use for finding pages containing specific words or phrases. Example query: \"machine learning\" AND python"</span>,
  <span style="color:#60a5fa">"parameters"</span>: {
    <span style="color:#60a5fa">"query"</span>: {
      <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>,
      <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Tantivy query string. Supports AND, OR, NOT, phrase quotes. Example: \"web crawling\" AND rust"</span>
    },
    <span style="color:#60a5fa">"limit"</span>: {
      <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"integer"</span>,
      <span style="color:#60a5fa">"minimum"</span>: <span style="color:#fbbf24">1</span>,
      <span style="color:#60a5fa">"maximum"</span>: <span style="color:#fbbf24">100</span>,
      <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Max results. Default: 20"</span>
    }
  }
}</code></pre>

The good version tells the model exactly what kind of string to send, constrains the numeric range, and includes an example in the description. The model doesn't have to guess the format -- it can copy the example and modify it.

<div class="note">
  <strong>Put examples in descriptions.</strong> A description that says "Tantivy query string" is less useful than one that says "Tantivy query string. Example: \"machine learning\" AND python". The model treats examples as templates. Give it a template and it'll produce well-formed calls. Make it guess and it'll invent its own syntax.
</div>

## Performance: latency budget for tool-use agents

Every tool call adds latency. The model needs time to think. The runtime needs time to validate and execute. The result needs to travel back. For a multi-step agent, these costs add up fast.

<table>
  <thead>
    <tr>
      <th>Operation</th>
      <th>Latency</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>LLM reasoning per turn</strong></td>
      <td>1-3s</td>
      <td>Depends on model size, prompt length</td>
    </tr>
    <tr>
      <td><strong>DuckDB query</strong></td>
      <td>5-50ms</td>
      <td>Simple lookups to moderate joins</td>
    </tr>
    <tr>
      <td><strong>Tantivy search</strong></td>
      <td>2-10ms</td>
      <td>BM25 over inverted index</td>
    </tr>
    <tr>
      <td><strong>Vald vector search</strong></td>
      <td>10-50ms</td>
      <td>ANN over embedding space</td>
    </tr>
    <tr>
      <td><strong>URL fetch</strong></td>
      <td>500ms-10s</td>
      <td>Network bound, highly variable</td>
    </tr>
    <tr>
      <td><strong>Crawl trigger</strong></td>
      <td>1-30s</td>
      <td>Dispatches async, returns job ID</td>
    </tr>
  </tbody>
</table>

For the 4-step climate change query from the composition section: 4 LLM turns (4-12s) + 3 DuckDB queries (15-150ms) + 1 Tantivy search (2-10ms) = roughly **5-13 seconds total**. Most of the time is the model thinking, not the tools executing.

Compare that to just running a SQL query directly: ~10ms. Agents are 100-1000x slower than direct tool access. The question isn't speed -- it's whether the flexibility is worth the cost. A user who knows SQL doesn't need an agent. A user who doesn't know SQL, doesn't know the schema, and can't write a recursive CTE? For them, 10 seconds to get a correct answer beats staring at a query editor forever.

<div class="note note-warn">
  <strong>Streaming helps.</strong> The 10-second wall feels shorter when the agent streams its reasoning. "Searching for climate change documents... found 10 results. Now looking up organizations mentioned in those pages..." gives the user feedback while the model works. A 10-second spinner feels broken. A 10-second narrated workflow feels productive.
</div>

## OpenIndex's tool surface

Here's where the existing OpenIndex components map to agent tools:

<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Tool</th>
      <th>Input</th>
      <th>Output</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>DuckDB</strong></td>
      <td><code>query_sql(sql)</code></td>
      <td>Arbitrary SQL string</td>
      <td>JSON rows</td>
    </tr>
    <tr>
      <td><strong>Tantivy</strong></td>
      <td><code>search_fulltext(query, filters)</code></td>
      <td>BM25 query + optional domain/date filters</td>
      <td>Ranked document list</td>
    </tr>
    <tr>
      <td><strong>Vald</strong></td>
      <td><code>search_vector(text, k)</code></td>
      <td>Natural language text + result count</td>
      <td>k nearest documents by embedding similarity</td>
    </tr>
    <tr>
      <td><strong>Knowledge Graph</strong></td>
      <td><code>query_entities(type, filters)</code></td>
      <td>Entity type + optional predicates</td>
      <td>Matching entities with properties</td>
    </tr>
    <tr>
      <td><strong>Knowledge Graph</strong></td>
      <td><code>query_triples(subject, predicate)</code></td>
      <td>Subject entity + optional predicate filter</td>
      <td>Related triples</td>
    </tr>
    <tr>
      <td><strong>OQL</strong></td>
      <td><code>query_oql(oql_string)</code></td>
      <td>Unified OQL query string</td>
      <td>Fused ranked results from all backends</td>
    </tr>
    <tr>
      <td><strong>Recrawler</strong></td>
      <td><code>crawl_urls(urls[])</code></td>
      <td>Array of URLs to crawl</td>
      <td>Job ID + status</td>
    </tr>
  </tbody>
</table>

These tools exist as Go functions today. `DuckDB` queries run against sharded databases. The knowledge graph stores triples. Tantivy and Vald have defined schemas. The recrawler fetches URLs at scale. What doesn't exist yet is the agent runtime -- the layer that takes a model's JSON output, maps it to these Go functions, handles errors, manages permissions, and feeds results back into context.

That's the next piece. The tools are built. The schemas are defined. The bridge between "model generates a tool call" and "Go function executes" is what comes next. It's not a model problem or a search problem. It's a systems engineering problem -- validation, serialization, error handling, retry logic, permission checks, rate limiting, streaming results. The boring parts that make the interesting parts work.

An LLM produces strings. Tools turn strings into actions. The architecture between them -- schemas that guide the model, validation that catches mistakes, error messages that enable recovery, permissions that prevent damage -- is where the engineering actually lives.
