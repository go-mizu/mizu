import type { KnowledgePanel, Fact, Link } from '../types';
import type { CacheStore } from '../store/cache';

interface WikipediaSummary {
  title: string;
  description?: string;
  extract: string;
  thumbnail?: {
    source: string;
    width: number;
    height: number;
  };
  content_urls?: {
    desktop?: { page: string };
    mobile?: { page: string };
  };
  type: string;
}

interface WikidataEntity {
  labels?: Record<string, { value: string }>;
  descriptions?: Record<string, { value: string }>;
  claims?: Record<string, WikidataClaim[]>;
  sitelinks?: Record<string, { title: string; url?: string }>;
}

interface WikidataClaim {
  mainsnak?: {
    datavalue?: {
      value: string | { text?: string; amount?: string; time?: string; id?: string };
      type: string;
    };
    datatype?: string;
  };
}

// Property IDs for common Wikidata facts
const WIKIDATA_PROPERTIES: Record<string, string> = {
  P569: 'Born',
  P570: 'Died',
  P19: 'Place of birth',
  P20: 'Place of death',
  P27: 'Nationality',
  P106: 'Occupation',
  P1412: 'Languages spoken',
  P17: 'Country',
  P36: 'Capital',
  P1082: 'Population',
  P571: 'Founded',
  P112: 'Founded by',
  P159: 'Headquarters',
  P452: 'Industry',
  P856: 'Official website',
  P1448: 'Official name',
  P18: 'Image',
};

export class KnowledgeService {
  private cache: CacheStore;

  constructor(cache: CacheStore) {
    this.cache = cache;
  }

  /**
   * Get a knowledge panel for a query.
   * Checks cache first, then tries Wikipedia and Wikidata APIs.
   */
  async getPanel(query: string): Promise<KnowledgePanel | null> {
    const normalizedQuery = query.trim().toLowerCase();

    if (!normalizedQuery) {
      return null;
    }

    // Check cache
    const cached = await this.cache.getKnowledge(normalizedQuery);
    if (cached) {
      return cached;
    }

    // Try Wikipedia summary API
    const panel = await this.fetchWikipediaPanel(normalizedQuery);
    if (!panel) {
      return null;
    }

    // Try to enrich with Wikidata facts
    const enrichedPanel = await this.enrichWithWikidata(panel, normalizedQuery);

    // Cache the result
    await this.cache.setKnowledge(normalizedQuery, enrichedPanel);

    return enrichedPanel;
  }

  private async fetchWikipediaPanel(query: string): Promise<KnowledgePanel | null> {
    // Encode the query for use in the URL, converting spaces to underscores
    // as expected by the Wikipedia API
    const encoded = encodeURIComponent(query.replace(/\s+/g, '_'));
    const url = `https://en.wikipedia.org/api/rest_v1/page/summary/${encoded}`;

    try {
      const response = await fetch(url, {
        headers: {
          'User-Agent': 'mizu-search/1.0 (search engine)',
          'Accept': 'application/json',
        },
      });

      if (!response.ok) {
        // Try a search-based fallback
        return this.searchWikipedia(query);
      }

      const data = (await response.json()) as WikipediaSummary;

      // Skip disambiguation and other non-article pages
      if (data.type === 'disambiguation' || data.type === 'no-extract') {
        return this.searchWikipedia(query);
      }

      if (!data.extract || data.extract.length < 20) {
        return null;
      }

      const links: Link[] = [];
      if (data.content_urls?.desktop?.page) {
        links.push({
          title: 'Wikipedia',
          url: data.content_urls.desktop.page,
          icon: 'wikipedia',
        });
      }

      return {
        title: data.title,
        subtitle: data.description,
        description: data.extract,
        image: data.thumbnail?.source,
        facts: [],
        links,
        source: 'Wikipedia',
      };
    } catch {
      return null;
    }
  }

  /**
   * Fallback: search Wikipedia and use the first result.
   */
  private async searchWikipedia(query: string): Promise<KnowledgePanel | null> {
    const encoded = encodeURIComponent(query);
    const url = `https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=${encoded}&format=json&utf8=1&srlimit=1&srprop=snippet`;

    try {
      const response = await fetch(url, {
        headers: {
          'User-Agent': 'mizu-search/1.0 (search engine)',
          'Accept': 'application/json',
        },
      });

      if (!response.ok) return null;

      const data = (await response.json()) as {
        query: {
          search: Array<{
            title: string;
            pageid: number;
            snippet: string;
          }>;
        };
      };

      const results = data.query?.search;
      if (!results || results.length === 0) return null;

      const firstResult = results[0];
      // Fetch the full summary using the found title
      const titleEncoded = encodeURIComponent(firstResult.title.replace(/\s+/g, '_'));
      const summaryUrl = `https://en.wikipedia.org/api/rest_v1/page/summary/${titleEncoded}`;

      const summaryResponse = await fetch(summaryUrl, {
        headers: {
          'User-Agent': 'mizu-search/1.0 (search engine)',
          'Accept': 'application/json',
        },
      });

      if (!summaryResponse.ok) return null;

      const summaryData = (await summaryResponse.json()) as WikipediaSummary;

      if (!summaryData.extract || summaryData.extract.length < 20) {
        return null;
      }

      const links: Link[] = [];
      if (summaryData.content_urls?.desktop?.page) {
        links.push({
          title: 'Wikipedia',
          url: summaryData.content_urls.desktop.page,
          icon: 'wikipedia',
        });
      }

      return {
        title: summaryData.title,
        subtitle: summaryData.description,
        description: summaryData.extract,
        image: summaryData.thumbnail?.source,
        facts: [],
        links,
        source: 'Wikipedia',
      };
    } catch {
      return null;
    }
  }

