import { describe, it, expect, beforeAll, beforeEach } from 'vitest';
import { BangService } from './bang';
import { KVStore } from '../store/kv';

// Create an in-memory KV namespace for testing
const createMockKV = (): KVNamespace => {
  const store = new Map<string, string>();
  return {
    get: async (key: string) => store.get(key) ?? null,
    put: async (key: string, value: string) => {
      store.set(key, value);
    },
    delete: async (key: string) => {
      store.delete(key);
    },
    list: async () => ({ keys: [], list_complete: true, cacheStatus: null }),
    getWithMetadata: async () => ({ value: null, metadata: null, cacheStatus: null }),
  } as unknown as KVNamespace;
};

describe('BangService', () => {
  let service: BangService;
  let kvStore: KVStore;
  let kv: KVNamespace;

  beforeEach(() => {
    kv = createMockKV();
    kvStore = new KVStore(kv);
    service = new BangService(kvStore);
  });

  describe('parse', () => {
    describe('no bang present', () => {
      it('returns original query when no bang', async () => {
        const result = await service.parse('hello world');
        expect(result.query).toBe('hello world');
        expect(result.redirect).toBeUndefined();
        expect(result.bang).toBeUndefined();
      });

      it('handles empty query', async () => {
        const result = await service.parse('');
        expect(result.query).toBe('');
      });

      it('handles whitespace only', async () => {
        const result = await service.parse('   ');
        expect(result.query).toBe('');
      });
    });

    describe('bang at start', () => {
      it('parses !g bang', async () => {
        const result = await service.parse('!g typescript');
        expect(result.redirect).toBe('https://www.google.com/search?q=typescript');
        expect(result.bang?.trigger).toBe('g');
        expect(result.bang?.name).toBe('Google');
        expect(result.query).toBe('typescript');
      });

      it('parses !yt bang', async () => {
        const result = await service.parse('!yt music video');
        expect(result.redirect).toBe('https://www.youtube.com/results?search_query=music%20video');
        expect(result.bang?.name).toBe('YouTube');
      });

      it('parses !w bang', async () => {
        const result = await service.parse('!w javascript');
        expect(result.redirect).toBe(
          'https://en.wikipedia.org/wiki/Special:Search?search=javascript'
        );
      });

      it('parses !gh bang', async () => {
        const result = await service.parse('!gh react');
        expect(result.redirect).toBe('https://github.com/search?q=react');
      });

      it('parses !r bang', async () => {
        const result = await service.parse('!r programming');
        expect(result.redirect).toBe('https://www.reddit.com/search/?q=programming');
      });

      it('parses !so bang', async () => {
        const result = await service.parse('!so python error');
        expect(result.redirect).toBe('https://stackoverflow.com/search?q=python%20error');
      });

      it('parses !npm bang', async () => {
        const result = await service.parse('!npm lodash');
        expect(result.redirect).toBe('https://www.npmjs.com/search?q=lodash');
      });

      it('parses !mdn bang', async () => {
        const result = await service.parse('!mdn array');
        expect(result.redirect).toBe('https://developer.mozilla.org/en-US/search?q=array');
      });

      it('parses !amz bang', async () => {
        const result = await service.parse('!amz laptop');
        expect(result.redirect).toBe('https://www.amazon.com/s?k=laptop');
      });

      it('parses !imdb bang', async () => {
        const result = await service.parse('!imdb inception');
        expect(result.redirect).toBe('https://www.imdb.com/find?q=inception');
      });

      it('parses !ddg bang', async () => {
        const result = await service.parse('!ddg privacy');
        expect(result.redirect).toBe('https://duckduckgo.com/?q=privacy');
      });

      it('parses !b (Bing) bang', async () => {
        const result = await service.parse('!b search term');
        expect(result.redirect).toBe('https://www.bing.com/search?q=search%20term');
      });
    });

    describe('bang at end', () => {
      it('parses bang at end of query', async () => {
        const result = await service.parse('typescript !g');
        expect(result.redirect).toBe('https://www.google.com/search?q=typescript');
        expect(result.bang?.trigger).toBe('g');
        expect(result.query).toBe('typescript');
      });

      it('parses multi-word query with bang at end', async () => {
        const result = await service.parse('how to code in python !yt');
        expect(result.redirect).toBe(
          'https://www.youtube.com/results?search_query=how%20to%20code%20in%20python'
        );
      });
    });

    describe('internal bangs', () => {
      it('handles !i (images) bang', async () => {
        const result = await service.parse('!i cats');
        expect(result.redirect).toBe('/images?q=cats');
        expect(result.category).toBe('internal');
        expect(result.query).toBe('cats');
      });

      it('handles !n (news) bang', async () => {
        const result = await service.parse('!n election');
        expect(result.redirect).toBe('/news?q=election');
        expect(result.category).toBe('internal');
      });

      it('handles !v (videos) bang', async () => {
        const result = await service.parse('!v tutorial');
        expect(result.redirect).toBe('/videos?q=tutorial');
        expect(result.category).toBe('internal');
      });
    });

    describe('URL encoding', () => {
      it('encodes special characters', async () => {
        const result = await service.parse('!g hello world');
        expect(result.redirect).toBe('https://www.google.com/search?q=hello%20world');
      });

      it('encodes ampersand', async () => {
        const result = await service.parse('!g cats & dogs');
        expect(result.redirect).toBe('https://www.google.com/search?q=cats%20%26%20dogs');
      });

      it('encodes plus sign', async () => {
        const result = await service.parse('!g c++ tutorial');
        expect(result.redirect).toContain('c%2B%2B');
      });
    });

    describe('unknown bangs', () => {
      it('returns original query for unknown bang', async () => {
        const result = await service.parse('!unknown test');
        expect(result.query).toBe('!unknown test');
        expect(result.redirect).toBeUndefined();
        expect(result.bang).toBeUndefined();
      });
    });

    describe('empty query with bang', () => {
      it('handles bang with no query', async () => {
        const result = await service.parse('!g');
        expect(result.redirect).toBe('https://www.google.com/search?q=');
        expect(result.query).toBe('');
      });
    });

    describe('custom bangs from KV', () => {
      it('looks up custom bangs from KV', async () => {
        // Create a custom bang in KV
        await kvStore.createBang({
          id: 1,
          trigger: 'custom',
          name: 'Custom Search',
          url_template: 'https://custom.example.com/?q={query}',
          category: 'custom',
          is_builtin: false,
          created_at: new Date().toISOString(),
        });

        const result = await service.parse('!custom test query');
        expect(result.redirect).toBe('https://custom.example.com/?q=test%20query');
        expect(result.bang?.name).toBe('Custom Search');
      });

      it('built-in bangs take precedence', async () => {
        // Even if there's a custom 'g' bang, built-in should work
        const result = await service.parse('!g test');
        expect(result.redirect).toBe('https://www.google.com/search?q=test');
      });
    });

    describe('case insensitivity', () => {
      it('handles uppercase bang', async () => {
        const result = await service.parse('!G test');
        expect(result.redirect).toBe('https://www.google.com/search?q=test');
      });

      it('handles mixed case bang', async () => {
        const result = await service.parse('!Yt video');
        expect(result.redirect).toContain('youtube.com');
      });
    });
  });

  describe('listBangs', () => {
    it('returns built-in bangs', async () => {
      const bangs = await service.listBangs();

      const triggers = bangs.map((b) => b.trigger);
      expect(triggers).toContain('g');
      expect(triggers).toContain('yt');
      expect(triggers).toContain('w');
      expect(triggers).toContain('gh');
      expect(triggers).toContain('r');
      expect(triggers).toContain('so');
      expect(triggers).toContain('npm');
      expect(triggers).toContain('amz');
      expect(triggers).toContain('imdb');
      expect(triggers).toContain('mdn');
      expect(triggers).toContain('ddg');
      expect(triggers).toContain('b');
      expect(triggers).toContain('i');
      expect(triggers).toContain('n');
      expect(triggers).toContain('v');
    });

    it('includes custom bangs', async () => {
      await kvStore.createBang({
        id: 100,
        trigger: 'mysite',
        name: 'My Site',
        url_template: 'https://mysite.com/?q={query}',
        category: 'custom',
        is_builtin: false,
        created_at: new Date().toISOString(),
      });

      const bangs = await service.listBangs();
      expect(bangs.some((b) => b.trigger === 'mysite')).toBe(true);
    });

    it('built-in bangs have is_builtin true', async () => {
      const bangs = await service.listBangs();
      const googleBang = bangs.find((b) => b.trigger === 'g');
      expect(googleBang?.is_builtin).toBe(true);
    });
  });

  describe('createBang', () => {
    it('creates a custom bang', async () => {
      const bang = {
        id: 100,
        trigger: 'test',
        name: 'Test',
        url_template: 'https://test.com/?q={query}',
        category: 'test',
        is_builtin: false,
        created_at: new Date().toISOString(),
      };

      await service.createBang(bang);

      // Verify it was created by parsing
      const result = await service.parse('!test hello');
      expect(result.redirect).toBe('https://test.com/?q=hello');
    });

    it('throws when trying to override built-in bang', async () => {
      const bang = {
        id: 100,
        trigger: 'g',
        name: 'Override Google',
        url_template: 'https://fake.com/?q={query}',
        category: 'test',
        is_builtin: false,
        created_at: new Date().toISOString(),
      };

      await expect(service.createBang(bang)).rejects.toThrow('Cannot override built-in bang: !g');
    });
  });

  describe('deleteBang', () => {
    it('deletes a custom bang', async () => {
      // First create a bang
      await kvStore.createBang({
        id: 100,
        trigger: 'deleteme',
        name: 'Delete Me',
        url_template: 'https://deleteme.com/?q={query}',
        category: 'custom',
        is_builtin: false,
        created_at: new Date().toISOString(),
      });

      // Verify it works
      let result = await service.parse('!deleteme test');
      expect(result.redirect).toBeTruthy();

      // Delete it
      await service.deleteBang('deleteme');

      // Verify it no longer works
      result = await service.parse('!deleteme test');
      expect(result.redirect).toBeUndefined();
    });

    it('throws when trying to delete built-in bang', async () => {
      await expect(service.deleteBang('g')).rejects.toThrow('Cannot delete built-in bang: !g');
    });
  });
});
