import { z } from 'zod';

// Rows are implemented as pages with databaseId set
// This file provides the row-specific schemas

export const RowSchema = z.object({
  id: z.string(),
  databaseId: z.string(),
  workspaceId: z.string(),
  title: z.string().default(''),
  icon: z.string().optional().nullable(),
  properties: z.record(z.unknown()).default({}),
  rowPosition: z.number(),
  createdBy: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreateRowSchema = z.object({
  databaseId: z.string(),
  title: z.string().optional(),
  icon: z.string().optional(),
  properties: z.record(z.unknown()).optional(),
  afterId: z.string().optional(),
});

export const UpdateRowSchema = z.object({
  title: z.string().optional(),
  icon: z.string().optional().nullable(),
  properties: z.record(z.unknown()).optional(),
});

export const MoveRowSchema = z.object({
  afterId: z.string().optional().nullable(),
  position: z.number().optional(),
});

export type Row = z.infer<typeof RowSchema>;
export type CreateRow = z.infer<typeof CreateRowSchema>;
export type UpdateRow = z.infer<typeof UpdateRowSchema>;
export type MoveRow = z.infer<typeof MoveRowSchema>;
