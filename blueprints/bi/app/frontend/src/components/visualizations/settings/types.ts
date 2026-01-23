// Shared types for visualization settings components

export interface BaseSettingsProps {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}

export interface ChartSettingsProps extends BaseSettingsProps {
  columns?: { name: string; type: string }[]
}

// Column mapping for explicit data configuration
export interface ColumnMapping {
  x?: string
  y?: string[]
  series?: string
  size?: string
  color?: string
  label?: string
}

// Series configuration
export interface SeriesConfig {
  name: string
  color?: string
  type?: 'line' | 'bar' | 'area'
  visible?: boolean
}
