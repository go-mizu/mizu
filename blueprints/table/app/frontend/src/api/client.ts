import type { User, Workspace, Base, Table, Field, TableRecord, View, Comment, Share, SelectOption } from '../types';

const API_BASE = '/api/v1';

// Token storage
let authToken: string | null = null;

export function setAuthToken(token: string | null) {
  authToken = token;
  if (token) {
    localStorage.setItem('token', token);
  } else {
    localStorage.removeItem('token');
  }
}

export function getAuthToken(): string | null {
  if (!authToken) {
    authToken = localStorage.getItem('token');
  }
  return authToken;
}

// Fetch wrapper with auth
async function apiFetch<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getAuthToken();
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  if (token) {
    (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || `Request failed: ${response.status}`);
  }

  return response.json();
}

// Auth API
export const authApi = {
  async register(email: string, name: string, password: string): Promise<{ token: string; user: User }> {
    const result = await apiFetch<{ token: string; user: User }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, name, password }),
    });
    setAuthToken(result.token);
    return result;
  },

  async login(email: string, password: string): Promise<{ token: string; user: User }> {
    const result = await apiFetch<{ token: string; user: User }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    setAuthToken(result.token);
    return result;
  },

  async logout(): Promise<void> {
    await apiFetch('/auth/logout', { method: 'POST' });
    setAuthToken(null);
  },

  async me(): Promise<{ user: User }> {
    return apiFetch('/auth/me');
  },
};

