import { z } from 'zod';

export const ParentTypeSchema = z.enum(['workspace', 'page', 'database']);

export const PageSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  parentId: z.string().optional().nullable(),
  parentType: ParentTypeSchema.default('workspace'),
  databaseId: z.string().optional().nullable(),
  rowPosition: z.number().optional().nullable(),
  title: z.string().default(''),
  icon: z.string().optional().nullable(),
  cover: z.string().optional().nullable(),
  coverY: z.number().min(0).max(1).default(0.5),
  properties: z.record(z.unknown()).default({}),
  isTemplate: z.boolean().default(false),
  isArchived: z.boolean().default(false),
  createdBy: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreatePageSchema = z.object({
  workspaceId: z.string(),
  parentId: z.string().optional(),
  parentType: ParentTypeSchema.optional(),
  databaseId: z.string().optional(),
  title: z.string().optional(),
  icon: z.string().optional(),
  cover: z.string().optional(),
  isTemplate: z.boolean().optional(),
  properties: z.record(z.unknown()).optional(),
});

export const UpdatePageSchema = z.object({
  title: z.string().optional(),
  icon: z.string().optional().nullable(),
  cover: z.string().optional().nullable(),
  coverY: z.number().min(0).max(1).optional(),
  properties: z.record(z.unknown()).optional(),
  isArchived: z.boolean().optional(),
});

export const MovePageSchema = z.object({
  parentId: z.string().optional().nullable(),
  parentType: ParentTypeSchema.optional(),
  position: z.number().optional(),
});

export type Page = z.infer<typeof PageSchema>;
export type ParentType = z.infer<typeof ParentTypeSchema>;
export type CreatePage = z.infer<typeof CreatePageSchema>;
export type UpdatePage = z.infer<typeof UpdatePageSchema>;
export type MovePage = z.infer<typeof MovePageSchema>;

// Page with hierarchy info
export interface PageWithHierarchy extends Page {
  breadcrumb: { id: string; title: string; icon?: string | null }[];
  children?: Page[];
}
