import { z } from 'zod';

// Rich text types
export const RichTextAnnotationsSchema = z.object({
  bold: z.boolean().optional(),
  italic: z.boolean().optional(),
  strikethrough: z.boolean().optional(),
  underline: z.boolean().optional(),
  code: z.boolean().optional(),
  color: z.string().optional(),
});

export const RichTextSchema = z.object({
  type: z.enum(['text', 'mention', 'equation']).default('text'),
  text: z
    .object({
      content: z.string(),
      link: z.string().optional(),
    })
    .optional(),
  mention: z
    .object({
      type: z.enum(['user', 'page', 'database', 'date']),
      id: z.string().optional(),
      date: z.string().optional(),
    })
    .optional(),
  equation: z
    .object({
      expression: z.string(),
    })
    .optional(),
  annotations: RichTextAnnotationsSchema.optional(),
  plainText: z.string().optional(),
});

export type RichText = z.infer<typeof RichTextSchema>;
export type RichTextAnnotations = z.infer<typeof RichTextAnnotationsSchema>;

// Pagination
export const PaginationSchema = z.object({
  cursor: z.string().optional(),
  limit: z.coerce.number().min(1).max(100).default(50),
});

export type Pagination = z.infer<typeof PaginationSchema>;

export interface PaginatedResult<T> {
  items: T[];
  nextCursor?: string;
  hasMore: boolean;
}

// Timestamps
export function nowISO(): string {
  return new Date().toISOString();
}

// ID validation
export const IdSchema = z.string().min(1).max(64);
