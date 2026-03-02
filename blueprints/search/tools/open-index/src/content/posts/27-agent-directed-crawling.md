---
slug: agent-directed-crawling
title: "Agent-Directed Crawling — Let the LLM Pick the Next URL"
date: 2026-03-17
summary: "Traditional crawlers follow every link. Agent-directed crawlers have goals. Instead of crawling the whole web, ask an LLM what's worth fetching next."
tags: [ai, crawling]
---

We crawled 2.5 million pages. Most of them are junk. Parked domains, cookie walls, duplicate content, thin affiliate pages, auto-generated tag indexes, login gates that return 200 OK with zero useful content. A traditional crawler doesn't know the difference between a page worth indexing and a page worth ignoring. It fetches everything and filters later.

That's a lot of "later." Entity extraction runs on every page. Embeddings get generated for every chunk. Storage grows linearly with page count. And at the end, maybe 30-40% of what we crawled actually contributes useful entities to the knowledge graph. The rest is compute spent producing nothing.

Agent-directed crawling flips this: evaluate before fetching. Instead of "crawl everything, filter after," ask a model "is this URL worth fetching?" before spending bandwidth on it.

## The frontier problem

Every crawl maintains a frontier -- the queue of URLs to fetch next. In a traditional crawler, the frontier is ordered by breadth-first, depth-first, or some PageRank-estimated priority. The ordering is mechanical. It doesn't know what's on the other side of the URL.

In an agent-directed crawler, the frontier is ordered by *relevance to a goal*. The agent looks at a URL, its anchor text, its domain, the context of the page that linked to it, and decides: worth fetching, or skip.

The decision is a scoring function:

<pre><code><span style="color:#888">// Traditional frontier scoring</span>
<span style="color:#60a5fa">score</span> = f(depth, domain_authority, discovery_time)

<span style="color:#888">// Agent-directed frontier scoring</span>
<span style="color:#60a5fa">score</span> = f(
  <span style="color:#4ade80">url_structure</span>,        <span style="color:#888">// /research/papers/ vs /terms-of-service</span>
  <span style="color:#4ade80">anchor_text</span>,          <span style="color:#888">// "published findings on..." vs "click here"</span>
  <span style="color:#4ade80">source_page_quality</span>,  <span style="color:#888">// entity-rich page? probably links to good stuff</span>
  <span style="color:#4ade80">domain_reputation</span>,    <span style="color:#888">// known high-quality domain? skip evaluation</span>
  <span style="color:#fbbf24">knowledge_gaps</span>        <span style="color:#888">// what's MISSING from the graph right now?</span>
)</code></pre>

That last signal -- knowledge gaps -- is the interesting one. The crawler doesn't just evaluate URLs in isolation. It asks: "given what we already know, what do we still need?"

## How an LLM evaluates a URL

The agent doesn't fetch the page to decide if it's worth fetching. That'd defeat the purpose. It uses signals available *before* the fetch:

**URL structure.** `/research/papers/2024/transformer-architectures` screams "technical content." `/privacy-policy` doesn't. URL path segments carry a surprising amount of signal. Patterns like `/blog/`, `/docs/`, `/wiki/`, `/publications/` correlate strongly with indexable content.

**Anchor text.** The text that links to a URL is a human-written summary of what's on the other side. "Their 2024 paper on efficient attention mechanisms" tells you more about the target page than the URL itself. "Click here" tells you nothing.

**Source page quality.** If the linking page has 40 extracted entities and 12 structured relationships, it's a high-quality source. Pages linked from high-quality sources tend to be worth fetching. Pages linked from thin content tend to be thin themselves.

**Domain reputation.** arxiv.org, nature.com, github.com -- some domains are known good. Skip the per-URL evaluation and fetch everything. Other domains are known bad. Parked domain registrars, link farms, content mills. Skip everything.

**Knowledge graph gaps.** This is where it gets genuinely useful. More on this in a moment.

Here's a concrete prompt that evaluates a batch of URLs:

