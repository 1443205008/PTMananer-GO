/**
 * TanStack Query 的集中式 query key 定义
 *
 * 集中管理便于失效（invalidate）时保持一致，避免字符串散落各处。
 */
export const queryKeys = {
  dashboard: (trendDays: number) => ['dashboard', { trendDays }] as const,
  accounts: () => ['accounts'] as const,
  account: (id: string) => ['accounts', id] as const,
  alerts: (unreadOnly?: boolean) => ['alerts', { unreadOnly }] as const,
  alertsUnreadCount: () => ['alerts', 'unread-count'] as const,
  settings: () => ['settings'] as const,
  settingsStatus: () => ['settings', 'status'] as const,
  torrents: (params: { accountId?: string; status?: string; page?: number; limit?: number }) =>
    ['torrents', params] as const,
};
