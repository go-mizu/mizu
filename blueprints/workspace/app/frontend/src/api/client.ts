const API_BASE = '/api/v1'

interface RequestOptions {
  method: string
  headers: Record<string, string>
  body?: string
  credentials: RequestCredentials
}

class ApiClient {
  private async request<T>(method: string, path: string, data?: unknown): Promise<T> {
    const options: RequestOptions = {
      method,
      headers: { 'Content-Type': 'application/json' },
      credentials: 'same-origin',
    }

    if (data) {
      options.body = JSON.stringify(data)
    }

    const response = await fetch(API_BASE + path, options)

    if (!response.ok) {
      let errorMessage = 'Request failed'
      try {
        const error = await response.json()
        errorMessage = error.error || errorMessage
      } catch {
        errorMessage = response.statusText || errorMessage
      }
      throw new Error(errorMessage)
    }

    // Handle empty responses
    const text = await response.text()
    if (!text) return {} as T

    return JSON.parse(text)
  }

  get<T>(path: string): Promise<T> {
    return this.request<T>('GET', path)
  }

  post<T>(path: string, data?: unknown): Promise<T> {
    return this.request<T>('POST', path, data)
  }

  put<T>(path: string, data?: unknown): Promise<T> {
    return this.request<T>('PUT', path, data)
  }

  patch<T>(path: string, data?: unknown): Promise<T> {
    return this.request<T>('PATCH', path, data)
  }

  delete<T>(path: string): Promise<T> {
    return this.request<T>('DELETE', path)
  }

  // File upload
  async upload(file: File, path: string = '/media/upload'): Promise<{ id: string; url: string }> {
    const formData = new FormData()
    formData.append('file', file)

    const response = await fetch(API_BASE + path, {
      method: 'POST',
      body: formData,
      credentials: 'same-origin',
    })

    if (!response.ok) {
      throw new Error('Upload failed')
    }

    return response.json()
  }
}

export const api = new ApiClient()

// Types for API responses
export interface Page {
  id: string
  workspace_id: string
  parent_type: string
  parent_id: string
  title: string
  icon?: string
  cover?: string
  position?: number
  is_favorite?: boolean
  created_at: string
  updated_at: string
}

export interface Block {
  id: string
  page_id: string
  type: string
  content: Record<string, unknown>
  position: number
  parent_id?: string
  created_at: string
  updated_at: string
}

export interface Database {
  id: string
  workspace_id: string
  name: string
  icon?: string
  description?: string
  properties: Property[]
  created_at: string
  updated_at: string
}

export interface Property {
  id: string
  name: string
  type: PropertyType
  options?: PropertyOption[]
  config?: Record<string, unknown>
}

export type PropertyType =
  | 'text'
  | 'number'
  | 'select'
  | 'multi_select'
  | 'date'
  | 'person'
  | 'checkbox'
  | 'url'
  | 'email'
  | 'phone'
  | 'files'
  | 'relation'
  | 'rollup'
  | 'formula'
  | 'created_time'
  | 'created_by'
  | 'last_edited_time'
  | 'last_edited_by'
  | 'status'

export interface PropertyOption {
  id: string
  name: string
  color: string
}

export interface DatabaseRow {
  id: string
  database_id: string
  properties: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface View {
  id: string
  database_id: string
  name: string
  type: 'table' | 'board' | 'list' | 'calendar' | 'gallery' | 'timeline' | 'chart'
  config: ViewConfig
}

export interface ViewConfig {
  filters?: Filter[]
  sorts?: Sort[]
  groupBy?: string | null
  hiddenProperties?: string[]
  propertyWidths?: Record<string, number>
}

export interface Filter {
  property: string
  operator: string
  value: unknown
}

export interface Sort {
  property: string
  direction: 'asc' | 'desc'
}

export interface Workspace {
  id: string
  name: string
  slug: string
  icon?: string
}

export interface User {
  id: string
  name: string
  email: string
  avatar?: string
}
