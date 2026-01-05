import type { Store } from '../store/types';
import type { Page, Block, CreateRow, UpdateRow, PaginatedResult } from '../models';
import { generateId } from '../utils/id';

export class RowService {
  constructor(private store: Store) {}

  async create(input: CreateRow, userId: string): Promise<Page> {
    const database = await this.store.databases.getById(input.databaseId);
    if (!database) {
      throw new Error('Database not found');
    }

    // Calculate position
    let rowPosition = 0;
    if (input.afterId) {
      const afterRow = await this.store.pages.getById(input.afterId);
      rowPosition = afterRow?.rowPosition ? afterRow.rowPosition + 1 : 0;
    } else {
      // Get max position
      const rows = await this.store.pages.listByDatabase(input.databaseId, { limit: 1 });
      rowPosition = rows.items.length > 0 ? (rows.items[0].rowPosition ?? 0) + 1 : 0;
    }

    return this.store.pages.create({
      id: generateId(),
      workspaceId: database.workspaceId,
      parentId: database.pageId,
      parentType: 'database',
      databaseId: input.databaseId,
      rowPosition,
      title: input.title ?? '',
      icon: input.icon,
      properties: input.properties ?? {},
      isTemplate: false,
      isArchived: false,
      createdBy: userId,
    });
  }

  async getById(id: string): Promise<Page | null> {
    const page = await this.store.pages.getById(id);
    if (!page || !page.databaseId) return null;
    return page;
  }

  async listByDatabase(databaseId: string, options?: {
    cursor?: string;
    limit?: number;
  }): Promise<PaginatedResult<Page>> {
    return this.store.pages.listByDatabase(databaseId, options);
  }

  async update(id: string, data: UpdateRow): Promise<Page> {
    return this.store.pages.update(id, data);
  }

  async delete(id: string): Promise<void> {
    // Delete blocks first
    await this.store.blocks.deleteByPage(id);
    // Delete the row (page)
    await this.store.pages.delete(id);
  }

  async duplicate(id: string, userId: string): Promise<Page> {
    const original = await this.store.pages.getById(id);
    if (!original || !original.databaseId) {
      throw new Error('Row not found');
    }

    // Calculate new position
    const rowPosition = (original.rowPosition ?? 0) + 0.5;

    // Create duplicate
    const duplicate = await this.store.pages.create({
      id: generateId(),
      workspaceId: original.workspaceId,
      parentId: original.parentId,
      parentType: original.parentType,
      databaseId: original.databaseId,
      rowPosition,
      title: original.title,
      icon: original.icon,
      properties: original.properties,
      isTemplate: false,
      isArchived: false,
      createdBy: userId,
    });

    // Copy blocks
    const blocks = await this.store.blocks.listByPage(id);
    const idMap = new Map<string, string>();

    for (const block of blocks) {
      const newId = generateId();
      idMap.set(block.id, newId);
    }

    for (const block of blocks) {
      await this.store.blocks.create({
        id: idMap.get(block.id)!,
        pageId: duplicate.id,
        parentId: block.parentId ? idMap.get(block.parentId) : undefined,
        type: block.type,
        content: block.content,
        position: block.position,
      });
    }

    return duplicate;
  }

  async getBlocks(id: string): Promise<Block[]> {
    return this.store.blocks.listByPage(id);
  }

  async createBlock(id: string, block: { type: string; content?: unknown }): Promise<Block> {
    const position = await this.store.blocks.getMaxPosition(id);
    return this.store.blocks.create({
      id: generateId(),
      pageId: id,
      type: block.type as Block['type'],
      content: (block.content ?? {}) as Block['content'],
      position,
    });
  }
}
