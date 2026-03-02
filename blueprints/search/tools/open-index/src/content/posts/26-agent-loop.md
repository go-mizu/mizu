---
slug: agent-loop
title: "The Agent Loop — Observe, Think, Act, Repeat"
date: 2026-03-16
summary: "An LLM that can only answer questions is a chatbot. An LLM that can observe its environment, decide what to do, execute actions, and learn from the results is an agent. Here's how the loop works."
tags: [ai, architecture]
---

Take any LLM. Ask it a question. It produces an answer. That's a chatbot. The model takes input, generates output, and forgets everything the moment the response is done. No memory, no environment awareness, no ability to go check something and come back.

Now give that same model access to tools -- a database, an API, a file system. Put it in a loop: observe the environment, decide what to do, execute an action, look at the result, decide again. Keep going until the goal is met. That's an agent.

The model is the same. The weights didn't change. The architecture around it changed. A chatbot is a single function call. An agent is a while loop.

## Chatbots vs agents

The distinction is worth hammering on because it's the source of most confusion about what "AI agents" are.

<table>
  <thead>
    <tr>
      <th>Property</th>
      <th>Chatbot</th>
      <th>Agent</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Interaction</strong></td>
      <td>Single request/response</td>
      <td>Multi-turn loop</td>
    </tr>
    <tr>
      <td><strong>Environment</strong></td>
      <td>None -- text in, text out</td>
      <td>Tools, databases, APIs, files</td>
    </tr>
    <tr>
      <td><strong>Memory</strong></td>
      <td>Conversation history only</td>
      <td>Conversation + tool results + state</td>
    </tr>
    <tr>
      <td><strong>Actions</strong></td>
      <td>Generate text</td>
      <td>Generate text, call functions, modify state</td>
    </tr>
    <tr>
      <td><strong>Stopping</strong></td>
      <td>After one response</td>
      <td>When goal is met (or budget exhausted)</td>
    </tr>
  </tbody>
</table>

A chatbot is stateless by design. An agent accumulates state. Every tool result, every error message, every partial answer feeds back into the next iteration. The model doesn't just answer -- it *reasons about what it still doesn't know* and takes action to fill the gaps.

## The loop, diagrammed

Every agent, regardless of framework or implementation, runs some variation of this:

<pre><code>
  <span style="color:#e0e0e0">User Goal</span>
      |
      v
  <span style="color:#60a5fa">┌─────────────────────────────┐</span>
  <span style="color:#60a5fa">│</span>  <strong>OBSERVE</strong>                    <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  Read environment state     <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  <span style="color:#888">(tool results, errors,</span>     <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  <span style="color:#888"> database state, etc.)</span>     <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">└──────────────┬──────────────┘</span>
               |
               v
  <span style="color:#60a5fa">┌─────────────────────────────┐</span>
  <span style="color:#60a5fa">│</span>  <strong>THINK</strong>                      <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  LLM decides next action    <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  <span style="color:#888">based on observations</span>     <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  <span style="color:#888">and original goal</span>         <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">└──────────────┬──────────────┘</span>
               |
               v
  <span style="color:#60a5fa">┌─────────────────────────────┐</span>
  <span style="color:#60a5fa">│</span>  <strong>ACT</strong>                        <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  Execute tool call          <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  <span style="color:#888">(query DB, call API,</span>      <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  <span style="color:#888"> write file, etc.)</span>        <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">└──────────────┬──────────────┘</span>
               |
               v
       <span style="color:#fbbf24">Goal met?</span> ──<span style="color:#e0e0e0">no</span>──> back to <strong>OBSERVE</strong>
           |
          <span style="color:#4ade80">yes</span>
           |
           v
    <span style="color:#4ade80">Return result</span>
</code></pre>

That's it. Observe, think, act, check. The entire field of "AI agents" is variations on this loop -- different tool sets, different stopping conditions, different ways of managing context. The loop itself is dead simple.

## What makes the loop work: tools

The LLM can't do anything by itself. It generates text. That's it. It can't query a database. It can't fetch a URL. It can't read a file. Tools give it hands.

A tool is a function the agent can call. Each tool has a name, a description (so the model knows when to use it), and a schema (so the model knows what arguments to pass). The agent runtime executes the function and feeds the result back into the context.

Here's what a tool definition looks like in practice:

