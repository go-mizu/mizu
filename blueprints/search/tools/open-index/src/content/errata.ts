import { icons, cardIcon } from '../icons'

export const errataPage = `
<h2>Errata</h2>
<p>This page documents known data quality issues, bugs, and caveats in OpenIndex crawls and tools. Issues will be listed here as they are discovered.</p>

<div class="note">
  No known issues yet. OpenIndex is a new project (started February 2026). As crawl data is produced and tools are used more widely, any data quality problems will be documented here transparently.
</div>

<hr>

<h2>Reporting Issues</h2>
<p>If you discover a data quality issue, please report it:</p>

<ul>
  <li><strong>GitHub Issues:</strong> <a href="https://github.com/nicholasgasior/gopher-crawl/issues">github.com/nicholasgasior/gopher-crawl/issues</a></li>
  <li>Include the crawl ID, affected URLs or domains, and a description of the problem.</li>
</ul>

<hr>

<h2>Known Caveats</h2>
<p>These are not bugs but inherent limitations to be aware of:</p>

<details>
  <summary>Common Crawl data reflects the web at crawl time</summary>
  <div class="details-body">
    <p>Pages may have changed or been removed since the crawl. WARC records are snapshots, not live data. Check <code>fetch_time</code> in the Parquet index to see when a page was crawled.</p>
  </div>
</details>

<details>
  <summary>~97% of CC index domains are unreachable from outside their region</summary>
  <div class="details-body">
    <p>When recrawling URLs from CC parquet files, the vast majority of domains (especially country-code TLDs) are dead or unreachable from outside their geographic region. This is expected behavior, not a bug. CC file partitioning is TLD-based (e.g., file 299 is mostly .cn domains, file 0 is mostly .ru).</p>
  </div>
</details>

<details>
  <summary>Per-domain connection limits affect recrawl success rate</summary>
  <div class="details-body">
    <p>Running 50K workers against ~73 unique domains means ~685 connections per domain. Most servers reject this. The default of 8 max connections per domain yields ~57.5% success rate (vs 0.8% without the limit).</p>
  </div>
</details>
`
