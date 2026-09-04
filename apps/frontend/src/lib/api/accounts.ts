import apiClient from '@/lib/api-client';
import type {
  Account,
  ConnectionTestResult,
  CreateAccountInput,
  UpdateAccountInput,
} from '@/types/api';

/**
 * 账户相关 API
 *
 * 后端使用 URI 版本控制（默认 v1），baseURL 为 `/api`，故此处路径带 `/v1` 前缀。
 * ⚠️ apiKey 仅在请求体中一次性传递，绝不落入任何前端持久化状态。
 */

export async function fetchAccounts(): Promise<Account[]> {
  const { data } = await apiClient.get<Account[]>('/v1/accounts');
  return data;
}

export async function fetchAccount(id: string): Promise<Account> {
  const { data } = await apiClient.get<Account>(`/v1/accounts/${id}`);
  return data;
}

export async function createAccount(input: CreateAccountInput): Promise<Account> {
  const { data } = await apiClient.post<Account>('/v1/accounts', input);
  return data;
}

export async function updateAccount(id: string, input: UpdateAccountInput): Promise<Account> {
  const { data } = await apiClient.patch<Account>(`/v1/accounts/${id}`, input);
  return data;
}

export async function deleteAccount(id: string): Promise<{ id: string }> {
  const { data } = await apiClient.delete<{ id: string }>(`/v1/accounts/${id}`);
  return data;
}

export async function testConnection(id: string): Promise<ConnectionTestResult> {
  const { data } = await apiClient.post<ConnectionTestResult>(`/v1/accounts/${id}/test-connection`);
  return data;
}

/** 手动触发同步（异步入队，立即返回） */
export async function triggerSync(id: string): Promise<{ queued: boolean; jobId: string }> {
  const { data } = await apiClient.post<{ queued: boolean; jobId: string }>(`/v1/sync/accounts/${id}`);
  return data;
}

/** 手动触发所有账户同步 */
export async function triggerSyncAll(): Promise<{ queued: number; jobIds: string[] }> {
  const { data } = await apiClient.post<{ queued: number; jobIds: string[] }>(`/v1/sync/all`);
  return data;
}
