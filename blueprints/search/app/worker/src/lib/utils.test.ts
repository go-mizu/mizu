import { describe, it, expect } from 'vitest';
import { generateId, sanitizeHtml, extractDomain, truncate } from './utils';

describe('utils', () => {
  describe('generateId', () => {
    it('generates a 16-character hex string', () => {
      const id = generateId();
      expect(id.length).toBe(16);
      expect(id).toMatch(/^[0-9a-f]+$/);
    });

    it('generates unique ids', () => {
      const ids = new Set<string>();
      for (let i = 0; i < 100; i++) {
        ids.add(generateId());
      }
      expect(ids.size).toBe(100);
    });
  });

  describe('sanitizeHtml', () => {
    it('removes script tags', () => {
      const html = '<div>Hello<script>alert("bad")</script>World</div>';
      const result = sanitizeHtml(html);
      expect(result).not.toContain('script');
      expect(result).not.toContain('alert');
    });

    it('removes style tags', () => {
      const html = '<div>Hello<style>.bad { color: red; }</style>World</div>';
      const result = sanitizeHtml(html);
      expect(result).not.toContain('style');
      expect(result).not.toContain('color');
    });

    it('removes iframe tags', () => {
      const html = '<div>Hello<iframe src="bad.html"></iframe>World</div>';
      const result = sanitizeHtml(html);
      expect(result).not.toContain('iframe');
    });

    it('removes object tags', () => {
      const html = '<div>Hello<object data="bad.swf"></object>World</div>';
      const result = sanitizeHtml(html);
      expect(result).not.toContain('object');
    });

    it('removes embed tags', () => {
      const html = '<div>Hello<embed src="bad.swf">World</div>';
      const result = sanitizeHtml(html);
      expect(result).not.toContain('embed');
    });

    it('removes form tags', () => {
      const html = '<div>Hello<form action="bad.php"><input></form>World</div>';
      const result = sanitizeHtml(html);
      expect(result).not.toContain('form');
    });

    it('preserves safe HTML', () => {
      const html = '<div><p>Hello <strong>World</strong></p></div>';
      const result = sanitizeHtml(html);
      expect(result).toContain('<p>');
      expect(result).toContain('<strong>');
      expect(result).toContain('Hello');
      expect(result).toContain('World');
    });

    it('handles empty input', () => {
      expect(sanitizeHtml('')).toBe('');
    });
  });

  describe('extractDomain', () => {
    it('extracts domain from https URL', () => {
      const domain = extractDomain('https://www.example.com/path/to/page');
      expect(domain).toBe('www.example.com');
    });

    it('extracts domain from http URL', () => {
      const domain = extractDomain('http://example.org/path');
      expect(domain).toBe('example.org');
    });

    it('extracts domain with port', () => {
      const domain = extractDomain('https://localhost:3000/api');
      expect(domain).toBe('localhost');
    });

    it('handles subdomain', () => {
      const domain = extractDomain('https://api.example.com/v1');
      expect(domain).toBe('api.example.com');
    });

    it('returns empty string for invalid URL', () => {
      const domain = extractDomain('not a url');
      expect(domain).toBe('');
    });

    it('returns empty string for empty input', () => {
      const domain = extractDomain('');
      expect(domain).toBe('');
    });
  });

  describe('truncate', () => {
    it('truncates long text', () => {
      const text = 'This is a very long string that needs to be truncated';
      const result = truncate(text, 20);
      expect(result.length).toBeLessThanOrEqual(23); // 20 + '...'
      expect(result.endsWith('...')).toBe(true);
    });

    it('does not truncate short text', () => {
      const text = 'Short';
      const result = truncate(text, 20);
      expect(result).toBe('Short');
    });

    it('handles exact length', () => {
      const text = '12345';
      const result = truncate(text, 5);
      expect(result).toBe('12345');
    });

    it('handles empty string', () => {
      expect(truncate('', 10)).toBe('');
    });

    it('adds ellipsis when truncating', () => {
      const text = 'Hello World';
      const result = truncate(text, 5);
      expect(result).toBe('Hello...');
    });
  });
});
