import type {
  SearchSettings,
  UserPreference,
  SearchLens,
  SearchHistory,
  Bang,
} from '../types';

export interface WidgetSettings {
  calculator: boolean;
  unit_converter: boolean;
  currency: boolean;
  weather: boolean;
  dictionary: boolean;
  time_zones: boolean;
  knowledge_panel: boolean;
}

const DEFAULT_SETTINGS: SearchSettings = {
  safe_search: 'moderate',
  results_per_page: 10,
  region: '',
  language: 'en',
  theme: 'system',
  open_in_new_tab: false,
  show_thumbnails: true,
};

const DEFAULT_WIDGET_SETTINGS: WidgetSettings = {
  calculator: true,
  unit_converter: true,
  currency: true,
  weather: true,
  dictionary: true,
  time_zones: true,
  knowledge_panel: true,
};

const MAX_HISTORY = 100;

export class KVStore {
  private kv: KVNamespace;

  constructor(kv: KVNamespace) {
    this.kv = kv;
  }

  // --- Settings ---

  async getSettings(): Promise<SearchSettings> {
    const raw = await this.kv.get('settings:default');
    if (!raw) {
      return { ...DEFAULT_SETTINGS };
    }
    return JSON.parse(raw) as SearchSettings;
  }

  async updateSettings(settings: Partial<SearchSettings>): Promise<SearchSettings> {
    const current = await this.getSettings();
    const merged: SearchSettings = { ...current, ...settings };
    await this.kv.put('settings:default', JSON.stringify(merged));
    return merged;
  }

  // --- Preferences ---

  async listPreferences(): Promise<UserPreference[]> {
    const index = await this.getIndex('preferences:_index');
    const preferences: UserPreference[] = [];
    for (const domain of index) {
      const pref = await this.getPreference(domain);
      if (pref) {
        preferences.push(pref);
      }
    }
    return preferences;
  }

  async getPreference(domain: string): Promise<UserPreference | null> {
    const raw = await this.kv.get(`preferences:${domain}`);
    if (!raw) return null;
    return JSON.parse(raw) as UserPreference;
  }

  async setPreference(pref: UserPreference): Promise<void> {
    await this.kv.put(`preferences:${pref.domain}`, JSON.stringify(pref));
    await this.addToIndex('preferences:_index', pref.domain);
  }

  async deletePreference(domain: string): Promise<void> {
    await this.kv.delete(`preferences:${domain}`);
    await this.removeFromIndex('preferences:_index', domain);
  }

  // --- Lenses ---

  async listLenses(): Promise<SearchLens[]> {
    const index = await this.getIndex('lenses:_index');
    const lenses: SearchLens[] = [];
    for (const id of index) {
      const lens = await this.getLens(id);
      if (lens) {
        lenses.push(lens);
      }
    }
    return lenses;
  }

  async getLens(id: string): Promise<SearchLens | null> {
    const raw = await this.kv.get(`lenses:${id}`);
    if (!raw) return null;
    return JSON.parse(raw) as SearchLens;
  }

  async createLens(lens: SearchLens): Promise<void> {
    await this.kv.put(`lenses:${lens.id}`, JSON.stringify(lens));
    await this.addToIndex('lenses:_index', lens.id);
  }

  async updateLens(id: string, lens: Partial<SearchLens>): Promise<SearchLens | null> {
    const current = await this.getLens(id);
    if (!current) return null;
    const updated: SearchLens = {
      ...current,
      ...lens,
      id,
      updated_at: new Date().toISOString(),
    };
    await this.kv.put(`lenses:${id}`, JSON.stringify(updated));
    return updated;
  }

  async deleteLens(id: string): Promise<void> {
    await this.kv.delete(`lenses:${id}`);
    await this.removeFromIndex('lenses:_index', id);
  }

  // --- History ---

