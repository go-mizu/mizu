// Field types
export type FieldType =
  | 'text'
  | 'long_text'
  | 'number'
  | 'currency'
  | 'percent'
  | 'single_select'
  | 'multi_select'
  | 'date'
  | 'datetime'
  | 'checkbox'
  | 'rating'
  | 'duration'
  | 'phone'
  | 'email'
  | 'url'
  | 'attachment'
  | 'user'
  | 'created_time'
  | 'last_modified_time'
  | 'created_by'
  | 'last_modified_by'
  | 'autonumber'
  | 'barcode'
  | 'button'
  | 'formula'
  | 'rollup'
  | 'count'
  | 'lookup'
  | 'link';

// View types
export type ViewType =
  | 'grid'
  | 'kanban'
  | 'calendar'
  | 'gallery'
  | 'form'
  | 'timeline'
  | 'list';

// User
export interface User {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  created_at: string;
}

// Workspace
export interface Workspace {
  id: string;
  name: string;
  slug: string;
  icon?: string;
  plan: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

// Base
export interface Base {
  id: string;
  workspace_id: string;
  name: string;
  description?: string;
  icon?: string;
  color: string;
  is_template: boolean;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// Table
export interface Table {
  id: string;
  base_id: string;
  name: string;
  description?: string;
  icon?: string;
  position: number;
  primary_field_id?: string;
  fields?: Field[];
  views?: View[];
  record_count?: number;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// Field
export interface Field {
  id: string;
  table_id: string;
  name: string;
  type: FieldType;
  description?: string;
  options?: FieldOptions;
  position: number;
  is_primary: boolean;
  is_computed: boolean;
  is_hidden: boolean;
  width: number;
  select_options?: SelectOption[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface FieldOptions {
  // Number options
  precision?: number;
  negative?: boolean;
  format?: string;

  // Currency options
  currency_symbol?: string;
  currency_code?: string;

  // Date options
  include_time?: boolean;
  time_format?: '12' | '24';
  date_format?: string;

  // Checkbox options
  icon?: string;
  color?: string;

  // Rating options
  max?: number;

  // Select options
  choices?: Array<{ id: string; name: string; color: string }>;

  // Link options
  linked_table_id?: string;
  is_reverse_link?: boolean;
  reverse_field_id?: string;

  // Formula options
  expression?: string;
  result_type?: FieldType;

  // Rollup options
  linked_field_id?: string;
  rollup_field_id?: string;
  aggregation?: string;
}

// Select option
export interface SelectOption {
  id: string;
  field_id: string;
  name: string;
  color: string;
  position: number;
}

// Table Record (named to avoid conflict with TypeScript's built-in Record<K,V>)
export interface TableRecord {
  id: string;
  table_id: string;
  position: number;
  values: { [fieldId: string]: CellValue };
  createdBy: string;
  createdAt: string;
  updatedBy: string;
  updatedAt: string;
}

// Cell value (can be various types)
export type CellValue =
  | string
  | number
  | boolean
  | Date
  | string[]
  | { id: string; name: string }[]
  | Attachment[]
  | null;

// Attachment
export interface Attachment {
  id: string;
  filename: string;
  size: number;
  mime_type: string;
  url: string;
  thumbnail_url?: string;
  width?: number;
  height?: number;
}

// View
export interface View {
  id: string;
  table_id: string;
  name: string;
  type: ViewType;
  filters: Filter[];
  sorts: Sort[];
  groups: Group[];
  field_config: FieldConfig[];
  settings: ViewSettings;
  config?: {
    groupBy?: string;
    dateField?: string;
    coverField?: string;
    [key: string]: unknown;
  };
  position: number;
  is_default: boolean;
  is_locked: boolean;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// Filter
export interface Filter {
  id?: string;
  field_id: string;
  operator: string;
  value: unknown;
}

// Sort
export interface Sort {
  field_id: string;
  direction: 'asc' | 'desc';
}

// Group
export interface Group {
  field_id: string;
  order?: 'asc' | 'desc';
  collapsed?: string[];
}

// Field config (per view)
export interface FieldConfig {
  field_id: string;
  visible: boolean;
  width?: number;
}

// View settings
export interface ViewSettings {
  // Grid
  row_height?: 'short' | 'medium' | 'tall' | 'extra_tall';
  frozen_columns?: number;
  show_row_numbers?: boolean;

  // Kanban
  card_size?: 'small' | 'medium' | 'large';
  card_cover_field_id?: string;
  hide_empty_columns?: boolean;
  color_columns?: boolean;

  // Calendar
  date_field_id?: string;
  end_date_field_id?: string;
  mode?: 'month' | 'week' | 'day';
  show_weekends?: boolean;
  week_start?: 0 | 1;

  // Gallery
  cover_field_id?: string;
  fit_image?: boolean;
  card_fields?: string[];

  // Timeline
  start_field_id?: string;
  end_field_id?: string;
  time_scale?: 'day' | 'week' | 'month' | 'quarter' | 'year';
  group_field_id?: string;

  // Form
  title?: string;
  description?: string;
  submit_button_text?: string;
  show_branding?: boolean;
  redirect_url?: string;
}

// Comment
export interface Comment {
  id: string;
  recordId: string;
  parentId?: string;
  userId: string;
  user?: User;
  content: string;
  isResolved: boolean;
  createdAt: string;
  updatedAt: string;
}

// Share
export interface Share {
  id: string;
  base_id: string;
  type: 'invite' | 'public_link';
  permission: 'read' | 'comment' | 'edit' | 'creator' | 'owner';
  email?: string;
  token?: string;
  expires_at?: string;
  created_by: string;
  created_at: string;
}

// API response types
export interface PaginatedResponse<T> {
  items: T[];
  next_cursor?: string;
  has_more: boolean;
}

export interface ApiError {
  error: string;
  message: string;
  status: number;
}
