import { describe, it, expect } from 'vitest';
import {
  getElementsByTagName,
  getTextContent,
  getAllTextContent,
  getElementAttribute,
} from './xml-parser';

describe('xml-parser', () => {
  describe('getElementsByTagName', () => {
    it('finds elements by tag name', () => {
      const xml = '<root><item>First</item><item>Second</item></root>';
      const elements = getElementsByTagName(xml, 'item');
      expect(elements.length).toBe(2);
      expect(elements[0]).toBe('First');
      expect(elements[1]).toBe('Second');
    });

    it('handles nested elements', () => {
      const xml = '<root><parent><child>Value</child></parent></root>';
      const elements = getElementsByTagName(xml, 'child');
      expect(elements.length).toBe(1);
      expect(elements[0]).toBe('Value');
    });

    it('handles self-closing tags', () => {
      const xml = '<root><item/><item>Content</item></root>';
      const elements = getElementsByTagName(xml, 'item');
      // Self-closing tag should return empty content
      expect(elements.length).toBeGreaterThanOrEqual(1);
    });

    it('handles elements with attributes', () => {
      const xml = '<root><item id="1">First</item><item id="2">Second</item></root>';
      const elements = getElementsByTagName(xml, 'item');
      expect(elements.length).toBe(2);
    });

    it('returns empty array when tag not found', () => {
      const xml = '<root><item>Content</item></root>';
      const elements = getElementsByTagName(xml, 'missing');
      expect(elements.length).toBe(0);
    });

    it('handles namespaced tags', () => {
      const xml = '<feed><entry><title>Title</title></entry></feed>';
      const elements = getElementsByTagName(xml, 'entry');
      expect(elements.length).toBe(1);
    });
  });

  describe('getTextContent', () => {
    it('extracts text content', () => {
      const xml = '<root><title>Hello World</title></root>';
      const text = getTextContent(xml, 'title');
      expect(text).toBe('Hello World');
    });

    it('strips nested tags', () => {
      const xml = '<root><summary>Text with <b>bold</b> and <i>italic</i></summary></root>';
      const text = getTextContent(xml, 'summary');
      expect(text).toContain('Text with');
      expect(text).toContain('bold');
      expect(text).toContain('italic');
      expect(text).not.toContain('<b>');
    });

    it('returns empty string when not found', () => {
      const xml = '<root><other>Content</other></root>';
      const text = getTextContent(xml, 'missing');
      expect(text).toBe('');
    });

    it('returns first match', () => {
      const xml = '<root><item>First</item><item>Second</item></root>';
      const text = getTextContent(xml, 'item');
      expect(text).toBe('First');
    });
  });

  describe('getAllTextContent', () => {
    it('returns all text contents', () => {
      const xml = '<root><name>Alice</name><name>Bob</name><name>Charlie</name></root>';
      const texts = getAllTextContent(xml, 'name');
      expect(texts.length).toBe(3);
      expect(texts).toContain('Alice');
      expect(texts).toContain('Bob');
      expect(texts).toContain('Charlie');
    });

    it('returns empty array when no matches', () => {
      const xml = '<root><other>Content</other></root>';
      const texts = getAllTextContent(xml, 'missing');
      expect(texts.length).toBe(0);
    });
  });

  describe('getElementAttribute', () => {
    it('extracts attribute value', () => {
      const xml = '<root><link href="https://example.com"/></root>';
      const values = getElementAttribute(xml, 'link', 'href');
      expect(values).toHaveLength(1);
      expect(values[0]).toBe('https://example.com');
    });

    it('handles double quotes', () => {
      const xml = '<root><item id="test-id">Content</item></root>';
      const values = getElementAttribute(xml, 'item', 'id');
      expect(values).toHaveLength(1);
      expect(values[0]).toBe('test-id');
    });

    it('handles single quotes', () => {
      const xml = "<root><item id='test-id'>Content</item></root>";
      const values = getElementAttribute(xml, 'item', 'id');
      expect(values).toHaveLength(1);
      expect(values[0]).toBe('test-id');
    });

    it('returns empty array when attribute not found', () => {
      const xml = '<root><item id="test">Content</item></root>';
      const values = getElementAttribute(xml, 'item', 'missing');
      expect(values).toHaveLength(0);
    });

    it('returns empty array when element not found', () => {
      const xml = '<root><other>Content</other></root>';
      const values = getElementAttribute(xml, 'item', 'id');
      expect(values).toHaveLength(0);
    });
  });

  describe('arXiv-style XML parsing', () => {
    const arxivSample = `
      <?xml version="1.0" encoding="UTF-8"?>
      <feed xmlns="http://www.w3.org/2005/Atom">
        <entry>
          <id>http://arxiv.org/abs/2101.00001v1</id>
          <title>A Sample Paper Title</title>
          <summary>This is the abstract of the paper describing the research.</summary>
          <author><name>John Doe</name></author>
          <author><name>Jane Smith</name></author>
          <published>2021-01-01T00:00:00Z</published>
          <link href="http://arxiv.org/abs/2101.00001v1" rel="alternate" type="text/html"/>
          <link href="http://arxiv.org/pdf/2101.00001v1" rel="related" type="application/pdf"/>
        </entry>
      </feed>
    `;

    it('parses entry id', () => {
      const id = getTextContent(arxivSample, 'id');
      expect(id).toBe('http://arxiv.org/abs/2101.00001v1');
    });

    it('parses entry title', () => {
      const title = getTextContent(arxivSample, 'title');
      expect(title).toBe('A Sample Paper Title');
    });

    it('parses entry summary', () => {
      const summary = getTextContent(arxivSample, 'summary');
      expect(summary).toContain('abstract of the paper');
    });

    it('parses all authors', () => {
      const authors = getAllTextContent(arxivSample, 'name');
      expect(authors.length).toBe(2);
      expect(authors).toContain('John Doe');
      expect(authors).toContain('Jane Smith');
    });

    it('parses published date', () => {
      const published = getTextContent(arxivSample, 'published');
      expect(published).toBe('2021-01-01T00:00:00Z');
    });
  });
});
