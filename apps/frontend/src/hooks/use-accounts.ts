'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  createAccount,
  deleteAccount,
  fetchAccount,
  fetchAccounts,
  testConnection,
  triggerSync,
  updateAccount,
} from '@/lib/api/accounts';
import { queryKeys } from '@/lib/query-keys';
import type { CreateAccountInput, UpdateAccountInput } from '@/types/api';

/**
 * 账户相关 Hooks
 *
 * mutation 成功后统一失效 accounts + dashboard 查询，保持 UI 与后端一致。
 * ⚠️ apiKey 仅在 mutation 入参中一次性传递，绝不写入任何缓存或持久化状态。
 */

export function useAccounts() {
  return useQuery({
    queryKey: queryKeys.accounts(),
    queryFn: fetchAccounts,
  });
}

export function useAccount(id: string) {
  return useQuery({
    queryKey: queryKeys.account(id),
    queryFn: () => fetchAccount(id),
    enabled: Boolean(id),
  });
}

/** 失效账户列表与仪表盘（dashboard 依赖账户聚合） */
function useInvalidateAccounts() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: queryKeys.accounts() });
    void qc.invalidateQueries({ queryKey: ['dashboard'] });
  };
}

export function useCreateAccount() {
  const invalidate = useInvalidateAccounts();
  return useMutation({
    mutationFn: (input: CreateAccountInput) => createAccount(input),
    onSuccess: invalidate,
  });
}

export function useUpdateAccount() {
  const invalidate = useInvalidateAccounts();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateAccountInput }) =>
      updateAccount(id, input),
    onSuccess: invalidate,
  });
}

export function useDeleteAccount() {
  const invalidate = useInvalidateAccounts();
  return useMutation({
    mutationFn: (id: string) => deleteAccount(id),
    onSuccess: invalidate,
  });
}

/** 连接测试：只读操作，不触发失效 */
export function useTestConnection() {
  return useMutation({
    mutationFn: (id: string) => testConnection(id),
  });
}

/** 手动触发同步：入队后延迟刷新，给后端 worker 处理时间 */
export function useTriggerSync() {
  const invalidate = useInvalidateAccounts();
  return useMutation({
    mutationFn: (id: string) => triggerSync(id),
    onSuccess: () => {
      // 同步为异步任务，稍后刷新以反映结果
      setTimeout(invalidate, 3_000);
    },
  });
}