<pre><code><span style="color:#888">// System prompt for batch URL evaluation</span>
<span style="color:#e0e0e0">You are a web crawl prioritizer for an open knowledge graph.</span>
<span style="color:#e0e0e0">Score each URL from 0-100 based on likelihood of containing</span>
<span style="color:#e0e0e0">useful, indexable content with extractable entities.</span>
<span style="color:#e0e0e0"></span>
<span style="color:#e0e0e0">Current knowledge gaps:</span>
<span style="color:#e0e0e0">- Organization entities missing location data: 188/200</span>
<span style="color:#e0e0e0">- Person entities missing affiliation: 340/500</span>
<span style="color:#e0e0e0">- Biotech sector: 0 entities (target: 50+)</span>
<span style="color:#e0e0e0"></span>
<span style="color:#e0e0e0">Return JSON array of {url, score, reason}.</span>
<span style="color:#e0e0e0">Score 0 = definitely skip. Score 100 = fetch immediately.</span></code></pre>

And the expected response:

<pre><code><span style="color:#e0e0e0">[</span>
  <span style="color:#e0e0e0">{</span>
    <span style="color:#60a5fa">"url"</span>: <span style="color:#4ade80">"https://biotech-weekly.com/companies/novo-nordisk-hq-relocation"</span>,
    <span style="color:#60a5fa">"score"</span>: <span style="color:#fbbf24">95</span>,
    <span style="color:#60a5fa">"reason"</span>: <span style="color:#4ade80">"Biotech org + location data. Fills two gaps."</span>
  <span style="color:#e0e0e0">},</span>
  <span style="color:#e0e0e0">{</span>
    <span style="color:#60a5fa">"url"</span>: <span style="color:#4ade80">"https://example.com/terms-of-service"</span>,
    <span style="color:#60a5fa">"score"</span>: <span style="color:#fbbf24">2</span>,
    <span style="color:#60a5fa">"reason"</span>: <span style="color:#4ade80">"Legal boilerplate. No extractable entities."</span>
  <span style="color:#e0e0e0">},</span>
  <span style="color:#e0e0e0">{</span>
    <span style="color:#60a5fa">"url"</span>: <span style="color:#4ade80">"https://stanford.edu/research/faculty/jane-doe"</span>,
    <span style="color:#60a5fa">"score"</span>: <span style="color:#fbbf24">88</span>,
    <span style="color:#60a5fa">"reason"</span>: <span style="color:#4ade80">"Faculty page. Person + affiliation + likely publications."</span>
  <span style="color:#e0e0e0">},</span>
  <span style="color:#e0e0e0">{</span>
    <span style="color:#60a5fa">"url"</span>: <span style="color:#4ade80">"https://cooking-blog.net/tag/easy-recipes"</span>,
    <span style="color:#60a5fa">"score"</span>: <span style="color:#fbbf24">5</span>,
    <span style="color:#60a5fa">"reason"</span>: <span style="color:#4ade80">"Tag index page. Thin content, no entities of interest."</span>
  <span style="color:#e0e0e0">},</span>
  <span style="color:#e0e0e0">{</span>
    <span style="color:#60a5fa">"url"</span>: <span style="color:#4ade80">"https://crunchbase.com/organization/moderna"</span>,
    <span style="color:#60a5fa">"score"</span>: <span style="color:#fbbf24">92</span>,
    <span style="color:#60a5fa">"reason"</span>: <span style="color:#4ade80">"Structured org data. Biotech, location, founders, funding."</span>
  <span style="color:#e0e0e0">}</span>
<span style="color:#e0e0e0">]</span></code></pre>

The model isn't fetching pages. It's reading URLs and metadata and guessing -- educated guessing based on patterns it's seen across billions of web pages during training. Those guesses are good enough to separate `/research/papers/` from `/cookie-consent/` without touching the network.

## Batch evaluation, not per-URL

Calling an LLM per URL is insanely expensive at crawl scale. One million frontier URLs at one LLM call each is one million API calls. Nobody's budget survives that.

Instead: batch 50-100 URLs per call. The model sees them together, scores them together, returns one JSON array.

The math:

<table>
  <thead>
    <tr>
      <th>Approach</th>
      <th>LLM Calls</th>
      <th>Cost (small model)</th>
      <th>Time</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Per-URL</strong></td>
      <td>1,000,000</td>
      <td>~$3,000</td>
      <td>Days</td>
    </tr>
    <tr>
      <td><strong>Batch of 20</strong></td>
      <td>50,000</td>
      <td>~$150</td>
      <td>Hours</td>
    </tr>
    <tr>
      <td><strong>Batch of 100</strong></td>
      <td>10,000</td>
      <td style="color:#4ade80"><strong>~$30</strong></td>
      <td>~30 min</td>
    </tr>
  </tbody>
