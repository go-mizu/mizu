const API_BASE = '/api/v1'

// Check if we're in dev mode without a backend
const isDevMode = import.meta.env?.DEV ?? false

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

    try {
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
    } catch (error) {
      // In dev mode, return mock data instead of crashing
      if (isDevMode) {
        console.warn(`[API Dev Mode] ${method} ${path} failed:`, error)
        return this.getMockResponse<T>(method, path, data)
      }
      throw error
    }
  }

  // Mock responses for dev mode when backend is not running
  private getMockResponse<T>(method: string, path: string, data?: unknown): T {
    // Return appropriate mock data based on endpoint
    if (path.startsWith('/pages/') && path.endsWith('/export')) {
      // Mock export response with download simulation
      const exportData = data as { format?: string } | undefined
      const format = exportData?.format || 'pdf'
      const filename = `export-${Date.now()}.${format}`
      // Create a mock blob URL for dev mode download
      const mockContent = format === 'pdf'
        ? 'Mock PDF content for development'
        : format === 'html'
        ? '<html><body><h1>Mock HTML Export</h1></body></html>'
        : '# Mock Markdown Export\n\nThis is a development mock export.'
      const blob = new Blob([mockContent], { type: format === 'pdf' ? 'application/pdf' : 'text/plain' })
      const downloadUrl = URL.createObjectURL(blob)
      return {
        id: `export-${Date.now()}`,
        download_url: downloadUrl,
        filename,
        size: mockContent.length,
        format,
        page_count: 1,
      } as T
    }
    if (path.startsWith('/pages/') && path.endsWith('/blocks')) {
      return { blocks: [] } as T
    }
    if (path.startsWith('/pages')) {
      return { id: 'mock-page', title: 'Mock Page', blocks: [] } as T
    }
    if (path.startsWith('/databases')) {
      return { id: 'mock-db', name: 'Untitled', records: [], properties: [], views: [] } as T
    }
    if (path.startsWith('/workspaces')) {
      return { id: 'mock-ws', name: 'Mock Workspace', slug: 'mock' } as T
    }
    if (path.startsWith('/search')) {
      return { results: [], users: [], databases: [] } as T
    }
    if (path.startsWith('/media/upload')) {
      return { id: 'mock-media', url: 'https://via.placeholder.com/400', filename: 'mock.jpg', type: 'image/jpeg' } as T
    }
    // Default empty response
    return {} as T
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
  async upload(file: File, path: string = '/media/upload'): Promise<{ id: string; url: string; filename: string; type: string }> {
    const formData = new FormData()
    formData.append('file', file)

    const response = await fetch(API_BASE + path, {
      method: 'POST',
      body: formData,
      credentials: 'same-origin',
    })

    if (!response.ok) {
      let errorMessage = 'Upload failed'
      try {
        const error = await response.json()
        errorMessage = error.error || errorMessage
      } catch {
        errorMessage = response.statusText || errorMessage
      }
      throw new Error(errorMessage)
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
  config?: PropertyConfig
}

export interface PropertyConfig {
  // Number format options
  numberFormat?: NumberFormat
  precision?: number
  currency?: string
  showThousandsSeparator?: boolean

  // Select/Status options (for backend compatibility)
  options?: PropertyOption[]

  // Relation options
  relatedDatabaseId?: string

  // Rollup options
  rollupProperty?: string
  rollupFunction?: 'count' | 'sum' | 'average' | 'min' | 'max' | 'percent_empty' | 'percent_not_empty'

  // Formula options
  formula?: string

  // Date options
  includeTime?: boolean
  dateFormat?: 'relative' | 'full' | 'month_day_year' | 'day_month_year' | 'year_month_day'
  timeFormat?: '12h' | '24h'
}

export type NumberFormat =
  | 'number'
  | 'number_with_commas'
  | 'percent'
  | 'dollar'
  | 'euro'
  | 'pound'
  | 'yen'
  | 'rupee'
  | 'won'
  | 'yuan'
  | 'peso'
  | 'franc'
  | 'kroner'
  | 'real'
  | 'ringgit'
  | 'ruble'
  | 'rupiah'
  | 'baht'
  | 'lira'
  | 'shekel'
  | 'rand'

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

export interface FilterGroup {
  operator: 'and' | 'or'
  filters: (Filter | FilterGroup)[]
}

// Advanced date filter values
export type DateFilterValue =
  | string // ISO date string
  | { type: 'relative'; value: RelativeDateValue }

export type RelativeDateValue =
  | 'today'
  | 'tomorrow'
  | 'yesterday'
  | 'one_week_ago'
  | 'one_week_from_now'
  | 'one_month_ago'
  | 'one_month_from_now'
  | 'this_week'
  | 'last_week'
  | 'next_week'
  | 'this_month'
  | 'last_month'
  | 'next_month'
  | 'this_year'
  | 'last_year'
  | 'next_year'

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

// File/Media types for database property
export interface FileAttachment {
  id: string
  name: string
  url: string
  type: string // MIME type
  size?: number
  thumbnailUrl?: string
}

export interface UploadResponse {
  id: string
  url: string
  filename: string
  type: string
}
