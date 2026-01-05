import type { Store } from '../store/types';
import type { Favorite, Page } from '../models';
import { generateId } from '../utils/id';

export class FavoriteService {
  constructor(private store: Store) {}

  async add(pageId: string, userId: string): Promise<Favorite> {
    // Check if already favorited
    const existing = await this.store.favorites.getByUserAndPage(userId, pageId);
    if (existing) {
      return existing;
    }

    const position = await this.store.favorites.getMaxPosition(userId);

    return this.store.favorites.create({
      id: generateId(),
      userId,
      pageId,
      position,
    });
  }

  async remove(pageId: string, userId: string): Promise<void> {
    await this.store.favorites.deleteByUserAndPage(userId, pageId);
  }

  async listByUser(userId: string, workspaceId: string): Promise<Favorite[]> {
    return this.store.favorites.listByUser(userId, workspaceId);
  }

  async listPagesWithFavorites(userId: string, workspaceId: string): Promise<(Favorite & { page: Page })[]> {
    const favorites = await this.store.favorites.listByUser(userId, workspaceId);
    const results: (Favorite & { page: Page })[] = [];

    for (const favorite of favorites) {
      const page = await this.store.pages.getById(favorite.pageId);
      if (page && !page.isArchived) {
        results.push({ ...favorite, page });
      }
    }

    return results;
  }

  async isFavorite(pageId: string, userId: string): Promise<boolean> {
    const favorite = await this.store.favorites.getByUserAndPage(userId, pageId);
    return favorite !== null;
  }

  async reorder(pageId: string, newPosition: number, userId: string): Promise<void> {
    const favorite = await this.store.favorites.getByUserAndPage(userId, pageId);
    if (!favorite) {
      throw new Error('Favorite not found');
    }

    // Update position (simplified - would need more logic for real reordering)
    await this.store.favorites.delete(favorite.id);
    await this.store.favorites.create({
      id: generateId(),
      userId,
      pageId,
      position: newPosition,
    });
  }
}
