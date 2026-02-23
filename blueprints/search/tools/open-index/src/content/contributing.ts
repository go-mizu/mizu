import { icons, cardIcon } from '../icons'

export const contributingPage = `
<h2>Contributing</h2>
<p>OpenIndex is open source. The code lives at <a href="https://github.com/nicholasgasior/gopher-crawl">github.com/nicholasgasior/gopher-crawl</a>. Contributions of any size are welcome -- from typo fixes to new packages.</p>

<hr>

<h3>Getting Started</h3>

<pre><code># Clone the repository
git clone https://github.com/nicholasgasior/gopher-crawl.git
cd gopher-crawl

# Run all tests
make test

# Run tests for a specific package
go test ./pkg/cc/...

# Run a single test
go test -run TestName ./...

# Build the CLI
make install</code></pre>

<h4>Prerequisites</h4>
<ul>
  <li><strong>Go 1.22+</strong> -- Required. The project uses Go 1.22+ ServeMux patterns.</li>
  <li><strong>DuckDB</strong> -- Used for sharded storage and Parquet queries.</li>
  <li><strong>Node.js 20+</strong> -- For the search worker (Hono/CF Workers) and frontend.</li>
</ul>

<hr>

<h3>Key Packages</h3>

<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('globe')} <span>pkg/cc</span></div>
    <p>Common Crawl integration. Downloads columnar index, CDX index, and WARC files. Smart caching, remote S3 queries via DuckDB httpfs, CDX API client.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('zap')} <span>pkg/recrawler</span></div>
    <p>High-throughput URL recrawler. 100K HTTP workers, 20K DNS workers, sharded DuckDB storage. Batch DNS, streaming probe, domain-level rate limiting.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('search')} <span>pkg/dcrawler</span></div>
    <p>Single-domain crawler with HTTP/2 multiplexing. Bloom filter frontier, errgroup workers, coordinator goroutine, sharded result storage.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('database')} <span>pkg/fineweb</span></div>
    <p>FineWeb dataset client. HuggingFace Hub + Viewer API integration. Parquet download, import, and analytics.</p>
  </div>
</div>

<p>The search worker lives in <code>blueprints/search/app/worker/</code> (Hono + TypeScript) with 70+ search engine adapters in <code>engines/</code>.</p>

<hr>

<h3>Code Style</h3>
<ul>
  <li>Go standard formatting. Run <code>gofmt</code> before committing.</li>
  <li>Follow <a href="https://go.dev/doc/effective_go">Effective Go</a> guidelines.</li>
  <li>Error messages: lowercase, no trailing punctuation.</li>
  <li>Commit messages: imperative mood ("Add CDX pagination" not "Added CDX pagination").</li>
  <li>One concern per PR. Do not mix unrelated changes.</li>
</ul>

<hr>

<h3>Pull Request Process</h3>
<ol>
  <li><strong>Open an issue first</strong> for non-trivial changes. This saves time for everyone.</li>
  <li><strong>Fork and branch</strong> -- Work on a descriptive branch name.</li>
  <li><strong>Write tests</strong> -- New features and bug fixes should include tests.</li>
  <li><strong>Submit a PR</strong> against <code>main</code> with a clear description.</li>
  <li><strong>Review</strong> -- The maintainer will review and provide feedback.</li>
</ol>

<hr>

<h3>Communication</h3>
<p>The primary channel is <a href="https://github.com/nicholasgasior/gopher-crawl/issues">GitHub Issues</a>. Use issues for bug reports, feature requests, and questions. For quick discussions, you can also reach out on <a href="https://discord.gg/openindex">Discord</a>.</p>

<div class="note">
  This is an early-stage project maintained by a single developer. Response times may vary, but every issue and PR gets attention.
</div>
`
