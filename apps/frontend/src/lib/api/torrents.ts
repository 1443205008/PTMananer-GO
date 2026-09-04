import apiClient from '@/lib/api-client';
import type { TorrentListResponse, TorrentStatus } from '@/types/api';

/**
 * 种子列表 API
 */

export interface FetchTorrentsParams {
  accountId?: string;
  status?: TorrentStatus;
  page?: number;
  limit?: number;
}

export async function fetchTorrents(params: FetchTorrentsParams = {}): Promise<TorrentListResponse> {
  const { data } = await apiClient.get<TorrentListResponse>('/v1/torrents', { params });
  return data;
}

/** 触发单账户种子同步 */
export async function triggerTorrentSync(accountId: string): Promise<{ queued: boolean; jobId: string }> {
  const { data } = await apiClient.post<{ queued: boolean; jobId: string }>(
    `/v1/sync/accounts/${accountId}/torrents`,
  );
  return data;
}
