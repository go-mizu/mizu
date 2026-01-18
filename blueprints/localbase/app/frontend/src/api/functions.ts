import { api } from './client';
import type {
  EdgeFunction,
  Deployment,
  Secret,
  FunctionLog,
  FunctionMetrics,
  FunctionTemplate,
  FunctionSource,
  FunctionTestRequest,
  FunctionTestResponse,
} from '../types';

export interface CreateFunctionRequest {
  name: string;
  slug?: string;
  entrypoint?: string;
  verify_jwt?: boolean;
  template_id?: string;
}

export interface UpdateFunctionRequest {
  name?: string;
  entrypoint?: string;
  verify_jwt?: boolean;
  status?: 'active' | 'inactive';
}

export interface DeployFunctionRequest {
  source_code: string;
  import_map?: string;
}

export interface CreateSecretRequest {
  name: string;
  value: string;
}

export interface BulkSecretRequest {
  secrets: Array<{ name: string; value: string }>;
}

export interface BulkSecretResponse {
  created: number;
  updated: number;
  total: number;
}

export interface UpdateSourceRequest {
  source_code: string;
  import_map?: string;
}

export const functionsApi = {
  // Function operations
  listFunctions: (): Promise<EdgeFunction[]> => {
    return api.get<EdgeFunction[]>('/api/functions');
  },

  getFunction: (id: string): Promise<EdgeFunction> => {
    return api.get<EdgeFunction>(`/api/functions/${id}`);
  },

  createFunction: (data: CreateFunctionRequest): Promise<EdgeFunction> => {
    return api.post<EdgeFunction>('/api/functions', data);
  },

  updateFunction: (id: string, data: UpdateFunctionRequest): Promise<EdgeFunction> => {
    return api.put<EdgeFunction>(`/api/functions/${id}`, data);
  },

  deleteFunction: (id: string): Promise<void> => {
    return api.delete(`/api/functions/${id}`);
  },

  // Source code operations
  getSource: (id: string): Promise<FunctionSource> => {
    return api.get<FunctionSource>(`/api/functions/${id}/source`);
  },

  updateSource: (id: string, data: UpdateSourceRequest): Promise<{ saved: boolean; is_draft: boolean }> => {
    return api.put<{ saved: boolean; is_draft: boolean }>(`/api/functions/${id}/source`, data);
  },

  // Deployment operations
  deployFunction: (id: string, data: DeployFunctionRequest): Promise<Deployment> => {
    return api.post<Deployment>(`/api/functions/${id}/deploy`, data);
  },

  listDeployments: (functionId: string): Promise<Deployment[]> => {
    return api.get<Deployment[]>(`/api/functions/${functionId}/deployments`);
  },

  downloadFunction: async (id: string): Promise<string> => {
    const response = await fetch(`/api/functions/${id}/download`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });
    return response.text();
  },

  // Testing
  testFunction: (id: string, data: FunctionTestRequest): Promise<FunctionTestResponse> => {
    return api.post<FunctionTestResponse>(`/api/functions/${id}/test`, data);
  },

  // Logs and metrics
  getLogs: (
    functionId: string,
    options?: { limit?: number; level?: string; since?: string }
  ): Promise<{ logs: FunctionLog[] }> => {
    const params = new URLSearchParams();
    if (options?.limit) params.set('limit', options.limit.toString());
    if (options?.level) params.set('level', options.level);
    if (options?.since) params.set('since', options.since);
    const query = params.toString();
    return api.get<{ logs: FunctionLog[] }>(`/api/functions/${functionId}/logs${query ? `?${query}` : ''}`);
  },

  getMetrics: (functionId: string, period?: string): Promise<FunctionMetrics> => {
    const query = period ? `?period=${period}` : '';
    return api.get<FunctionMetrics>(`/api/functions/${functionId}/metrics${query}`);
  },

  // Secret operations
  listSecrets: (): Promise<Secret[]> => {
    return api.get<Secret[]>('/api/functions/secrets');
  },

  createSecret: (data: CreateSecretRequest): Promise<Secret> => {
    return api.post<Secret>('/api/functions/secrets', data);
  },

  bulkUpdateSecrets: (data: BulkSecretRequest): Promise<BulkSecretResponse> => {
    return api.put<BulkSecretResponse>('/api/functions/secrets/bulk', data);
  },

  deleteSecret: (name: string): Promise<void> => {
    return api.delete(`/api/functions/secrets/${name}`);
  },

  // Templates
  listTemplates: (): Promise<{ templates: FunctionTemplate[] }> => {
    return api.get<{ templates: FunctionTemplate[] }>('/api/function-templates');
  },

  getTemplate: (id: string): Promise<{ id: string; source_code: string; import_map?: string }> => {
    return api.get<{ id: string; source_code: string; import_map?: string }>(`/api/function-templates/${id}`);
  },

  // Invoke function (public endpoint)
  invokeFunction: async (name: string, options?: {
    method?: string;
    headers?: Record<string, string>;
    body?: any;
  }): Promise<any> => {
    const response = await fetch(`/functions/v1/${name}`, {
      method: options?.method || 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      body: options?.body ? JSON.stringify(options.body) : undefined,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.message || 'Function invocation failed');
    }

    return response.json();
  },
};
