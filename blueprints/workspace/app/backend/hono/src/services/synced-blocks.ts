import type { Store } from '../store/types';
import type { SyncedBlock, CreateSyncedBlock, UpdateSyncedBlock } from '../models';
import { generateId } from '../utils/id';

export class SyncedBlockService {
  constructor(private store: Store) {}

  async create(input: CreateSyncedBlock, userId: string): Promise<SyncedBlock> {
    return this.store.syncedBlocks.create({
      id: generateId(),
      workspaceId: input.workspaceId,
      sourceBlockId: input.sourceBlockId,
      content: input.content,
      createdBy: userId,
    });
  }

  async getById(id: string): Promise<SyncedBlock | null> {
    return this.store.syncedBlocks.getById(id);
  }

  async listByWorkspace(workspaceId: string): Promise<SyncedBlock[]> {
    return this.store.syncedBlocks.listByWorkspace(workspaceId);
  }

  async listByPage(pageId: string): Promise<SyncedBlock[]> {
    return this.store.syncedBlocks.listByPage(pageId);
  }

  async update(id: string, data: UpdateSyncedBlock): Promise<SyncedBlock> {
    return this.store.syncedBlocks.update(id, data);
  }

  async delete(id: string): Promise<void> {
    await this.store.syncedBlocks.delete(id);
  }

  async sync(id: string, content: unknown): Promise<SyncedBlock> {
    // Update the synced block content
    // This would propagate to all instances of this synced block
    return this.store.syncedBlocks.update(id, { content });
  }
}
