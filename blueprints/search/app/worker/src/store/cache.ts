import type {
  SearchResponse,
  Suggestion,
  KnowledgePanel,
  InstantAnswer,
  ImageSearchResponse,
  VideoSearchResponse,
} from '../types';

export class CacheStore {
  private kv: KVNamespace;

  constructor(kv: KVNamespace) {
    this.kv = kv;
  }

  // --- Search results cache (permanent) ---

  async getSearch(hash: string): Promise<SearchResponse | null> {
    const raw = await this.kv.get(`cache:search:${hash}`);
    if (!raw) return null;
    return JSON.parse(raw) as SearchResponse;
  }

  async setSearch(hash: string, response: SearchResponse): Promise<void> {
    await this.kv.put(`cache:search:${hash}`, JSON.stringify(response));
  }

  // --- Image search results cache (permanent) ---

  async getImageSearch(hash: string): Promise<ImageSearchResponse | null> {
    const raw = await this.kv.get(`cache:imgsearch:${hash}`);
    if (!raw) return null;
    return JSON.parse(raw) as ImageSearchResponse;
  }

  async setImageSearch(hash: string, response: ImageSearchResponse): Promise<void> {
    await this.kv.put(`cache:imgsearch:${hash}`, JSON.stringify(response));
  }

  // --- Video search results cache (permanent) ---

  async getVideoSearch(hash: string): Promise<VideoSearchResponse | null> {
    const raw = await this.kv.get(`cache:vidsearch:${hash}`);
    if (!raw) return null;
    return JSON.parse(raw) as VideoSearchResponse;
  }

  async setVideoSearch(hash: string, response: VideoSearchResponse): Promise<void> {
    await this.kv.put(`cache:vidsearch:${hash}`, JSON.stringify(response));
  }

  // --- Suggestions cache (permanent) ---

  async getSuggest(hash: string): Promise<Suggestion[] | null> {
    const raw = await this.kv.get(`cache:suggest:${hash}`);
    if (!raw) return null;
    return JSON.parse(raw) as Suggestion[];
  }

  async setSuggest(hash: string, suggestions: Suggestion[]): Promise<void> {
    await this.kv.put(`cache:suggest:${hash}`, JSON.stringify(suggestions));
  }

  // --- Knowledge panel cache (permanent) ---

  async getKnowledge(query: string): Promise<KnowledgePanel | null> {
    const raw = await this.kv.get(`cache:knowledge:${query}`);
    if (!raw) return null;
    return JSON.parse(raw) as KnowledgePanel;
  }

  async setKnowledge(query: string, panel: KnowledgePanel): Promise<void> {
    await this.kv.put(`cache:knowledge:${query}`, JSON.stringify(panel));
  }

  // --- Instant answer cache (permanent) ---

  async getInstant(hash: string): Promise<InstantAnswer | null> {
    const raw = await this.kv.get(`cache:instant:${hash}`);
    if (!raw) return null;
    return JSON.parse(raw) as InstantAnswer;
  }

  async setInstant(hash: string, answer: InstantAnswer): Promise<void> {
    await this.kv.put(`cache:instant:${hash}`, JSON.stringify(answer));
  }
}
