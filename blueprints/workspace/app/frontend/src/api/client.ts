const API_BASE = '/api/v1'

// Helper to convert blocks to markdown (for dev mode export)
function blocksToMarkdown(blocks: Array<{ type: string; content?: Record<string, unknown>; children?: unknown[] }>, indent = ''): string {
  let md = ''
  let listCounter = 0

  for (const block of blocks) {
    const richText = block.content?.rich_text as Array<{ text: string; annotations?: Record<string, boolean> }> | undefined
    const text = richText?.map(rt => {
      let t = rt.text || ''
      if (rt.annotations?.bold) t = `**${t}**`
      if (rt.annotations?.italic) t = `*${t}*`
      if (rt.annotations?.code) t = `\`${t}\``
      if (rt.annotations?.strikethrough) t = `~~${t}~~`
      return t
    }).join('') || ''

    switch (block.type) {
      case 'paragraph':
        if (text) md += `${indent}${text}\n\n`
        break
      case 'heading_1':
        md += `# ${text}\n\n`
        listCounter = 0
        break
      case 'heading_2':
        md += `## ${text}\n\n`
        listCounter = 0
        break
      case 'heading_3':
        md += `### ${text}\n\n`
        listCounter = 0
        break
      case 'bulleted_list_item':
        md += `${indent}- ${text}\n`
        if (block.children) {
          md += blocksToMarkdown(block.children as Array<{ type: string; content?: Record<string, unknown>; children?: unknown[] }>, indent + '  ')
        }
        break
      case 'numbered_list_item':
        listCounter++
        md += `${indent}${listCounter}. ${text}\n`
        if (block.children) {
          md += blocksToMarkdown(block.children as Array<{ type: string; content?: Record<string, unknown>; children?: unknown[] }>, indent + '  ')
        }
        break
      case 'to_do':
        md += `${indent}- [${block.content?.checked ? 'x' : ' '}] ${text}\n`
        break
      case 'toggle':
        md += `<details>\n<summary>${text}</summary>\n\n`
        if (block.children) {
          md += blocksToMarkdown(block.children as Array<{ type: string; content?: Record<string, unknown>; children?: unknown[] }>)
        }
        md += `</details>\n\n`
        break
      case 'quote':
        md += `> ${text}\n\n`
        break
      case 'callout':
        md += `> ${block.content?.icon || '💡'} ${text}\n\n`
        break
      case 'code':
        md += `\`\`\`${block.content?.language || ''}\n${text}\n\`\`\`\n\n`
        break
      case 'divider':
        md += `---\n\n`
        listCounter = 0
        break
      case 'image':
        md += `![${block.content?.caption || 'Image'}](${block.content?.url || ''})\n\n`
        break
      case 'bookmark':
        md += `[${block.content?.title || block.content?.url || 'Bookmark'}](${block.content?.url || ''})\n\n`
        break
      case 'equation':
        md += `$$\n${block.content?.expression || ''}\n$$\n\n`
        break
      default:
        if (text) md += `${text}\n\n`
    }
  }
  return md
}

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
      // Always throw the error - don't silently return mock data
      // This ensures backend connection issues are visible during development
      console.error(`[API] ${method} ${path} failed:`, error)
      throw error
    }
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

  // Table view
  frozen_columns?: number
  row_height?: 'small' | 'medium' | 'tall' | 'extra_tall'
  wrap_cells?: boolean
  wrap_columns?: Record<string, boolean>
  calculations?: ColumnCalculation[]
  property_order?: string[]

  // Board view
  card_size?: 'small' | 'medium' | 'large'
  card_preview?: 'none' | 'page_cover' | 'page_content' | 'files'
  card_preview_property?: string
  fit_card_image?: boolean
  color_columns?: boolean
  hide_empty_groups?: boolean
  card_properties?: string[]
  column_order?: string[]

  // Timeline view
  time_scale?: 'hours' | 'days' | 'weeks' | 'months' | 'quarters' | 'years'
  start_date_property?: string
  end_date_property?: string
  show_table_panel?: boolean
  table_panel_width?: number
  table_panel_properties?: string[]
  dependencies?: Dependency[]
  show_dependencies?: boolean
  show_milestones?: boolean

  // Calendar view
  calendar_mode?: 'month' | 'week' | 'day'
  start_week_on_monday?: boolean
  event_color_property?: string
  show_weekends?: boolean

  // Gallery view
  gallery_card_size?: 'small' | 'medium' | 'large'
  preview_source?: 'page_cover' | 'page_content' | 'files' | 'none'
  files_property_id?: string
  fit_image?: boolean
  show_title?: boolean
  hide_card_names?: boolean

  // Chart view
  chart_type?: 'vertical_bar' | 'horizontal_bar' | 'line' | 'donut'
  chart_x_axis?: AxisConfig
  chart_y_axis?: AxisConfig
  chart_group_by?: string
  chart_style?: ChartStyleConfig
  chart_aggregation?: 'count' | 'sum' | 'average' | 'min' | 'max'

  // List view
  list_show_properties?: string[]
  list_compact?: boolean
}

export interface Dependency {
  from_row_id: string
  to_row_id: string
  type: 'finish_to_start' | 'start_to_start' | 'finish_to_finish' | 'start_to_finish'
}

export type CalculationType =
  | 'none'
  | 'count_all'
  | 'count_values'
  | 'count_unique'
  | 'count_empty'
  | 'count_not_empty'
  | 'percent_empty'
  | 'percent_not_empty'
  | 'sum'
  | 'average'
  | 'median'
  | 'min'
  | 'max'
  | 'range'
  | 'earliest_date'
  | 'latest_date'
  | 'date_range'

export interface ColumnCalculation {
  property_id: string
  type: CalculationType
}

export interface AxisConfig {
  property_id: string
  sort?: 'ascending' | 'descending' | 'none'
  visible_groups?: string[]
  omit_zero_values?: boolean
  cumulative?: boolean
}

export interface ChartStyleConfig {
  height?: 'small' | 'medium' | 'large' | 'extra_large'
  grid_lines?: boolean
  x_axis_labels?: boolean
  y_axis_labels?: boolean
  data_labels?: boolean
  smooth_line?: boolean
  gradient_area?: boolean
  show_center_value?: boolean
  show_legend?: boolean
  color_scheme?: string
  color_by_value?: boolean
  stacked?: boolean
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
