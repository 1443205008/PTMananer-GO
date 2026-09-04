'use client';

import { useQuery, useMutation } from '@tanstack/react-query';
import { searchTorrents, genDlToken, fetchTeamList, type SearchTorrentsParams } from '@/lib/api/search';

export function useSearchTorrents(params: SearchTorrentsParams, enabled = false) {
  return useQuery({
    queryKey: ['search-torrents', params],
    queryFn: () => searchTorrents(params),
    enabled,
    staleTime: 60_000,
    retry: false,
  });
}

export function useTeamList() {
  return useQuery({
    queryKey: ['search-teams'],
    queryFn: fetchTeamList,
    staleTime: 30 * 60_000, // 30 分钟，制作组列表变化不频繁
  });
}

export function useGenDlToken() {
  return useMutation({
    mutationFn: (torrentId: string) => genDlToken(torrentId),
  });
}
