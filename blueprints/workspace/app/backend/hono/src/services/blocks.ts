import type { Store } from '../store/types';
import type { Block, BlockTree, CreateBlock, UpdateBlock, MoveBlock, UpdateBlocks } from '../models';
import { generateId } from '../utils/id';
import { nowISO } from '../models/common';

export class BlockService {
  constructor(private store: Store) {}

  async create(input: CreateBlock): Promise<Block> {
    let position = input.position;

    if (position === undefined) {
      if (input.afterId) {
        const afterBlock = await this.store.blocks.getById(input.afterId);
        position = afterBlock ? afterBlock.position + 1 : 1;
      } else {
        position = await this.store.blocks.getMaxPosition(input.pageId, input.parentId);
      }
    }

    return this.store.blocks.create({
      id: generateId(),
      pageId: input.pageId,
      parentId: input.parentId,
      type: input.type,
      content: input.content ?? {},
      position,
    });
  }

  async getById(id: string): Promise<Block | null> {
    return this.store.blocks.getById(id);
  }

  async listByPage(pageId: string): Promise<Block[]> {
    return this.store.blocks.listByPage(pageId);
  }

  async getBlockTree(pageId: string): Promise<BlockTree[]> {
    const blocks = await this.store.blocks.listByPage(pageId);
    return this.buildTree(blocks);
  }

  async update(id: string, data: UpdateBlock): Promise<Block> {
    return this.store.blocks.update(id, data);
  }

  async move(id: string, data: MoveBlock): Promise<Block> {
    const block = await this.store.blocks.getById(id);
    if (!block) {
      throw new Error('Block not found');
    }

    const updates: Partial<Block> = {};

    if (data.parentId !== undefined) {
      updates.parentId = data.parentId ?? undefined;
    }

    if (data.position !== undefined) {
      updates.position = data.position;
    } else if (data.afterId !== undefined) {
      if (data.afterId === null) {
        updates.position = 0;
      } else {
        const afterBlock = await this.store.blocks.getById(data.afterId);
        updates.position = afterBlock ? afterBlock.position + 0.5 : 1;
      }
    }

    return this.store.blocks.update(id, updates);
  }

  async delete(id: string): Promise<void> {
    // Delete children first
    const children = await this.store.blocks.listByParent(id);
    for (const child of children) {
      await this.delete(child.id);
    }
    await this.store.blocks.delete(id);
  }

  async updateBlocks(pageId: string, data: UpdateBlocks): Promise<Block[]> {
    const now = nowISO();
    const blocks: Block[] = [];

    const processBlock = (
      blockData: UpdateBlocks['blocks'][number],
      parentId: string | undefined,
      position: number
    ): Block => {
      const block: Block = {
        id: blockData.id ?? generateId(),
        pageId,
        parentId,
        type: blockData.type,
        content: blockData.content ?? {},
        position,
        createdAt: now,
        updatedAt: now,
      };
      blocks.push(block);

      if (blockData.children) {
        blockData.children.forEach((child, idx) => {
          processBlock(child as UpdateBlocks['blocks'][number], block.id, idx);
        });
      }

      return block;
    };

    data.blocks.forEach((block, idx) => {
      processBlock(block, undefined, idx);
    });

    await this.store.blocks.batchUpsert(pageId, blocks);

    return blocks;
  }

  private buildTree(blocks: Block[]): BlockTree[] {
    const blockMap = new Map<string, BlockTree>();
    const roots: BlockTree[] = [];

    // Initialize all blocks with empty children
    for (const block of blocks) {
      blockMap.set(block.id, { ...block, children: [] });
    }

    // Build tree
    for (const block of blocks) {
      const node = blockMap.get(block.id)!;
      if (block.parentId) {
        const parent = blockMap.get(block.parentId);
        if (parent) {
          parent.children.push(node);
        } else {
          roots.push(node);
        }
      } else {
        roots.push(node);
      }
    }

    // Sort children by position
    const sortChildren = (nodes: BlockTree[]) => {
      nodes.sort((a, b) => a.position - b.position);
      for (const node of nodes) {
        sortChildren(node.children);
      }
    };

    sortChildren(roots);

    return roots;
  }
}
