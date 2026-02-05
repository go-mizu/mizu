import { describe, it, expect } from 'vitest';
import {
  extractText,
  extractAttribute,
  findElements,
  decodeHtmlEntities,
  extractLinks,
} from './html-parser';

describe('html-parser', () => {
  describe('extractText', () => {
    it('extracts text from simple HTML', () => {
      const html = '<p>Hello World</p>';
      expect(extractText(html)).toBe('Hello World');
    });

    it('handles multiple elements', () => {
      const html = '<div><p>First</p><p>Second</p></div>';
      const result = extractText(html);
      expect(result).toContain('First');
      expect(result).toContain('Second');
    });

    it('removes script tags', () => {
      const html = '<div>Text<script>alert("bad")</script>More</div>';
      const result = extractText(html);
      expect(result).toContain('Text');
      expect(result).toContain('More');
      expect(result).not.toContain('alert');
    });

    it('removes style tags', () => {
      const html = '<div>Text<style>.class { color: red; }</style>More</div>';
      const result = extractText(html);
      expect(result).not.toContain('color');
    });

    it('decodes HTML entities', () => {
      const html = '<p>Hello &amp; World &lt;test&gt;</p>';
      const result = extractText(html);
      expect(result).toContain('&');
      expect(result).toContain('<test>');
    });

    it('collapses whitespace', () => {
      const html = '<p>Hello     World\n\n\nTest</p>';
      const result = extractText(html);
      expect(result).toBe('Hello World Test');
    });

    it('handles empty input', () => {
      expect(extractText('')).toBe('');
    });
  });

  describe('extractAttribute', () => {
    it('extracts href from anchor', () => {
      const html = '<a href="https://example.com">Link</a>';
      expect(extractAttribute(html, 'a', 'href')).toBe('https://example.com');
    });

    it('extracts src from img', () => {
      const html = '<img src="/image.png" alt="test">';
      expect(extractAttribute(html, 'img', 'src')).toBe('/image.png');
    });

    it('handles double quotes', () => {
      const html = '<div data-value="test value">Content</div>';
      expect(extractAttribute(html, 'div', 'data-value')).toBe('test value');
    });

    it('handles single quotes', () => {
      const html = "<div data-value='test value'>Content</div>";
      expect(extractAttribute(html, 'div', 'data-value')).toBe('test value');
    });

    it('returns empty string when not found', () => {
      const html = '<div>Content</div>';
      expect(extractAttribute(html, 'div', 'data-missing')).toBe('');
    });

    it('finds first occurrence', () => {
      const html = '<a href="first.html">A</a><a href="second.html">B</a>';
      expect(extractAttribute(html, 'a', 'href')).toBe('first.html');
    });
  });

  describe('findElements', () => {
    it('finds elements by tag name', () => {
      const html = '<div><p>First</p><p>Second</p></div>';
      const elements = findElements(html, 'p');
      expect(elements.length).toBe(2);
      expect(elements[0]).toContain('First');
      expect(elements[1]).toContain('Second');
    });

    it('finds elements by class', () => {
      const html = '<div class="item">A</div><div class="item">B</div><div class="other">C</div>';
      const elements = findElements(html, '.item');
      expect(elements.length).toBe(2);
    });

    it('finds elements by id', () => {
      const html = '<div id="unique">Content</div><div id="other">Other</div>';
      const elements = findElements(html, '#unique');
      expect(elements.length).toBe(1);
      expect(elements[0]).toContain('Content');
    });

    it('finds elements by attribute', () => {
      const html =
        '<div data-type="special">A</div><div data-type="normal">B</div><div data-type="special">C</div>';
      const elements = findElements(html, "div[data-type='special']");
      expect(elements.length).toBe(2);
    });

    it('handles nested elements', () => {
      const html = '<div class="outer"><div class="inner">Nested</div></div>';
      const elements = findElements(html, '.outer');
      expect(elements.length).toBe(1);
      expect(elements[0]).toContain('inner');
    });

    it('returns empty array when no matches', () => {
      const html = '<div>Content</div>';
      const elements = findElements(html, '.nonexistent');
      expect(elements.length).toBe(0);
    });
  });

  describe('decodeHtmlEntities', () => {
    it('decodes named entities', () => {
      expect(decodeHtmlEntities('&amp;')).toBe('&');
      expect(decodeHtmlEntities('&lt;')).toBe('<');
      expect(decodeHtmlEntities('&gt;')).toBe('>');
      expect(decodeHtmlEntities('&quot;')).toBe('"');
      expect(decodeHtmlEntities('&#39;')).toBe("'");
      expect(decodeHtmlEntities('&nbsp;')).toBe(' ');
    });

    it('decodes typography entities', () => {
      expect(decodeHtmlEntities('&mdash;')).toBe('—');
      expect(decodeHtmlEntities('&ndash;')).toBe('–');
      expect(decodeHtmlEntities('&hellip;')).toBe('…');
    });

    it('decodes symbol entities', () => {
      expect(decodeHtmlEntities('&copy;')).toBe('©');
      expect(decodeHtmlEntities('&reg;')).toBe('®');
      expect(decodeHtmlEntities('&trade;')).toBe('™');
    });

    it('decodes decimal numeric entities', () => {
      expect(decodeHtmlEntities('&#65;')).toBe('A');
      expect(decodeHtmlEntities('&#97;')).toBe('a');
      expect(decodeHtmlEntities('&#169;')).toBe('©');
    });

    it('decodes hexadecimal numeric entities', () => {
      expect(decodeHtmlEntities('&#x41;')).toBe('A');
      expect(decodeHtmlEntities('&#x61;')).toBe('a');
      expect(decodeHtmlEntities('&#xA9;')).toBe('©');
    });

    it('handles multiple entities', () => {
      expect(decodeHtmlEntities('&lt;div&gt;')).toBe('<div>');
      expect(decodeHtmlEntities('Tom &amp; Jerry')).toBe('Tom & Jerry');
    });

    it('preserves unrecognized entities', () => {
      expect(decodeHtmlEntities('&unknown;')).toBe('&unknown;');
    });

    it('handles text without entities', () => {
      expect(decodeHtmlEntities('Hello World')).toBe('Hello World');
    });
  });

  describe('extractLinks', () => {
    it('extracts href from anchors', () => {
      const html = '<a href="https://example.com">Link</a>';
      const links = extractLinks(html);
      expect(links).toContain('https://example.com');
    });

    it('extracts multiple links', () => {
      const html = `
        <a href="https://a.com">A</a>
        <a href="https://b.com">B</a>
        <a href="https://c.com">C</a>
      `;
      const links = extractLinks(html);
      expect(links.length).toBe(3);
      expect(links).toContain('https://a.com');
      expect(links).toContain('https://b.com');
      expect(links).toContain('https://c.com');
    });

    it('handles single quoted hrefs', () => {
      const html = "<a href='https://example.com'>Link</a>";
      const links = extractLinks(html);
      expect(links).toContain('https://example.com');
    });

    it('returns empty array when no links', () => {
      const html = '<div>No links here</div>';
      const links = extractLinks(html);
      expect(links.length).toBe(0);
    });
  });
});
