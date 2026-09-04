'use client';

import { useState } from 'react';
import { Plus, Globe, AlertTriangle, RefreshCw, Loader2 } from 'lucide-react';
import { AccountCard } from '@/components/accounts/account-card';
import { AccountFormDialog } from '@/components/accounts/account-form-dialog';
import { useAccounts } from '@/hooks/use-accounts';
import type { Account } from '@/types/api';

/**
 * 站点账户管理页
 *
 * 列出所有已接入账户（Phase 1 仅 M-Team），支持新建 / 编辑 / 删除 / 测试连接 / 手动同步。
 * ⚠️ 账户列表绝不含任何凭据字段；API Key 仅在表单提交时一次性传递。
 */
export default function SitesPage() {
  const { data: accounts, isLoading, isError, error, refetch, isFetching } = useAccounts();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Account | null>(null);

  function openCreate() {
    setEditing(null);
    setDialogOpen(true);
  }

  function openEdit(account: Account) {
    setEditing(account);
    setDialogOpen(true);
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* ── Page heading ── */}
      <div className="page-header">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold text-fg">站点账户</h2>
          <p className="mt-0.5 text-sm text-fg-subtle">
            管理已接入的 PT 站点账户
            {accounts && ` · 共 ${accounts.length} 个`}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <button onClick={() => void refetch()} className="btn-toolbar">
            <RefreshCw className={`h-3.5 w-3.5 ${isFetching ? 'animate-spin' : ''}`} />
            刷新
          </button>
          <button onClick={openCreate} className="btn-accent">
            <Plus className="h-3.5 w-3.5" />
            添加账户
          </button>
        </div>
      </div>

      {/* ── Content ── */}
      {isLoading ? (
        <SitesSkeleton />
      ) : isError ? (
        <ErrorState
          message={(error as Error)?.message ?? '无法连接后端服务'}
          onRetry={() => void refetch()}
        />
      ) : !accounts || accounts.length === 0 ? (
        <EmptyState onAdd={openCreate} />
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {accounts.map((account) => (
            <AccountCard key={account.id} account={account} onEdit={openEdit} />
          ))}
        </div>
      )}

      {/* ── Create / edit dialog ── */}
      <AccountFormDialog open={dialogOpen} onOpenChange={setDialogOpen} account={editing} />
    </div>
  );
}

// ── Sub-components ─────────────────────────────────────────────────────────────

function SitesSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      {Array.from({ length: 2 }).map((_, i) => (
        <div key={i} className="h-52 animate-pulse rounded-xl bg-bg-card" />
      ))}
    </div>
  );
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex min-h-[360px] flex-col items-center justify-center gap-4 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-danger/10">
        <AlertTriangle className="h-6 w-6 text-danger" />
      </div>
      <div>
        <p className="text-sm font-medium text-fg">加载失败</p>
        <p className="mt-1 text-xs text-fg-subtle">{message}</p>
      </div>
      <button onClick={onRetry} className="btn-toolbar px-4">
        <Loader2 className="h-3.5 w-3.5" />
        重试
      </button>
    </div>
  );
}

function EmptyState({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="flex min-h-[360px] flex-col items-center justify-center gap-4 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-accent/10">
        <Globe className="h-6 w-6 text-accent" />
      </div>
      <div>
        <p className="text-sm font-medium text-fg">还没有接入任何账户</p>
        <p className="mt-1 text-xs text-fg-subtle">添加你的第一个 M-Team 账户开始监控</p>
      </div>
      <button onClick={onAdd} className="btn-accent px-4">
        <Plus className="h-3.5 w-3.5" />
        添加账户
      </button>
    </div>
  );
}
