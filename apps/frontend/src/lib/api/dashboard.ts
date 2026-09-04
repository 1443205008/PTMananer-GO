import apiClient from '@/lib/api-client';
import type { Dashboard } from '@/types/api';

/**
 * 仪表盘聚合 API
 *
 * 后端使用 URI 版本控制（默认 v1），baseURL 为 `/api`，故路径带 `/v1` 前缀。
 */
export async function fetchDashboard(trendDays = 30): Promise<Dashboard> {
  const { data } = await apiClient.get<Dashboard>('/v1/dashboard', {
    params: { trendDays },
  });
  return data;
}
