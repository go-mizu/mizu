import { icons, cardIcon } from '../icons'

export const missionPage = `
<h2>Mission</h2>

<p>Web intelligence should not be locked behind corporate APIs.</p>

<p>Today, if you want to search, analyze, or understand the web at scale, you need access to infrastructure that only a handful of companies control. That is a problem. The web was built in the open. The tools to understand it should be open too.</p>

<hr>

<h3>What We Believe</h3>

<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('globe')} <span>Open Data</span></div>
    <p>The open web belongs to everyone. Crawl data, indices, and derived knowledge should be freely accessible -- not paywalled or gatekept.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('code')} <span>Open Tools</span></div>
    <p>Every component is open source. The crawler, the indexer, the search worker, the CLI. You can read every line, run your own instance, or build on top of it.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('bookOpen')} <span>Open Knowledge</span></div>
    <p>Raw data is not enough. We build structured indices, knowledge graphs, and semantic search -- so the web becomes queryable, not just downloadable.</p>
  </div>
</div>

<hr>

<h3>Built in the Open</h3>

<p>OpenIndex started in 2026 as a solo project. There is no company behind it, no funding round, no board of directors. Just a developer who thinks the web deserves better tools.</p>

<p>Everything is public from day one: the code, the roadmap, the decisions, the mistakes. If something is broken, you can see it. If something can be improved, you can change it.</p>

<p>This is not a product. It is infrastructure for anyone who wants to understand the web.</p>

<hr>

<h3>For the Open Web</h3>
<ul>
  <li>A researcher should be able to query web-scale data without a corporate partnership.</li>
  <li>A small team should be able to build search without petabytes of proprietary crawl data.</li>
  <li>A student should have the same access to web intelligence as an engineer at a tech giant.</li>
  <li>Anyone, anywhere, should be able to understand what is on the web and how it connects.</li>
</ul>

<p>That is the mission. We are early, and there is a lot of work to do. If this matters to you, <a href="/contributing">contribute</a>.</p>
`
