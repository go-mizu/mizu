/**
 * Lightweight HTML text extraction utilities for Cloudflare Workers.
 * Uses string operations and regex -- no DOM parser available.
 */

/**
 * Decode common HTML entities to their text equivalents.
 */
export function decodeHtmlEntities(text: string): string {
  let result = text;
  // Named entities
  result = result.replace(/&amp;/g, '&');
  result = result.replace(/&lt;/g, '<');
  result = result.replace(/&gt;/g, '>');
  result = result.replace(/&quot;/g, '"');
  result = result.replace(/&#39;/g, "'");
  result = result.replace(/&apos;/g, "'");
  result = result.replace(/&nbsp;/g, ' ');
  result = result.replace(/&mdash;/g, '\u2014');
  result = result.replace(/&ndash;/g, '\u2013');
  result = result.replace(/&laquo;/g, '\u00AB');
  result = result.replace(/&raquo;/g, '\u00BB');
  result = result.replace(/&hellip;/g, '\u2026');
  result = result.replace(/&copy;/g, '\u00A9');
  result = result.replace(/&reg;/g, '\u00AE');
  result = result.replace(/&trade;/g, '\u2122');

  // Numeric entities (decimal): &#NNN;
  result = result.replace(/&#(\d+);/g, (_match, code) => {
    const num = parseInt(code, 10);
    return num > 0 && num < 0x10ffff ? String.fromCodePoint(num) : '';
  });

  // Numeric entities (hexadecimal): &#xHHH;
  result = result.replace(/&#x([0-9a-fA-F]+);/g, (_match, code) => {
    const num = parseInt(code, 16);
    return num > 0 && num < 0x10ffff ? String.fromCodePoint(num) : '';
  });

  return result;
}

/**
 * Strip all HTML tags from text and decode entities.
 */
export function extractText(html: string): string {
  // Remove script and style blocks entirely
  let result = html.replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '');
  result = result.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '');

  // Replace <br>, <p>, <div>, <li> tags with spaces for readability
  result = result.replace(/<br\s*\/?>/gi, ' ');
  result = result.replace(/<\/?(p|div|li|h[1-6]|tr|td|th)\b[^>]*>/gi, ' ');

  // Strip remaining tags
  result = result.replace(/<[^>]+>/g, '');

  // Decode entities
  result = decodeHtmlEntities(result);

  // Collapse whitespace
  result = result.replace(/\s+/g, ' ').trim();

  return result;
}

/**
 * Extract the value of a specific attribute from the first occurrence of a tag.
 * Returns empty string if not found.
 *
 * Example: extractAttribute('<a href="https://example.com">', 'a', 'href')
 *          => 'https://example.com'
 */
export function extractAttribute(
  html: string,
  tag: string,
  attr: string
): string {
  // Match the opening tag
  const tagPattern = new RegExp(
    `<${tag}\\b[^>]*?\\b${attr}\\s*=\\s*(?:"([^"]*)"|'([^']*)')`,
    'i'
  );
  const match = html.match(tagPattern);
  if (match) {
    return decodeHtmlEntities(match[1] ?? match[2] ?? '');
  }
  return '';
}

/**
 * Find all elements matching a simple selector.
 * Supports:
 *   - Tag name: "div", "a", "h2"
 *   - Class: ".className" or "div.className"
 *   - ID: "#idName" or "div#idName"
 *   - Attribute: "div[role='link']"
 *   - Data attribute: "div[data-sncf='1']"
 *
 * Returns an array of the outer HTML strings for each matched element.
 */
export function findElements(html: string, selector: string): string[] {
  let tag = '';
  let className = '';
  let idName = '';
  let attrName = '';
  let attrValue = '';

  // Parse selector
  const attrMatch = selector.match(
    /^(\w+)?\[([a-zA-Z0-9_-]+)\s*=\s*['"]([^'"]*)['"]\]$/
  );
  if (attrMatch) {
    tag = attrMatch[1] || '';
    attrName = attrMatch[2];
    attrValue = attrMatch[3];
  } else if (selector.includes('#')) {
    const parts = selector.split('#');
    tag = parts[0] || '';
    idName = parts[1];
  } else if (selector.includes('.')) {
    const parts = selector.split('.');
    tag = parts[0] || '';
    className = parts[1];
  } else {
    tag = selector;
  }

  const results: string[] = [];
  const tagToSearch = tag || '[a-zA-Z][a-zA-Z0-9]*';

  // Build pattern to find opening tags
  let openPattern: RegExp;
  if (attrName) {
    openPattern = new RegExp(
      `<(${tagToSearch})\\b[^>]*?\\b${attrName}\\s*=\\s*["']${escapeRegex(attrValue)}["'][^>]*>`,
      'gi'
    );
  } else if (idName) {
    openPattern = new RegExp(
      `<(${tagToSearch})\\b[^>]*?\\bid\\s*=\\s*["']${escapeRegex(idName)}["'][^>]*>`,
      'gi'
    );
  } else if (className) {
    openPattern = new RegExp(
      `<(${tagToSearch})\\b[^>]*?\\bclass\\s*=\\s*["'][^"']*\\b${escapeRegex(className)}\\b[^"']*["'][^>]*>`,
      'gi'
    );
  } else {
    openPattern = new RegExp(`<(${tagToSearch})\\b[^>]*>`, 'gi');
  }

  let openMatch: RegExpExecArray | null;
  while ((openMatch = openPattern.exec(html)) !== null) {
    const startIndex = openMatch.index;
    const matchedTag = openMatch[1].toLowerCase();

    // Check for self-closing
    if (openMatch[0].endsWith('/>')) {
      results.push(openMatch[0]);
      continue;
    }

    // Find the matching closing tag, handling nesting
    let depth = 1;
    let searchPos = startIndex + openMatch[0].length;
    const closeTagPattern = new RegExp(
      `<(/?)${matchedTag}\\b[^>]*>`,
      'gi'
    );
    closeTagPattern.lastIndex = searchPos;

    let closeMatch: RegExpExecArray | null;
    while (depth > 0 && (closeMatch = closeTagPattern.exec(html)) !== null) {
      if (closeMatch[1] === '/') {
        depth--;
      } else if (!closeMatch[0].endsWith('/>')) {
        depth++;
      }
      if (depth === 0) {
        const endIndex = closeMatch.index + closeMatch[0].length;
        results.push(html.slice(startIndex, endIndex));
      }
    }

    // If we could not find the closing tag, take a reasonable chunk
    if (depth > 0) {
      const maxChunk = Math.min(startIndex + 10000, html.length);
      results.push(html.slice(startIndex, maxChunk));
    }
  }

  return results;
}

/**
 * Escape special regex characters in a string.
 */
function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Extract all href values from anchor tags in an HTML fragment.
 */
export function extractLinks(html: string): string[] {
  const links: string[] = [];
  const re = /<a\b[^>]*?\bhref\s*=\s*(?:"([^"]*)"|'([^']*)')/gi;
  let match: RegExpExecArray | null;
  while ((match = re.exec(html)) !== null) {
    const href = match[1] ?? match[2];
    if (href) {
      links.push(decodeHtmlEntities(href));
    }
  }
  return links;
}
