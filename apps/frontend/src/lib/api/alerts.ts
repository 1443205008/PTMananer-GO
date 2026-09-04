import apiClient from '@/lib/api-client';
import type { AlertDto } from '@/types/api';

/**
 * 告警 API
 */

export async function fetchAlerts(unreadOnly = false): Promise<AlertDto[]> {
  const { data } = await apiClient.get<AlertDto[]>('/v1/alerts', {
    params: unreadOnly ? { unreadOnly: 'true' } : {},
  });
  return data;
}

export async function fetchUnreadCount(): Promise<{ count: number }> {
  const { data } = await apiClient.get<{ count: number }>('/v1/alerts/unread-count');
  return data;
}

export async function markAlertRead(id: string): Promise<AlertDto> {
  const { data } = await apiClient.patch<AlertDto>(`/v1/alerts/${id}/read`);
  return data;
}

export async function markAllAlertsRead(): Promise<{ updated: number }> {
  const { data } = await apiClient.patch<{ updated: number }>('/v1/alerts/read-all');
  return data;
}

export async function dismissAlert(id: string): Promise<void> {
  await apiClient.delete(`/v1/alerts/${id}`);
}