<pre><code><span style="color:#888">// Tool definitions for a web intelligence agent</span>
[
  {
    <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"query_knowledge_graph"</span>,
    <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Execute SQL against the triples table. Returns entity relationships."</span>,
    <span style="color:#60a5fa">"parameters"</span>: {
      <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"object"</span>,
      <span style="color:#60a5fa">"properties"</span>: {
        <span style="color:#60a5fa">"sql"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>, <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"DuckDB SQL query"</span> }
      }
    }
  },
  {
    <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"search_index"</span>,
    <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Full-text search via Tantivy. Returns ranked documents."</span>,
    <span style="color:#60a5fa">"parameters"</span>: {
      <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"object"</span>,
      <span style="color:#60a5fa">"properties"</span>: {
        <span style="color:#60a5fa">"query"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
        <span style="color:#60a5fa">"limit"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"integer"</span>, <span style="color:#60a5fa">"default"</span>: <span style="color:#fbbf24">20</span> }
      }
    }
  },
  {
    <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"fetch_url"</span>,
    <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Fetch a URL and return its content."</span>,
    <span style="color:#60a5fa">"parameters"</span>: {
      <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"object"</span>,
      <span style="color:#60a5fa">"properties"</span>: {
        <span style="color:#60a5fa">"url"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> }
      }
    }
  },
  {
    <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"extract_entities"</span>,
    <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"Run NER on text. Returns typed entity mentions."</span>,
    <span style="color:#60a5fa">"parameters"</span>: {
      <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"object"</span>,
      <span style="color:#60a5fa">"properties"</span>: {
        <span style="color:#60a5fa">"text"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
        <span style="color:#60a5fa">"types"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"array"</span>, <span style="color:#60a5fa">"items"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> } }
      }
    }
  }
]</code></pre>

The model reads these definitions and decides which tool to call, with what arguments, at each step. It doesn't execute the function -- the runtime does. The model produces a structured tool call (basically a JSON blob with name and args), the runtime executes it, and the result goes back into the model's context for the next iteration.

This is why tool design matters more than model selection. A well-designed tool set with clear descriptions and constrained inputs makes a mediocre model look smart. A bad tool set makes a frontier model look confused.

## A concrete example

Let's walk through the full loop. A user asks: **"Which AI companies are in San Francisco?"**

This isn't a single query. The agent needs to decompose it, execute multiple lookups, filter results, and synthesize an answer.

**Turn 1 -- THINK**

The model receives the question and the tool definitions. It reasons: "I need to find entities of type Organization, related to the topic 'artificial intelligence', located in San Francisco. Let me start by querying the knowledge graph for organizations in SF."

**Turn 1 -- ACT**

<pre><code><span style="color:#888">// Tool call</span>
<span style="color:#60a5fa">query_knowledge_graph</span>({
  <span style="color:#60a5fa">"sql"</span>: <span style="color:#4ade80">"SELECT DISTINCT t1.subject AS company
          FROM triples t1
          JOIN triples t2 ON t2.subject = t1.subject
          WHERE t1.predicate = 'oi:locatedIn'
            AND t1.object = 'entity/san_francisco'
            AND t2.predicate = 'rdf:type'
            AND t2.object = 'oi:Organization'"</span>
})</code></pre>

**Turn 2 -- OBSERVE**

<pre><code><span style="color:#888">// Tool result: 47 rows</span>
[
  { <span style="color:#60a5fa">"company"</span>: <span style="color:#4ade80">"openai"</span> },
  { <span style="color:#60a5fa">"company"</span>: <span style="color:#4ade80">"anthropic"</span> },
  { <span style="color:#60a5fa">"company"</span>: <span style="color:#4ade80">"stripe"</span> },
  { <span style="color:#60a5fa">"company"</span>: <span style="color:#4ade80">"uber"</span> },
  <span style="color:#888">// ... 43 more rows</span>
]</code></pre>

**Turn 2 -- THINK**

"47 organizations in San Francisco, but the question asks specifically about AI companies. Stripe and Uber aren't AI companies. I need to filter by topic. Let me also add a confidence threshold -- some of these triples might be low-quality NER extractions."

**Turn 2 -- ACT**

<pre><code><span style="color:#888">// Refined query</span>
<span style="color:#60a5fa">query_knowledge_graph</span>({
  <span style="color:#60a5fa">"sql"</span>: <span style="color:#4ade80">"SELECT DISTINCT t1.subject AS company
          FROM triples t1
          JOIN triples t2 ON t2.subject = t1.subject
          JOIN triples t3 ON t3.subject = t1.subject
          WHERE t1.predicate = 'oi:locatedIn'
            AND t1.object = 'entity/san_francisco'
            AND t1.confidence > 0.8
            AND t2.predicate = 'rdf:type'
            AND t2.object = 'oi:Organization'
            AND t3.predicate = 'oi:topicOf'
            AND t3.object = 'entity/artificial_intelligence'
            AND t3.confidence > 0.8
          ORDER BY t1.subject"</span>
})</code></pre>

