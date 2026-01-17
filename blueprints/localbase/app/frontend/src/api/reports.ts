import { api } from './client';

// Chart type enum
export type ChartType = 'line' | 'area' | 'stacked_area' | 'bar' | 'stacked_bar' | 'table';

// Metric data point
export interface MetricDataPoint {
  timestamp: string;
  value: number;
  values?: Record<string, number>;
}

// Chart configuration
export interface ChartConfig {
  id: string;
  title: string;
  type: ChartType;
  metric?: string;
  metrics?: string[];
  unit: string;
}

// Chart data with configuration
export interface ChartData extends ChartConfig {
  data: MetricDataPoint[];
}

// Report configuration
export interface ReportConfig {
  id: string;
  name: string;
  description?: string;
  report_type: string;
  charts: ChartConfig[];
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

// Report type definition
export interface ReportType {
  id: string;
  name: string;
  description: string;
}

// Complete report with data
export interface Report {
  report_type: string;
  from: string;
  to: string;
  interval: string;
  charts: ChartData[];
}

// Report filter parameters
export interface ReportFilter {
  from?: string;
  to?: string;
  time_range?: string;
  interval?: string;
}

// Build URL search params from filter
function buildSearchParams(filter?: ReportFilter): URLSearchParams {
  const params = new URLSearchParams();
  if (filter?.from) params.set('from', filter.from);
  if (filter?.to) params.set('to', filter.to);
  if (filter?.time_range) params.set('time_range', filter.time_range);
  if (filter?.interval) params.set('interval', filter.interval);
  return params;
}

export const reportsApi = {
  // List available report types
  listReportTypes: (): Promise<ReportType[]> => {
    return api.get('/api/reports');
  },

  // Get a report with data
  getReport: (reportType: string, filter?: ReportFilter): Promise<Report> => {
    const params = buildSearchParams(filter);
    const query = params.toString();
    return api.get(`/api/reports/${reportType}${query ? `?${query}` : ''}`);
  },

  // Get a single chart's data
  getReportChart: (reportType: string, chartId: string, filter?: ReportFilter): Promise<ChartData> => {
    const params = buildSearchParams(filter);
    const query = params.toString();
    return api.get(`/api/reports/${reportType}/chart/${chartId}${query ? `?${query}` : ''}`);
  },

  // List report configurations
  listReportConfigs: (): Promise<ReportConfig[]> => {
    return api.get('/api/reports/configs');
  },

  // Get report configuration by type
  getReportConfig: (reportType: string): Promise<ReportConfig> => {
    return api.get(`/api/reports/configs/${reportType}`);
  },

  // Get Prometheus metrics (text format)
  getPrometheusMetrics: (): Promise<string> => {
    return api.get('/customer/v1/privileged/metrics');
  },
};
