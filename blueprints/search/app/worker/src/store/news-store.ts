/**
 * KV store for news user preferences and reading history.
 */

import type {
  NewsUserPreferences,
  ReadingHistoryEntry,
  NewsCategory,
} from '../types';

const MAX_HISTORY = 200;
const MAX_INTERESTS = 20;

const DEFAULT_PREFERENCES: Omit<NewsUserPreferences, 'userId' | 'createdAt' | 'updatedAt'> = {
  followedTopics: [],
  followedSources: [],
  hiddenSources: [],
  interests: [],
  language: 'en',
  region: 'US',
};

export class NewsStore {
  private kv: KVNamespace;

  constructor(kv: KVNamespace) {
    this.kv = kv;
  }

  // --- User Preferences ---

  private prefKey(userId: string): string {
    return `news:user:${userId}`;
  }

  private historyKey(userId: string): string {
    return `news:history:${userId}`;
  }

  async getPreferences(userId: string): Promise<NewsUserPreferences> {
    const raw = await this.kv.get(this.prefKey(userId));
    if (!raw) {
      return this.createDefaultPreferences(userId);
    }
    return JSON.parse(raw) as NewsUserPreferences;
  }

  private createDefaultPreferences(userId: string): NewsUserPreferences {
    const now = new Date().toISOString();
    return {
      userId,
      ...DEFAULT_PREFERENCES,
      createdAt: now,
      updatedAt: now,
    };
  }

  async updatePreferences(
    userId: string,
    updates: Partial<Omit<NewsUserPreferences, 'userId' | 'createdAt'>>
  ): Promise<NewsUserPreferences> {
    const current = await this.getPreferences(userId);
    const updated: NewsUserPreferences = {
      ...current,
      ...updates,
      userId,
      createdAt: current.createdAt,
      updatedAt: new Date().toISOString(),
    };
    await this.kv.put(this.prefKey(userId), JSON.stringify(updated));
    return updated;
  }

  // --- Follow/Unfollow Topics ---

  async followTopic(userId: string, topic: string): Promise<void> {
    const prefs = await this.getPreferences(userId);
    if (!prefs.followedTopics.includes(topic)) {
      prefs.followedTopics.push(topic);
      await this.updatePreferences(userId, { followedTopics: prefs.followedTopics });
    }
  }

  async unfollowTopic(userId: string, topic: string): Promise<void> {
    const prefs = await this.getPreferences(userId);
    prefs.followedTopics = prefs.followedTopics.filter((t) => t !== topic);
    await this.updatePreferences(userId, { followedTopics: prefs.followedTopics });
  }

  // --- Follow/Unfollow Sources ---

  async followSource(userId: string, source: string): Promise<void> {
    const prefs = await this.getPreferences(userId);
    if (!prefs.followedSources.includes(source)) {
      prefs.followedSources.push(source);
      await this.updatePreferences(userId, { followedSources: prefs.followedSources });
    }
  }

  async unfollowSource(userId: string, source: string): Promise<void> {
    const prefs = await this.getPreferences(userId);
    prefs.followedSources = prefs.followedSources.filter((s) => s !== source);
    await this.updatePreferences(userId, { followedSources: prefs.followedSources });
  }

  // --- Hide/Unhide Sources ---

  async hideSource(userId: string, source: string): Promise<void> {
    const prefs = await this.getPreferences(userId);
    if (!prefs.hiddenSources.includes(source)) {
      prefs.hiddenSources.push(source);
      await this.updatePreferences(userId, { hiddenSources: prefs.hiddenSources });
    }
  }

  async unhideSource(userId: string, source: string): Promise<void> {
    const prefs = await this.getPreferences(userId);
    prefs.hiddenSources = prefs.hiddenSources.filter((s) => s !== source);
    await this.updatePreferences(userId, { hiddenSources: prefs.hiddenSources });
  }

  // --- Location ---

  async setLocation(
    userId: string,
    location: NewsUserPreferences['location']
  ): Promise<void> {
    await this.updatePreferences(userId, { location });
  }

  async addLocation(
    userId: string,
    location: NonNullable<NewsUserPreferences['location']>
  ): Promise<void> {
    const prefs = await this.getPreferences(userId);
    const locations = prefs.locations || [];
    // Check if location already exists
    const exists = locations.some(
      (l) => l.city === location.city && l.country === location.country
    );
    if (!exists) {
      locations.push(location);
      await this.updatePreferences(userId, { locations });
    }
  }

  async removeLocation(userId: string, city: string, country: string): Promise<void> {
    const prefs = await this.getPreferences(userId);
    const locations = (prefs.locations || []).filter(
      (l) => !(l.city === city && l.country === country)
    );
    await this.updatePreferences(userId, { locations });
  }

  // --- Reading History ---

  async getReadingHistory(userId: string, limit = 50): Promise<ReadingHistoryEntry[]> {
    const raw = await this.kv.get(this.historyKey(userId));
    if (!raw) return [];
    const history = JSON.parse(raw) as ReadingHistoryEntry[];
    return history.slice(0, limit);
  }

  async addToReadingHistory(userId: string, entry: ReadingHistoryEntry): Promise<void> {
    const history = await this.getReadingHistory(userId, MAX_HISTORY);

    // Remove duplicate if exists
    const filtered = history.filter((h) => h.articleId !== entry.articleId);

    // Prepend new entry
    const updated = [entry, ...filtered].slice(0, MAX_HISTORY);

    await this.kv.put(this.historyKey(userId), JSON.stringify(updated));

    // Update interests based on reading
    await this.updateInterestsFromReading(userId, entry);
  }

  async clearReadingHistory(userId: string): Promise<void> {
    await this.kv.delete(this.historyKey(userId));
  }

  // --- Interest Inference ---

  private async updateInterestsFromReading(
    userId: string,
    entry: ReadingHistoryEntry
  ): Promise<void> {
    const prefs = await this.getPreferences(userId);
    const interests = [...prefs.interests];

    // Add category as interest if not already present
    if (!interests.includes(entry.category)) {
      interests.push(entry.category);
    }

    // Add source as interest (normalized)
    const normalizedSource = entry.source.toLowerCase().replace(/\s+/g, '-');
    if (!interests.includes(normalizedSource)) {
      interests.push(normalizedSource);
    }

    // Trim to max interests (keep most recent)
    const trimmed = interests.slice(-MAX_INTERESTS);

    if (trimmed.length !== prefs.interests.length) {
      await this.updatePreferences(userId, { interests: trimmed });
    }
  }

  // --- Compute personalized score ---

  computePersonalizedScore(
    article: { category: NewsCategory; source: string; score: number },
    prefs: NewsUserPreferences
  ): number {
    let score = article.score;

    // Boost if source is followed
    const normalizedSource = article.source.toLowerCase().replace(/\s+/g, '-');
    if (prefs.followedSources.some((s) => s.toLowerCase() === normalizedSource)) {
      score *= 1.5;
    }

    // Boost if category is followed topic
    if (prefs.followedTopics.includes(article.category)) {
      score *= 1.3;
    }

    // Boost if in interests
    if (prefs.interests.includes(article.category)) {
      score *= 1.1;
    }
    if (prefs.interests.includes(normalizedSource)) {
      score *= 1.1;
    }

    // Penalize hidden sources (should be filtered but just in case)
    if (prefs.hiddenSources.some((s) => s.toLowerCase() === normalizedSource)) {
      score = 0;
    }

    return score;
  }
}
