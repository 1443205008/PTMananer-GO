'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchTorrents, triggerTorrentSync, type FetchTorrentsParams } from '@/lib/api/torrents';
import { queryKeys } from '@/lib/query-keys';
import type { TorrentStatus } from '@/types/api';

export function useTorrents(params: FetchTorrentsParams = {}) {
  return useQuery({
    queryKey: queryKeys.torrents(params),
    queryFn: () => fetchTorrents(params),
    staleTime: 30_000,
  });
}

export function useTriggerTorrentSync() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (accountId: string) => triggerTorrentSync(accountId),
    onSuccess: () => {
      // 同步为异步 BullMQ 任务，实际耗时约 15~25s（API冷却 + 拉取 + 写库）
      // 分三次刷新：15s / 25s / 40s，确保任一完成时间都能捕到
      [15_000, 25_000, 40_000].forEach((delay) => {
        setTimeout(() => {
          void qc.invalidateQueries({ queryKey: ['torrents'] });
        }, delay);
      });
    },
  });
}