</table>

At $0.003 per call with a small model (batches of 100 URLs, ~2K input tokens per batch), evaluating 1M frontier URLs costs about $30. Compare that to the cost of actually fetching all 1M URLs: bandwidth, storage, compute for entity extraction on 600K pages that turn out to be junk. The evaluation cost is a rounding error next to the wasted downstream processing it prevents.

<div class="note">
  <strong>Batch size tradeoff.</strong> Larger batches are cheaper but less accurate. At 100 URLs per batch, the model gives each URL about 20 tokens of attention. At 20 per batch, it can reason more carefully about each one. We found 50 URLs per batch to be the sweet spot in early testing -- cheap enough to be practical, accurate enough to beat random selection by 3x on precision.
</div>

## Goal-directed vs. exhaustive

<table>
  <thead>
    <tr>
      <th>Property</th>
      <th>Traditional Crawling</th>
      <th>Agent-Directed Crawling</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Coverage</strong></td>
      <td>Exhaustive (everything reachable)</td>
      <td>Selective (high-value pages only)</td>
    </tr>
    <tr>
      <td><strong>Precision</strong></td>
      <td>Low (~30-40% useful)</td>
      <td style="color:#4ade80">High (~70-80% useful, estimated)</td>
    </tr>
    <tr>
      <td><strong>Cost per useful page</strong></td>
      <td>High (fetch 3 pages to get 1 good one)</td>
      <td style="color:#4ade80">Low (fetch 1.3 pages to get 1 good one)</td>
    </tr>
    <tr>
      <td><strong>Frontier ordering</strong></td>
      <td>BFS / DFS / PageRank estimate</td>
      <td>Goal-relevance scoring</td>
    </tr>
    <tr>
      <td><strong>Stopping condition</strong></td>
      <td>Frontier exhausted or budget hit</td>
      <td>Goal satisfied or diminishing returns</td>
    </tr>
    <tr>
      <td><strong>Best for</strong></td>
      <td>Archival, general-purpose indexing</td>
      <td>Knowledge graph completion, domain-specific corpus building</td>
    </tr>
  </tbody>
</table>

The key difference isn't intelligence vs. stupidity. Traditional crawlers are perfectly good at what they do -- broad coverage. Agent-directed crawlers are good at something different: targeted coverage with a budget constraint. If you've got infinite bandwidth and storage, crawl everything. If you don't, be selective.

## Knowledge graph gap analysis

The most interesting application. Instead of crawling based on URL features alone, query the knowledge graph to find what's missing, then crawl specifically to fill those gaps.

<pre><code><span style="color:#888">-- What entity types are we thin on?</span>
<span style="color:#60a5fa">SELECT</span> object <span style="color:#60a5fa">AS</span> entity_type,
       <span style="color:#60a5fa">COUNT</span>(*) <span style="color:#60a5fa">AS</span> count
<span style="color:#60a5fa">FROM</span> triples
<span style="color:#60a5fa">WHERE</span> predicate = <span style="color:#4ade80">'rdf:type'</span>
<span style="color:#60a5fa">GROUP BY</span> object
<span style="color:#60a5fa">ORDER BY</span> count <span style="color:#60a5fa">ASC</span>
<span style="color:#60a5fa">LIMIT</span> <span style="color:#fbbf24">10</span>;

<span style="color:#888">-- Result:</span>
<span style="color:#888">-- MedicalEntity         12</span>
<span style="color:#888">-- GovernmentOrg          28</span>
<span style="color:#888">-- AcademicInstitution    45</span>
<span style="color:#888">-- SportsTeam             51</span>
<span style="color:#888">-- ...</span></code></pre>

Twelve medical entities in a corpus of 2.5M pages? That's a gap. Now find what's missing at the relationship level:

<pre><code><span style="color:#888">-- Which entities are missing key relationships?</span>
<span style="color:#60a5fa">SELECT</span>
  e.subject <span style="color:#60a5fa">AS</span> entity,
  e.object  <span style="color:#60a5fa">AS</span> entity_type,
  <span style="color:#60a5fa">COUNT</span>(<span style="color:#60a5fa">DISTINCT</span> r.predicate) <span style="color:#60a5fa">AS</span> relationship_count
<span style="color:#60a5fa">FROM</span> triples e
<span style="color:#60a5fa">LEFT JOIN</span> triples r
  <span style="color:#60a5fa">ON</span> r.subject = e.subject
  <span style="color:#60a5fa">AND</span> r.predicate != <span style="color:#4ade80">'rdf:type'</span>
