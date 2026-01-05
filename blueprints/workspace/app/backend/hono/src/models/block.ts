import { z } from 'zod';
import { RichTextSchema } from './common';

export const BlockTypeSchema = z.enum([
  'paragraph',
  'heading_1',
  'heading_2',
  'heading_3',
  'bulleted_list',
  'numbered_list',
  'to_do',
  'toggle',
  'quote',
  'callout',
  'divider',
  'code',
  'image',
  'video',
  'file',
  'bookmark',
  'table',
  'table_row',
  'column_list',
  'column',
  'synced_block',
  'embed',
]);

export const BlockContentSchema = z.object({
  richText: z.array(RichTextSchema).optional(),
  checked: z.boolean().optional(),
  language: z.string().optional(),
  url: z.string().optional(),
  caption: z.array(RichTextSchema).optional(),
  icon: z.string().optional(),
  color: z.string().optional(),
  // Table specific
  tableWidth: z.number().optional(),
  hasColumnHeader: z.boolean().optional(),
  hasRowHeader: z.boolean().optional(),
  cells: z.array(z.array(RichTextSchema)).optional(),
  // Synced block
  syncedBlockId: z.string().optional(),
  // Embed
  embedUrl: z.string().optional(),
  // File/Image specific
  fileId: z.string().optional(),
  fileName: z.string().optional(),
  fileSize: z.number().optional(),
  mimeType: z.string().optional(),
  width: z.number().optional(),
  height: z.number().optional(),
});

export const BlockSchema = z.object({
  id: z.string(),
  pageId: z.string(),
  parentId: z.string().optional().nullable(),
  type: BlockTypeSchema,
  content: BlockContentSchema.default({}),
  position: z.number(),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreateBlockSchema = z.object({
  pageId: z.string(),
  parentId: z.string().optional(),
  type: BlockTypeSchema,
  content: BlockContentSchema.optional(),
  position: z.number().optional(),
  afterId: z.string().optional(),
});

export const UpdateBlockSchema = z.object({
  type: BlockTypeSchema.optional(),
  content: BlockContentSchema.optional(),
});

export const MoveBlockSchema = z.object({
  parentId: z.string().optional().nullable(),
  afterId: z.string().optional().nullable(),
  position: z.number().optional(),
});

export const UpdateBlocksSchema = z.object({
  blocks: z.array(
    z.object({
      id: z.string().optional(),
      type: BlockTypeSchema,
      content: BlockContentSchema.optional(),
      parentId: z.string().optional().nullable(),
      children: z.array(z.lazy(() => z.any())).optional(),
    })
  ),
});

export type Block = z.infer<typeof BlockSchema>;
export type BlockType = z.infer<typeof BlockTypeSchema>;
export type BlockContent = z.infer<typeof BlockContentSchema>;
export type CreateBlock = z.infer<typeof CreateBlockSchema>;
export type UpdateBlock = z.infer<typeof UpdateBlockSchema>;
export type MoveBlock = z.infer<typeof MoveBlockSchema>;
export type UpdateBlocks = z.infer<typeof UpdateBlocksSchema>;

// Block with children (nested structure)
export interface BlockTree extends Block {
  children: BlockTree[];
}
