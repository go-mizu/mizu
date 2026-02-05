/**
 * Simple XML parser for Cloudflare Workers.
 * Provides basic element extraction by tag name.
 * Used primarily for arXiv ATOM feed parsing.
 */

import { decodeHtmlEntities } from './html-parser';

export interface XmlElement {
  tag: string;
  attributes: Record<string, string>;
  content: string;
  children: XmlElement[];
}

/**
 * Extract all elements with a given tag name from XML text.
 * Returns the full inner content (including nested tags) as a string.
 */
export function getElementsByTagName(
  xml: string,
  tagName: string
): string[] {
  const results: string[] = [];
  const openPattern = new RegExp(`<${tagName}(?:\\s[^>]*)?>`, 'gi');
  const closeTag = `</${tagName}>`;

  let openMatch: RegExpExecArray | null;
  while ((openMatch = openPattern.exec(xml)) !== null) {
    const contentStart = openMatch.index + openMatch[0].length;

    // Check for self-closing tag
    if (openMatch[0].endsWith('/>')) {
      results.push('');
      continue;
    }

    // Find matching close tag, handling nesting
    let depth = 1;
    let pos = contentStart;

    while (depth > 0 && pos < xml.length) {
      const nextOpen = xml.indexOf(`<${tagName}`, pos);
      const nextClose = xml.toLowerCase().indexOf(closeTag.toLowerCase(), pos);

      if (nextClose === -1) {
        // No closing tag found, take rest of string
        break;
      }

      if (nextOpen !== -1 && nextOpen < nextClose) {
        // Check if it is truly an opening tag (not a different tag starting with same prefix)
        const afterTag = xml[nextOpen + tagName.length + 1];
        if (
          afterTag === '>' ||
          afterTag === ' ' ||
          afterTag === '/' ||
          afterTag === '\n' ||
          afterTag === '\r' ||
          afterTag === '\t'
        ) {
          // Check for self-closing
          const openEnd = xml.indexOf('>', nextOpen);
          if (openEnd !== -1 && xml[openEnd - 1] !== '/') {
            depth++;
          }
        }
        pos = nextOpen + tagName.length + 1;
      } else {
        depth--;
        if (depth === 0) {
          results.push(xml.slice(contentStart, nextClose));
        }
        pos = nextClose + closeTag.length;
      }
    }
  }

  return results;
}

/**
 * Get the text content of the first element with the given tag name.
 * Strips inner XML tags and decodes entities.
 */
export function getTextContent(xml: string, tagName: string): string {
  const elements = getElementsByTagName(xml, tagName);
  if (elements.length === 0) {
    return '';
  }
  // Strip all XML/HTML tags
  const stripped = elements[0].replace(/<[^>]+>/g, '');
  return decodeHtmlEntities(stripped).trim();
}

/**
 * Get text content of all elements with the given tag name.
 */
export function getAllTextContent(xml: string, tagName: string): string[] {
  const elements = getElementsByTagName(xml, tagName);
  return elements.map((el) => {
    const stripped = el.replace(/<[^>]+>/g, '');
    return decodeHtmlEntities(stripped).trim();
  });
}

/**
 * Get an attribute value from the first element with a given tag name.
 * If attrName is specified, returns the value of that attribute.
 */
export function getElementAttribute(
  xml: string,
  tagName: string,
  attrName: string
): string[] {
  const results: string[] = [];
  const pattern = new RegExp(
    `<${tagName}\\b[^>]*?\\b${attrName}\\s*=\\s*(?:"([^"]*)"|'([^']*)')`,
    'gi'
  );

  let match: RegExpExecArray | null;
  while ((match = pattern.exec(xml)) !== null) {
    results.push(decodeHtmlEntities(match[1] ?? match[2] ?? ''));
  }

  return results;
}

/**
 * Parse an XML string into a simplified element structure.
 * This is a lightweight parser -- it does not handle all XML edge cases,
 * but works well for clean XML like arXiv ATOM feeds.
 */
export function parseXml(xml: string): XmlElement[] {
  const elements: XmlElement[] = [];
  const tagPattern = /<(\w[\w:-]*)((?:\s+[\w:-]+\s*=\s*(?:"[^"]*"|'[^']*'))*)\s*(\/?)>/g;

  let match: RegExpExecArray | null;
  while ((match = tagPattern.exec(xml)) !== null) {
    const tag = match[1];
    const attrsRaw = match[2] || '';
    const selfClosing = match[3] === '/';

    // Parse attributes
    const attributes: Record<string, string> = {};
    const attrPattern = /([\w:-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')/g;
    let attrMatch: RegExpExecArray | null;
    while ((attrMatch = attrPattern.exec(attrsRaw)) !== null) {
      attributes[attrMatch[1]] = decodeHtmlEntities(
        attrMatch[2] ?? attrMatch[3] ?? ''
      );
    }

    if (selfClosing) {
      elements.push({ tag, attributes, content: '', children: [] });
    } else {
      // Find content until closing tag
      const closeTag = `</${tag}>`;
      const closeIndex = xml
        .toLowerCase()
        .indexOf(closeTag.toLowerCase(), match.index + match[0].length);
      if (closeIndex !== -1) {
        const content = xml.slice(
          match.index + match[0].length,
          closeIndex
        );
        elements.push({
          tag,
          attributes,
          content,
          children: parseXml(content),
        });
      }
    }
  }

  return elements;
}
