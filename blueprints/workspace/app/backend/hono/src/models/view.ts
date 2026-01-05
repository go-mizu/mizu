import { z } from 'zod';

export const ViewTypeSchema = z.enum([
  'table',
  'board',
  'calendar',
  'gallery',
  'list',
  'timeline',
]);

export const FilterOperatorSchema = z.enum([
  'equals',
  'not_equals',
  'contains',
  'not_contains',
  'starts_with',
  'ends_with',
  'is_empty',
  'is_not_empty',
  'greater_than',
  'less_than',
  'greater_than_or_equal',
  'less_than_or_equal',
  'date_is',
  'date_is_before',
  'date_is_after',
  'date_is_on_or_before',
  'date_is_on_or_after',
  'date_is_within',
]);

export const FilterConditionSchema = z.object({
  propertyId: z.string(),
  operator: FilterOperatorSchema,
  value: z.unknown().optional(),
});

export const FilterGroupSchema: z.ZodType<FilterGroup> = z.lazy(() =>
  z.object({
    type: z.enum(['and', 'or']),
    conditions: z.array(
      z.union([FilterConditionSchema, FilterGroupSchema])
    ),
  })
);

export interface FilterGroup {
  type: 'and' | 'or';
  conditions: (FilterCondition | FilterGroup)[];
}

export type FilterCondition = z.infer<typeof FilterConditionSchema>;

export const SortDirectionSchema = z.enum(['ascending', 'descending']);

export const SortSchema = z.object({
  propertyId: z.string(),
  direction: SortDirectionSchema,
});

export const ViewPropertySchema = z.object({
  propertyId: z.string(),
  visible: z.boolean().default(true),
  width: z.number().optional(),
});

export const ViewSchema = z.object({
  id: z.string(),
  databaseId: z.string(),
  name: z.string().min(1).max(255),
  type: ViewTypeSchema.default('table'),
  filter: FilterGroupSchema.optional().nullable(),
  sorts: z.array(SortSchema).optional().nullable(),
  properties: z.array(ViewPropertySchema).optional().nullable(),
  groupBy: z.string().optional().nullable(),
  calendarBy: z.string().optional().nullable(),
  position: z.number(),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreateViewSchema = z.object({
  databaseId: z.string(),
  name: z.string().min(1).max(255),
  type: ViewTypeSchema.optional(),
  filter: FilterGroupSchema.optional().nullable(),
  sorts: z.array(SortSchema).optional().nullable(),
  groupBy: z.string().optional().nullable(),
  calendarBy: z.string().optional().nullable(),
});

export const UpdateViewSchema = z.object({
  name: z.string().min(1).max(255).optional(),
  type: ViewTypeSchema.optional(),
  filter: FilterGroupSchema.optional().nullable(),
  sorts: z.array(SortSchema).optional().nullable(),
  properties: z.array(ViewPropertySchema).optional().nullable(),
  groupBy: z.string().optional().nullable(),
  calendarBy: z.string().optional().nullable(),
});

export const QueryViewSchema = z.object({
  filter: FilterGroupSchema.optional(),
  sorts: z.array(SortSchema).optional(),
  cursor: z.string().optional(),
  limit: z.coerce.number().min(1).max(100).optional(),
});

export type View = z.infer<typeof ViewSchema>;
export type ViewType = z.infer<typeof ViewTypeSchema>;
export type Sort = z.infer<typeof SortSchema>;
export type SortDirection = z.infer<typeof SortDirectionSchema>;
export type ViewProperty = z.infer<typeof ViewPropertySchema>;
export type CreateView = z.infer<typeof CreateViewSchema>;
export type UpdateView = z.infer<typeof UpdateViewSchema>;
export type QueryView = z.infer<typeof QueryViewSchema>;
