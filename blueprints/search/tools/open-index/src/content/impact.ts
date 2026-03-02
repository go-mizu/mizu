import { icons, cardIcon } from '../icons'

export const impactPage = `
<h2>Impact</h2>

<p>OpenIndex launched in 2026. We are just getting started.</p>

<p>There are no impressive stats to share yet -- no millions of API queries, no hundreds of research papers, no thousands of users. That is honest. What we can share is what we have built, what we hope to enable, and why it matters.</p>

<hr>

<h3>What Exists Today</h3>

<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('terminal')} <span>Working Tools</span></div>
    <p>A Go CLI with Common Crawl integration, high-throughput recrawler, domain crawler, and sharded DuckDB storage. These are real, tested, running tools.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('search')} <span>Search Worker</span></div>
    <p>A Hono-based search worker deployed on Cloudflare Workers with 70+ search engine adapters. Running at the edge, globally distributed.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('database')} <span>Data Pipeline</span></div>
    <p>Parquet columnar index, CDX API integration, WARC fetching, and sharded result storage. The plumbing for web-scale data processing.</p>
  </div>
</div>

<hr>

<h3>What We Hope to Enable</h3>

<ul>
  <li><strong>Independent research</strong> -- Researchers who want to study the web at scale without depending on commercial APIs or institutional partnerships.</li>
  <li><strong>Small-team search</strong> -- Teams that want to build search products on top of open infrastructure, not proprietary crawl data.</li>
  <li><strong>Open knowledge graphs</strong> -- Structured, queryable representations of web content that anyone can use, extend, and improve.</li>
  <li><strong>Web literacy</strong> -- Tools that help people understand what is on the web, how it connects, and how it changes over time.</li>
</ul>

<hr>

<h3>Be Part of It</h3>

<p>The best way to have impact on a young project is to show up early. If you believe web intelligence should be open, there are ways to help:</p>

<ul>
  <li><strong>Use the tools</strong> -- Run the crawler, query the data, report what breaks.</li>
  <li><strong>Contribute code</strong> -- See the <a href="/contributing">contributing guide</a>.</li>
  <li><strong>Share ideas</strong> -- Open an <a href="https://github.com/nicholasgasior/gopher-crawl/issues">issue</a> with suggestions, use cases, or feedback.</li>
  <li><strong>Spread the word</strong> -- Tell someone who cares about open data.</li>
</ul>

<div class="note">
  We would rather show honest progress than fabricate milestones. Check back. This page will grow as the project does.
</div>
`
