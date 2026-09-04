'use client';

import { useState } from 'react';
import {
  Files,
  RefreshCw,
  Filter,
  Loader2,
  AlertTriangle,
} from 'lucide-react';
import { useTorrents, useTriggerTorrentSync } from '@/hooks/use-torrents';
import { useAccounts } from '@/hooks/use-accounts';
import { useSettings } from '@/hooks/use-settings';
import { formatBytes, formatRatio, formatRelativeTime } from '@/lib/formatters';
import { cn } from '@/lib/utils';
import type { TorrentDto, TorrentStatus } from '@/types/api';

// ── Status 配置 ───────────────────────────────────────────────────────────────

const STATUS_CONFIG: Record<
  TorrentStatus,
  { label: string; dot: string; badge: string }
> = {
  SEEDING: {
    label: '做种中',
    dot: 'bg-success',
    badge: 'bg-success/10 text-success border-success/20',
  },
  LEECHING: {
    label: '下载中',
    dot: 'bg-info',
    badge: 'bg-info/10 text-info border-info/20',
  },
  COMPLETED: {
    label: '已完成',
    dot: 'bg-fg-subtle',
    badge: 'bg-fg-subtle/10 text-fg-muted border-border',
  },
  STOPPED: {
    label: '已停止',
    dot: 'bg-fg-subtle',
    badge: 'bg-fg-subtle/10 text-fg-muted border-border',
  },
  UNKNOWN: {
    label: '未知',
    dot: 'bg-fg-subtle',
    badge: 'bg-fg-subtle/10 text-fg-muted border-border',
  },
};

const STATUS_OPTIONS: { value: TorrentStatus | ''; label: string }[] = [
  { value: '', label: '全部状态' },
  { value: 'SEEDING', label: '做种中' },
  { value: 'LEECHING', label: '下载中' },
  { value: 'COMPLETED', label: '已完成' },
  { value: 'STOPPED', label: '已停止' },
];

// ── Page ─────────────────────────────────────────────────────────────────────

