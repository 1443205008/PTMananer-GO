import apiClient from '@/lib/api-client';
import type { TorrentSearchResponse, SearchMode, SearchDiscount, SearchSortField, SearchTeam } from '@/types/api';

export interface SearchTorrentsParams {
  keyword?: string;
  page?: number;
  pageSize?: number;
  mode?: SearchMode;
  discount?: SearchDiscount;
  sortField?: SearchSortField;
  sortDirection?: 'ASC' | 'DESC';
  teams?: number[];
}

export async function searchTorrents(params: SearchTorrentsParams): Promise<TorrentSearchResponse> {
  const { teams, ...rest } = params;
  const { data } = await apiClient.get<TorrentSearchResponse>('/v1/search/torrents', {
    params: { ...rest, ...(teams?.length ? { teams: teams.join(',') } : {}) },
  });
  return data;
}

export async function fetchTeamList(): Promise<SearchTeam[]> {
  const { data } = await apiClient.get<SearchTeam[]>('/v1/search/teams');
  return data;
}

export async function genDlToken(torrentId: string): Promise<{ url: string }> {
  const { data } = await apiClient.post<{ url: string }>('/v1/search/dl-token', { torrentId });
  return data;
}
