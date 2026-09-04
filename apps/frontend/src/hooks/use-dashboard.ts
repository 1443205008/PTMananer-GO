'use client';

import { useQuery } from '@tanstack/react-query';
import { fetchDashboard } from '@/lib/api/dashboard';
import { queryKeys } from '@/lib/query-keys';

/**
 * 仪表盘数据 Hook
 *
 * 默认拉取近 30 天趋势；staleTime 由 QueryProvider 全局配置（60s）。
 * refetchInterval 让页面在后台同步完成后自动刷新。
 */
export function useDashboard(trendDays = 30) {
  return useQuery({
    queryKey: queryKeys.dashboard(trendDays),
    queryFn: () => fetchDashboard(trendDays),
    // 后端同步为异步任务，前台每 60s 自动刷新一次以反映最新结果
    refetchInterval: 60_000,
  });
}
