import apiClient from '@/lib/api-client';
import type { FakeSeedJob } from '@/types/api';

export async function listFakeSeedJobs(accountId?: string): Promise<FakeSeedJob[]> {
  const { data } = await apiClient.get<FakeSeedJob[]>('/v1/fake-seed', {
    params: accountId ? { accountId } : undefined,
  });
  return data;
}

export async function startFakeSeed(
  accountId: string, torrentId: string, torrentName: string,
): Promise<FakeSeedJob> {
  const { data } = await apiClient.post<FakeSeedJob>('/v1/fake-seed', {
    accountId, torrentId, torrentName,
  });
  return data;
}

export async function stopFakeSeed(id: string): Promise<{ id: string; status: string }> {
  const { data } = await apiClient.delete(`/v1/fake-seed/${id}/stop`);
  return data;
}

export async function removeFakeSeed(id: string): Promise<{ id: string }> {
  const { data } = await apiClient.delete(`/v1/fake-seed/${id}`);
  return data;
}
