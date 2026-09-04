'use client';

import { useEffect, useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { X, Loader2, Eye, EyeOff } from 'lucide-react';
import { useCreateAccount, useUpdateAccount } from '@/hooks/use-accounts';
import type { Account } from '@/types/api';

interface AccountFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 传入则为编辑模式，否则为新建模式 */
  account?: Account | null;
}

/**
 * 账户新建 / 编辑对话框
 *
 * ⚠️ 安全约束：
 * - API Key 输入框默认脱敏（password 类型），仅提交时传递
 * - 编辑模式下 API Key 留空表示不修改；不回显任何已存凭据
 * - 提交后立即清空本地 apiKey state
 */
export function AccountFormDialog({ open, onOpenChange, account }: AccountFormDialogProps) {
  const isEdit = Boolean(account);
  const createMutation = useCreateAccount();
  const updateMutation = useUpdateAccount();

  const [accountName, setAccountName] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [remark, setRemark] = useState('');
  const [showApiKey, setShowApiKey] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // 打开 / 切换账户时同步表单初值（apiKey 永不回显）
  useEffect(() => {
    if (open) {
      setAccountName(account?.accountName ?? '');
      setRemark(account?.remark ?? '');
      setApiKey('');
      setShowApiKey(false);
      setFormError(null);
    }
  }, [open, account]);

  const isPending = createMutation.isPending || updateMutation.isPending;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);

    try {
      if (isEdit && account) {
        await updateMutation.mutateAsync({
          id: account.id,
          input: {
            accountName: accountName.trim(),
            remark: remark.trim() || undefined,
            // 留空表示不轮换 Key
            ...(apiKey.trim() ? { apiKey: apiKey.trim() } : {}),
          },
        });
      } else {
        await createMutation.mutateAsync({
          siteCode: 'MTEAM',
          accountName: accountName.trim(),
          apiKey: apiKey.trim(),
          remark: remark.trim() || undefined,
        });
      }
      // 提交成功后立即清空敏感输入
      setApiKey('');
      onOpenChange(false);
    } catch (err) {
      const message =
        (err as { response?: { data?: { message?: string | string[] } } })?.response?.data
          ?.message ?? (err as Error)?.message ?? '提交失败';
      setFormError(Array.isArray(message) ? message.join('; ') : message);
    }
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm data-[state=open]:animate-fade-in data-[state=closed]:animate-fade-out" />
        <Dialog.Content className="fixed inset-x-0 bottom-0 z-50 max-h-[90dvh] overflow-y-auto rounded-t-2xl border border-border bg-bg-card p-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))] shadow-2xl focus:outline-none max-sm:data-[state=closed]:animate-sheet-down max-sm:data-[state=open]:animate-sheet-up sm:inset-auto sm:left-1/2 sm:top-1/2 sm:w-[90vw] sm:max-w-md sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-xl sm:p-6 sm:pb-6">
          <div className="mb-5 flex items-start justify-between gap-3">
            <div className="min-w-0">
              <Dialog.Title className="text-base font-semibold text-fg">
                {isEdit ? '编辑账户' : '添加 M-Team 账户'}
              </Dialog.Title>
              <Dialog.Description className="mt-1 text-xs text-fg-subtle">
                {isEdit
                  ? 'API Key 留空表示不修改'
                  : 'API Key 将加密存储，绝不明文保存或回传前端'}
              </Dialog.Description>
            </div>
            <Dialog.Close className="icon-btn -mr-1 -mt-1">
              <X className="h-4 w-4" />
            </Dialog.Close>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            {/* 站点（Phase 1 仅 M-Team，固定展示） */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-fg-muted">站点</label>
              <div className="flex items-center gap-2 rounded-lg border border-border bg-bg-elevated px-3 py-2.5 text-base text-fg-muted sm:py-2 sm:text-sm">
                <span className="h-2 w-2 shrink-0 rounded-full bg-accent" />
                M-Team
              </div>
            </div>

            {/* 账户名称 */}
            <div>
              <label htmlFor="accountName" className="mb-1.5 block text-xs font-medium text-fg-muted">
                账户名称
              </label>
              <input
                id="accountName"
                type="text"
                required
                maxLength={64}
                value={accountName}
                onChange={(e) => setAccountName(e.target.value)}
                placeholder="如：主账户"
                className="field bg-bg-elevated sm:py-2"
              />
            </div>

            {/* API Key */}
            <div>
              <label htmlFor="apiKey" className="mb-1.5 block text-xs font-medium text-fg-muted">
                API Key {isEdit && <span className="text-fg-subtle">(留空不修改)</span>}
              </label>
              <div className="relative">
                <input
                  id="apiKey"
                  type={showApiKey ? 'text' : 'password'}
                  required={!isEdit}
                  autoComplete="off"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  placeholder={isEdit ? '••••••••' : 'x-api-key'}
                  className="field bg-bg-elevated pr-12 sm:py-2 sm:pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowApiKey((v) => !v)}
                  aria-label={showApiKey ? '隐藏 API Key' : '显示 API Key'}
                  className="icon-btn absolute right-1 top-1/2 -translate-y-1/2 sm:right-2"
                >
                  {showApiKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              <p className="mt-1 text-xs text-fg-subtle">
                在 M-Team「控制台 → 实验室 → API 密钥」生成
              </p>
            </div>

            {/* 备注 */}
            <div>
              <label htmlFor="remark" className="mb-1.5 block text-xs font-medium text-fg-muted">
                备注 <span className="text-fg-subtle">(可选)</span>
              </label>
              <input
                id="remark"
                type="text"
                maxLength={256}
                value={remark}
                onChange={(e) => setRemark(e.target.value)}
                className="field bg-bg-elevated sm:py-2"
              />
            </div>

            {formError && (
              <div className="break-anywhere rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-xs text-danger">
                {formError}
              </div>
            )}

            <div className="flex flex-col gap-2 pt-2 sm:flex-row sm:justify-end">
              <Dialog.Close
                type="button"
                className="btn-toolbar w-full bg-bg-elevated px-4 text-sm sm:w-auto"
              >
                取消
              </Dialog.Close>
              <button
                type="submit"
                disabled={isPending}
                className="btn-accent w-full px-4 text-sm sm:w-auto"
              >
                {isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                {isEdit ? '保存' : '添加'}
              </button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