<span style="color:#60a5fa">WHERE</span> e.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> e.object = <span style="color:#4ade80">'oi:Organization'</span>
<span style="color:#60a5fa">GROUP BY</span> e.subject, e.object
<span style="color:#60a5fa">HAVING</span> relationship_count < <span style="color:#fbbf24">3</span>
<span style="color:#60a5fa">ORDER BY</span> relationship_count <span style="color:#60a5fa">ASC</span>;

<span style="color:#888">-- Result: 188 organizations with fewer than 3 relationships.</span>
<span style="color:#888">-- Most are missing: location, founder, industry.</span></code></pre>

Now check specifically for location gaps:

<pre><code><span style="color:#888">-- Organizations missing location data</span>
<span style="color:#60a5fa">SELECT</span> e.subject <span style="color:#60a5fa">AS</span> org
<span style="color:#60a5fa">FROM</span> triples e
<span style="color:#60a5fa">WHERE</span> e.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> e.object = <span style="color:#4ade80">'oi:Organization'</span>
  <span style="color:#60a5fa">AND</span> e.subject <span style="color:#60a5fa">NOT IN</span> (
    <span style="color:#60a5fa">SELECT</span> subject <span style="color:#60a5fa">FROM</span> triples
    <span style="color:#60a5fa">WHERE</span> predicate = <span style="color:#4ade80">'oi:locatedIn'</span>
  );

<span style="color:#888">-- 188 out of 200 organizations have no location.</span>
<span style="color:#888">-- Priority: find pages about these orgs that mention location.</span></code></pre>

These queries produce a concrete shopping list. The agent turns it into frontier priorities: "search for Moderna headquarters," "find Stripe office locations," "look up where Anthropic is based." It seeds the frontier with URLs likely to contain the missing data -- company about pages, Crunchbase profiles, Wikipedia infoboxes.

## The feedback loop

This is the key insight. The crawl informs the graph, and the graph directs the next crawl.

<pre><code>  <span style="color:#60a5fa">Gap Analysis</span>
       |
       |  <span style="color:#888">"188 orgs missing location data"</span>
       v
  <span style="color:#4ade80">Seed Frontier</span>
       |
       |  <span style="color:#888">URLs likely to contain org + location info</span>
       v
  <span style="color:#fbbf24">Agent Evaluation</span>
       |
       |  <span style="color:#888">Score and prioritize 10K candidate URLs</span>
       v
  <span style="color:#4ade80">Targeted Crawl</span>
       |
       |  <span style="color:#888">Fetch top 1,000 URLs (score > 70)</span>
       v
  <span style="color:#60a5fa">Entity Extraction</span>
       |
       |  <span style="color:#888">+3,000 new entities, +800 location relationships</span>
       v
  <span style="color:#4ade80">Knowledge Graph Updated</span>
       |
       |  <span style="color:#888">Orgs missing location: 188 → 94</span>
       v
  <span style="color:#60a5fa">Gap Analysis (again)</span>
       |
       |  <span style="color:#888">"94 orgs still missing. Also: 0 biotech entities."</span>
       v
  <span style="color:#888">... repeat ...</span></code></pre>

Each cycle is more targeted than the last. The first round casts a wider net because the graph is sparse and the gaps are broad. By the third or fourth round, the agent is filling specific holes -- "find the headquarters of these 12 remaining organizations" -- and the frontier gets narrower and more precise.

The stopping condition isn't "frontier exhausted." It's "diminishing returns." When a crawl cycle adds fewer than N new entities per 100 pages fetched, the remaining gaps are probably unfillable from the open web, and it's time to stop.

## When NOT to use agent-directed crawling

This isn't a replacement for traditional crawling. It's a complement. Several scenarios where agent direction is the wrong tool:

**Initial seed crawl.** You need data before the agent can evaluate anything. The knowledge graph gap analysis requires a knowledge graph, which requires entities, which require crawled pages. The first crawl has to be brute-force. Crawl broadly, extract entities, build the initial graph. Agent direction kicks in from the second crawl onward.

**Exhaustive archival.** If the goal is "index everything on this domain," just crawl it all. The domain crawler already handles this at 275 pages/s. Agent evaluation adds latency and cost for zero benefit -- you're fetching everything anyway.

