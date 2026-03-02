import { icons, cardIcon } from '../icons'

export const teamPage = `
<h2>Team</h2>

<p>OpenIndex is currently a solo project.</p>

<hr>

<div class="team-grid">
  <div class="team-card">
    <div class="team-avatar">${icons.user}</div>
    <h4>Maintainer</h4>
    <p>Creator and sole developer. Building OpenIndex as part of the <a href="https://github.com/nicholasgasior/gopher-crawl">Mizu ecosystem</a>.</p>
  </div>
</div>

<hr>

<h3>Why Solo?</h3>
<p>Not every project needs a team page full of headshots and titles. OpenIndex started because one developer wanted better open tools for web intelligence. That is the entire origin story.</p>

<p>The codebase is real. The tools work. The ambition is large. But it is honest to say: right now, this is one person writing code.</p>

<hr>

<h3>Want to Join?</h3>
<p>This does not have to stay a solo project. If you care about open web data, there is meaningful work to do across the stack:</p>

<ul>
  <li>Go backend -- crawlers, indexers, data pipelines</li>
  <li>TypeScript -- search worker, frontend, engine adapters</li>
  <li>Systems programming -- Zig recrawler, performance optimization</li>
  <li>Data engineering -- DuckDB, Parquet, analytics</li>
  <li>Design -- ontology, knowledge graph schema, query language</li>
</ul>

<p>No application process. No interviews. Just open a PR or start a conversation on <a href="https://github.com/nicholasgasior/gopher-crawl/issues">GitHub Issues</a>.</p>

<p>See the <a href="/contributing">Contributing Guide</a> to get started.</p>
`