**Turn 3 -- OBSERVE**

<pre><code><span style="color:#888">// Tool result: 23 rows</span>
[
  { <span style="color:#60a5fa">"company"</span>: <span style="color:#4ade80">"anthropic"</span> },
  { <span style="color:#60a5fa">"company"</span>: <span style="color:#4ade80">"cohere"</span> },
  { <span style="color:#60a5fa">"company"</span>: <span style="color:#4ade80">"hugging_face"</span> },
  { <span style="color:#60a5fa">"company"</span>: <span style="color:#4ade80">"openai"</span> },
  { <span style="color:#60a5fa">"company"</span>: <span style="color:#4ade80">"scale_ai"</span> },
  <span style="color:#888">// ... 18 more rows</span>
]</code></pre>

**Turn 3 -- THINK**

"23 high-confidence AI companies in San Francisco. The question is answered. Let me format and return the result."

**Turn 3 -- Return**

The agent produces a natural language answer listing the 23 companies, grouped or ranked however makes sense. Three iterations of the loop. Two tool calls. One answer that required decomposition, query refinement, and filtering -- none of which a single query could handle cleanly.

The model didn't just translate the question into SQL. It *evaluated its own results*, decided they were too broad, and refined the query. That's the part that makes it an agent and not a template engine.

## Stopping conditions

An agent loop needs to end. Without explicit stopping conditions, a confused model loops forever -- calling tools that return errors, retrying the same query, chasing its own tail.

Four mechanisms:

<table>
  <thead>
    <tr>
      <th>Condition</th>
      <th>What Triggers It</th>
      <th>Typical Value</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Goal achieved</strong></td>
      <td>Model decides the answer is complete</td>
      <td>N/A -- model's judgment</td>
    </tr>
    <tr>
      <td><strong>Max iterations</strong></td>
      <td>Hard cap on loop cycles</td>
      <td>10-25 turns</td>
    </tr>
    <tr>
      <td><strong>Token budget</strong></td>
      <td>Context window nearing capacity</td>
      <td>~80% of window size</td>
    </tr>
    <tr>
      <td><strong>Error threshold</strong></td>
      <td>Consecutive tool call failures</td>
      <td>3-5 failures in a row</td>
    </tr>
  </tbody>
</table>

Goal achievement is the happy path. The model looks at the accumulated observations, decides it has enough information, and returns a final answer. This works most of the time.

Max iterations is the safety net. Set it too low (3-5) and complex questions can't be answered -- the agent runs out of turns mid-reasoning. Set it too high (50+) and a confused agent wastes tokens for minutes before giving up. 15-20 is a reasonable default for most workloads.

Token budget matters more than people think. Every observation, every tool result, every thought the model produces goes into the context. Hit the window limit and the agent either crashes or starts losing early context -- which means it forgets the original question.

Error threshold catches tool failures. If the database is down, or the agent keeps generating invalid SQL, three consecutive errors should trigger an early exit with an honest "I couldn't answer this" rather than burning 20 more iterations on a broken tool.

<div class="note note-warn">
  <strong>The most common failure mode isn't infinite loops.</strong> It's the agent deciding it's done too early. The model produces a plausible-sounding answer after one tool call when the question actually required three. Stopping conditions prevent runaway loops, but they don't prevent premature exits. That's a prompt engineering problem, not an architecture problem.
</div>

## The context window is the bottleneck

This is the part that doesn't get enough attention.

Every iteration of the loop adds to the context. The system prompt (tool definitions, instructions) costs tokens up front. Each tool call adds tokens. Each tool result adds more. The model's reasoning -- even when it's just "thinking" before choosing a tool -- adds tokens. It all accumulates.

A back-of-the-envelope calculation for a 200K token window:

<pre><code><span style="color:#888">// Context budget breakdown</span>
<span style="color:#e0e0e0">System prompt + tool definitions</span>    <span style="color:#fbbf24">~3,000 tokens</span>
<span style="color:#e0e0e0">User question</span>                       <span style="color:#fbbf24">~100 tokens</span>
<span style="color:#e0e0e0">Per iteration (think + act + observe)</span>
  <span style="color:#e0e0e0">Model reasoning</span>                  <span style="color:#fbbf24">~300 tokens</span>
  <span style="color:#e0e0e0">Tool call</span>                        <span style="color:#fbbf24">~150 tokens</span>
  <span style="color:#e0e0e0">Tool result</span>                      <span style="color:#fbbf24">~500-5,000 tokens</span>

