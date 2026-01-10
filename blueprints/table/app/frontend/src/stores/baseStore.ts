import { create } from 'zustand';
import type { Workspace, Base, Table, Field, View, TableRecord, SelectOption, Comment, Filter, Sort, CellValue } from '../types';
import { workspacesApi, basesApi, tablesApi, fieldsApi, recordsApi, viewsApi, commentsApi } from '../api/client';

interface BaseState {
  // Data
  workspaces: Workspace[];
  currentWorkspace: Workspace | null;
  bases: Base[];
  currentBase: Base | null;
  tables: Table[];
  currentTable: Table | null;
  fields: Field[];
  views: View[];
  currentView: View | null;
  records: TableRecord[];
  comments: Comment[];

  // Filter, Sort, Group
  filters: Filter[];
  filterConjunction: 'and' | 'or';
  sorts: Sort[];
  groupBy: string | null;

  // Pagination
  nextCursor: string | null;
  hasMore: boolean;

  // Loading states
  isLoading: boolean;
  isLoadingRecords: boolean;
  error: string | null;

  // Actions - Workspaces
  loadWorkspaces: () => Promise<void>;
  selectWorkspace: (id: string) => Promise<void>;
  createWorkspace: (name: string, slug: string) => Promise<Workspace>;

  // Actions - Bases
  loadBases: (workspaceId: string) => Promise<void>;
  selectBase: (id: string) => Promise<void>;
  createBase: (name: string, color?: string) => Promise<Base>;
  updateBase: (id: string, data: Partial<Base>) => Promise<void>;
  deleteBase: (id: string) => Promise<void>;

  // Actions - Tables
  selectTable: (id: string) => Promise<void>;
  createTable: (name: string) => Promise<Table>;
  updateTable: (id: string, data: Partial<Table>) => Promise<void>;
  deleteTable: (id: string) => Promise<void>;

  // Actions - Fields
  loadFields: (tableId: string) => Promise<void>;
  createField: (name: string, type: string, options?: Record<string, unknown>) => Promise<Field>;
  updateField: (id: string, data: Partial<Field>) => Promise<void>;
  deleteField: (id: string) => Promise<void>;
  reorderFields: (fromIndex: number, toIndex: number) => void;

  // Actions - Select Options
  createSelectOption: (fieldId: string, name: string, color?: string) => Promise<SelectOption>;
  updateSelectOption: (fieldId: string, optionId: string, name: string, color: string) => Promise<void>;
  deleteSelectOption: (fieldId: string, optionId: string) => Promise<void>;

  // Actions - Views
  selectView: (id: string) => Promise<void>;
  createView: (name: string, type: string) => Promise<View>;
  updateView: (id: string, data: Partial<View>) => Promise<void>;
  deleteView: (id: string) => Promise<void>;

  // Actions - Records
  loadRecords: (tableId: string, reset?: boolean) => Promise<void>;
  loadMoreRecords: () => Promise<void>;
  createRecord: (fields?: Record<string, unknown>) => Promise<TableRecord>;
  updateRecord: (id: string, fields: Record<string, unknown>) => Promise<void>;
  deleteRecord: (id: string) => Promise<void>;
  updateCellValue: (recordId: string, fieldId: string, value: unknown) => Promise<void>;

  // Actions - Comments
  fetchComments: (recordId: string) => Promise<void>;
  createComment: (recordId: string, content: string) => Promise<Comment>;
  deleteComment: (commentId: string) => Promise<void>;

  // Actions - Filter, Sort, Group
  setFilters: (filters: Filter[], conjunction?: 'and' | 'or') => void;
  setSorts: (sorts: Sort[]) => void;
  setGroupBy: (fieldId: string | null) => void;
  getFilteredRecords: () => TableRecord[];
  getSortedRecords: () => TableRecord[];
  getGroupedRecords: () => { group: string; records: TableRecord[] }[];

  // Utilities
  clearError: () => void;
}