// Workspaces API
export const workspacesApi = {
  async list(): Promise<{ workspaces: Workspace[] }> {
    return apiFetch('/workspaces');
  },

  async create(name: string, slug: string): Promise<{ workspace: Workspace }> {
    return apiFetch('/workspaces', {
      method: 'POST',
      body: JSON.stringify({ name, slug }),
    });
  },

  async get(id: string): Promise<{ workspace: Workspace }> {
    return apiFetch(`/workspaces/${id}`);
  },

  async update(id: string, data: Partial<Workspace>): Promise<{ workspace: Workspace }> {
    return apiFetch(`/workspaces/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async delete(id: string): Promise<void> {
    await apiFetch(`/workspaces/${id}`, { method: 'DELETE' });
  },

  async getBases(id: string): Promise<{ bases: Base[] }> {
    return apiFetch(`/workspaces/${id}/bases`);
  },
};

// Bases API
export const basesApi = {
  async create(workspaceId: string, name: string, color?: string): Promise<{ base: Base }> {
    return apiFetch('/bases', {
      method: 'POST',
      body: JSON.stringify({ workspace_id: workspaceId, name, color }),
    });
  },

  async get(id: string): Promise<{ base: Base; tables: Table[] }> {
    return apiFetch(`/bases/${id}`);
  },

  async update(id: string, data: Partial<Base>): Promise<{ base: Base }> {
    return apiFetch(`/bases/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async delete(id: string): Promise<void> {
    await apiFetch(`/bases/${id}`, { method: 'DELETE' });
  },

  async getTables(id: string): Promise<{ tables: Table[] }> {
    return apiFetch(`/bases/${id}/tables`);
  },
};

// Tables API
export const tablesApi = {
  async create(baseId: string, name: string): Promise<{ table: Table }> {
    return apiFetch('/tables', {
      method: 'POST',
      body: JSON.stringify({ base_id: baseId, name }),
    });
  },

  async get(id: string): Promise<{ table: Table; fields?: Field[]; views?: View[] }> {
    return apiFetch(`/tables/${id}`);
  },

  async update(id: string, data: Partial<Table>): Promise<{ table: Table }> {
    return apiFetch(`/tables/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async delete(id: string): Promise<void> {
    await apiFetch(`/tables/${id}`, { method: 'DELETE' });
  },

  async getFields(id: string): Promise<{ fields: Field[] }> {
    return apiFetch(`/tables/${id}/fields`);
  },

  async getViews(id: string): Promise<{ views: View[] }> {
    return apiFetch(`/tables/${id}/views`);
  },
};

// Fields API
export const fieldsApi = {
  async create(tableId: string, name: string, type: string, options?: Record<string, unknown>): Promise<{ field: Field }> {
    return apiFetch('/fields', {
      method: 'POST',
      body: JSON.stringify({ table_id: tableId, name, type, options }),
    });
  },

  async get(id: string): Promise<{ field: Field }> {
    return apiFetch(`/fields/${id}`);
  },

  async update(id: string, data: Partial<Field>): Promise<{ field: Field }> {
    return apiFetch(`/fields/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async delete(id: string): Promise<void> {
    await apiFetch(`/fields/${id}`, { method: 'DELETE' });
  },

  async reorder(tableId: string, fieldIds: string[]): Promise<void> {
    await apiFetch(`/fields/${tableId}/reorder`, {
      method: 'POST',
      body: JSON.stringify({ field_ids: fieldIds }),
    });
  },

  // Select options
  async getOptions(fieldId: string): Promise<{ options: SelectOption[] }> {
    return apiFetch(`/fields/${fieldId}/options`);
  },

  async createOption(fieldId: string, name: string, color?: string): Promise<{ option: SelectOption }> {
    return apiFetch(`/fields/${fieldId}/options`, {
      method: 'POST',
      body: JSON.stringify({ name, color }),
    });
  },

  async updateOption(fieldId: string, optionId: string, name: string, color: string): Promise<{ option: SelectOption }> {
    return apiFetch(`/fields/${fieldId}/options/${optionId}`, {
      method: 'PATCH',
      body: JSON.stringify({ name, color }),
    });
  },

  async deleteOption(fieldId: string, optionId: string): Promise<void> {
    await apiFetch(`/fields/${fieldId}/options/${optionId}`, { method: 'DELETE' });
  },
};

// Server-side filter types
export interface ServerFilter {
  field_id: string;
  operator: string;
  value?: unknown;
}

export interface ServerSort {
  field_id: string;
  direction: 'asc' | 'desc';
}

export interface ListRecordsOptions {
  cursor?: string;
  limit?: number;
  filters?: ServerFilter[];
  filter_logic?: 'and' | 'or';
  sorts?: ServerSort[];
  search?: string;
  group_by?: string;
}

// Records API
export const recordsApi = {
  async list(tableId: string, options?: ListRecordsOptions): Promise<{ records: TableRecord[]; next_cursor?: string; has_more: boolean }> {
    const params = new URLSearchParams({ table_id: tableId });

    if (options?.cursor) params.set('cursor', options.cursor);
    if (options?.limit) params.set('limit', String(options.limit));
    if (options?.filters && options.filters.length > 0) {
      params.set('filters', JSON.stringify(options.filters));
    }
    if (options?.filter_logic) params.set('filter_logic', options.filter_logic);
    if (options?.sorts && options.sorts.length > 0) {
      params.set('sorts', JSON.stringify(options.sorts));
    }
    if (options?.search) params.set('search', options.search);
    if (options?.group_by) params.set('group_by', options.group_by);

    return apiFetch(`/records?${params}`);
  },

  async create(tableId: string, fields?: { [key: string]: unknown }): Promise<{ record: TableRecord }> {
    return apiFetch('/records', {
      method: 'POST',
      body: JSON.stringify({ table_id: tableId, fields }),
    });
  },

  async get(id: string): Promise<{ record: TableRecord }> {
    return apiFetch(`/records/${id}`);
  },

  async update(id: string, fields: { [key: string]: unknown }): Promise<{ record: TableRecord }> {
    return apiFetch(`/records/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ fields }),
    });
  },

  async delete(id: string): Promise<void> {
    await apiFetch(`/records/${id}`, { method: 'DELETE' });
  },

  async batchCreate(tableId: string, records: Array<{ fields?: { [key: string]: unknown } }>): Promise<{ records: TableRecord[] }> {
    return apiFetch('/records/batch', {
      method: 'POST',
      body: JSON.stringify({ table_id: tableId, records }),
    });
  },

  async batchUpdate(records: Array<{ id: string; fields: { [key: string]: unknown } }>): Promise<{ records: TableRecord[] }> {
    return apiFetch('/records/batch', {
      method: 'PATCH',
      body: JSON.stringify({ records }),
    });
  },

  async batchDelete(ids: string[]): Promise<void> {
    await apiFetch('/records/batch', {
      method: 'DELETE',
      body: JSON.stringify({ ids }),
    });
  },
};

// Views API
export const viewsApi = {
  async create(tableId: string, name: string, type: string): Promise<{ view: View }> {
    return apiFetch('/views', {
      method: 'POST',
      body: JSON.stringify({ table_id: tableId, name, type }),
    });
  },

  async get(id: string): Promise<{ view: View }> {
    return apiFetch(`/views/${id}`);
  },

  async update(id: string, data: Partial<View>): Promise<{ view: View }> {
    return apiFetch(`/views/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async delete(id: string): Promise<void> {
    await apiFetch(`/views/${id}`, { method: 'DELETE' });
  },

  async duplicate(id: string): Promise<{ view: View }> {
    return apiFetch(`/views/${id}/duplicate`, { method: 'POST' });
  },

  async reorder(tableId: string, viewIds: string[]): Promise<void> {
    await apiFetch(`/views/${tableId}/reorder`, {
      method: 'POST',
      body: JSON.stringify({ view_ids: viewIds }),
    });
  },

  async setFilters(id: string, filters: View['filters']): Promise<{ view: View }> {
    return apiFetch(`/views/${id}/filters`, {
      method: 'PATCH',
      body: JSON.stringify({ filters }),
    });
  },

  async setSorts(id: string, sorts: View['sorts']): Promise<{ view: View }> {
    return apiFetch(`/views/${id}/sorts`, {
      method: 'PATCH',
      body: JSON.stringify({ sorts }),
    });
  },

  async setGroups(id: string, groups: View['groups']): Promise<{ view: View }> {
    return apiFetch(`/views/${id}/groups`, {
      method: 'PATCH',
      body: JSON.stringify({ groups }),
    });
  },

  async setFieldConfig(id: string, fieldConfig: View['field_config']): Promise<{ view: View }> {
    return apiFetch(`/views/${id}/field-config`, {
      method: 'PATCH',
      body: JSON.stringify({ field_config: fieldConfig }),
    });
  },

  async setConfig(id: string, config: View['config']): Promise<{ view: View }> {
    return apiFetch(`/views/${id}/config`, {
      method: 'PATCH',
      body: JSON.stringify({ config }),
    });
  },
};

// Comments API
export const commentsApi = {
  async list(recordId: string): Promise<{ comments: Comment[] }> {
    return apiFetch(`/comments/record/${recordId}`);
  },

  async create(recordId: string, content: string, parentId?: string): Promise<{ comment: Comment }> {
    return apiFetch('/comments', {
      method: 'POST',
      body: JSON.stringify({ record_id: recordId, content, parent_id: parentId }),
    });
  },

  async update(id: string, content?: { text: string }, isResolved?: boolean): Promise<{ comment: Comment }> {
    return apiFetch(`/comments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ content, is_resolved: isResolved }),
    });
  },

  async delete(id: string): Promise<void> {
    await apiFetch(`/comments/${id}`, { method: 'DELETE' });
  },

  async resolve(id: string): Promise<{ comment: Comment }> {
    return apiFetch(`/comments/${id}/resolve`, { method: 'POST' });
  },

  async unresolve(id: string): Promise<{ comment: Comment }> {
    return apiFetch(`/comments/${id}/unresolve`, { method: 'POST' });
  },
};

// Shares API
export const sharesApi = {
  async list(baseId: string): Promise<{ shares: Share[] }> {
    return apiFetch(`/shares/base/${baseId}`);
  },

  async create(baseId: string, type: string, permission: string, email?: string): Promise<{ share: Share }> {
    return apiFetch('/shares', {
      method: 'POST',
      body: JSON.stringify({ base_id: baseId, type, permission, email }),
    });
  },

  async delete(id: string): Promise<void> {
    await apiFetch(`/shares/${id}`, { method: 'DELETE' });
  },

  async getByToken(token: string): Promise<{ share: Share; base: Base; tables: Table[] }> {
    return apiFetch(`/shares/token/${token}`);
  },
};
