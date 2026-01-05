import type { Store } from '../store/types';
import type { Page, Database } from '../models';

export interface SearchResult {
  type: 'page' | 'database';
  id: string;
  title: string;
  icon?: string | null;
  parentTitle?: string;
  snippet?: string;
}

export interface RecentItem {
  type: 'page' | 'database';
  id: string;
  title: string;
  icon?: string | null;
  updatedAt: string;
}

export class SearchService {
  constructor(private store: Store) {}

  async search(workspaceId: string, query: string): Promise<SearchResult[]> {
    const results: SearchResult[] = [];
    const lowerQuery = query.toLowerCase();

    // Search pages (excluding database rows)
    const pages = await this.store.pages.listByWorkspace(workspaceId, { includeArchived: false });
    for (const page of pages) {
      if (!page.databaseId && page.title.toLowerCase().includes(lowerQuery)) {
        results.push({
          type: 'page',
          id: page.id,
          title: page.title || 'Untitled',
          icon: page.icon,
        });
      }
    }

    // Search databases and their rows
    const databases = await this.store.databases.listByWorkspace(workspaceId);
    for (const db of databases) {
      if (db.title.toLowerCase().includes(lowerQuery)) {
        results.push({
          type: 'database',
          id: db.id,
          title: db.title || 'Untitled Database',
          icon: db.icon,
        });
      }

      // Search database rows
      const rows = await this.store.pages.listByDatabase(db.id, { limit: 100 });
      for (const row of rows.items) {
        // Check row title or properties.title
        const rowTitle = row.title || (row.properties?.title as string) || '';
        if (rowTitle.toLowerCase().includes(lowerQuery)) {
          results.push({
            type: 'page',
            id: row.id,
            title: rowTitle || 'Untitled',
            icon: row.icon,
            parentTitle: db.title,
          });
        }
      }
    }

    // Sort by relevance (exact match first, then by title length)
    results.sort((a, b) => {
      const aExact = a.title.toLowerCase() === lowerQuery;
      const bExact = b.title.toLowerCase() === lowerQuery;
      if (aExact && !bExact) return -1;
      if (!aExact && bExact) return 1;
      return a.title.length - b.title.length;
    });

    return results.slice(0, 20);
  }

  async quickSearch(workspaceId: string, query: string): Promise<SearchResult[]> {
    // Quick search returns fewer results for fast autocomplete
    const results = await this.search(workspaceId, query);
    return results.slice(0, 5);
  }

  async getRecent(workspaceId: string, userId: string, limit = 10): Promise<RecentItem[]> {
    const pages = await this.store.pages.listByWorkspace(workspaceId, { includeArchived: false });

    // Sort by updated_at descending
    const sortedPages = pages
      .filter((p) => !p.databaseId) // Exclude database rows
      .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
      .slice(0, limit);

    return sortedPages.map((page) => ({
      type: 'page' as const,
      id: page.id,
      title: page.title || 'Untitled',
      icon: page.icon,
      updatedAt: page.updatedAt,
    }));
  }

  async searchInDatabase(
    databaseId: string,
    query: string
  ): Promise<Page[]> {
    const result = await this.store.pages.listByDatabase(databaseId, { limit: 100 });
    const lowerQuery = query.toLowerCase();

    return result.items.filter((row) => {
      // Search in title
      if (row.title.toLowerCase().includes(lowerQuery)) return true;

      // Search in properties
      for (const value of Object.values(row.properties)) {
        if (typeof value === 'string' && value.toLowerCase().includes(lowerQuery)) {
          return true;
        }
      }

      return false;
    });
  }
}
