const ESC: Record<string, string> = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }
function esc(s: string): string { return s.replace(/[&<>"']/g, c => ESC[c] || c) }

export function renderMarkdown(md: string, citationCount: number = 0): string {
  const lines = md.split('\n')
  const out: string[] = []
  let inCode = false
  let codeLang = ''
  let codeLines: string[] = []
  let inList = ''
  let inBlockquote = false

  function closeList() {
    if (inList) { out.push(inList === 'ul' ? '</ul>' : '</ol>'); inList = '' }
  }
  function closeBlockquote() {
    if (inBlockquote) { out.push('</blockquote>'); inBlockquote = false }
  }

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]

    // Code blocks
    if (line.startsWith('```')) {
      if (inCode) {
        out.push(`<div class="cb"><div class="cb-h"><span>${esc(codeLang || 'code')}</span><button onclick="navigator.clipboard.writeText(this.closest('.cb').querySelector('code').textContent)" class="cb-cp">Copy</button></div><pre><code>${esc(codeLines.join('\n'))}</code></pre></div>`)
        inCode = false
        codeLines = []
        codeLang = ''
      } else {
        closeList(); closeBlockquote()
        inCode = true
        codeLang = line.slice(3).trim()
      }
      continue
    }
    if (inCode) { codeLines.push(line); continue }

    // Blank line
    if (!line.trim()) { closeList(); closeBlockquote(); continue }

    // Blockquote
    if (line.startsWith('> ')) {
      closeList()
      if (!inBlockquote) { out.push('<blockquote>'); inBlockquote = true }
      out.push(`<p>${inline(line.slice(2), citationCount)}</p>`)
      continue
    }
    closeBlockquote()

    // Headers
    const hm = line.match(/^(#{1,6})\s+(.+)/)
    if (hm) {
      closeList()
      const level = hm[1].length
      out.push(`<h${level}>${inline(hm[2], citationCount)}</h${level}>`)
      continue
    }

    // Horizontal rule
    if (/^[-*_]{3,}\s*$/.test(line)) { closeList(); out.push('<hr>'); continue }

    // Unordered list
    const ulm = line.match(/^(\s*)[-*+]\s+(.+)/)
    if (ulm) {
      if (inList !== 'ul') { closeList(); out.push('<ul>'); inList = 'ul' }
      out.push(`<li>${inline(ulm[2], citationCount)}</li>`)
      continue
    }

    // Ordered list
    const olm = line.match(/^(\s*)\d+[.)]\s+(.+)/)
    if (olm) {
      if (inList !== 'ol') { closeList(); out.push('<ol>'); inList = 'ol' }
      out.push(`<li>${inline(olm[2], citationCount)}</li>`)
      continue
    }

    // Table
    if (line.includes('|') && line.trim().startsWith('|')) {
      closeList()
      const rows: string[][] = []
      let j = i
      while (j < lines.length && lines[j].trim().startsWith('|')) {
        const cells = lines[j].trim().replace(/^\||\|$/g, '').split('|').map(c => c.trim())
        if (!/^[-:\s|]+$/.test(lines[j])) rows.push(cells)
        j++
      }
      if (rows.length > 0) {
        out.push('<div class="table-wrap"><table>')
        out.push('<thead><tr>' + rows[0].map(c => `<th>${inline(c, citationCount)}</th>`).join('') + '</tr></thead>')
        if (rows.length > 1) {
          out.push('<tbody>')
          for (let k = 1; k < rows.length; k++) {
            out.push('<tr>' + rows[k].map(c => `<td>${inline(c, citationCount)}</td>`).join('') + '</tr>')
          }
          out.push('</tbody>')
        }
        out.push('</table></div>')
      }
      i = j - 1
      continue
    }

    // Paragraph
    closeList()
    out.push(`<p>${inline(line, citationCount)}</p>`)
  }

  closeList(); closeBlockquote()
  if (inCode && codeLines.length > 0) {
    out.push(`<div class="cb"><pre><code>${esc(codeLines.join('\n'))}</code></pre></div>`)
  }

  return out.join('\n')
}

function inline(text: string, citationCount: number): string {
  let s = esc(text)

  // Citation references [1], [2] etc → clickable superscripts
  if (citationCount > 0) {
    s = s.replace(/\[(\d+)\]/g, (_, n) => {
      const num = parseInt(n)
      if (num >= 1 && num <= citationCount) {
        return `<a href="#cite-${num}" class="cr" title="Source ${num}">${num}</a>`
      }
      return `[${n}]`
    })
  }

  // Bold
  s = s.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  // Italic
  s = s.replace(/\*(.+?)\*/g, '<em>$1</em>')
  s = s.replace(/_(.+?)_/g, '<em>$1</em>')
  // Strikethrough
  s = s.replace(/~~(.+?)~~/g, '<del>$1</del>')
  // Inline code
  s = s.replace(/`([^`]+)`/g, '<code>$1</code>')
  // Links
  s = s.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>')
  // Auto-link URLs
  s = s.replace(/(^|[\s(])(https?:\/\/[^\s)<]+)/g, '$1<a href="$2" target="_blank" rel="noopener">$2</a>')

  return s
}
