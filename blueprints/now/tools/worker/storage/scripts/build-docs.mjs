import { readFileSync, writeFileSync } from "node:fs";
import { marked } from "marked";

const md = readFileSync("src/docs.md", "utf8");
const tokens = marked.lexer(md);
const HTTP = ["GET", "POST", "PUT", "DELETE", "HEAD", "PATCH"];

// ── Helpers ──────────────────────────────────────────────────────────────

function esc(s) {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function textOf(token) {
  if (!token) return "";
  if (token.type === "text" || token.type === "codespan" || token.type === "escape")
    return token.text || "";
  if (token.tokens) return token.tokens.map(textOf).join("");
  return token.text || "";
}

function renderInline(toks) {
  if (!toks) return "";
  return toks
    .map((t) => {
      switch (t.type) {
        case "text":
          return esc(t.text);
        case "codespan":
          return `<code>${esc(t.text)}</code>`;
        case "strong":
          return `<strong>${renderInline(t.tokens)}</strong>`;
        case "em":
          return `<em>${renderInline(t.tokens)}</em>`;
        case "link":
          return `<a href="${esc(t.href)}">${renderInline(t.tokens)}</a>`;
        case "escape":
          return esc(t.text);
        case "br":
          return "<br>";
        case "html":
          return t.raw;
        default:
          return esc(t.raw || "");
      }
    })
    .join("");
}

// ── Syntax highlighting (single-pass, regex-based) ───────────────────────

function hlJSON(code) {
  return code.replace(
    /"([^"\\]*(?:\\.[^"\\]*)*)"\s*:|"([^"\\]*(?:\\.[^"\\]*)*)"|(-?\d+(?:\.\d+)?)\b|\b(true|false|null)\b/g,
    (m, prop, str, num, kw) => {
      if (prop !== undefined)
        return `<span class="hl-prop">"${prop}"</span>:`;
      if (str !== undefined)
        return `<span class="hl-str">"${str}"</span>`;
      if (num !== undefined) return `<span class="hl-num">${num}</span>`;
      if (kw !== undefined) return `<span class="hl-kw">${kw}</span>`;
      return m;
    },
  );
}

function hlBash(code) {
  return code.replace(
    /(#[^\n]*)|('(?:[^'\\]|\\.)*')|("(?:[^"\\]|\\.)*")/g,
    (m, comment, sq, dq) => {
      if (comment !== undefined && comment !== null)
        return `<span class="hl-comment">${comment}</span>`;
      if (sq) return `<span class="hl-str">${sq}</span>`;
      if (dq) return `<span class="hl-str">${dq}</span>`;
      return m;
    },
  );
}

function hlJS(code) {
  return code.replace(
    /(\/\/[^\n]*)|("(?:[^"\\]|\\.)*")|(`(?:[^`\\]|\\.)*`)|(\b(?:const|let|var|await|async|function|return|if|else|new|throw)\b)|(\b\d+\b)|\b(true|false|null)\b/g,
    (m, comment, dq, tl, kw, num, lit) => {
      if (comment !== undefined && comment !== null)
        return `<span class="hl-comment">${comment}</span>`;
      if (dq) return `<span class="hl-str">${dq}</span>`;
      if (tl) return `<span class="hl-str">${tl}</span>`;
      if (kw) return `<span class="hl-kw">${kw}</span>`;
      if (num) return `<span class="hl-num">${num}</span>`;
      if (lit) return `<span class="hl-kw">${lit}</span>`;
      return m;
    },
  );
}

function highlight(code, lang) {
  const s = esc(code);
  if (lang === "json") return hlJSON(s);
  if (lang === "bash") return hlBash(s);
  if (lang === "javascript" || lang === "js") return hlJS(s);
  return s;
}

// ── Type & required badges ───────────────────────────────────────────────

function typeBadge(raw) {
  const t = raw.toLowerCase();
  let cls = "type";
  if (t.includes("string") && (t.includes("[]") || t.includes("array")))
    cls += " type--arr";
  else if (t.includes("string")) cls += " type--str";
  else if (t.includes("int") || t === "number") cls += " type--int";
  else if (t.includes("bool")) cls += " type--bool";
  else if (t.includes("[]") || t.includes("array")) cls += " type--arr";
  else if (t.includes("object") || t === "json") cls += " type--obj";
  return `<span class="${cls}">${esc(raw)}</span>`;
}

function reqBadge(val) {
  return val.toLowerCase().trim() === "yes"
    ? '<span class="req req--yes">required</span>'
    : '<span class="req req--no">optional</span>';
}

function tabLabel(lang) {
  if (lang === "bash") return "curl";
  if (lang === "javascript" || lang === "js") return "JavaScript";
  if (lang === "json") return "JSON";
  return lang || "Code";
}

// ── State ────────────────────────────────────────────────────────────────

const sidebar = [];
let html = "";
let codeQueue = [];
let tableCtx = null;
let respStatus = "200 OK";

// ── Code-group flushing ──────────────────────────────────────────────────

function preBlock(code, lang) {
  const hl = highlight(code, lang);
  return `<pre><button class="cb" onclick="cp(this)">Copy</button><code>${hl}</code></pre>`;
}

function flushCodes() {
  if (codeQueue.length === 0) return "";
  const blocks = [...codeQueue];
  codeQueue = [];

  // Single block → standalone <pre>
  if (blocks.length === 1) {
    return preBlock(blocks[0].text, blocks[0].lang) + "\n";
  }

  // Multiple blocks: split into request code-group + optional JSON response
  const lastIsJSON = blocks[blocks.length - 1].lang === "json";
  const reqBlocks = lastIsJSON ? blocks.slice(0, -1) : blocks;
  const resBlock = lastIsJSON ? blocks[blocks.length - 1] : null;

  let out = "";

  if (reqBlocks.length > 0) {
    out += '<div class="code-group"><div class="code-tabs">';
    reqBlocks.forEach((b, i) => {
      out += `<button class="code-tab${i === 0 ? " active" : ""}" onclick="switchTab(this)">${tabLabel(b.lang)}</button>`;
    });
    out += "</div>";
    reqBlocks.forEach((b, i) => {
      out += `<div class="code-panel${i === 0 ? " active" : ""}">${preBlock(b.text, b.lang)}</div>`;
    });
    out += "</div>\n";
  }

  if (resBlock) {
    out += `<div class="response"><div class="response-header"><span class="response-status">${respStatus}</span> Response</div>${preBlock(resBlock.text, "json")}</div>\n`;
  }

  return out;
}

// ── Section-label detection ──────────────────────────────────────────────

const PARAM_LABELS = [
  "request body",
  "path parameters",
  "query parameters",
  "headers",
  "response headers",
  "response",
];

function isParamLabel(label) {
  const lower = label.toLowerCase();
  return PARAM_LABELS.some((l) => lower.startsWith(l));
}

function extractStatus(text) {
  const m = text.match(/(\d{3})\s+(\w+)/);
  return m ? `${m[1]} ${m[2]}` : "200 OK";
}

// ── Table renderers ──────────────────────────────────────────────────────

function renderParamsTable(t) {
  const headers = t.header.map((h) => renderInline(h.tokens));
  const headerTexts = t.header.map((h) => textOf(h).toLowerCase());
  const typeIdx = headerTexts.indexOf("type");
  const reqIdx = headerTexts.indexOf("required");
  const hasRequired = reqIdx >= 0;

  let out = '<table class="params"><thead><tr>';
  for (let j = 0; j < headers.length; j++) {
    if (j === reqIdx) continue; // skip Required column header
    out += `<th>${headers[j]}</th>`;
  }
  out += "</tr></thead><tbody>";

  for (const row of t.rows) {
    out += "<tr>";
    for (let j = 0; j < row.length; j++) {
      if (j === reqIdx) continue; // skip Required cell

      if (j === typeIdx) {
        // Type badge
        const rawType = row[j].tokens.map(textOf).join("");
        out += `<td>${typeBadge(rawType)}</td>`;
      } else if (j === 0 && hasRequired) {
        // Name cell: append required badge
        const rawReq = row[reqIdx].tokens.map(textOf).join("");
        out += `<td>${renderInline(row[j].tokens)} ${reqBadge(rawReq)}</td>`;
      } else {
        out += `<td>${renderInline(row[j].tokens)}</td>`;
      }
    }
    out += "</tr>";
  }
  out += "</tbody></table>\n";
  return out;
}

function renderRefTable(t) {
  let out = '<table class="ref-table"><thead><tr>';
  for (const h of t.header) {
    out += `<th>${renderInline(h.tokens)}</th>`;
  }
  out += "</tr></thead><tbody>";
  for (const row of t.rows) {
    out += "<tr>";
    for (const cell of row) {
      out += `<td>${renderInline(cell.tokens)}</td>`;
    }
    out += "</tr>";
  }
  out += "</tbody></table>\n";
  return out;
}

// ── Token walker ─────────────────────────────────────────────────────────

function walk(toks) {
  for (let i = 0; i < toks.length; i++) {
    const t = toks[i];

    // Flush code buffer on any non-code, non-space token
    if (t.type !== "code" && t.type !== "space" && codeQueue.length > 0) {
      html += flushCodes();
    }

    switch (t.type) {
      case "heading": {
        const idMatch = t.text.match(/\s*\{#([\w-]+)\}\s*$/);
        const id = idMatch
          ? idMatch[1]
          : t.text
              .replace(/<[^>]+>/g, "")
              .toLowerCase()
              .replace(/[^\w]+/g, "-")
              .replace(/^-|-$/g, "");
        let cleanText = idMatch
          ? t.text.replace(/\s*\{#[\w-]+\}\s*$/, "")
          : t.text;

        let sidebarText = cleanText;

        // h2: detect HTTP method prefix → add badges
        if (t.depth === 2) {
          for (const m of HTTP) {
            if (cleanText.startsWith(m + " ")) {
              cleanText = `<span class="method method--${m.toLowerCase()}">${m}</span> ${cleanText.slice(m.length + 1)}`;
              sidebarText = `<span class="sm sm-${m.toLowerCase()}">${m}</span>${sidebarText.slice(m.length + 1)}`;
              break;
            }
          }
        }

        if (t.depth === 1) {
          sidebar.push({ type: "group", id, text: cleanText });
          html += `<h1 id="${id}">${cleanText}</h1>\n`;
        } else if (t.depth <= 2) {
          sidebar.push({ type: "link", id, text: sidebarText });
          html += `<h2 id="${id}">${cleanText}</h2>\n`;
        } else {
          html += `<h${t.depth} id="${id}">${cleanText}</h${t.depth}>\n`;
        }
        tableCtx = null;
        respStatus = "200 OK";
        break;
      }

      case "paragraph": {
        // Check for bold section label: **Label** or **Label** with desc
        if (t.tokens && t.tokens[0]?.type === "strong") {
          const label = textOf(t.tokens[0]);
          if (isParamLabel(label)) {
            tableCtx = "params";
            if (label.toLowerCase().startsWith("response")) {
              const restText = t.tokens.slice(1).map(textOf).join("");
              respStatus = extractStatus(label + restText);
            }
            html += `<h3>${esc(label)}</h3>\n`;
            // Render description after the label
            const restHtml = renderInline(t.tokens.slice(1))
              .replace(/^\s*[—–\-]\s*/, "")
              .trim();
            if (restHtml) html += `<p>${restHtml}</p>\n`;
            break;
          }
        }

        // Base URL detection
        const raw = t.tokens ? t.tokens.map(textOf).join("") : t.text || "";
        if (raw.startsWith("Base URL:")) {
          const codeTok = t.tokens?.find((tk) => tk.type === "codespan");
          const url = codeTok ? codeTok.text : "";
          html += `<div class="base-url"><span class="base-url-label">Base URL</span><span class="base-url-val">${esc(url)}</span></div>\n`;
          break;
        }

        // Regular paragraph
        html += `<p>${renderInline(t.tokens)}</p>\n`;
        break;
      }

      case "code": {
        codeQueue.push({ text: t.text, lang: t.lang || "" });
        break;
      }

      case "table": {
        if (tableCtx === "params") {
          html += renderParamsTable(t);
          tableCtx = null;
        } else {
          html += renderRefTable(t);
        }
        break;
      }

      case "list": {
        const tag = t.ordered ? "ol" : "ul";
        html += `<${tag}>`;
        for (const item of t.items) {
          html += `<li>${item.tokens ? item.tokens.map((sub) => (sub.type === "text" ? renderInline(sub.tokens || [{ type: "text", text: sub.text }]) : "")).join("") : ""}</li>`;
        }
        html += `</${tag}>\n`;
        break;
      }

      case "blockquote": {
        html += "<blockquote>";
        walk(t.tokens);
        html += "</blockquote>\n";
        break;
      }

      case "hr":
        html += "<hr>\n";
        break;

      case "space":
        break;

      default:
        break;
    }
  }

  // Flush remaining code blocks
  if (codeQueue.length > 0) {
    html += flushCodes();
  }
}

// ── Run ──────────────────────────────────────────────────────────────────

walk(tokens);

const sidebarHtml = sidebar
  .map((e) =>
    e.type === "group"
      ? `  <a href="#${e.id}" class="sg">${e.text}</a>`
      : `  <a href="#${e.id}" class="sub">${e.text}</a>`,
  )
  .join("\n");

const output = `// Auto-generated from docs.md - do not edit
export const docsContentHtml = ${JSON.stringify(html)};
export const docsSidebarHtml = ${JSON.stringify(sidebarHtml)};
`;

writeFileSync("src/pages/docs-content.ts", output);
console.log("✓ Built src/pages/docs-content.ts from src/docs.md");
