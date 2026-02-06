import { describe, it, expect } from 'vitest';
import {
  GoogleNewsRSSEngine,
  parseGoogleNewsRss,
  buildSearchFeedUrl,
  buildCategoryFeedUrl,
} from './google-news-rss';
import type { EngineParams } from './engine';

describe('GoogleNewsRSSEngine', () => {
  const engine = new GoogleNewsRSSEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '' as const,
    engineData: {},
  };

  // -- Metadata --

  it('should have correct metadata', () => {
    expect(engine.name).toMatch(/^google news rss/);
    expect(engine.categories).toContain('news');
    expect(engine.supportsPaging).toBe(false);
    expect(engine.maxPage).toBe(1);
  });

  // -- buildSearchFeedUrl --

  describe('buildSearchFeedUrl', () => {
    it('should generate correct URL with query, language, and region', () => {
      const url = buildSearchFeedUrl('artificial intelligence', 'en', 'US');
      expect(url).toBe(
        'https://news.google.com/rss/search?q=artificial%20intelligence&hl=en&gl=US&ceid=US:en'
      );
    });

    it('should encode special characters in query', () => {
      const url = buildSearchFeedUrl('C++ & Rust', 'en', 'US');
      expect(url).toContain('q=C%2B%2B%20%26%20Rust');
    });

    it('should use provided language and region', () => {
      const url = buildSearchFeedUrl('news', 'de', 'DE');
      expect(url).toContain('hl=de');
      expect(url).toContain('gl=DE');
      expect(url).toContain('ceid=DE:de');
    });
  });

  // -- buildCategoryFeedUrl --

  describe('buildCategoryFeedUrl', () => {
    it('should generate headlines URL for top category', () => {
      const url = buildCategoryFeedUrl('top', 'en', 'US');
      expect(url).toBe(
        'https://news.google.com/rss?hl=en&gl=US&ceid=US:en'
      );
      // Top category should NOT include /topics/ path
      expect(url).not.toContain('/topics/');
    });

    it('should generate topic URL for technology category', () => {
      const url = buildCategoryFeedUrl('technology', 'en', 'US');
      expect(url).toBe(
        'https://news.google.com/rss/topics/TECHNOLOGY?hl=en&gl=US&ceid=US:en'
      );
    });

    it('should generate topic URL for business category', () => {
      const url = buildCategoryFeedUrl('business', 'en', 'US');
      expect(url).toContain('/topics/BUSINESS');
    });

    it('should generate topic URL for world category', () => {
      const url = buildCategoryFeedUrl('world', 'en', 'US');
      expect(url).toContain('/topics/WORLD');
    });

    it('should respect language and region parameters', () => {
      const url = buildCategoryFeedUrl('science', 'fr', 'FR');
      expect(url).toContain('hl=fr');
      expect(url).toContain('gl=FR');
      expect(url).toContain('ceid=FR:fr');
    });
  });

  // -- parseGoogleNewsRss --

  describe('parseGoogleNewsRss', () => {
    const rssXml = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
  <title>Google News</title>
  <item>
    <title>AI Makes Breakthrough - TechCrunch</title>
    <link>https://news.google.com/rss/articles/abc123</link>
    <pubDate>Thu, 06 Feb 2026 10:00:00 GMT</pubDate>
    <description>&lt;a href="https://example.com"&gt;Full article&lt;/a&gt; Some description text</description>
    <source url="https://techcrunch.com">TechCrunch</source>
    <guid>tag:news.google.com,2005:cluster=abc123</guid>
  </item>
  <item>
    <title>Second Article - CNN</title>
    <link>https://news.google.com/rss/articles/def456</link>
    <pubDate>Wed, 05 Feb 2026 15:30:00 GMT</pubDate>
    <description>Another article description</description>
    <source url="https://cnn.com">CNN</source>
    <media:content url="https://example.com/image.jpg" medium="image"/>
    <guid>tag:news.google.com,2005:cluster=def456</guid>
  </item>
</channel>
</rss>`;

    it('should parse RSS items correctly', () => {
      const items = parseGoogleNewsRss(rssXml);
      expect(items).toHaveLength(2);
    });

    it('should extract title, removing " - Source" suffix', () => {
      const items = parseGoogleNewsRss(rssXml);
      expect(items[0].title).toBe('AI Makes Breakthrough');
      expect(items[1].title).toBe('Second Article');
    });

    it('should extract source from title suffix', () => {
      const items = parseGoogleNewsRss(rssXml);
      // Source is extracted from the " - Source" suffix in the title
      expect(items[0].source).toBe('TechCrunch');
      expect(items[1].source).toBe('CNN');
    });

    it('should extract source URL from <source> element', () => {
      const items = parseGoogleNewsRss(rssXml);
      expect(items[0].sourceUrl).toBe('https://techcrunch.com');
      expect(items[1].sourceUrl).toBe('https://cnn.com');
    });

    it('should extract published date from pubDate', () => {
      const items = parseGoogleNewsRss(rssXml);
      expect(items[0].publishedAt).toBe(
        new Date('Thu, 06 Feb 2026 10:00:00 GMT').toISOString()
      );
      expect(items[1].publishedAt).toBe(
        new Date('Wed, 05 Feb 2026 15:30:00 GMT').toISOString()
      );
    });

    it('should extract images from media:content', () => {
      const items = parseGoogleNewsRss(rssXml);
      // First item has no media:content
      expect(items[0].imageUrl).toBeUndefined();
      // Second item has media:content
      expect(items[1].imageUrl).toBe('https://example.com/image.jpg');
    });

    it('should extract link URL', () => {
      const items = parseGoogleNewsRss(rssXml);
      expect(items[0].url).toBe('https://news.google.com/rss/articles/abc123');
      expect(items[1].url).toBe('https://news.google.com/rss/articles/def456');
    });

    it('should extract snippet from description with HTML stripped', () => {
      const items = parseGoogleNewsRss(rssXml);
      // First item description has HTML entities that get decoded then tags stripped
      expect(items[0].snippet).toContain('Some description text');
      // Second item has plain text description
      expect(items[1].snippet).toBe('Another article description');
    });

    it('should handle CDATA sections', () => {
      const cdataRss = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
  <item>
    <title><![CDATA[Breaking News Update - Reuters]]></title>
    <link>https://news.google.com/rss/articles/xyz789</link>
    <pubDate>Thu, 06 Feb 2026 12:00:00 GMT</pubDate>
    <description>CDATA test</description>
    <source url="https://reuters.com">Reuters</source>
  </item>
</channel>
</rss>`;

      const items = parseGoogleNewsRss(cdataRss);
      expect(items).toHaveLength(1);
      // The XML parser strips CDATA markers via tag-stripping regex;
      // verify the item is still parsed (link is present so item is not null)
      expect(items[0].url).toBe('https://news.google.com/rss/articles/xyz789');
    });

    it('should return empty array for feed with no items', () => {
      const emptyRss = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
  <title>Google News</title>
</channel>
</rss>`;
      const items = parseGoogleNewsRss(emptyRss);
      expect(items).toHaveLength(0);
    });

    it('should skip items without a link', () => {
      const noLinkRss = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
  <item>
    <title>No Link Article - Source</title>
    <description>Missing link element</description>
  </item>
</channel>
</rss>`;
      const items = parseGoogleNewsRss(noLinkRss);
      expect(items).toHaveLength(0);
    });
  });

  // -- parseResponse --

  describe('parseResponse', () => {
    const rssXml = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
  <title>Google News</title>
  <item>
    <title>Tech Giant Announces New Product - The Verge</title>
    <link>https://news.google.com/rss/articles/aaa111</link>
    <pubDate>Thu, 06 Feb 2026 08:00:00 GMT</pubDate>
    <description>A major tech company has unveiled its latest product.</description>
    <source url="https://theverge.com">The Verge</source>
    <media:content url="https://example.com/product.jpg" medium="image"/>
    <guid>tag:news.google.com,2005:cluster=aaa111</guid>
  </item>
  <item>
    <title>Markets Rally on Good Earnings - Bloomberg</title>
    <link>https://news.google.com/rss/articles/bbb222</link>
    <pubDate>Wed, 05 Feb 2026 20:00:00 GMT</pubDate>
    <description>Stock markets surged after positive earnings reports.</description>
    <source url="https://bloomberg.com">Bloomberg</source>
    <guid>tag:news.google.com,2005:cluster=bbb222</guid>
  </item>
</channel>
</rss>`;

    it('should return EngineResults with proper fields', () => {
      const results = engine.parseResponse(rssXml, defaultParams);

      expect(results.results).toHaveLength(2);
      expect(results.suggestions).toEqual([]);
      expect(results.corrections).toEqual([]);
    });

    it('should set category to news and template to news on each result', () => {
      const results = engine.parseResponse(rssXml, defaultParams);

      for (const result of results.results) {
        expect(result.category).toBe('news');
        expect(result.template).toBe('news');
      }
    });

    it('should populate result fields correctly', () => {
      const results = engine.parseResponse(rssXml, defaultParams);
      const first = results.results[0];

      expect(first.url).toBe('https://news.google.com/rss/articles/aaa111');
      expect(first.title).toBe('Tech Giant Announces New Product');
      expect(first.content).toContain('latest product');
      expect(first.engine).toMatch(/^google news rss/);
      expect(first.score).toBeGreaterThan(0);
      expect(first.source).toBe('The Verge');
      expect(first.thumbnailUrl).toBe('https://example.com/product.jpg');
      expect(first.publishedAt).toBeTruthy();
    });

    it('should include metadata with sourceUrl and newsCategory', () => {
      const results = engine.parseResponse(rssXml, defaultParams);
      const first = results.results[0];

      expect(first.metadata).toBeDefined();
      expect(first.metadata!.sourceUrl).toBe('https://theverge.com');
      expect(first.metadata!.newsCategory).toBe('top');
      expect(first.metadata!.articleId).toBeTruthy();
    });

    it('should handle empty feed gracefully', () => {
      const emptyRss = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
  <title>Google News</title>
</channel>
</rss>`;

      const results = engine.parseResponse(emptyRss, defaultParams);
      expect(results.results).toHaveLength(0);
      expect(results.suggestions).toEqual([]);
      expect(results.corrections).toEqual([]);
    });
  });

  // -- buildRequest --

  describe('buildRequest', () => {
    it('should build a search URL when query is provided', () => {
      const config = engine.buildRequest('climate change', defaultParams);

      expect(config.url).toContain('/rss/search');
      expect(config.url).toContain('q=climate%20change');
      expect(config.url).toContain('hl=en');
      expect(config.url).toContain('gl=US');
      expect(config.method).toBe('GET');
    });

    it('should build a category URL when query is empty', () => {
      const config = engine.buildRequest('', defaultParams);

      // Default category is 'top', which uses headlines URL
      expect(config.url).toContain('news.google.com/rss');
      expect(config.url).not.toContain('/search');
      expect(config.method).toBe('GET');
    });

    it('should extract language and region from locale', () => {
      const config = engine.buildRequest('test', {
        ...defaultParams,
        locale: 'fr-FR',
      });

      expect(config.url).toContain('hl=fr');
      expect(config.url).toContain('gl=FR');
      expect(config.url).toContain('ceid=FR:fr');
    });

    it('should include appropriate headers', () => {
      const config = engine.buildRequest('test', defaultParams);

      expect(config.headers['User-Agent']).toBeTruthy();
      expect(config.headers['Accept']).toContain('rss+xml');
    });

    it('should have empty cookies array', () => {
      const config = engine.buildRequest('test', defaultParams);
      expect(config.cookies).toEqual([]);
    });
  });

  // -- Category-specific engines --

  describe('category-specific engines', () => {
    it('should use category-specific name', () => {
      const techEngine = new GoogleNewsRSSEngine('technology');
      expect(techEngine.name).toBe('google news rss (technology)');
    });

    it('should build category feed URL when query is empty', () => {
      const techEngine = new GoogleNewsRSSEngine('technology');
      const config = techEngine.buildRequest('', defaultParams);
      expect(config.url).toContain('/topics/TECHNOLOGY');
    });

    it('should use search URL even for category engine when query is provided', () => {
      const techEngine = new GoogleNewsRSSEngine('technology');
      const config = techEngine.buildRequest('AI news', defaultParams);
      expect(config.url).toContain('/rss/search');
      expect(config.url).toContain('q=AI%20news');
    });
  });
});
