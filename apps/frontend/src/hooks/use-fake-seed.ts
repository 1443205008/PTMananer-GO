'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listFakeSeedJobs, startFakeSeed, stopFakeSeed, removeFakeSeed,
} from '@/lib/api/fake-seed';

const KEY = ['fake-seed'];

export function useFakeSeedJobs(accountId?: string) {
  return useQuery({
    queryKey: [...KEY, accountId],
    queryFn: () => listFakeSeedJobs(accountId),
    refetchInterval: 15_000, // 每15秒刷新一次状态
  });
}

export function useStartFakeSeed() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ accountId, torrentId, torrentName }: {
      accountId: string; torrentId: string; torrentName: string;
    }) => startFakeSeed(accountId, torrentId, torrentName),
    onSuccess: () => void qc.invalidateQueries({ queryKey: KEY }),
  });
}

export function useStopFakeSeed() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => stopFakeSeed(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: KEY }),
  });
}

export function useRemoveFakeSeed() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => removeFakeSeed(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: KEY }),
  });
}
