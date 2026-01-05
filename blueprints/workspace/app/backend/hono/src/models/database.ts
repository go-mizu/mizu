import { z } from 'zod';

export const PropertyTypeSchema = z.enum([
  'title',
  'rich_text',
  'number',
  'select',
  'multi_select',
  'date',
  'person',
  'files',
  'checkbox',
  'url',
  'email',
  'phone',
  'formula',
  'relation',
  'rollup',
  'status',
  'created_time',
  'created_by',
  'last_edited_time',
  'last_edited_by',
]);

export const SelectOptionSchema = z.object({
  id: z.string(),
  name: z.string(),
  color: z.string().optional(),
});

export const StatusGroupSchema = z.object({
  id: z.string(),
  name: z.string(),
  color: z.string(),
  optionIds: z.array(z.string()),
});

export const PropertyConfigSchema = z.object({
  // Select/Multi-select
  options: z.array(SelectOptionSchema).optional(),
  // Status
  groups: z.array(StatusGroupSchema).optional(),
  // Number
  format: z.string().optional(),
  // Formula
  expression: z.string().optional(),
  // Relation
  databaseId: z.string().optional(),
  // Rollup
  relationPropertyId: z.string().optional(),
  rollupPropertyId: z.string().optional(),
  function: z.string().optional(),
});

export const PropertySchema = z.object({
  id: z.string(),
  name: z.string().min(1).max(255),
  type: PropertyTypeSchema,
  config: PropertyConfigSchema.optional(),
});

export const DatabaseSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  pageId: z.string(),
  title: z.string().default(''),
  icon: z.string().optional().nullable(),
  cover: z.string().optional().nullable(),
  isInline: z.boolean().default(false),
  properties: z.array(PropertySchema).default([]),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const CreateDatabaseSchema = z.object({
  workspaceId: z.string(),
  pageId: z.string().optional(), // Parent page for inline databases; if not provided, creates standalone database page
  title: z.string().optional(),
  icon: z.string().optional(),
  isInline: z.boolean().optional(),
  properties: z.array(PropertySchema).optional(),
});

export const UpdateDatabaseSchema = z.object({
  title: z.string().optional(),
  icon: z.string().optional().nullable(),
  cover: z.string().optional().nullable(),
});

export const AddPropertySchema = z.object({
  name: z.string().min(1).max(255),
  type: PropertyTypeSchema,
  config: PropertyConfigSchema.optional(),
});

export const UpdatePropertySchema = z.object({
  name: z.string().min(1).max(255).optional(),
  config: PropertyConfigSchema.optional(),
});

export type Database = z.infer<typeof DatabaseSchema>;
export type Property = z.infer<typeof PropertySchema>;
export type PropertyType = z.infer<typeof PropertyTypeSchema>;
export type PropertyConfig = z.infer<typeof PropertyConfigSchema>;
export type SelectOption = z.infer<typeof SelectOptionSchema>;
export type StatusGroup = z.infer<typeof StatusGroupSchema>;
export type CreateDatabase = z.infer<typeof CreateDatabaseSchema>;
export type UpdateDatabase = z.infer<typeof UpdateDatabaseSchema>;
export type AddProperty = z.infer<typeof AddPropertySchema>;
export type UpdateProperty = z.infer<typeof UpdatePropertySchema>;

// Default properties for a new database
export function defaultDatabaseProperties(): Property[] {
  return [
    { id: 'title', name: 'Name', type: 'title' },
  ];
}
