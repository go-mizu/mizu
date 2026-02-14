#!/usr/bin/env node
// Offline processor for Common Crawl cluster.idx → KV bulk JSON
// Usage: node scripts/process-cluster.mjs [CRAWL_ID]
//
// Downloads cluster.idx (~110MB), parses SURT keys, aggregates by domain,
// outputs paginated JSON files for wrangler KV bulk upload.

import { writeFile } from 'fs/promises'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))

const CRAWL_ID = process.argv[2] || 'CC-MAIN-2026-04'
const KV_PREFIX = 'v2:'
const DOMAINS_PER_PAGE = 100
const ENTRIES_PER_PAGE = 100
const MAX_ENTRY_PAGES = 100 // Store first 100 pages of raw entries for CDX browse preview
const KV_NAMESPACE_ID = 'a412dc6f75e245e09c90944e156c5cf6'

const CLUSTER_URL = `https://data.commoncrawl.org/cc-index/collections/${CRAWL_ID}/indexes/cluster.idx`

function surtToDomain(surtHost) {
  return surtHost.split(',').reverse().join('.')
}

async function main() {
  console.log(`Processing cluster.idx for ${CRAWL_ID}`)
  console.log(`URL: ${CLUSTER_URL}\n`)

  // HEAD to get size
  const head = await fetch(CLUSTER_URL, { method: 'HEAD' })
  if (!head.ok) throw new Error(`HEAD failed: ${head.status} ${head.statusText}`)
  const totalBytes = parseInt(head.headers.get('content-length') || '0')
  console.log(`Size: ${(totalBytes / 1024 / 1024).toFixed(1)} MB`)

  // Stream download and process line by line
  console.log('Downloading and processing...')
  const res = await fetch(CLUSTER_URL)
  if (!res.ok) throw new Error(`GET failed: ${res.status} ${res.statusText}`)

  const domainCounts = new Map()
  const rawEntries = []
  const maxRawEntries = MAX_ENTRY_PAGES * ENTRIES_PER_PAGE
  let lineCount = 0
  let buffer = ''
  let bytesRead = 0

  const reader = res.body.getReader()
  const decoder = new TextDecoder()

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    bytesRead += value.byteLength
    buffer += decoder.decode(value, { stream: true })

    const lines = buffer.split('\n')
    buffer = lines.pop() // keep incomplete last line

    for (const line of lines) {
      if (!line) continue
      lineCount++

      // Format: SURT_KEY TIMESTAMP\tCDX_FILE\tOFFSET\tLENGTH\tPAGE_NUM
      const tabParts = line.split('\t')
      if (tabParts.length < 2) continue

      const firstPart = tabParts[0]
      const cdxFile = tabParts[1] || ''
      const pageNum = parseInt(tabParts[4] || '0')

      const parenIdx = firstPart.indexOf(')')
      if (parenIdx === -1) continue

      const surtHost = firstPart.substring(0, parenIdx)
      const domain = surtToDomain(surtHost)

      domainCounts.set(domain, (domainCounts.get(domain) || 0) + 1)

      if (rawEntries.length < maxRawEntries) {
        const spaceIdx = firstPart.lastIndexOf(' ')
        rawEntries.push({
          surtKey: spaceIdx > parenIdx ? firstPart.substring(0, spaceIdx) : firstPart,
          domain,
          cdxFile,
          pageNum,
        })
      }
    }

    // Progress
    if (lineCount % 50000 === 0) {
      const pct = totalBytes > 0 ? ((bytesRead / totalBytes) * 100).toFixed(0) : '?'
      process.stdout.write(`\r  ${lineCount} lines, ${(bytesRead / 1024 / 1024).toFixed(0)} MB (${pct}%), ${domainCounts.size} domains`)
    }
  }

  // Process remaining buffer
  if (buffer.trim()) {
    lineCount++
    const tabParts = buffer.split('\t')
    if (tabParts.length >= 2) {
      const firstPart = tabParts[0]
      const cdxFile = tabParts[1] || ''
      const pageNum = parseInt(tabParts[4] || '0')
      const parenIdx = firstPart.indexOf(')')
      if (parenIdx !== -1) {
        const surtHost = firstPart.substring(0, parenIdx)
        const domain = surtToDomain(surtHost)
        domainCounts.set(domain, (domainCounts.get(domain) || 0) + 1)
      }
    }
  }

  console.log(`\n\nTotal lines: ${lineCount}`)
  console.log(`Unique domains: ${domainCounts.size}`)

  // Sort domains by entry count descending
  const sortedDomains = Array.from(domainCounts.entries())
    .map(([domain, count]) => ({ domain, pages: count, size: 0 }))
    .sort((a, b) => b.pages - a.pages)

  const totalDomains = sortedDomains.length
  const totalDomainPages = Math.ceil(totalDomains / DOMAINS_PER_PAGE)
  const totalEntryPages = Math.min(Math.ceil(rawEntries.length / ENTRIES_PER_PAGE), MAX_ENTRY_PAGES)

  console.log(`Domain pages: ${totalDomainPages} (${DOMAINS_PER_PAGE}/page)`)
  console.log(`Entry pages: ${totalEntryPages} (${ENTRIES_PER_PAGE}/page)`)
  console.log(`Top 10 domains:`)
  for (const d of sortedDomains.slice(0, 10)) {
    console.log(`  ${d.domain}: ${d.pages} entries`)
  }

  // Build KV bulk entries
  const kvEntries = []

  // Meta
  kvEntries.push({
    key: `${KV_PREFIX}cluster:${CRAWL_ID}:meta`,
    value: JSON.stringify({ totalDomains, totalPages: totalDomainPages, totalEntries: lineCount, entriesPages: totalEntryPages }),
  })

  // Domain pages
  for (let p = 0; p < totalDomainPages; p++) {
    const start = p * DOMAINS_PER_PAGE
    kvEntries.push({
      key: `${KV_PREFIX}cluster:${CRAWL_ID}:domains:${p}`,
      value: JSON.stringify(sortedDomains.slice(start, start + DOMAINS_PER_PAGE)),
    })
  }

  // Entry pages (for CDX browse landing)
  for (let p = 0; p < totalEntryPages; p++) {
    const start = p * ENTRIES_PER_PAGE
    kvEntries.push({
      key: `${KV_PREFIX}cluster:${CRAWL_ID}:entries:${p}`,
      value: JSON.stringify(rawEntries.slice(start, start + ENTRIES_PER_PAGE)),
    })
  }

  console.log(`\nTotal KV keys: ${kvEntries.length}`)

  // Write bulk JSON files (wrangler kv bulk put limit: 10,000 keys per file)
  const BATCH_SIZE = 10000
  const outFiles = []
  for (let i = 0; i < kvEntries.length; i += BATCH_SIZE) {
    const batch = kvEntries.slice(i, i + BATCH_SIZE)
    const filename = `kv-cluster-${Math.floor(i / BATCH_SIZE)}.json`
    const outPath = join(__dirname, filename)
    await writeFile(outPath, JSON.stringify(batch))
    const sizeMB = (JSON.stringify(batch).length / 1024 / 1024).toFixed(1)
    console.log(`Wrote ${filename} (${batch.length} keys, ${sizeMB} MB)`)
    outFiles.push(outPath)
  }

  console.log('\nUpload with:')
  for (const f of outFiles) {
    console.log(`  npx wrangler kv bulk put "${f}" --namespace-id ${KV_NAMESPACE_ID}`)
  }
}

main().catch(err => { console.error(err); process.exit(1) })
