import { describe, it, expect } from 'vitest';
import type { EngineResult } from '../engines/engine';

/**
 * The toNewsResult function is module-private in search.ts.
 * We replicate the exact logic here so we can unit-test the conversion
 * independently without needing to stand up the full SearchService.
 */

function extractDomain(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    return '';
  }
}

function toNewsResult(r: EngineResult, index: number) {
  const domain = extractDomain(r.url);
  return {
    id: `${Date.now().toString(36)}-${index}`,
    url: r.url,
    title: r.title,
    snippet: r.content,
    source: r.source || domain,
    source_domain: domain,
    author: r.author || (r.metadata?.authors as string[])?.join(', ') || undefined,
    image_url: r.thumbnailUrl || undefined,
    thumbnail_url: r.thumbnailUrl || undefined,
    published_at: r.publishedAt || new Date().toISOString(),
    engine: r.engine,
    engines: [r.engine],
    metadata: r.metadata,
  };
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeEngineResult(overrides: Partial<EngineResult> = {}): EngineResult {
  return {
    url: 'https://example.com/article',
    title: 'Test Article',
    content: 'Some article content',
    engine: 'google_news',
    score: 1.0,
    category: 'news',
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('toNewsResult conversion', () => {
  it('maps all fields correctly when every field is present', () => {
    const input = makeEngineResult({
      url: 'https://www.reuters.com/world/breaking-news',
      title: 'Breaking News Title',
      content: 'Full article summary goes here.',
      source: 'Reuters',
      publishedAt: '2026-01-15T10:30:00Z',
      thumbnailUrl: 'https://cdn.reuters.com/thumb.jpg',
      author: 'Jane Doe',
      engine: 'bing_news',
      metadata: { language: 'en', section: 'world' },
    });

    const result = toNewsResult(input, 0);

    expect(result).toEqual({
      id: expect.any(String),
      url: 'https://www.reuters.com/world/breaking-news',
      title: 'Breaking News Title',
      snippet: 'Full article summary goes here.',
      source: 'Reuters',
      source_domain: 'www.reuters.com',
      author: 'Jane Doe',
      image_url: 'https://cdn.reuters.com/thumb.jpg',
      thumbnail_url: 'https://cdn.reuters.com/thumb.jpg',
      published_at: '2026-01-15T10:30:00Z',
      engine: 'bing_news',
      engines: ['bing_news'],
      metadata: { language: 'en', section: 'world' },
    });
  });

  it('falls back to domain when source is missing', () => {
    const input = makeEngineResult({
      url: 'https://www.bbc.co.uk/news/tech-123',
      source: undefined,
    });

    const result = toNewsResult(input, 1);

    expect(result.source).toBe('www.bbc.co.uk');
    expect(result.source_domain).toBe('www.bbc.co.uk');
  });

  it('sets author to undefined when both author and metadata.authors are missing', () => {
    const input = makeEngineResult({
      author: undefined,
      metadata: undefined,
    });

    const result = toNewsResult(input, 2);

    expect(result.author).toBeUndefined();
  });

  it('sets image_url and thumbnail_url to undefined when thumbnailUrl is missing', () => {
    const input = makeEngineResult({
      thumbnailUrl: undefined,
    });

    const result = toNewsResult(input, 3);

    expect(result.image_url).toBeUndefined();
    expect(result.thumbnail_url).toBeUndefined();
  });

  it('derives author from metadata.authors when author field is absent', () => {
    const input = makeEngineResult({
      author: undefined,
      metadata: { authors: ['John', 'Jane'] },
    });

    const result = toNewsResult(input, 4);

    expect(result.author).toBe('John, Jane');
  });

  it('prefers direct author field over metadata.authors', () => {
    const input = makeEngineResult({
      author: 'Direct Author',
      metadata: { authors: ['Meta Author 1', 'Meta Author 2'] },
    });

    const result = toNewsResult(input, 5);

    expect(result.author).toBe('Direct Author');
  });

  it('handles single metadata author correctly', () => {
    const input = makeEngineResult({
      author: undefined,
      metadata: { authors: ['Solo Writer'] },
    });

    const result = toNewsResult(input, 6);

    expect(result.author).toBe('Solo Writer');
  });
});

describe('extractDomain / source_domain', () => {
  const cases: Array<{ url: string; expected: string }> = [
    { url: 'https://www.nytimes.com/2026/01/15/article.html', expected: 'www.nytimes.com' },
    { url: 'https://techcrunch.com/some-path', expected: 'techcrunch.com' },
    { url: 'http://news.bbc.co.uk/world/', expected: 'news.bbc.co.uk' },
    { url: 'https://sub.domain.example.org/page?q=1', expected: 'sub.domain.example.org' },
    { url: 'https://localhost:3000/test', expected: 'localhost' },
  ];

  for (const { url, expected } of cases) {
    it(`extracts "${expected}" from "${url}"`, () => {
      const input = makeEngineResult({ url });
      const result = toNewsResult(input, 0);
      expect(result.source_domain).toBe(expected);
    });
  }

  it('returns empty string for invalid URLs', () => {
    const input = makeEngineResult({ url: 'not-a-valid-url' });
    const result = toNewsResult(input, 0);
    expect(result.source_domain).toBe('');
  });
});

describe('published_at fallback', () => {
  it('uses publishedAt directly when provided', () => {
    const input = makeEngineResult({
      publishedAt: '2026-02-01T08:00:00Z',
    });

    const result = toNewsResult(input, 0);

    expect(result.published_at).toBe('2026-02-01T08:00:00Z');
  });

  it('falls back to a valid ISO date string when publishedAt is undefined', () => {
    const input = makeEngineResult({
      publishedAt: undefined,
    });

    const before = new Date().toISOString();
    const result = toNewsResult(input, 0);
    const after = new Date().toISOString();

    // The fallback should be a valid ISO date string
    expect(() => new Date(result.published_at)).not.toThrow();
    const ts = new Date(result.published_at).getTime();
    expect(ts).toBeGreaterThanOrEqual(new Date(before).getTime());
    expect(ts).toBeLessThanOrEqual(new Date(after).getTime());
  });
});

describe('source field mapping', () => {
  it('maps engine source directly to source field', () => {
    const input = makeEngineResult({
      source: 'Associated Press',
      url: 'https://apnews.com/article/123',
    });

    const result = toNewsResult(input, 0);

    expect(result.source).toBe('Associated Press');
    expect(result.source_domain).toBe('apnews.com');
  });

  it('falls back to domain when source is empty string', () => {
    const input = makeEngineResult({
      source: '',
      url: 'https://www.theguardian.com/news/story',
    });

    const result = toNewsResult(input, 0);

    expect(result.source).toBe('www.theguardian.com');
  });
});

describe('output shape matches NewsResult type', () => {
  it('has all required fields with correct types', () => {
    const input = makeEngineResult({
      url: 'https://example.com/news',
      title: 'Title',
      content: 'Content',
      engine: 'test_engine',
    });

    const result = toNewsResult(input, 0);

    // Required string fields
    expect(typeof result.id).toBe('string');
    expect(result.id.length).toBeGreaterThan(0);
    expect(typeof result.url).toBe('string');
    expect(typeof result.title).toBe('string');
    expect(typeof result.snippet).toBe('string');
    expect(typeof result.source).toBe('string');
    expect(typeof result.source_domain).toBe('string');
    expect(typeof result.published_at).toBe('string');
    expect(typeof result.engine).toBe('string');

    // engines is a string array
    expect(Array.isArray(result.engines)).toBe(true);
    expect(result.engines.length).toBe(1);
    expect(typeof result.engines[0]).toBe('string');
  });

  it('snippet maps from content', () => {
    const input = makeEngineResult({
      content: 'This is the article body content.',
    });

    const result = toNewsResult(input, 0);

    expect(result.snippet).toBe('This is the article body content.');
  });

  it('engines array contains exactly the single engine name', () => {
    const input = makeEngineResult({ engine: 'duckduckgo_news' });

    const result = toNewsResult(input, 0);

    expect(result.engines).toEqual(['duckduckgo_news']);
  });

  it('id is unique across different indices', () => {
    const input = makeEngineResult();

    const result0 = toNewsResult(input, 0);
    const result1 = toNewsResult(input, 1);

    expect(result0.id).not.toBe(result1.id);
  });

  it('passes metadata through unchanged', () => {
    const meta = { category: 'tech', tags: ['ai', 'ml'], count: 42 };
    const input = makeEngineResult({ metadata: meta });

    const result = toNewsResult(input, 0);

    expect(result.metadata).toBe(meta);
  });

  it('metadata is undefined when not provided on engine result', () => {
    const input = makeEngineResult({ metadata: undefined });

    const result = toNewsResult(input, 0);

    expect(result.metadata).toBeUndefined();
  });
});