<span style="color:#888">// 15 iterations × 2,000 avg tokens/iteration = 30,000 tokens</span>
<span style="color:#888">// Total: ~33,000 tokens used out of 200K</span>
<span style="color:#888">// Sounds fine until the tool results get large...</span>

<span style="color:#888">// Real scenario: query returns 50 rows of entity data</span>
<span style="color:#e0e0e0">Tool result (50 rows × ~100 tokens/row)</span>  <span style="color:#fbbf24">~5,000 tokens</span>
<span style="color:#e0e0e0">Do that 8 times</span>                          <span style="color:#fbbf24">~40,000 tokens</span>
<span style="color:#888">// Now we've used 25% of the window on tool results alone</span></code></pre>

200K tokens sounds huge. It isn't, once the agent starts doing real work. Three strategies keep context manageable:

**Truncate large results.** If a query returns 500 rows, the agent doesn't need all 500 in context. Return the first 20 with a count: "Showing 20 of 500 results." The model can request more if needed.

**Summarize old observations.** After turn 10, the observations from turns 1-3 probably aren't relevant anymore. Replace them with a one-line summary: "Earlier: queried organizations in SF, got 47 results, filtered to 23 AI companies."

**Keep tool results out of history.** On subsequent turns, include the model's reasoning and the tool call, but compress the tool result to its summary. The model already processed the result when it made its next decision -- it doesn't need the raw data sitting in context forever.

## Single-agent vs multi-agent

One agent doing everything is the simplest architecture. One loop, one context, one set of tools. For most tasks, that's enough.

But some workflows are naturally parallel. Consider a web intelligence pipeline:

<pre><code>
  <span style="color:#4ade80">┌─────────────┐</span>     <span style="color:#fbbf24">┌─────────────┐</span>     <span style="color:#60a5fa">┌─────────────┐</span>
  <span style="color:#4ade80">│ Crawl Agent │</span>     <span style="color:#fbbf24">│ Extract     │</span>     <span style="color:#60a5fa">│ Resolution  │</span>
  <span style="color:#4ade80">│             │</span>     <span style="color:#fbbf24">│ Agent       │</span>     <span style="color:#60a5fa">│ Agent       │</span>
  <span style="color:#4ade80">│ Tools:      │</span>     <span style="color:#fbbf24">│ Tools:      │</span>     <span style="color:#60a5fa">│ Tools:      │</span>
  <span style="color:#4ade80">│ - fetch_url │</span>     <span style="color:#fbbf24">│ - parse_html│</span>     <span style="color:#60a5fa">│ - query_kg  │</span>
  <span style="color:#4ade80">│ - check_robots│</span>   <span style="color:#fbbf24">│ - run_ner   │</span>     <span style="color:#60a5fa">│ - merge     │</span>
  <span style="color:#4ade80">│ - store_page│</span>     <span style="color:#fbbf24">│ - extract_  │</span>     <span style="color:#60a5fa">│ - dedupe    │</span>
  <span style="color:#4ade80">│             │</span>     <span style="color:#fbbf24">│   schema_org│</span>     <span style="color:#60a5fa">│             │</span>
  <span style="color:#4ade80">└──────┬──────┘</span>     <span style="color:#fbbf24">└──────┬──────┘</span>     <span style="color:#60a5fa">└──────┬──────┘</span>
         |                   |                   |
         v                   v                   v
  <span style="color:#888">╔═══════════════════════════════════════════════════╗</span>
  <span style="color:#888">║              Shared State (DuckDB)                ║</span>
  <span style="color:#888">║  pages table │ triples table │ entities table     ║</span>
  <span style="color:#888">╚═══════════════════════════════════════════════════╝</span>
                         |
                         v
              <span style="color:#e0e0e0">┌─────────────────┐</span>
              <span style="color:#e0e0e0">│  Search Agent   │</span>
              <span style="color:#e0e0e0">│  Tools:         │</span>
              <span style="color:#e0e0e0">│  - search_index │</span>
              <span style="color:#e0e0e0">│  - query_kg     │</span>
              <span style="color:#e0e0e0">│  - vector_search│</span>
              <span style="color:#e0e0e0">└─────────────────┘</span>
