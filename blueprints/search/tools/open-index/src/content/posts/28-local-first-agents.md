---
slug: local-first-agents
title: "Local-First AI Agents — Your Hardware, Your Data, Your Rules"
date: 2026-03-18
summary: "Cloud AI is convenient until it isn't. Run your agents locally, query your own DuckDB, keep your data on your own disk. Here's the architecture."
tags: [ai, architecture]
---

Most AI agent frameworks assume cloud everything. Your LLM runs on someone else's GPU. Your vector search lives in Pinecone. Your data sits in Supabase. Every query leaves your network, crosses the internet, hits an API, and comes back -- if the API is up, if you haven't blown your rate limit, if the pricing hasn't changed since last Tuesday.

This works fine until it doesn't. The API goes down during a production run. The provider changes their terms of service to claim training rights on your queries. Your monthly bill triples because an agent loop ran longer than expected. You discover your competitive intelligence queries have been logged on someone else's server.

Local-first means flipping the default. The core loop runs on your hardware. Data stays on your disk. Cloud is an optional enhancement, not a load-bearing dependency.

## What "local-first" actually means

The term gets thrown around loosely. Here's what we mean precisely:

- **The agent loop runs on your machine.** Not a cloud function, not a managed service. A process on hardware you control.
- **Data stays on your disk.** DuckDB files, not a hosted database. Parquet files, not a data warehouse API.
- **Tool execution is local.** Shell commands, file I/O, local HTTP servers. The agent calls tools that exist on the same machine.
- **LLM inference *can* be local or remote -- your choice.** Ollama for privacy, Claude API for quality. The architecture doesn't assume either.
- **The agent works offline for everything except LLM inference** (if you're using a remote model). Data access, tool execution, state persistence -- all local.

"Local-first" doesn't mean "no internet ever." It means the default is local, and cloud is opt-in. If your internet goes down, the agent can still read its memory, query its database, and execute its tools. It just can't ask an LLM for new inferences until connectivity returns -- and if you're running Ollama, even that keeps working.

## The local AI stack

Here's the concrete technology at each layer, with honest trade-offs:

<table>
  <thead>
    <tr>
      <th>Layer</th>
      <th>Cloud Option</th>
      <th>Local Option</th>
      <th>Trade-off</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>LLM inference</strong></td>
      <td>Claude / GPT API</td>
      <td>Ollama + Llama 3.3 70B</td>
      <td>Quality vs. cost vs. privacy</td>
    </tr>
    <tr>
      <td><strong>Embeddings</strong></td>
      <td>OpenAI embeddings API</td>
      <td>nomic-embed-text local</td>
      <td>Speed vs. quality</td>
    </tr>
    <tr>
      <td><strong>Vector search</strong></td>
      <td>Pinecone / Weaviate</td>
      <td>Vald or SQLite-VSS</td>
      <td>Scale vs. simplicity</td>
    </tr>
    <tr>
      <td><strong>Database</strong></td>
      <td>PostgreSQL cloud</td>
      <td>DuckDB local</td>
      <td>Ops burden vs. zero-config</td>
    </tr>
    <tr>
      <td><strong>Full-text search</strong></td>
      <td>Elasticsearch cloud</td>
      <td>Tantivy local</td>
      <td>Cluster mgmt vs. single binary</td>
    </tr>
  </tbody>
</table>

Let's not pretend local is always better. Cloud GPT-4o and Claude are significantly more capable than local Llama 3.3 for complex reasoning, multi-step planning, and nuanced language tasks. That's just reality.

But for many agent tasks -- entity extraction, SQL generation, document summarization, classification -- local models are good enough. A 7B model can extract named entities from HTML with 90%+ accuracy. It doesn't need Claude-level reasoning to find email addresses on a contact page. Match the model to the task.

## DuckDB as agent memory

We've written about DuckDB for crawl analytics (post 6). Turns out it's also the perfect agent memory store. No server process. Single file on disk. SQL interface means the agent reads and writes memory with standard queries -- no custom serialization, no proprietary format.

Here's what an agent memory table looks like:

<pre><code><span style="color:#60a5fa">CREATE TABLE</span> agent_memory (
  id            <span style="color:#fbbf24">INTEGER</span> <span style="color:#60a5fa">PRIMARY KEY</span>,
  created_at    <span style="color:#fbbf24">TIMESTAMP</span> <span style="color:#60a5fa">DEFAULT</span> current_timestamp,
  category      <span style="color:#fbbf24">TEXT</span>,      <span style="color:#888">-- 'conversation', 'tool_result', 'observation'</span>
  content       <span style="color:#fbbf24">TEXT</span>,      <span style="color:#888">-- the actual memory content</span>
  metadata      <span style="color:#fbbf24">JSON</span>,      <span style="color:#888">-- structured data (tool name, params, etc.)</span>
  session_id    <span style="color:#fbbf24">TEXT</span>,      <span style="color:#888">-- group memories by agent session</span>
  relevance     <span style="color:#fbbf24">FLOAT</span>      <span style="color:#888">-- for scoring/retrieval</span>
);</code></pre>

The agent stores conversation history, tool results, and observations. Everything persists across restarts -- just reopen the `.duckdb` file:

<pre><code><span style="color:#888">-- Store a tool result</span>
<span style="color:#60a5fa">INSERT INTO</span> agent_memory (category, content, metadata, session_id)
<span style="color:#60a5fa">VALUES</span> (
  <span style="color:#4ade80">'tool_result'</span>,
  <span style="color:#4ade80">'Found 12,841 pages from en.wikipedia.org with status 200'</span>,
  <span style="color:#4ade80">'{"tool": "duckdb_query", "query": "SELECT COUNT(*) FROM pages WHERE domain = ..."}'</span>,
  <span style="color:#4ade80">'session_20260318_001'</span>
);

<span style="color:#888">-- Retrieve recent memories for context injection</span>
<span style="color:#60a5fa">SELECT</span> content, metadata
<span style="color:#60a5fa">FROM</span> agent_memory
<span style="color:#60a5fa">WHERE</span> session_id = <span style="color:#4ade80">'session_20260318_001'</span>
<span style="color:#60a5fa">ORDER BY</span> created_at <span style="color:#60a5fa">DESC</span>
<span style="color:#60a5fa">LIMIT</span> <span style="color:#fbbf24">20</span>;

<span style="color:#888">-- Query the knowledge graph locally</span>
<span style="color:#60a5fa">SELECT</span> m.content, p.url, p.title
<span style="color:#60a5fa">FROM</span> agent_memory m
<span style="color:#60a5fa">JOIN</span> pages p <span style="color:#60a5fa">ON</span> m.metadata->><span style="color:#4ade80">'source_url'</span> = p.url
<span style="color:#60a5fa">WHERE</span> m.category = <span style="color:#4ade80">'observation'</span>
  <span style="color:#60a5fa">AND</span> m.content <span style="color:#60a5fa">LIKE</span> <span style="color:#4ade80">'%climate%'</span>;</code></pre>

No ORM. No migration framework. No connection pooling. The agent opens a file, runs SQL, closes the file. If the agent crashes, the data's still there. If you want to inspect what the agent knows, open the same file in the DuckDB CLI and browse.

<div class="note">
  <strong>Why not SQLite?</strong> SQLite works fine for row-level operations. But agent memory often involves analytical queries -- "what are the top domains I've seen?" or "summarize all tool results from the last hour." DuckDB's columnar engine handles these aggregations much faster. And since we're already using DuckDB for crawl data, it's one less dependency.
</div>

## Running LLMs locally: the 2026 landscape

The local inference scene has matured significantly. Here's what's actually usable today, not what's theoretically possible.

**Ollama** is the simplest path. Install it, run `ollama run llama3.3:70b`, and you've got an HTTP API compatible with the OpenAI client format. Dead simple. Handles quantization, model management, and GPU offloading automatically.

**llama.cpp** gives lower-level control. You pick the GGUF quantization level, manage context windows, tune batch sizes. More work, better performance tuning. Good if you're running multiple models simultaneously and need fine-grained resource allocation.

**MLX** is Apple's framework for M-series Macs. If you're on a MacBook Pro with 64GB or 96GB unified memory, MLX gets you surprisingly good inference speeds because the CPU and GPU share the same memory pool. No PCIe bottleneck.

**vLLM** is the production-grade option. Continuous batching, PagedAttention, tensor parallelism across multiple GPUs. Overkill for a single agent, but if you're running a shared inference server for a team, it's the right tool.

<table>
  <thead>
    <tr>
      <th>Model</th>
      <th>RAM Required</th>
      <th>Speed (tokens/s)</th>
      <th>Good For</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Llama 3.3 8B Q4</strong></td>
      <td>~6 GB</td>
      <td>40-80 tok/s (CPU)</td>
      <td>NER, classification, simple extraction</td>
    </tr>
    <tr>
      <td><strong>Llama 3.3 70B Q4</strong></td>
      <td>~42 GB</td>
      <td>10-20 tok/s (GPU)</td>
      <td>Reasoning, planning, complex Q&amp;A</td>
    </tr>
    <tr>
      <td><strong>Mistral 7B Q4</strong></td>
      <td>~5 GB</td>
      <td>50-90 tok/s (CPU)</td>
      <td>Fast classification, code generation</td>
    </tr>
    <tr>
      <td><strong>Qwen 2.5 72B Q4</strong></td>
      <td>~44 GB</td>
      <td>8-15 tok/s (GPU)</td>
      <td>Multilingual tasks, long context</td>
    </tr>
    <tr>
      <td><strong>nomic-embed-text</strong></td>
      <td>~0.5 GB</td>
      <td>500+ embeddings/s</td>
      <td>Local embeddings (137M params)</td>
    </tr>
  </tbody>
</table>

Be honest with yourself: 7B models are fast but limited. They handle structured tasks well -- pull entities from text, classify a document, generate a SQL query from a template. They struggle with multi-step reasoning, ambiguous queries, and anything that requires world knowledge beyond their training data.

70B models are genuinely good for complex tasks, but they need 48GB+ of RAM (quantized) or a GPU with 24GB+ VRAM. For a laptop, that means a MacBook Pro with 64GB unified memory or a desktop with an RTX 4090.

For many agent workflows, the right answer isn't picking one model -- it's using both.

## The hybrid architecture

The pragmatic approach: run the agent loop locally, use local models for cheap tasks, call cloud APIs for hard tasks. The agent itself decides which backend to use based on task complexity.

<pre><code>  <span style="color:#60a5fa">Agent Loop</span> (local process)
       |
       | classify task complexity
       v
  <span style="color:#e0e0e0">┌──────────────────────────────────────────┐</span>
  <span style="color:#e0e0e0">│</span>  <span style="color:#fbbf24">Router</span>                                  <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Simple task?  → local 7B   <span style="color:#4ade80">(free)</span>       <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Complex task? → cloud API  <span style="color:#fbbf24">($0.01)</span>     <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Embeddings?   → local nomic <span style="color:#4ade80">(free)</span>     <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Data query?   → local DuckDB <span style="color:#4ade80">(free)</span>    <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">└──────────────────────────────────────────┘</span>
       |                    |
       v                    v
  <span style="color:#4ade80">┌──────────┐</span>     <span style="color:#60a5fa">┌──────────────┐</span>
  <span style="color:#4ade80">│ Ollama   │</span>     <span style="color:#60a5fa">│ Claude API   │</span>
  <span style="color:#4ade80">│ 7B local │</span>     <span style="color:#60a5fa">│ (when needed)│</span>
  <span style="color:#4ade80">└──────────┘</span>     <span style="color:#60a5fa">└──────────────┘</span>
       |                    |
       v                    v
  <span style="color:#e0e0e0">┌──────────────────────────────────────────┐</span>
  <span style="color:#e0e0e0">│</span>  <span style="color:#4ade80">DuckDB</span> (always local)                   <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Agent memory + crawl data + results     <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">└──────────────────────────────────────────┘</span></code></pre>

The decision logic in practice:

- **Simple classification/extraction** -- local 7B model. "Is this page about technology?" "Extract the author name from this HTML." Free, fast, private. Runs in 200ms.
- **Complex reasoning/planning** -- cloud API. "Given these 50 search results, identify the three most relevant to the user's question and explain why." Better quality, costs a penny per call.
- **Embeddings** -- always local with nomic-embed-text. 500+ embeddings per second, no API cost. Quality is within 5% of OpenAI's ada-002 on most benchmarks.
- **Data storage and queries** -- always local DuckDB. No question. Your data never leaves.

This gives you privacy by default. 80-90% of agent operations touch the local model and local database only. The cloud API sees only the complex reasoning prompts -- not your raw data, not your crawl results, not your query history.

## Privacy and security

This isn't abstract. Here's what local-first means concretely for your data:

- Your crawl data doesn't leave your network. The 16-shard DuckDB files sit on your disk. Period.
- Entity extraction results stay local. The NER model runs on your machine, writes to your database.
- No third-party has a copy of your knowledge graph. The relationships between entities, the link structure, the domain metadata -- all local.
- No API provider can see your queries. When you search your index for "competitor product pricing," that query doesn't show up in anyone's logs.
- Your search history isn't someone else's training data. Every major AI provider has some form of data retention policy. Local execution has a simple policy: your disk, your rules.

<div class="note note-warn">
  <strong>The cloud API still sees your complex prompts.</strong> If you route hard tasks to a cloud LLM, that provider sees those specific prompts. The point isn't zero cloud exposure -- it's minimizing it. Entity extraction, SQL queries, classification, embeddings -- all local. Only the reasoning step hits the cloud, and only when you choose.
</div>

If you're building a competitive intelligence tool on your crawl index, this matters a lot. You don't want "show me all pages where competitor X mentions pricing changes" going through a third-party API. With local-first, the query runs against your DuckDB, the extraction runs on your local model, and the results never leave your machine.

## The cost equation

Let's do real math. Running an AI agent that processes 10,000 queries per day, with entity extraction, embeddings, and occasional complex reasoning.

<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Cloud-Everything (monthly)</th>
      <th>Local-First (monthly)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>LLM inference</strong></td>
      <td>$300-600 (API calls)</td>
      <td>$15 electricity (local GPU)</td>
    </tr>
    <tr>
      <td><strong>Embeddings</strong></td>
      <td>$50-100 (OpenAI API)</td>
      <td>$0 (local nomic-embed)</td>
    </tr>
    <tr>
      <td><strong>Vector search</strong></td>
      <td>$70-150 (Pinecone)</td>
      <td>$0 (Vald on same machine)</td>
    </tr>
    <tr>
      <td><strong>Database</strong></td>
      <td>$50-200 (managed Postgres)</td>
      <td>$0 (DuckDB file)</td>
    </tr>
    <tr>
      <td><strong>Full-text search</strong></td>
      <td>$100-300 (Elastic Cloud)</td>
      <td>$0 (Tantivy binary)</td>
    </tr>
    <tr>
      <td style="color:#fbbf24"><strong>Total</strong></td>
      <td style="color:#fbbf24"><strong>$570-1,350/mo</strong></td>
      <td style="color:#4ade80"><strong>$15/mo + hardware</strong></td>
    </tr>
  </tbody>
</table>

The one-time hardware cost for local-first: a workstation with 64GB RAM and an RTX 4090 runs about $2,500-3,000. A Mac Studio with 64GB unified memory is around $2,000. Even at the low end of cloud costs ($570/mo), the hardware pays for itself in 4-5 months.

For the hybrid approach -- local models for cheap tasks, cloud API for complex reasoning -- you're looking at maybe $30-80/month in API costs instead of $570+. The cloud API handles maybe 10-20% of queries (the hard ones), while everything else runs free on local hardware.

<div class="note">
  <strong>The math gets better over time.</strong> Cloud costs scale linearly with usage. Hardware costs are fixed. Double your query volume and cloud costs double. Double your query volume locally and your electricity bill goes up a few dollars. For personal or small-team deployments, local-first is obviously cheaper. For larger deployments, it depends on utilization -- but the break-even point is lower than most people expect.
</div>

## OpenIndex is built for this

Everything in the OpenIndex stack was chosen with local-first as a constraint, not an afterthought.

**DuckDB**: embedded, zero-config, single file. No database server to run. The agent opens a file, runs SQL, gets results. The 16-shard architecture handles high-throughput writes from crawlers. The same files serve as agent memory.

**Tantivy**: a Rust library that compiles to a single binary. No Elasticsearch cluster, no JVM, no YAML configuration files. Build the inverted index, query it, done.

**Vald**: runs on a single Kubernetes node for small deployments. Agents, gateway, and index all on one machine. Scale out to multiple nodes when you need it, but start with one.

The entire stack fits on a laptop. A crawl index, full-text search, vector search, and agent memory -- all running locally. An agent layer on top is a Go binary that runs the loop, calls local tools, queries DuckDB, and optionally hits a cloud API when the task exceeds what a local model can handle.

No Kubernetes required (unless you want Vald at scale). No containers required (unless you prefer them). No cloud account required (unless you want cloud LLM inference). The infrastructure is local-first today. Every component runs on a single machine with no network dependencies.

<pre><code>  <span style="color:#60a5fa">Agent Binary</span> (Go)
       |
       +---> <span style="color:#4ade80">DuckDB</span>   <span style="color:#888">(agent memory + crawl data, local files)</span>
       |
       +---> <span style="color:#4ade80">Tantivy</span>  <span style="color:#888">(full-text search, local binary)</span>
       |
       +---> <span style="color:#4ade80">Vald</span>     <span style="color:#888">(vector search, single node)</span>
       |
       +---> <span style="color:#4ade80">Ollama</span>   <span style="color:#888">(local LLM, optional)</span>
       |
       +---> <span style="color:#fbbf24">Cloud API</span> <span style="color:#888">(remote LLM, optional)</span></code></pre>

The agent layer is planned. The infrastructure underneath -- the part that actually stores data, indexes it, and makes it queryable -- is local-first today. When the agent ships, it won't need to phone home to work. Your hardware, your data, your rules.
