import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { ulid } from 'ulid';
import { z } from 'zod';
import type { Env, Variables } from '../types/index.js';
import { CreateFieldSchema, UpdateFieldSchema, CreateSelectOptionSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { authRequired } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';

const fields = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

// All field routes require authentication
fields.use('*', authRequired);

/**
 * POST /fields - Create field in table
 */
fields.post('/', zValidator('json', CreateFieldSchema.extend({ table_id: z.string() })), async (c) => {
  const user = c.get('user');
  const db = c.get('db');
  const { table_id, ...input } = c.req.valid('json');

  // Verify table exists
  const table = await db.getTable(table_id);
  if (!table) {
    throw ApiError.notFound('Table not found');
  }

  // Get next position
  const maxPos = await db.getMaxFieldPosition(table_id);
  const position = maxPos + 1;

  const field = await db.createField({
    id: ulid(),
    table_id,
    ...input,
    position,
    created_by: user.id,
  });

  // If select type, create default options
  if ((input.type === 'single_select' || input.type === 'multi_select') && input.options) {
    const optionsInput = input.options as { options?: Array<{ name: string; color?: string }> };
    if (optionsInput.options) {
      for (let i = 0; i < optionsInput.options.length; i++) {
        const opt = optionsInput.options[i];
        await db.createSelectOption({
          id: ulid(),
          field_id: field.id,
          name: opt.name,
          color: opt.color,
          position: i,
        });
      }
    }
  }

  // Get select options if applicable
  let selectOptions;
  if (input.type === 'single_select' || input.type === 'multi_select') {
    selectOptions = await db.getSelectOptionsByField(field.id);
  }

  return c.json({ field: { ...field, select_options: selectOptions } }, 201);
});

/**
 * GET /fields/:id - Get field
 */
fields.get('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const field = await db.getField(id);
  if (!field) {
    throw ApiError.notFound('Field not found');
  }

  // Get select options if applicable
  let selectOptions;
  if (field.type === 'single_select' || field.type === 'multi_select') {
    selectOptions = await db.getSelectOptionsByField(id);
  }

  return c.json({ field: { ...field, select_options: selectOptions } });
});

/**
 * PATCH /fields/:id - Update field
 */
fields.patch('/:id', zValidator('json', UpdateFieldSchema), async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();
  const input = c.req.valid('json');

  const existingField = await db.getField(id);
  if (!existingField) {
    throw ApiError.notFound('Field not found');
  }

  // Prevent modifying primary field type
  if (existingField.is_primary && input.options) {
    // Primary field options can still be modified
  }

  const field = await db.updateField(id, input);

  return c.json({ field });
});

/**
 * DELETE /fields/:id - Delete field
 */
fields.delete('/:id', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const field = await db.getField(id);
  if (!field) {
    throw ApiError.notFound('Field not found');
  }

  // Prevent deleting primary field
  if (field.is_primary) {
    throw ApiError.badRequest('Cannot delete primary field');
  }

  // Delete all cell values for this field
  await db.deleteCellValuesByField(id);

  await db.deleteField(id);
  return c.json({ message: 'Field deleted' });
});

/**
 * POST /fields/:tableId/reorder - Reorder fields
 */
fields.post('/:tableId/reorder', zValidator('json', z.object({ field_ids: z.array(z.string()) })), async (c) => {
  const db = c.get('db');
  const { tableId } = c.req.param();
  const { field_ids } = c.req.valid('json');

  await db.reorderFields(tableId, field_ids);
  return c.json({ message: 'Fields reordered' });
});

// ============================================================================
// Select Options
// ============================================================================

/**
 * GET /fields/:id/options - List select options
 */
fields.get('/:id/options', async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();

  const field = await db.getField(id);
  if (!field) {
    throw ApiError.notFound('Field not found');
  }

  if (field.type !== 'single_select' && field.type !== 'multi_select') {
    throw ApiError.badRequest('Field is not a select type');
  }

  const options = await db.getSelectOptionsByField(id);
  return c.json({ options });
});

/**
 * POST /fields/:id/options - Create select option
 */
fields.post('/:id/options', zValidator('json', CreateSelectOptionSchema), async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();
  const input = c.req.valid('json');

  const field = await db.getField(id);
  if (!field) {
    throw ApiError.notFound('Field not found');
  }

  if (field.type !== 'single_select' && field.type !== 'multi_select') {
    throw ApiError.badRequest('Field is not a select type');
  }

  // Get next position
  const existingOptions = await db.getSelectOptionsByField(id);
  const position = existingOptions.length;

  // Default colors
  const colors = ['#CFDFFF', '#D0F0FD', '#C2F5E9', '#D1F7C4', '#FEE2A8', '#FFD1E5', '#EDE2FE', '#E0E0E0'];
  const color = input.color || colors[position % colors.length];

  const option = await db.createSelectOption({
    id: ulid(),
    field_id: id,
    name: input.name,
    color,
    position,
  });

  return c.json({ option }, 201);
});

/**
 * PATCH /fields/:fieldId/options/:optionId - Update select option
 */
fields.patch('/:fieldId/options/:optionId', zValidator('json', z.object({ name: z.string().optional(), color: z.string().optional() })), async (c) => {
  const db = c.get('db');
  const { optionId } = c.req.param();
  const { name, color } = c.req.valid('json');

  // Get current option to merge values
  const options = await db.getSelectOptionsByField(c.req.param('fieldId'));
  const currentOption = options.find(o => o.id === optionId);
  if (!currentOption) {
    throw ApiError.notFound('Option not found');
  }

  const option = await db.updateSelectOption(
    optionId,
    name ?? currentOption.name,
    color ?? currentOption.color
  );

  return c.json({ option });
});

/**
 * DELETE /fields/:fieldId/options/:optionId - Delete select option
 */
fields.delete('/:fieldId/options/:optionId', async (c) => {
  const db = c.get('db');
  const { optionId } = c.req.param();

  await db.deleteSelectOption(optionId);
  return c.json({ message: 'Option deleted' });
});

/**
 * POST /fields/:id/options/reorder - Reorder select options
 */
fields.post('/:id/options/reorder', zValidator('json', z.object({ option_ids: z.array(z.string()) })), async (c) => {
  const db = c.get('db');
  const { id } = c.req.param();
  const { option_ids } = c.req.valid('json');

  await db.reorderSelectOptions(id, option_ids);
  return c.json({ message: 'Options reordered' });
});

export { fields };
