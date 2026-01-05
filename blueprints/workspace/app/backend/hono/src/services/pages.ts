import type { Store } from '../store/types';
import type { Page, CreatePage, UpdatePage, PageWithHierarchy } from '../models';
import { generateId } from '../utils/id';

export class PageService {
  constructor(private store: Store) {}

  async create(input: CreatePage, userId: string): Promise<Page> {
    return this.store.pages.create({
      id: generateId(),
      workspaceId: input.workspaceId,
      parentId: input.parentId,
      parentType: input.parentType ?? 'workspace',
      databaseId: input.databaseId,
      title: input.title ?? '',
      icon: input.icon,
      cover: input.cover,
      coverY: 0.5,
      properties: input.properties ?? {},
      isTemplate: input.isTemplate ?? false,
      isArchived: false,
      createdBy: userId,
    });
  }

  async getById(id: string): Promise<Page | null> {
    return this.store.pages.getById(id);
  }

  async getWithHierarchy(id: string): Promise<PageWithHierarchy | null> {
    const page = await this.store.pages.getById(id);
    if (!page) return null;

    const breadcrumb = await this.store.pages.getHierarchy(id);
    const children = await this.store.pages.listByParent(id, 'page');

    return { ...page, breadcrumb, children };
  }

  async listByWorkspace(
    workspaceId: string,
    options?: { parentId?: string | null; includeArchived?: boolean }
  ): Promise<Page[]> {
    return this.store.pages.listByWorkspace(workspaceId, options);
  }

  async listByParent(parentId: string): Promise<Page[]> {
    return this.store.pages.listByParent(parentId, 'page');
  }

  async update(id: string, data: UpdatePage): Promise<Page> {
    return this.store.pages.update(id, data);
  }

  async archive(id: string): Promise<Page> {
    return this.store.pages.update(id, { isArchived: true });
  }

  async restore(id: string): Promise<Page> {
    return this.store.pages.update(id, { isArchived: false });
  }

  async delete(id: string): Promise<void> {
    // Delete blocks first
    await this.store.blocks.deleteByPage(id);
    // Delete page
    await this.store.pages.delete(id);
  }

  async duplicate(id: string, userId: string): Promise<Page> {
    const original = await this.store.pages.getById(id);
    if (!original) {
      throw new Error('Page not found');
    }

    // Create duplicate page
    const duplicate = await this.store.pages.create({
      id: generateId(),
      workspaceId: original.workspaceId,
      parentId: original.parentId,
      parentType: original.parentType,
      title: `${original.title} (copy)`,
      icon: original.icon,
      cover: original.cover,
      coverY: original.coverY,
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

  async getHierarchy(id: string): Promise<{ id: string; title: string; icon?: string | null }[]> {
    return this.store.pages.getHierarchy(id);
  }
}
