import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '../client'

// Types
export interface XRayCard {
  id: string
  title: string
  description?: string
  visualization: string
  query?: string
  data?: {
    columns: { name: string; display_name: string; type: string }[]
    rows: Record<string, any>[]
    row_count: number
  }
  width: number
  height: number
  row: number
  col: number
  settings?: Record<string, any>
}

export interface XRayNavLink {
  type: 'zoom_in' | 'zoom_out' | 'related'
  label: string
  target_type: 'table' | 'field'
  target_id: string
}

export interface XRayTableStats {
  row_count: number
  column_count: number
  nullable_count: number
  last_updated?: string
}

export interface XRayResult {
  title: string
  description: string
  table_id: string
  table_name: string
  generated_at: string
  cards: XRayCard[]
  navigation: XRayNavLink[]
  stats?: XRayTableStats
}

// X-Ray Table
export function useXRayTable(datasourceId: string, tableId: string, enabled = true) {
  return useQuery({
    queryKey: ['xray', 'table', datasourceId, tableId],
    queryFn: () => api.post<XRayResult>(
      `/xray/${datasourceId}/table/${tableId}`,
      { include_data: true }
    ),
    enabled: enabled && !!datasourceId && !!tableId,
    staleTime: 1000 * 60 * 5, // Cache for 5 minutes
  })
}

// X-Ray Field
export function useXRayField(datasourceId: string, columnId: string, enabled = true) {
  return useQuery({
    queryKey: ['xray', 'field', datasourceId, columnId],
    queryFn: () => api.post<XRayResult>(
      `/xray/${datasourceId}/field/${columnId}`,
      { include_data: true }
    ),
    enabled: enabled && !!datasourceId && !!columnId,
    staleTime: 1000 * 60 * 5,
  })
}

// X-Ray Compare
export interface XRayCompareParams {
  datasourceId: string
  tableId: string
  column: string
  value1: string
  value2: string
}

export function useXRayCompare(params: XRayCompareParams, enabled = true) {
  return useQuery({
    queryKey: ['xray', 'compare', params],
    queryFn: () => api.post<XRayResult>(
      `/xray/${params.datasourceId}/table/${params.tableId}/compare`,
      {
        column: params.column,
        value1: params.value1,
        value2: params.value2,
        include_data: true
      }
    ),
    enabled: enabled && !!params.datasourceId && !!params.tableId && !!params.column,
    staleTime: 1000 * 60 * 5,
  })
}

// Save X-Ray as Dashboard
export function useSaveXRay() {
  return useMutation({
    mutationFn: (data: { xray: XRayResult; name?: string; collectionId?: string }) =>
      api.post<{ dashboard_id: string; name: string; message: string }>(
        '/xray/save',
        {
          xray: data.xray,
          name: data.name,
          collection_id: data.collectionId
        }
      ),
  })
}

// Trigger X-Ray generation (mutation-based for on-demand generation)
export function useGenerateXRay() {
  return useMutation({
    mutationFn: ({ datasourceId, tableId, limit }: { datasourceId: string; tableId: string; limit?: number }) =>
      api.post<XRayResult>(
        `/xray/${datasourceId}/table/${tableId}`,
        { include_data: true, limit: limit || 10000 }
      ),
  })
}

export function useGenerateFieldXRay() {
  return useMutation({
    mutationFn: ({ datasourceId, columnId }: { datasourceId: string; columnId: string }) =>
      api.post<XRayResult>(
        `/xray/${datasourceId}/field/${columnId}`,
        { include_data: true }
      ),
  })
}