  /**
   * Enrich a knowledge panel with Wikidata structured facts.
   */
  private async enrichWithWikidata(
    panel: KnowledgePanel,
    query: string,
  ): Promise<KnowledgePanel> {
    try {
      // Search Wikidata for the entity
      const searchEncoded = encodeURIComponent(query);
      const searchUrl = `https://www.wikidata.org/w/api.php?action=wbsearchentities&search=${searchEncoded}&language=en&format=json&limit=1`;

      const searchResponse = await fetch(searchUrl, {
        headers: {
          'User-Agent': 'mizu-search/1.0 (search engine)',
          'Accept': 'application/json',
        },
      });

      if (!searchResponse.ok) return panel;

      const searchData = (await searchResponse.json()) as {
        search: Array<{ id: string; label: string; description?: string }>;
      };

      if (!searchData.search || searchData.search.length === 0) {
        return panel;
      }

      const entityId = searchData.search[0].id;

      // Fetch entity details
      const entityUrl = `https://www.wikidata.org/w/api.php?action=wbgetentities&ids=${entityId}&languages=en&format=json&props=claims|labels|descriptions|sitelinks`;

      const entityResponse = await fetch(entityUrl, {
        headers: {
          'User-Agent': 'mizu-search/1.0 (search engine)',
          'Accept': 'application/json',
        },
      });

      if (!entityResponse.ok) return panel;

      const entityData = (await entityResponse.json()) as {
        entities: Record<string, WikidataEntity>;
      };

      const entity = entityData.entities[entityId];
      if (!entity || !entity.claims) return panel;

      // Extract facts from claims
      const facts: Fact[] = [];
      for (const [propId, label] of Object.entries(WIKIDATA_PROPERTIES)) {
        const claims = entity.claims[propId];
        if (!claims || claims.length === 0) continue;

        const claim = claims[0];
        const value = this.extractClaimValue(claim);
        if (value) {
          facts.push({ label, value });
        }
      }

      // Add Wikidata link
      const links = [...(panel.links ?? [])];
      links.push({
        title: 'Wikidata',
        url: `https://www.wikidata.org/wiki/${entityId}`,
        icon: 'wikidata',
      });

      return {
        ...panel,
        facts: facts.length > 0 ? facts : panel.facts,
        links,
      };
    } catch {
      // If Wikidata enrichment fails, return original panel
      return panel;
    }
  }

  /**
   * Extract a human-readable value from a Wikidata claim.
   */
  private extractClaimValue(claim: WikidataClaim): string | null {
    const datavalue = claim.mainsnak?.datavalue;
    if (!datavalue) return null;

    switch (datavalue.type) {
      case 'string':
        return typeof datavalue.value === 'string' ? datavalue.value : null;

      case 'monolingualtext':
        if (typeof datavalue.value === 'object' && datavalue.value?.text) {
          return datavalue.value.text;
        }
        return null;

      case 'quantity':
        if (typeof datavalue.value === 'object' && datavalue.value?.amount) {
          const amount = datavalue.value.amount.replace(/^\+/, '');
          // Format large numbers with commas
          const num = parseFloat(amount);
          if (!isNaN(num)) {
            return num.toLocaleString('en-US');
          }
          return amount;
        }
        return null;

      case 'time':
        if (typeof datavalue.value === 'object' && datavalue.value?.time) {
          // Wikidata time format: +YYYY-MM-DDT00:00:00Z
          const timeStr = datavalue.value.time;
          const match = timeStr.match(/^\+?(-?\d{4})-(\d{2})-(\d{2})/);
          if (match) {
            const year = parseInt(match[1], 10);
            const month = parseInt(match[2], 10);
            const day = parseInt(match[3], 10);
            if (month > 0 && day > 0) {
              const date = new Date(year, month - 1, day);
              return date.toLocaleDateString('en-US', {
                year: 'numeric',
                month: 'long',
                day: 'numeric',
              });
            }
            return String(year);
          }
          return null;
        }
        return null;

      case 'wikibase-entityid':
        // This is a reference to another entity -- just return the ID
        // A full implementation would resolve these, but that would require
        // additional API calls for each one
        if (typeof datavalue.value === 'object' && datavalue.value?.id) {
          return datavalue.value.id;
        }
        return null;

      default:
        if (typeof datavalue.value === 'string') {
          return datavalue.value;
        }
        return null;
    }
  }
}