  async listHistory(limit?: number): Promise<SearchHistory[]> {
    const index = await this.getIndex('history:_index');
    const sliced = limit ? index.slice(0, limit) : index;
    const entries: SearchHistory[] = [];
    for (const id of sliced) {
      const entry = await this.getHistoryEntry(id);
      if (entry) {
        entries.push(entry);
      }
    }
    return entries;
  }

  private async getHistoryEntry(id: string): Promise<SearchHistory | null> {
    const raw = await this.kv.get(`history:${id}`);
    if (!raw) return null;
    return JSON.parse(raw) as SearchHistory;
  }

  async addHistory(entry: SearchHistory): Promise<void> {
    await this.kv.put(`history:${entry.id}`, JSON.stringify(entry));

    const index = await this.getIndex('history:_index');
    // Prepend newest first
    const updated = [entry.id, ...index.filter((id) => id !== entry.id)];
    // Trim to max history size and clean up old entries
    if (updated.length > MAX_HISTORY) {
      const removed = updated.slice(MAX_HISTORY);
      for (const id of removed) {
        await this.kv.delete(`history:${id}`);
      }
    }
    await this.setIndex('history:_index', updated.slice(0, MAX_HISTORY));
  }

  async deleteHistory(id: string): Promise<void> {
    await this.kv.delete(`history:${id}`);
    await this.removeFromIndex('history:_index', id);
  }

  async clearHistory(): Promise<void> {
    const index = await this.getIndex('history:_index');
    for (const id of index) {
      await this.kv.delete(`history:${id}`);
    }
    await this.setIndex('history:_index', []);
  }

  // --- Bangs ---

  async listBangs(): Promise<Bang[]> {
    const index = await this.getIndex('bangs:_index');
    const bangs: Bang[] = [];
    for (const trigger of index) {
      const bang = await this.getBang(trigger);
      if (bang) {
        bangs.push(bang);
      }
    }
    return bangs;
  }

  async getBang(trigger: string): Promise<Bang | null> {
    const raw = await this.kv.get(`bangs:${trigger}`);
    if (!raw) return null;
    return JSON.parse(raw) as Bang;
  }

  async createBang(bang: Bang): Promise<void> {
    await this.kv.put(`bangs:${bang.trigger}`, JSON.stringify(bang));
    await this.addToIndex('bangs:_index', bang.trigger);
    if (!bang.is_builtin) {
      await this.addToIndex('bangs:_custom', bang.trigger);
    }
  }

  async deleteBang(trigger: string): Promise<void> {
    await this.kv.delete(`bangs:${trigger}`);
    await this.removeFromIndex('bangs:_index', trigger);
    await this.removeFromIndex('bangs:_custom', trigger);
  }

  // --- Widget Settings ---

  async getWidgetSettings(): Promise<WidgetSettings> {
    const raw = await this.kv.get('widgets:settings');
    if (!raw) {
      return { ...DEFAULT_WIDGET_SETTINGS };
    }
    return JSON.parse(raw) as WidgetSettings;
  }

  async updateWidgetSettings(settings: Partial<WidgetSettings>): Promise<WidgetSettings> {
    const current = await this.getWidgetSettings();
    const merged: WidgetSettings = { ...current, ...settings };
    await this.kv.put('widgets:settings', JSON.stringify(merged));
    return merged;
  }

  // --- Index helpers ---

  private async getIndex(key: string): Promise<string[]> {
    const raw = await this.kv.get(key);
    if (!raw) return [];
    return JSON.parse(raw) as string[];
  }

  private async setIndex(key: string, index: string[]): Promise<void> {
    await this.kv.put(key, JSON.stringify(index));
  }

  private async addToIndex(key: string, value: string): Promise<void> {
    const index = await this.getIndex(key);
    if (!index.includes(value)) {
      index.push(value);
      await this.setIndex(key, index);
    }
  }

  private async removeFromIndex(key: string, value: string): Promise<void> {
    const index = await this.getIndex(key);
    const filtered = index.filter((item) => item !== value);
    if (filtered.length !== index.length) {
      await this.setIndex(key, filtered);
    }
  }
}
