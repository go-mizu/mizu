import type { ApiError } from '../types';

// Default Supabase local development service key
const DEFAULT_SERVICE_KEY = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU';

export class ApiClientError extends Error {
  public status: number;
  public code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = 'ApiClientError';
    this.status = status;
    this.code = code;
  }
}

interface RequestConfig {
  headers?: Record<string, string>;
}

class ApiClient {
  private getServiceKey(): string {
    return localStorage.getItem('serviceKey') || DEFAULT_SERVICE_KEY;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    config?: RequestConfig
  ): Promise<T> {
    const serviceKey = this.getServiceKey();

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'apikey': serviceKey,
      'Authorization': `Bearer ${serviceKey}`,
      ...config?.headers,
    };

    const response = await fetch(path, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    // Handle empty responses
    const contentType = response.headers.get('content-type');
    let data: any = null;

    if (contentType?.includes('application/json')) {
      const text = await response.text();
      if (text) {
        data = JSON.parse(text);
      }
    }

    if (!response.ok) {
      const error = data as ApiError | null;
      throw new ApiClientError(
        response.status,
        error?.msg || error?.message || error?.error || 'Request failed',
        error?.error_code
      );
    }

    return data as T;
  }

  async get<T>(path: string, config?: RequestConfig): Promise<T> {
    return this.request<T>('GET', path, undefined, config);
  }

  async post<T>(path: string, body?: unknown, config?: RequestConfig): Promise<T> {
    return this.request<T>('POST', path, body, config);
  }

  async put<T>(path: string, body?: unknown, config?: RequestConfig): Promise<T> {
    return this.request<T>('PUT', path, body, config);
  }

  async patch<T>(path: string, body?: unknown, config?: RequestConfig): Promise<T> {
    return this.request<T>('PATCH', path, body, config);
  }

  async delete<T>(path: string, config?: RequestConfig): Promise<T> {
    return this.request<T>('DELETE', path, undefined, config);
  }

  // Special method for file uploads
  async uploadFile(path: string, file: File): Promise<any> {
    const serviceKey = this.getServiceKey();

    const response = await fetch(path, {
      method: 'POST',
      headers: {
        'apikey': serviceKey,
        'Authorization': `Bearer ${serviceKey}`,
      },
      body: file,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new ApiClientError(
        response.status,
        error?.message || error?.error || 'Upload failed',
        error?.error_code
      );
    }

    return response.json();
  }

  // Get download URL with auth
  getAuthenticatedUrl(path: string): string {
    const serviceKey = this.getServiceKey();
    const url = new URL(path, window.location.origin);
    url.searchParams.set('apikey', serviceKey);
    return url.toString();
  }
}

export const api = new ApiClient();
