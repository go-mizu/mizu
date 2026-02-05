import { describe, it, expect, beforeEach } from 'vitest';
import { KVStore } from './kv';
import type { SearchLens, UserPreference, SearchHistory, Bang } from '../types';

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

describe('KVStore', () => {
  let kvStore: KVStore;
  let kv: KVNamespace;

  beforeEach(() => {
    kv = createMockKV();
    kvStore = new KVStore(kv);
  });

  describe('Settings', () => {
    it('returns default settings when none stored', async () => {
      const settings = await kvStore.getSettings();

      expect(settings.safe_search).toBe('moderate');
      expect(settings.results_per_page).toBe(10);
      expect(settings.region).toBe('');
      expect(settings.language).toBe('en');
      expect(settings.theme).toBe('system');
      expect(settings.open_in_new_tab).toBe(false);
      expect(settings.show_thumbnails).toBe(true);
    });

    it('updates and retrieves settings', async () => {
      await kvStore.updateSettings({
        safe_search: 'strict',
        results_per_page: 20,
      });

      const settings = await kvStore.getSettings();
      expect(settings.safe_search).toBe('strict');
      expect(settings.results_per_page).toBe(20);
      // Other defaults should remain
      expect(settings.language).toBe('en');
    });

    it('merges partial updates', async () => {
      await kvStore.updateSettings({ theme: 'dark' });
      await kvStore.updateSettings({ language: 'fr' });

      const settings = await kvStore.getSettings();
      expect(settings.theme).toBe('dark');
      expect(settings.language).toBe('fr');
    });
  });

  describe('Preferences', () => {
    it('creates and retrieves a preference', async () => {
      const pref: UserPreference = {
        id: 'pref1',
        domain: 'example.com',
        action: 'block',
        level: -2,
        created_at: new Date().toISOString(),
      };

      await kvStore.setPreference(pref);

      const retrieved = await kvStore.getPreference('example.com');
      expect(retrieved).toEqual(pref);
    });

    it('lists all preferences', async () => {
      await kvStore.setPreference({
        id: 'pref1',
        domain: 'site1.com',
        action: 'block',
        level: -2,
        created_at: new Date().toISOString(),
      });

      await kvStore.setPreference({
        id: 'pref2',
        domain: 'site2.com',
        action: 'boost',
        level: 1,
        created_at: new Date().toISOString(),
      });

      const prefs = await kvStore.listPreferences();
      expect(prefs.length).toBe(2);
      expect(prefs.map((p) => p.domain)).toContain('site1.com');
      expect(prefs.map((p) => p.domain)).toContain('site2.com');
    });

    it('deletes a preference', async () => {
      await kvStore.setPreference({
        id: 'pref1',
        domain: 'todelete.com',
        action: 'block',
        level: -2,
        created_at: new Date().toISOString(),
      });

      await kvStore.deletePreference('todelete.com');

      const retrieved = await kvStore.getPreference('todelete.com');
      expect(retrieved).toBeNull();
    });

    it('returns null for non-existent preference', async () => {
      const pref = await kvStore.getPreference('nonexistent.com');
      expect(pref).toBeNull();
    });
  });

  describe('Lenses', () => {
    const createLens = (id: string, name: string): SearchLens => ({
      id,
      name,
      description: `Description for ${name}`,
      domains: ['example.com'],
      is_public: false,
      is_built_in: false,
      is_shared: false,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });

    it('creates and retrieves a lens', async () => {
      const lens = createLens('lens1', 'My Lens');
      await kvStore.createLens(lens);

      const retrieved = await kvStore.getLens('lens1');
      expect(retrieved).toEqual(lens);
    });

    it('lists all lenses', async () => {
      await kvStore.createLens(createLens('lens1', 'Lens 1'));
      await kvStore.createLens(createLens('lens2', 'Lens 2'));

      const lenses = await kvStore.listLenses();
      expect(lenses.length).toBe(2);
    });

    it('updates a lens', async () => {
      await kvStore.createLens(createLens('lens1', 'Original Name'));

      const updated = await kvStore.updateLens('lens1', { name: 'Updated Name' });

      expect(updated?.name).toBe('Updated Name');
      expect(updated?.updated_at).not.toBe(updated?.created_at);
    });

    it('returns null when updating non-existent lens', async () => {
      const result = await kvStore.updateLens('nonexistent', { name: 'Test' });
      expect(result).toBeNull();
    });

    it('deletes a lens', async () => {
      await kvStore.createLens(createLens('lens1', 'To Delete'));
      await kvStore.deleteLens('lens1');

      const retrieved = await kvStore.getLens('lens1');
      expect(retrieved).toBeNull();
    });

    it('returns null for non-existent lens', async () => {
      const lens = await kvStore.getLens('nonexistent');
      expect(lens).toBeNull();
    });
  });

  describe('History', () => {
    const createHistoryEntry = (id: string, query: string): SearchHistory => ({
      id,
      query,
      results: Math.floor(Math.random() * 1000),
      searched_at: new Date().toISOString(),
    });

    it('adds and retrieves history', async () => {
      await kvStore.addHistory(createHistoryEntry('h1', 'test query'));

      const history = await kvStore.listHistory();
      expect(history.length).toBe(1);
      expect(history[0].query).toBe('test query');
    });

    it('orders history newest first', async () => {
      await kvStore.addHistory(createHistoryEntry('h1', 'first'));
      await kvStore.addHistory(createHistoryEntry('h2', 'second'));
      await kvStore.addHistory(createHistoryEntry('h3', 'third'));

      const history = await kvStore.listHistory();
      expect(history[0].query).toBe('third');
      expect(history[1].query).toBe('second');
      expect(history[2].query).toBe('first');
    });

    it('limits history retrieval', async () => {
      for (let i = 0; i < 10; i++) {
        await kvStore.addHistory(createHistoryEntry(`h${i}`, `query ${i}`));
      }

      const history = await kvStore.listHistory(5);
      expect(history.length).toBe(5);
    });

    it('deletes specific history entry', async () => {
      await kvStore.addHistory(createHistoryEntry('h1', 'first'));
      await kvStore.addHistory(createHistoryEntry('h2', 'second'));

      await kvStore.deleteHistory('h1');

      const history = await kvStore.listHistory();
      expect(history.length).toBe(1);
      expect(history[0].id).toBe('h2');
    });

    it('clears all history', async () => {
      await kvStore.addHistory(createHistoryEntry('h1', 'first'));
      await kvStore.addHistory(createHistoryEntry('h2', 'second'));

      await kvStore.clearHistory();

      const history = await kvStore.listHistory();
      expect(history.length).toBe(0);
    });
  });

  describe('Bangs', () => {
    const createBang = (trigger: string, name: string): Bang => ({
      id: Math.floor(Math.random() * 1000),
      trigger,
      name,
      url_template: `https://${trigger}.example.com/?q={query}`,
      category: 'custom',
      is_builtin: false,
      created_at: new Date().toISOString(),
    });

    it('creates and retrieves a bang', async () => {
      const bang = createBang('test', 'Test Bang');
      await kvStore.createBang(bang);

      const retrieved = await kvStore.getBang('test');
      expect(retrieved).toEqual(bang);
    });

    it('lists all bangs', async () => {
      await kvStore.createBang(createBang('b1', 'Bang 1'));
      await kvStore.createBang(createBang('b2', 'Bang 2'));

      const bangs = await kvStore.listBangs();
      expect(bangs.length).toBe(2);
    });

    it('deletes a bang', async () => {
      await kvStore.createBang(createBang('todelete', 'To Delete'));
      await kvStore.deleteBang('todelete');

      const retrieved = await kvStore.getBang('todelete');
      expect(retrieved).toBeNull();
    });

    it('returns null for non-existent bang', async () => {
      const bang = await kvStore.getBang('nonexistent');
      expect(bang).toBeNull();
    });
  });

  describe('Widget Settings', () => {
    it('returns default widget settings', async () => {
      const settings = await kvStore.getWidgetSettings();

      expect(settings.calculator).toBe(true);
      expect(settings.unit_converter).toBe(true);
      expect(settings.currency).toBe(true);
      expect(settings.weather).toBe(true);
      expect(settings.dictionary).toBe(true);
      expect(settings.time_zones).toBe(true);
      expect(settings.knowledge_panel).toBe(true);
    });

    it('updates widget settings', async () => {
      await kvStore.updateWidgetSettings({
        calculator: false,
        weather: false,
      });

      const settings = await kvStore.getWidgetSettings();
      expect(settings.calculator).toBe(false);
      expect(settings.weather).toBe(false);
      // Other defaults should remain
      expect(settings.currency).toBe(true);
    });
  });
});