</code></pre>

Each agent has its own loop, its own tools, its own context window. They don't talk to each other directly. They communicate through the database -- the crawl agent writes pages, the extraction agent reads pages and writes triples, the resolution agent reads triples and writes canonical entities. The search agent reads everything.

<table>
  <thead>
    <tr>
      <th>Approach</th>
      <th>Pros</th>
      <th>Cons</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Single agent</strong></td>
      <td>Simple, easy to debug, one context</td>
      <td>Serial execution, context fills fast, one failure stalls everything</td>
    </tr>
    <tr>
      <td><strong>Multi-agent</strong></td>
      <td>Parallel, specialized tools per agent, isolated failures</td>
      <td>Coordination complexity, shared state management, harder to debug</td>
    </tr>
  </tbody>
</table>

The key insight: multi-agent coordination through shared state (a database) is dramatically simpler than direct agent-to-agent messaging. No protocol negotiation. No message queues. No "agent A waits for agent B to finish." Agent A writes rows. Agent B reads rows. The database handles concurrency. This isn't a new idea -- it's how microservices work, just with LLMs in the loop.

## Where this fits in OpenIndex

The knowledge graph from [post 11](/blog/knowledge-graph) already has entities and relationships. OQL from [post 12](/blog/oql-query-language) already has a query interface. Full-text search, vector search, graph queries -- the backends exist or are being built.

What doesn't exist yet is the reasoning layer. The layer that takes a natural language question and decomposes it into the right sequence of tool calls.

Consider: "Find researchers who published papers on transformer architectures and are affiliated with companies that went public in 2025."

That's not one query. It's at least four:

<pre><code><span style="color:#888">// Step 1: Find people who published papers on transformers</span>
<span style="color:#60a5fa">query_knowledge_graph</span>(<span style="color:#4ade80">"
  SELECT DISTINCT t1.subject AS researcher
  FROM triples t1
  JOIN triples t2 ON t2.subject = t1.subject
  WHERE t1.predicate = 'oi:authorOf'
    AND t2.predicate = 'oi:topicOf'
    AND t2.object = 'entity/transformer_architecture'
"</span>)

<span style="color:#888">// Step 2: Find their affiliations</span>
<span style="color:#60a5fa">query_knowledge_graph</span>(<span style="color:#4ade80">"
  SELECT t.subject AS researcher, t.object AS company
  FROM triples t
  WHERE t.predicate = 'oi:affiliatedWith'
    AND t.subject IN (... researchers from step 1 ...)
"</span>)

<span style="color:#888">// Step 3: Check which companies went public in 2025</span>
<span style="color:#60a5fa">search_index</span>({ <span style="color:#60a5fa">"query"</span>: <span style="color:#4ade80">"IPO 2025"</span> })

<span style="color:#888">// Step 4: Cross-reference and filter</span>
<span style="color:#60a5fa">query_knowledge_graph</span>(<span style="color:#4ade80">"
  SELECT ... WHERE company IN (... IPO companies from step 3 ...)
"</span>)</code></pre>

No single query language handles this. OQL can execute each individual query. An agent handles the decomposition -- figuring out the right order, feeding results from one step into the next, deciding when it has enough information to answer.

<div class="note">
  <strong>Status: designed, not built.</strong> The agent layer is on the roadmap, but the infrastructure comes first. DuckDB, the triple store, Tantivy, Vald, OQL -- these are the tools an agent would use. Building the agent before the tools exist is like hiring a carpenter before buying lumber. The infrastructure is what we're shipping now. The reasoning layer comes after the tools are solid.
</div>

## The loop is the easy part

Here's the thing about the agent loop: it's maybe 50 lines of code. Read the goal, call the model with tools, execute the tool call, append the result, repeat. The loop itself is trivial.

What's hard is everything else. Tool design that gives the model enough capability without enough rope to hang itself. Context management that keeps relevant information accessible without blowing the token budget. Stopping conditions that prevent both runaway loops and premature exits. Evaluation -- how do you even measure whether an agent answered a question correctly when the answer required six tool calls and intermediate reasoning?

The model isn't the bottleneck. The loop isn't the bottleneck. The tools, the context, and the evaluation are the bottleneck. An agent is only as good as the environment it operates in. Build a great tool set with clean schemas and well-structured data, and a decent model will do impressive things. Give a frontier model bad tools and noisy data, and it'll hallucinate confidently.

We're building the environment first. The agent comes after.