export const useBaseStore = create<BaseState>((set, get) => ({
  // Initial state
  workspaces: [],
  currentWorkspace: null,
  bases: [],
  currentBase: null,
  tables: [],
  currentTable: null,
  fields: [],
  views: [],
  currentView: null,
  records: [],
  comments: [],
  filters: [],
  filterConjunction: 'and',
  sorts: [],
  groupBy: null,
  nextCursor: null,
  hasMore: false,
  isLoading: false,
  isLoadingRecords: false,
  error: null,

  // Workspaces
  loadWorkspaces: async () => {
    set({ isLoading: true });
    try {
      const { workspaces } = await workspacesApi.list();
      set({ workspaces, isLoading: false });
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  selectWorkspace: async (id: string) => {
    const workspace = get().workspaces.find(w => w.id === id);
    if (!workspace) return;

    set({ currentWorkspace: workspace, currentBase: null, tables: [], currentTable: null });
    await get().loadBases(id);
  },

  createWorkspace: async (name: string, slug: string) => {
    const { workspace } = await workspacesApi.create(name, slug);
    set({ workspaces: [...get().workspaces, workspace] });
    return workspace;
  },

  // Bases
  loadBases: async (workspaceId: string) => {
    set({ isLoading: true });
    try {
      const { bases } = await workspacesApi.getBases(workspaceId);
      set({ bases, isLoading: false });
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  selectBase: async (id: string) => {
    set({ isLoading: true });
    try {
      const { base, tables } = await basesApi.get(id);
      set({ currentBase: base, tables, isLoading: false });

      // Auto-select first table
      if (tables.length > 0) {
        await get().selectTable(tables[0].id);
      }
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  createBase: async (name: string, color?: string) => {
    const workspace = get().currentWorkspace;
    if (!workspace) throw new Error('No workspace selected');

    const { base } = await basesApi.create(workspace.id, name, color);
    set({ bases: [...get().bases, base] });
    return base;
  },

  updateBase: async (id: string, data: Partial<Base>) => {
    const { base } = await basesApi.update(id, data);
    set({
      bases: get().bases.map(b => b.id === id ? base : b),
      currentBase: get().currentBase?.id === id ? base : get().currentBase,
    });
  },

  deleteBase: async (id: string) => {
    await basesApi.delete(id);
    set({
      bases: get().bases.filter(b => b.id !== id),
      currentBase: get().currentBase?.id === id ? null : get().currentBase,
    });
  },

  // Tables
  selectTable: async (id: string) => {
    set({ isLoading: true });
    try {
      const { table } = await tablesApi.get(id);
      set({
        currentTable: table,
        fields: table.fields || [],
        views: table.views || [],
        isLoading: false,
      });

      // Auto-select default view or first view
      const views = table.views || [];
      const defaultView = views.find(v => v.is_default) || views[0];
      if (defaultView) {
        set({ currentView: defaultView });
      }

      // Load records
      await get().loadRecords(id, true);
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  createTable: async (name: string) => {
    const base = get().currentBase;
    if (!base) throw new Error('No base selected');

    const { table } = await tablesApi.create(base.id, name);
    set({ tables: [...get().tables, table] });
    return table;
  },

  updateTable: async (id: string, data: Partial<Table>) => {
    const { table } = await tablesApi.update(id, data);
    set({
      tables: get().tables.map(t => t.id === id ? table : t),
      currentTable: get().currentTable?.id === id ? table : get().currentTable,
    });
  },

  deleteTable: async (id: string) => {
    await tablesApi.delete(id);
    set({
      tables: get().tables.filter(t => t.id !== id),
      currentTable: get().currentTable?.id === id ? null : get().currentTable,
    });
  },

  // Fields
  loadFields: async (tableId: string) => {
    const { fields } = await tablesApi.getFields(tableId);
    set({ fields });
  },

  createField: async (name: string, type: string, options?: Record<string, unknown>) => {
    const table = get().currentTable;
    if (!table) throw new Error('No table selected');

    const { field } = await fieldsApi.create(table.id, name, type, options);
    set({ fields: [...get().fields, field] });
    return field;
  },

  updateField: async (id: string, data: Partial<Field>) => {
    const { field } = await fieldsApi.update(id, data);
    set({ fields: get().fields.map(f => f.id === id ? field : f) });
  },

  deleteField: async (id: string) => {
    await fieldsApi.delete(id);
    set({ fields: get().fields.filter(f => f.id !== id) });
  },

  reorderFields: (fromIndex: number, toIndex: number) => {
    const fields = [...get().fields];
    const [removed] = fields.splice(fromIndex, 1);
    fields.splice(toIndex, 0, removed);
    set({ fields });
  },

  // Select Options
  createSelectOption: async (fieldId: string, name: string, color?: string) => {
    const { option } = await fieldsApi.createOption(fieldId, name, color);

    // Update the field's select_options
    set({
      fields: get().fields.map(f => {
        if (f.id === fieldId) {
          return {
            ...f,
            select_options: [...(f.select_options || []), option],
          };
        }
        return f;
      }),
    });

    return option;
  },

  updateSelectOption: async (fieldId: string, optionId: string, name: string, color: string) => {
    const { option } = await fieldsApi.updateOption(fieldId, optionId, name, color);

    set({
      fields: get().fields.map(f => {
        if (f.id === fieldId) {
          return {
            ...f,
            select_options: (f.select_options || []).map(o =>
              o.id === optionId ? option : o
            ),
          };
        }
        return f;
      }),
    });
  },

  deleteSelectOption: async (fieldId: string, optionId: string) => {
    await fieldsApi.deleteOption(fieldId, optionId);

    set({
      fields: get().fields.map(f => {
        if (f.id === fieldId) {
          return {
            ...f,
            select_options: (f.select_options || []).filter(o => o.id !== optionId),
          };
        }
        return f;
      }),
    });
  },

  // Views
  selectView: async (id: string) => {
    const view = get().views.find(v => v.id === id);
    if (view) {
      set({ currentView: view });
    }
  },

  createView: async (name: string, type: string) => {
    const table = get().currentTable;
    if (!table) throw new Error('No table selected');

    const { view } = await viewsApi.create(table.id, name, type);
    set({ views: [...get().views, view] });
    return view;
  },

  updateView: async (id: string, data: Partial<View>) => {
    const { view } = await viewsApi.update(id, data);
    set({
      views: get().views.map(v => v.id === id ? view : v),
      currentView: get().currentView?.id === id ? view : get().currentView,
    });
  },

  deleteView: async (id: string) => {
    await viewsApi.delete(id);
    const views = get().views.filter(v => v.id !== id);
    set({
      views,
      currentView: get().currentView?.id === id ? views[0] : get().currentView,
    });
  },

  // Records
  loadRecords: async (tableId: string, reset = false) => {
    set({ isLoadingRecords: true });
    try {
      const cursor = reset ? undefined : get().nextCursor || undefined;
      const { records, next_cursor, has_more } = await recordsApi.list(tableId, cursor);

      set({
        records: reset ? records : [...get().records, ...records],
        nextCursor: next_cursor || null,
        hasMore: has_more,
        isLoadingRecords: false,
      });
    } catch (err) {
      set({ error: (err as Error).message, isLoadingRecords: false });
    }
  },

  loadMoreRecords: async () => {
    const table = get().currentTable;
    if (!table || !get().hasMore) return;
    await get().loadRecords(table.id);
  },

  createRecord: async (fields?: Record<string, unknown>) => {
    const table = get().currentTable;
    if (!table) throw new Error('No table selected');

    const { record } = await recordsApi.create(table.id, fields);
    set({ records: [...get().records, record] });
    return record;
  },

  updateRecord: async (id: string, fields: Record<string, unknown>) => {
    const { record } = await recordsApi.update(id, fields);
    set({ records: get().records.map(r => r.id === id ? record : r) });
  },

  deleteRecord: async (id: string) => {
    await recordsApi.delete(id);
    set({ records: get().records.filter(r => r.id !== id) });
  },

  updateCellValue: async (recordId: string, fieldId: string, value: unknown) => {
    await get().updateRecord(recordId, { [fieldId]: value });
  },

  // Comments
  fetchComments: async (recordId: string) => {
    try {
      const { comments } = await commentsApi.list(recordId);
      set({ comments });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },

  createComment: async (recordId: string, content: string) => {
    const { comment } = await commentsApi.create(recordId, content);
    set({ comments: [...get().comments, comment] });
    return comment;
  },

  deleteComment: async (commentId: string) => {
    await commentsApi.delete(commentId);
    set({ comments: get().comments.filter(c => c.id !== commentId) });
  },

  // Filter, Sort, Group
  setFilters: (filters: Filter[], conjunction: 'and' | 'or' = 'and') => {
    set({ filters, filterConjunction: conjunction });
  },

  setSorts: (sorts: Sort[]) => {
    set({ sorts });
  },

  setGroupBy: (fieldId: string | null) => {
    set({ groupBy: fieldId });
  },

  getFilteredRecords: () => {
    const { records, filters, filterConjunction, fields } = get();
    if (filters.length === 0) return records;

    return records.filter(record => {
      const results = filters.map(filter => {
        const field = fields.find(f => f.id === filter.field_id);
        if (!field) return true;

        const value = record.values[filter.field_id] as CellValue;
        return evaluateFilter(value, filter.operator, filter.value as CellValue, field);
      });

      return filterConjunction === 'and'
        ? results.every(Boolean)
        : results.some(Boolean);
    });
  },

  getSortedRecords: () => {
    const { sorts, fields } = get();
    const filtered = get().getFilteredRecords();
    if (sorts.length === 0) return filtered;

    return [...filtered].sort((a, b) => {
      for (const sort of sorts) {
        const field = fields.find(f => f.id === sort.field_id);
        if (!field) continue;

        const aVal = a.values[sort.field_id];
        const bVal = b.values[sort.field_id];
        const comparison = compareValues(aVal, bVal, field.type);

        if (comparison !== 0) {
          return sort.direction === 'asc' ? comparison : -comparison;
        }
      }
      return 0;
    });
  },

  getGroupedRecords: () => {
    const { groupBy, fields } = get();
    const sorted = get().getSortedRecords();

    if (!groupBy) {
      return [{ group: '', records: sorted }];
    }

    const field = fields.find(f => f.id === groupBy);
    if (!field) {
      return [{ group: '', records: sorted }];
    }

    const groups = new Map<string, TableRecord[]>();
    const uncategorized: TableRecord[] = [];

    sorted.forEach(record => {
      const value = record.values[groupBy];
      if (value === null || value === undefined) {
        uncategorized.push(record);
      } else {
        const groupKey = String(value);
        if (!groups.has(groupKey)) {
          groups.set(groupKey, []);
        }
        groups.get(groupKey)!.push(record);
      }
    });

    const result: { group: string; records: TableRecord[] }[] = [];

    // Add grouped records
    groups.forEach((records, group) => {
      result.push({ group, records });
    });

    // Add uncategorized at the end
    if (uncategorized.length > 0) {
      result.push({ group: '(Empty)', records: uncategorized });
    }

    return result;
  },

  // Utilities
  clearError: () => set({ error: null }),
}));

// Helper function to evaluate a filter condition
function evaluateFilter(value: CellValue, operator: string, filterValue: CellValue, _field: Field): boolean {
  const isEmpty = value === null || value === undefined || value === '';
  const isArray = Array.isArray(value);

  switch (operator) {
    case 'is_empty':
      return isEmpty || (isArray && value.length === 0);
    case 'is_not_empty':
      return !isEmpty && (!isArray || value.length > 0);
    case 'is_checked':
      return value === true;
    case 'is_not_checked':
      return value !== true;
    case 'contains':
      return String(value || '').toLowerCase().includes(String(filterValue || '').toLowerCase());
    case 'not_contains':
      return !String(value || '').toLowerCase().includes(String(filterValue || '').toLowerCase());
    case 'is':
      if (typeof value === 'string' && typeof filterValue === 'string') {
        return value.toLowerCase() === filterValue.toLowerCase();
      }
      return value === filterValue;
    case 'is_not':
      if (typeof value === 'string' && typeof filterValue === 'string') {
        return value.toLowerCase() !== filterValue.toLowerCase();
      }
      return value !== filterValue;
    case 'starts_with':
      return String(value || '').toLowerCase().startsWith(String(filterValue || '').toLowerCase());
    case 'ends_with':
      return String(value || '').toLowerCase().endsWith(String(filterValue || '').toLowerCase());
    case 'eq':
      return Number(value) === Number(filterValue);
    case 'neq':
      return Number(value) !== Number(filterValue);
    case 'lt':
      return Number(value) < Number(filterValue);
    case 'lte':
      return Number(value) <= Number(filterValue);
    case 'gt':
      return Number(value) > Number(filterValue);
    case 'gte':
      return Number(value) >= Number(filterValue);
    case 'is_any_of':
      if (Array.isArray(filterValue)) {
        const filterArr = filterValue as string[];
        if (Array.isArray(value) && typeof value[0] === 'string') {
          const valueArr = value as string[];
          return filterArr.some((fv) => valueArr.includes(fv));
        }
        return filterArr.includes(value as string);
      }
      return false;
    case 'is_none_of':
      if (Array.isArray(filterValue)) {
        const filterArr = filterValue as string[];
        if (Array.isArray(value) && typeof value[0] === 'string') {
          const valueArr = value as string[];
          return !filterArr.some((fv) => valueArr.includes(fv));
        }
        return !filterArr.includes(value as string);
      }
      return true;
    case 'is_before':
      return new Date(value as string) < new Date(filterValue as string);
    case 'is_after':
      return new Date(value as string) > new Date(filterValue as string);
    case 'is_on_or_before':
      return new Date(value as string) <= new Date(filterValue as string);
    case 'is_on_or_after':
      return new Date(value as string) >= new Date(filterValue as string);
    case 'is_within':
      return evaluateDateWithin(value as string, filterValue as string);
    default:
      return true;
  }
}

// Helper function to compare values for sorting
function compareValues(a: CellValue, b: CellValue, fieldType: string): number {
  // Handle nulls
  if (a === null || a === undefined) return 1;
  if (b === null || b === undefined) return -1;

  // Number fields
  if (['number', 'currency', 'percent', 'rating', 'duration'].includes(fieldType)) {
    return Number(a) - Number(b);
  }

  // Date fields
  if (['date', 'datetime', 'created_time', 'last_modified_time'].includes(fieldType)) {
    return new Date(a as string).getTime() - new Date(b as string).getTime();
  }

  // Checkbox
  if (fieldType === 'checkbox') {
    return (a === true ? 1 : 0) - (b === true ? 1 : 0);
  }

  // Default: string comparison
  return String(a).localeCompare(String(b));
}

// Helper function to evaluate date "is within" condition
function evaluateDateWithin(value: string, period: string): boolean {
  if (!value) return false;

  const date = new Date(value);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());

  switch (period) {
    case 'today':
      return date >= today && date < new Date(today.getTime() + 86400000);
    case 'yesterday':
      const yesterday = new Date(today.getTime() - 86400000);
      return date >= yesterday && date < today;
    case 'tomorrow':
      const tomorrow = new Date(today.getTime() + 86400000);
      return date >= tomorrow && date < new Date(tomorrow.getTime() + 86400000);
    case 'past_week':
      return date >= new Date(today.getTime() - 7 * 86400000) && date < now;
    case 'past_month':
      return date >= new Date(today.getTime() - 30 * 86400000) && date < now;
    case 'past_year':
      return date >= new Date(today.getTime() - 365 * 86400000) && date < now;
    case 'next_week':
      return date > now && date <= new Date(today.getTime() + 7 * 86400000);
    case 'next_month':
      return date > now && date <= new Date(today.getTime() + 30 * 86400000);
    case 'next_year':
      return date > now && date <= new Date(today.getTime() + 365 * 86400000);
    default:
      return true;
  }
}
