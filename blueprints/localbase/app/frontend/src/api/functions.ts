import { api } from './client';
import type { EdgeFunction, Deployment, Secret } from '../types';

export interface CreateFunctionRequest {
  name: string;
  slug?: string;
  entrypoint?: string;
  verify_jwt?: boolean;
}

export interface UpdateFunctionRequest {
  name?: string;
  entrypoint?: string;
  verify_jwt?: boolean;
  status?: 'active' | 'inactive';
}

export interface DeployFunctionRequest {
  source_code: string;
}

export interface CreateSecretRequest {
  name: string;
  value: string;
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

  deployFunction: (id: string, data: DeployFunctionRequest): Promise<Deployment> => {
    return api.post<Deployment>(`/api/functions/${id}/deploy`, data);
  },

  listDeployments: (functionId: string): Promise<Deployment[]> => {
    return api.get<Deployment[]>(`/api/functions/${functionId}/deployments`);
  },

  // Secret operations
  listSecrets: (): Promise<Secret[]> => {
    return api.get<Secret[]>('/api/functions/secrets');
  },

  createSecret: (data: CreateSecretRequest): Promise<Secret> => {
    return api.post<Secret>('/api/functions/secrets', data);
  },

  deleteSecret: (name: string): Promise<void> => {
    return api.delete(`/api/functions/secrets/${name}`);
  },

  // Invoke function (public endpoint)
  invokeFunction: async (name: string, body?: any): Promise<any> => {
    const response = await fetch(`/functions/v1/${name}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.message || 'Function invocation failed');
    }

    return response.json();
  },
};
