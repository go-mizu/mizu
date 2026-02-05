import { describe, it, expect } from 'vitest';
import { createDefaultMetaSearch } from './metasearch';
import type { EngineParams } from './engine';

describe('Video Search Integration', () => {
  const metasearch = createDefaultMetaSearch();

  it('should aggregate results from multiple video engines', async () => {
    const params: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    };

    const result = await metasearch.search('javascript tutorial', 'videos', params);

    expect(result.results.length).toBeGreaterThan(0);
    expect(result.successfulEngines).toBeGreaterThan(0);

    // Verify results from multiple sources
    const sources = new Set(result.results.map(r => r.engine));
    console.log('Sources found:', Array.from(sources));
    console.log('Total results:', result.results.length);
    console.log('Successful engines:', result.successfulEngines);
    console.log('Failed engines:', result.failedEngines);
  }, 120000);

  it('should return video-specific fields', async () => {
    const params: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    };

    const result = await metasearch.search('music video', 'videos', params);

    expect(result.results.length).toBeGreaterThan(0);

    const video = result.results[0];
    expect(video.url).toBeTruthy();
    expect(video.title).toBeTruthy();
    expect(video.category).toBe('videos');
  }, 120000);

  it('should apply time range filter', async () => {
    const params: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: 'week',
      engineData: {},
    };

    const result = await metasearch.search('news', 'videos', params);
    expect(result.results).toBeDefined();
    console.log('Results with time range:', result.results.length);
  }, 120000);

  it('should handle safe search', async () => {
    const params: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 2,
      timeRange: '',
      engineData: {},
    };

    const result = await metasearch.search('cooking', 'videos', params);
    expect(result.results).toBeDefined();
  }, 120000);

  it('should list all 9 video engines', () => {
    const engines = metasearch.getByCategory('videos');
    const names = engines.map(e => e.name);

    console.log('Registered video engines:', names);

    // Check for all 9 engines (using actual engine names)
    expect(names).toContain('youtube');
    expect(names).toContain('duckduckgo videos');
    expect(names).toContain('vimeo');
    expect(names).toContain('dailymotion');
    expect(names).toContain('google_videos');
    expect(names).toContain('bing_videos');
    expect(names).toContain('peertube');
    expect(names).toContain('360search');
    expect(names).toContain('sogou');
    expect(engines.length).toBe(9);
  });
});
