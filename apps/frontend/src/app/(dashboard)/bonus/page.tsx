'use client';

import { Sparkles } from 'lucide-react';
import { useAccounts } from '@/hooks/use-accounts';
import { useDashboard } from '@/hooks/use-dashboard';
import { TrendChart, type TrendChartPoint } from '@/components/dashboard/trend-chart';
import { formatBonus, formatRelativeTime } from '@/lib/formatters';
import type { Account, TrendPoint } from '@/types/api';

export default function BonusPage() {
  const { data: accounts, isLoading } = useAccounts();
  const { data: dashboard } = useDashboard();

  const totalBonus = accounts?.reduce((sum, a) => sum + (a.stats?.bonus ?? 0), 0) ?? 0;
  const bonusTrend: TrendChartPoint[] =
    dashboard?.trend.map((p: TrendPoint) => ({ date: p.date, value: p.bonus })) ?? [];

  if (isLoading) return <BonusSkeleton />;

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* Heading */}
      <div>
        <h2 className="text-lg font-semibold text-fg">魔力值</h2>
        <p className="mt-0.5 text-sm text-fg-subtle">
          {accounts?.length ?? 0} 个账户 · 总计 {formatBonus(totalBonus)}
        </p>
      </div>

      {/* Trend chart — 有数据才渲染 */}
      {bonusTrend.length > 0 && (
        <div className="bento-card flex min-h-[200px] flex-col">
          <p className="mb-3 text-xs font-medium uppercase tracking-wide text-fg-subtle">
            魔力值趋势（近 30 天）
          </p>
          <TrendChart
            data={bonusTrend}
            gradientId="bonus-page-grad"
            color="#8B5CF6"
            formatValue={(v) => formatBonus(v)}
          />
        </div>
      )}

      {/* Per-account cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {accounts?.map((account) => <BonusCard key={account.id} account={account} />)}
      </div>
    </div>
  );
}

function BonusCard({ account }: { account: Account }) {
  const bonus = account.stats?.bonus ?? 0;
  const level = account.stats?.levelName;
  const syncedAt = account.stats?.syncedAt;
  const hourlyRate = account.stats?.bonusHourlyRate;

  return (
    <div className="bento-card flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-accent/10">
            <Sparkles className="h-3.5 w-3.5 text-accent" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-fg">{account.accountName}</p>
            <p className="truncate text-xs text-fg-subtle">{account.siteName}</p>
          </div>
        </div>
        {level && (
          <span className="shrink-0 rounded-full border border-accent/20 bg-accent/10 px-2 py-0.5 text-xs text-accent">
            {level}
          </span>
        )}
      </div>
      <div className="min-w-0">
        <p className="truncate text-xl font-bold tracking-tight text-fg sm:text-2xl">
          {formatBonus(bonus)}
        </p>
        {hourlyRate != null && (
          <p className="mt-0.5 text-xs font-medium text-accent">
            +{formatBonus(hourlyRate)}/h · 时魔值
          </p>
        )}
        <p className="mt-0.5 text-xs text-fg-subtle">
          {syncedAt ? `更新于 ${formatRelativeTime(syncedAt)}` : '尚未同步'}
        </p>
      </div>
    </div>
  );
}

function BonusSkeleton() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="h-10 w-40 animate-pulse rounded-lg bg-bg-card" />
      <div className="h-48 animate-pulse rounded-xl bg-bg-card" />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 2 }).map((_, i) => (
          <div key={i} className="h-32 animate-pulse rounded-xl bg-bg-card" />
        ))}
      </div>
    </div>
  );
}
