import { readFileSync, writeFileSync } from "node:fs";
import { Marked } from "marked";

const md = readFileSync("src/docs.md", "utf8");

const sidebarEntries = [];

const marked = new Marked({
  renderer: {
    heading({ text, depth }) {
      // Parse {#custom-id} from rendered text
      const idMatch = text.match(/\s*\{#([\w-]+)\}\s*$/);
      const id = idMatch
        ? idMatch[1]
        : text
            .replace(/<[^>]+>/g, "")
            .toLowerCase()
            .replace(/[^\w]+/g, "-")
            .replace(/^-|-$/g, "");
      const cleanText = idMatch
        ? text.replace(/\s*\{#[\w-]+\}\s*$/, "")
        : text;

      if (depth <= 2) {
        sidebarEntries.push({ id, text: cleanText, level: depth });
      }

      return `<h${depth} id="${id}">${cleanText}</h${depth}>\n`;
    },

    code({ text }) {
      const escaped = text
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
      return `<pre><button class="cb" onclick="cp(this)">Copy</button><code>${escaped}</code></pre>\n`;
    },

    table({ header, body }) {
      return `<table class="ref-table">\n${header}${body}</table>\n`;
    },
  },
});

const contentHtml = marked.parse(md);

const sidebarHtml = sidebarEntries
  .map((e) => `  <a href="#${e.id}">${e.text}</a>`)
  .join("\n");

const output = `// Auto-generated from docs.md — do not edit
export const docsContentHtml = ${JSON.stringify(contentHtml)};
export const docsSidebarHtml = ${JSON.stringify(sidebarHtml)};
`;

writeFileSync("src/docs-content.ts", output);
console.log("✓ Built src/docs-content.ts from src/docs.md");
