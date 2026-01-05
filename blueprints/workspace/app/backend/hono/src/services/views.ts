import type { Store } from '../store/types';
import type { View, CreateView, UpdateView, QueryView, Page, PaginatedResult } from '../models';
import { generateId } from '../utils/id';

export class ViewService {
  constructor(private store: Store) {}

  async create(input: CreateView): Promise<View> {
    const position = await this.store.views.getMaxPosition(input.databaseId);

    return this.store.views.create({
      id: generateId(),
      databaseId: input.databaseId,
      name: input.name,
      type: input.type ?? 'table',
      position,
    });
  }

  async getById(id: string): Promise<View | null> {
    return this.store.views.getById(id);
  }

  async listByDatabase(databaseId: string): Promise<View[]> {
    return this.store.views.listByDatabase(databaseId);
  }

  async update(id: string, data: UpdateView): Promise<View> {
    return this.store.views.update(id, data);
  }

  async delete(id: string): Promise<void> {
    await this.store.views.delete(id);
  }

  async query(id: string, input: QueryView): Promise<PaginatedResult<Page>> {
    const view = await this.store.views.getById(id);
    if (!view) {
      throw new Error('View not found');
    }

    // Merge view filters/sorts with query filters/sorts
    const filter = input.filter ?? view.filter ?? undefined;
    const sorts = input.sorts ?? view.sorts ?? undefined;

    return this.store.pages.listByDatabase(view.databaseId, {
      cursor: input.cursor,
      limit: input.limit,
      filter,
      sorts,
    });
  }
}