**The fetch hot path.** The recrawler runs 50K concurrent workers doing 100K+ URLs/s. There isn't room for an LLM evaluation in that loop. Agent direction works on the frontier *before* handing URLs to the recrawler, not during the fetch. The recrawler stays dumb and fast. The intelligence sits upstream.

<div class="note">
  <strong>Honest about the gap.</strong> Agent-directed crawling works best when you have a specific goal (fill knowledge graph gaps) and a clear quality signal (entity extraction results). For general-purpose "crawl the web" workloads, traditional breadth-first with PageRank-estimated priority is still the right approach. The agent adds value when the goal is precision, not coverage.
</div>

## Architecture sketch

Where agent-directed crawling fits in the existing stack:

<pre><code>  <span style="color:#888">┌───────────────────────────────────────────────┐</span>
  <span style="color:#888">│</span>  <span style="color:#fbbf24">Agent Layer</span> <span style="color:#888">(planned)</span>                         <span style="color:#888">│</span>
  <span style="color:#888">│</span>                                               <span style="color:#888">│</span>
  <span style="color:#888">│</span>  Knowledge Graph  ──>  Gap Analysis            <span style="color:#888">│</span>
  <span style="color:#888">│</span>         |                    |                  <span style="color:#888">│</span>
  <span style="color:#888">│</span>         v                    v                  <span style="color:#888">│</span>
  <span style="color:#888">│</span>  Entity Store         LLM URL Scoring           <span style="color:#888">│</span>
  <span style="color:#888">│</span>         |                    |                  <span style="color:#888">│</span>
  <span style="color:#888">│</span>         v                    v                  <span style="color:#888">│</span>
  <span style="color:#888">│</span>  Feedback Loop    ──>  Curated Frontier          <span style="color:#888">│</span>
  <span style="color:#888">│</span>                              |                  <span style="color:#888">│</span>
  <span style="color:#888">└──────────────────────────────┼──────────────────┘</span>
                               v
  <span style="color:#888">┌───────────────────────────────────────────────┐</span>
  <span style="color:#888">│</span>  <span style="color:#4ade80">Recrawler</span> <span style="color:#888">(existing, unchanged)</span>               <span style="color:#888">│</span>
  <span style="color:#888">│</span>                                               <span style="color:#888">│</span>
  <span style="color:#888">│</span>  50K workers  ·  64 transport shards           <span style="color:#888">│</span>
  <span style="color:#888">│</span>  16-shard DuckDB  ·  per-domain semaphores     <span style="color:#888">│</span>
  <span style="color:#888">│</span>  DNS cache  ·  dead domain tracking             <span style="color:#888">│</span>
  <span style="color:#888">│</span>                                               <span style="color:#888">│</span>
  <span style="color:#888">└───────────────────────────────────────────────┘</span></code></pre>

The recrawler doesn't change. It takes a list of URLs and fetches them as fast as possible. The agent layer sits above it, curating which URLs make it into that list. Today, the frontier is populated by Common Crawl parquet files -- every URL in the index. Tomorrow, the agent filters that list before the recrawler sees it.

<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Brute-force recrawler</td>
      <td><span style="color:#4ade80">Battle-tested (50K workers, 100K+ URLs/s)</span></td>
    </tr>
    <tr>
      <td>Knowledge graph + triple store</td>
      <td><span style="color:#4ade80">Designed</span></td>
    </tr>
    <tr>
      <td>Entity extraction pipeline</td>
      <td><span style="color:#fbbf24">JSON-LD parser prototyped, NER planned</span></td>
    </tr>
    <tr>
      <td>Agent-directed frontier curation</td>
      <td><span style="color:#888">Designed concept, not built</span></td>
    </tr>
    <tr>
      <td>Gap analysis → crawl feedback loop</td>
      <td><span style="color:#888">Depends on knowledge graph</span></td>
    </tr>
  </tbody>
</table>

The order matters. The recrawler and knowledge graph come first -- they're prerequisites. Agent direction is a layer on top. It can't evaluate URLs without a knowledge graph to identify gaps, and it can't verify its own effectiveness without entity extraction to measure what each crawl cycle adds.

But the design is in place, and the interface is clean: the agent produces a list of scored URLs. The recrawler consumes a list of URLs. Everything between those two boundaries is where the intelligence lives. Everything outside them stays exactly as it is -- fast, dumb, and reliable.