export default function TorrentsPage() {
  const [accountId, setAccountId] = useState('');
  const [status, setStatus] = useState<TorrentStatus | ''>('');
  const [page, setPage] = useState(1);
  const { data: settings } = useSettings();

  const { data: result, isLoading, isError, refetch, isFetching } = useTorrents({
    accountId: accountId || undefined,
    status: (status as TorrentStatus) || undefined,
    page,
    limit: settings?.defaultPageSize ?? 20,
  });
  const { data: accounts } = useAccounts();
  const syncTorrents = useTriggerTorrentSync();

  const totalPages = result ? Math.ceil(result.total / result.limit) : 1;

  function handleFilterChange() {
    setPage(1); // 筛选变化时重置页码
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* Heading */}
      <div className="page-header">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold text-fg">种子列表</h2>
          <p className="mt-0.5 text-sm text-fg-subtle">
            {result ? `共 ${result.total} 条` : '加载中...'}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => void refetch()} className="btn-toolbar flex-1 sm:flex-none">
            <RefreshCw className={cn('h-3.5 w-3.5', isFetching && 'animate-spin')} />
            刷新
          </button>
          {/* 未选账户时保持禁用态，但仍是可聚焦的 button（原来是 span，键盘拿不到） */}
          <button
            onClick={() => accountId && void syncTorrents.mutateAsync(accountId)}
            disabled={!accountId || syncTorrents.isPending}
            title={accountId ? undefined : '请先在筛选器选择账户'}
            className="btn-accent flex-1 sm:flex-none"
          >
            {syncTorrents.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Files className="h-3.5 w-3.5" />
            )}
            {syncTorrents.isPending ? '同步中…' : '同步种子'}
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <div className="flex items-center gap-2">
          <Filter className="h-3.5 w-3.5 shrink-0 text-fg-subtle" />
          {/* Account filter */}
          <select
            value={accountId}
            onChange={(e) => { setAccountId(e.target.value); handleFilterChange(); }}
            className="field sm:w-auto"
          >
            <option value="">全部账户</option>
            {accounts?.map((a) => (
              <option key={a.id} value={a.id}>
                {a.accountName}
              </option>
            ))}
          </select>
        </div>
        {/* Status filter —— 窄屏横向滚动，不再挤压 */}
        <div className="seg-bar">
          {STATUS_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => { setStatus(opt.value); handleFilterChange(); }}
              className={cn(
                'seg-btn',
                status === opt.value ? 'seg-btn-active' : 'seg-btn-idle',
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      {isLoading ? (
        <TorrentsSkeleton />
      ) : isError ? (
        <div className="flex min-h-[300px] flex-col items-center justify-center gap-3">
          <AlertTriangle className="h-8 w-8 text-danger" />
          <p className="text-sm text-fg-subtle">加载失败</p>
          <button onClick={() => void refetch()} className="btn-toolbar">
            重试
          </button>
        </div>
      ) : !result || result.data.length === 0 ? (
        <EmptyState accountId={accountId} />
      ) : (
        <>
          {/* 窄屏：卡片列表（表格 7 列在手机上必然横向溢出） */}
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 md:hidden">
            {result.data.map((torrent) => (
              <TorrentCard key={torrent.id} torrent={torrent} showAccount={!accountId} />
            ))}
          </div>

          {/* md 起：表格（此时内容区 ≥720px，容得下 640px 的表） */}
          <div className="bento-card hidden overflow-hidden p-0 md:block">
            <div className="overflow-x-auto">
              <table className="w-full min-w-[640px] text-xs">
                <thead>
                  <tr className="border-b border-border bg-bg-elevated text-fg-subtle">
                    <th className="px-4 py-2.5 text-left font-medium">种子名称</th>
                    <th className="px-4 py-2.5 text-right font-medium">大小</th>
                    <th className="px-4 py-2.5 text-right font-medium">已上传</th>
                    <th className="px-4 py-2.5 text-right font-medium">分享率</th>
                    <th className="px-4 py-2.5 text-right font-medium">做种时间</th>
                    <th className="px-4 py-2.5 text-left font-medium">状态</th>
                    <th className="px-4 py-2.5 text-right font-medium">最后活动</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {result.data.map((torrent) => (
                    <TorrentRow key={torrent.id} torrent={torrent} showAccount={!accountId} />
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-3">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="btn-toolbar min-w-[88px]"
              >
                上一页
              </button>
              <span className="shrink-0 text-xs tabular-nums text-fg-subtle">
                {page} / {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="btn-toolbar min-w-[88px]"
              >
                下一页
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ── TorrentCard（窄屏） ───────────────────────────────────────────────────────

function TorrentCard({ torrent, showAccount }: { torrent: TorrentDto; showAccount: boolean }) {
  const cfg = STATUS_CONFIG[torrent.status];
  const seedHours = Math.round(torrent.seedTimeSecs / 3600);

  return (
    <div className="data-card">
      {/* 名称 */}
      <p className="line-clamp-2 break-anywhere text-sm font-medium leading-snug text-fg">
        {torrent.name}
      </p>

      {/* 次要信息行 */}
      <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-fg-subtle">
        {showAccount && (
          <>
            <span className="max-w-[45%] truncate">{torrent.accountName}</span>
            <span aria-hidden>·</span>
          </>
        )}
        <span className="tabular-nums">{formatBytes(BigInt(torrent.sizeBytes))}</span>
        <span aria-hidden>·</span>
        <span className="tabular-nums">做种 {seedHours > 0 ? `${seedHours}h` : '-'}</span>
      </div>

      {/* 关键指标 */}
      <div className="mt-2.5 grid grid-cols-2 gap-2">
        <div className="metric">
          <p className="metric-label">已上传</p>
          <p className="metric-value">{formatBytes(BigInt(torrent.uploadedBytes))}</p>
        </div>
        <div className="metric">
          <p className="metric-label">分享率</p>
          <p className="metric-value">{formatRatio(torrent.ratio)}</p>
        </div>
      </div>

      {/* 状态 + 最后活动 */}
      <div className="mt-2.5 flex items-center justify-between gap-2 text-xs">
        <span className="flex min-w-0 items-center gap-1.5">
          <span className={cn('h-1.5 w-1.5 shrink-0 rounded-full', cfg.dot)} />
          <span className="truncate text-fg-muted">{cfg.label}</span>
        </span>
        <span className="shrink-0 text-fg-subtle">
          {torrent.lastActivityAt ? formatRelativeTime(torrent.lastActivityAt) : '-'}
        </span>
      </div>
    </div>
  );
}

// ── TorrentRow（md 起的表格行） ───────────────────────────────────────────────

function TorrentRow({ torrent, showAccount }: { torrent: TorrentDto; showAccount: boolean }) {
  const cfg = STATUS_CONFIG[torrent.status];
  const seedHours = Math.round(torrent.seedTimeSecs / 3600);

  return (
    <tr className="transition-colors hover:bg-bg-elevated">
      <td className="px-4 py-2.5">
        <div className="max-w-[260px]">
          <p className="truncate font-medium text-fg">{torrent.name}</p>
          {showAccount && (
            <p className="truncate text-fg-subtle">{torrent.accountName}</p>
          )}
        </div>
      </td>
      <td className="px-4 py-2.5 text-right tabular-nums text-fg-muted">
        {formatBytes(BigInt(torrent.sizeBytes))}
      </td>
      <td className="px-4 py-2.5 text-right tabular-nums text-fg-muted">
        {formatBytes(BigInt(torrent.uploadedBytes))}
      </td>
      <td className="px-4 py-2.5 text-right tabular-nums text-fg-muted">
        {formatRatio(torrent.ratio)}
      </td>
      <td className="px-4 py-2.5 text-right tabular-nums text-fg-muted">
        {seedHours > 0 ? `${seedHours}h` : '-'}
      </td>
      <td className="px-4 py-2.5">
        <span className={cn('rounded-full border px-2 py-0.5 text-[10px] font-medium', cfg.badge)}>
          {cfg.label}
        </span>
      </td>
      <td className="px-4 py-2.5 text-right text-fg-subtle">
        {torrent.lastActivityAt ? formatRelativeTime(torrent.lastActivityAt) : '-'}
      </td>
    </tr>
  );
}

// ── EmptyState ────────────────────────────────────────────────────────────────

function EmptyState({ accountId }: { accountId: string }) {
  return (
    <div className="flex min-h-[300px] flex-col items-center justify-center gap-3 px-4 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-accent/10">
        <Files className="h-6 w-6 text-accent" />
      </div>
      <p className="text-sm font-medium text-fg">暂无种子数据</p>
      <p className="text-xs text-fg-subtle">
        {accountId
          ? '点击右上角「同步种子」按钮拉取数据'
          : '选择账户后点击「同步种子」拉取数据'}
      </p>
    </div>
  );
}

// ── Skeleton ──────────────────────────────────────────────────────────────────

function TorrentsSkeleton() {
  return (
    <div className="space-y-2">
      <div className="h-10 animate-pulse rounded-lg bg-bg-card" />
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="h-28 animate-pulse rounded-xl bg-bg-card md:h-12 md:rounded-lg" />
      ))}
    </div>
  );
}
