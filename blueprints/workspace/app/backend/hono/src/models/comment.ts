import { z } from 'zod';
import { RichTextSchema } from './common';

export const CommentTargetTypeSchema = z.enum(['page', 'block', 'database_row']);

export const CommentSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  targetType: CommentTargetTypeSchema,
  targetId: z.string(),
  parentId: z.string().optional().nullable(),
  content: z.array(RichTextSchema),
  authorId: z.string(),
  isResolved: z.boolean().default(false),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreateCommentSchema = z.object({
  workspaceId: z.string(),
  targetType: CommentTargetTypeSchema,
  targetId: z.string(),
  parentId: z.string().optional(),
  content: z.array(RichTextSchema),
});

export const UpdateCommentSchema = z.object({
  content: z.array(RichTextSchema).optional(),
});

export type Comment = z.infer<typeof CommentSchema>;
export type CommentTargetType = z.infer<typeof CommentTargetTypeSchema>;
export type CreateComment = z.infer<typeof CreateCommentSchema>;
export type UpdateComment = z.infer<typeof UpdateCommentSchema>;

// Comment with author info
export interface CommentWithAuthor extends Comment {
  author: {
    id: string;
    name: string;
    avatarUrl?: string | null;
  };
  replies?: CommentWithAuthor[];
}
