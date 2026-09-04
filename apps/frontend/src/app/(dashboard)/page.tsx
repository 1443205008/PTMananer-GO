'use client';

import Link from 'next/link';
import {
  ArrowUpDown,
  HardDrive,
  TrendingUp,
  Zap,
  AlertTriangle,
  Globe,
  Plus,
  RefreshCw,
} from 'lucide-react';
import { StatCard } from '@/components/dashboard/stat-card';
import { TrendChart, type TrendChartPoint } from '@/components/dashboard/trend-chart';
import { useDashboard } from '@/hooks/use-dashboard';
import { formatBytes, formatRatio, formatBonus, formatRelativeTime } from '@/lib/formatters';
import type { Account, TrendPoint } from '@/types/api';

export default function DashboardPage() {
  const { data, isLoading, isError, error, refetch, isFetching } = useDashboard();

  if (isLoading) return <DashboardSkeleton />;

  if (isError) {
    return (
      <ErrorState
        message={(error as Error)?.message ?? '无法连接后端服务'}
        onRetry={() => void refetch()}
      />
    );
  }

  const summary = data!.summary;

  // 无账户 → 引导添加
  if (summary.accountCount === 0) {
    return <EmptyState />;
  }

  // ── 趋势数据映射（bytes → GB 数值，tooltip/轴用 formatBytes 还原）──
  const uploadTrend: TrendChartPoint[] = data!.trend.map((p: TrendPoint) => ({
    date: p.date,
    value: Number(BigInt(p.uploadBytes)),
  }));
  const bonusTrend: TrendChartPoint[] = data!.trend.map((p: TrendPoint) => ({
    date: p.date,
    value: p.bonus,
  }));

  // 增长率：取趋势最后两日的环比（累计值，环比反映当日净增）
  const uploadGrowth = dailyGrowth(data!.trend.map((p) => Number(BigInt(p.uploadBytes))));
  const bonusGrowth = dailyGrowth(data!.trend.map((p) => p.bonus));

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* ── Page heading ── */}
      <div className="page-header">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold text-fg">总览</h2>
          <p className="mt-0.5 text-sm text-fg-subtle">
            {summary.accountCount} 个账户 · {summary.siteCount} 个站点
            {summary.lastSyncAt && ` · 最近同步 ${formatRelativeTime(summary.lastSyncAt)}`}
          </p>
        </div>
        <button onClick={() => void refetch()} className="btn-toolbar">
          <RefreshCw className={`h-3.5 w-3.5 ${isFetching ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {/* ── Bento Grid — row 1: 4 key metrics ── */}
      <div className="grid grid-cols-2 gap-3 sm:gap-4 sm:grid-cols-4">
        <StatCard
          title="总上传"
          value={formatBytes(BigInt(summary.totalUploadBytes))}
          growth={uploadGrowth}
          growthLabel="vs 昨日"
          icon={ArrowUpDown}
          accent
        />
        <StatCard
          title="总体分享率"
          value={formatRatio(summary.overallRatio)}
          icon={TrendingUp}
        />
        <StatCard
          title="魔力值"
          value={formatBonus(summary.totalBonus)}
          growth={bonusGrowth}
          growthLabel="vs 昨日"
          icon={Zap}
        />
        <StatCard
          title="做种数"
          value={summary.totalSeedingCount.toString()}
          subtitle={formatBytes(BigInt(summary.totalSeedingBytes))}
          icon={HardDrive}
        />
      </div>

      {/* ── Bento Grid — row 2: secondary metrics + status ── */}
      <div className="grid grid-cols-2 gap-3 sm:gap-4 sm:grid-cols-3">
        <StatCard
          title="H&R 警告"
          value={summary.totalHitAndRun === 0 ? '无' : summary.totalHitAndRun.toString()}
          subtitle={summary.totalHitAndRun === 0 ? '状态良好' : '需要处理'}
          icon={AlertTriangle}
          className={summary.totalHitAndRun > 0 ? 'border-warning/30 bg-warning/5' : ''}
        />
        <StatCard
          title="总下载"
          value={formatBytes(BigInt(summary.totalDownloadBytes))}
          icon={Globe}
        />
        <div className="bento-card col-span-2 sm:col-span-1">
          <p className="text-xs font-medium uppercase tracking-wide text-fg-subtle">账户状态</p>
          <div className="mt-3 space-y-1.5">
            {data!.accounts.map((acc: Account) => (
              <SyncRow
                key={acc.id}
                label={acc.accountName}
                status={statusToDot(acc.status)}
                lastSync={acc.lastSyncAt ? formatRelativeTime(acc.lastSyncAt) : '从未同步'}
              />
            ))}
          </div>
        </div>
      </div>

      {/* ── Bento Grid — row 3: trend charts ── */}
      <div className="grid grid-cols-1 gap-3 sm:gap-4 sm:grid-cols-2">
        <div className="bento-card flex min-h-[190px] flex-col sm:min-h-[220px]">
          <p className="mb-3 text-xs font-medium uppercase tracking-wide text-fg-subtle">
            上传趋势 (近 30 天)
          </p>
          <TrendChart
            data={uploadTrend}
            gradientId="upload-grad"
            color="#6366F1"
            formatValue={(v) => formatBytes(v)}
          />
        </div>
        <div className="bento-card flex min-h-[190px] flex-col sm:min-h-[220px]">
          <p className="mb-3 text-xs font-medium uppercase tracking-wide text-fg-subtle">
            魔力值趋势 (近 30 天)
          </p>
          <TrendChart
            data={bonusTrend}
            gradientId="bonus-grad"
            color="#8B5CF6"
            formatValue={(v) => formatBonus(v)}
          />
        </div>
      </div>
    </div>
  );
}

// ── Helpers ──────────────────────────────────────────────────────────────────

/** 取序列最后两点的环比增长率（%）；不足两点或前值为 0 时返回 null */
function dailyGrowth(series: number[]): number | null {
  if (series.length < 2) return null;
  const prev = series[series.length - 2];
  const last = series[series.length - 1];
  if (prev === 0) return null;
  return ((last - prev) / prev) * 100;
}

function statusToDot(status: Account['status']): 'synced' | 'syncing' | 'error' | 'never' {
  switch (status) {
    case 'ACTIVE':
      return 'synced';
    case 'CREDENTIAL_INVALID':
    case 'SYNC_ERROR':
      return 'error';
    case 'INACTIVE':
    default:
      return 'never';
  }
}

// ── Sub-components ─────────────────────────────────────────────────────────────

function SyncRow({
  label,
  status,
  lastSync,
}: {
  label: string;
  status: 'synced' | 'syncing' | 'error' | 'never';
  lastSync?: string;
}) {
  const dot: Record<typeof status, string> = {
    synced: 'bg-success',
    syncing: 'bg-warning animate-pulse',
    error: 'bg-danger',
    never: 'bg-fg-subtle',
  };
  return (
    <div className="flex items-center justify-between text-xs">
      <div className="flex min-w-0 items-center gap-2">
        <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${dot[status]}`} />
        <span className="truncate text-fg-muted">{label}</span>
      </div>
      <span className="shrink-0 text-fg-subtle">{lastSync}</span>
    </div>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="h-10 w-40 animate-pulse rounded-lg bg-bg-card sm:w-48" />
      <div className="grid grid-cols-2 gap-3 sm:gap-4 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-28 animate-pulse rounded-xl bg-bg-card" />
        ))}
      </div>
      <div className="grid grid-cols-1 gap-3 sm:gap-4 sm:grid-cols-2">
        {Array.from({ length: 2 }).map((_, i) => (
          <div key={i} className="h-44 animate-pulse rounded-xl bg-bg-card sm:h-56" />
        ))}
      </div>
    </div>
  );
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center gap-4 px-4 text-center">
      <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-danger/10">
        <AlertTriangle className="h-6 w-6 text-danger" />
      </div>
      <div className="min-w-0">
        <p className="text-sm font-medium text-fg">加载失败</p>
        <p className="mt-1 break-anywhere text-xs text-fg-subtle">{message}</p>
      </div>
      <button onClick={onRetry} className="btn-toolbar px-4">
        <RefreshCw className="h-3.5 w-3.5" />
        重试
      </button>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center gap-4 px-4 text-center">
      <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-accent/10">
        <Globe className="h-6 w-6 text-accent" />
      </div>
      <div className="min-w-0">
        <p className="text-sm font-medium text-fg">还没有接入任何账户</p>
        <p className="mt-1 text-xs text-fg-subtle">添加你的第一个 M-Team 账户开始监控</p>
      </div>
      <Link href="/sites" className="btn-accent px-4">
        <Plus className="h-3.5 w-3.5" />
        添加账户
      </Link>
    </div>
  );
}
