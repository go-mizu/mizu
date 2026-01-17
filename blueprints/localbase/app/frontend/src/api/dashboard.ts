import { api } from './client';
import type { DashboardStats, HealthStatus } from '../types';

export const dashboardApi = {
  // Get dashboard statistics
  getStats: (): Promise<DashboardStats> => {
    return api.get<DashboardStats>('/api/dashboard/stats');
  },

  // Get health status
  getHealth: (): Promise<HealthStatus> => {
    return api.get<HealthStatus>('/api/dashboard/health');
  },
};
