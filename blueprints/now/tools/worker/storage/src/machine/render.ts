/**
 * Convert raw markdown text to styled HTML for the machine-view panel.
 *
 * Strategy: extract code blocks first (replacing with placeholders),
 * then apply inline transforms, then re-insert code blocks.
 * This prevents headings/bold/links inside code from being styled.
 */
export function markdownToHtml(md: string): string {
  // 1. Extract fenced code blocks before any other processing
  const codeBlocks: string[] = [];
  let text = md.replace(/^```(\w*)\n([\s\S]*?)^```$/gm, (_, lang, code) => {
    const escaped = code
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
    const langTag = lang
      ? `<span class="code-lang">${lang}</span>\n`
      : "";
    const idx = codeBlocks.length;
    codeBlocks.push(
      `<div class="code-fence">${langTag}${escaped}</div>`,
    );
    return `%%CODEBLOCK_${idx}%%`;
  });

  // 2. Extract tables before inline processing
  const tables: string[] = [];
  text = text.replace(
    /^(\|.+\|)\n(\|[-| :]+\|)\n((?:\|.+\|\n?)*)/gm,
    (_, header, _sep, body) => {
      const hCells = header
        .split("|")
        .filter((c: string) => c.trim())
        .map((c: string) => c.trim());
      const rows = body
        .trim()
        .split("\n")
        .filter((r: string) => r.trim())
        .map((r: string) =>
          r
            .split("|")
            .filter((c: string) => c.trim())
            .map((c: string) => c.trim()),
        );

      let html = '<div class="tbl">';
      html +=
        '<div class="tbl-h">' +
        hCells.map((c: string) => `<span class="tbl-c">${esc(c)}</span>`).join("") +
        "</div>";
      for (const row of rows) {
        html +=
          '<div class="tbl-r">' +
          row.map((c: string) => `<span class="tbl-c">${esc(c)}</span>`).join("") +
          "</div>";
      }
      html += "</div>";
      const idx = tables.length;
      tables.push(html);
      return `%%TABLE_${idx}%%`;
    },
  );

  // 3. HTML-escape the remaining text
  text = text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");

  // 4. Apply inline markdown transforms
  text = text
    // Headings
    .replace(/^(#{1,3}) (.+)$/gm, (_, hashes, content) => {
      const level = hashes.length;
      return `<span class="h${level}">${hashes} ${content}</span>`;
    })

    // Blockquotes (&gt; after escaping)
    .replace(/^&gt; (.+)$/gm, '<span class="bq">$1</span>')

    // Bold
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")

    // Links: [text](url)
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a class="link" href="$2">$1</a>')

    // Inline code
    .replace(/`([^`]+)`/g, '<code class="ic">$1</code>')

    // Unordered list items
    .replace(/^- (.+)$/gm, '<span class="li">- $1</span>')

    // Ordered list items
    .replace(/^(\d+)\. (.+)$/gm, '<span class="li">$1. $2</span>');

  // 5. Re-insert tables (after HTML escaping, placeholders became %%TABLE_0%%)
  for (let i = 0; i < tables.length; i++) {
    text = text.replace(`%%TABLE_${i}%%`, tables[i]);
  }

  // 6. Re-insert code blocks
  for (let i = 0; i < codeBlocks.length; i++) {
    text = text.replace(`%%CODEBLOCK_${i}%%`, codeBlocks[i]);
  }

  return text;
}

/** Escape HTML entities and render inline code within table cells. */
function esc(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/`([^`]+)`/g, '<code class="ic">$1</code>');
}
