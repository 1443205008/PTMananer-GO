'use client';

import { HardDrive, ArrowUpDown, Layers } from 'lucide-react';
import { useAccounts } from '@/hooks/use-accounts';
import { formatBytes } from '@/lib/formatters';
import type { Account } from '@/types/api';

export default function SeedingPage() {
  const { data: accounts, isLoading } = useAccounts();

  const totalSeedingCount =
    accounts?.reduce((sum, a) => sum + (a.stats?.seedingCount ?? 0), 0) ?? 0;
  const totalSeedingBytes =
    accounts?.reduce((sum, a) => sum + BigInt(a.stats?.seedingBytes ?? '0'), BigInt(0)) ?? BigInt(0);
  const totalLeechingCount =
    accounts?.reduce((sum, a) => sum + (a.stats?.leechingCount ?? 0), 0) ?? 0;

  if (isLoading) return <SeedingSkeleton />;

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* Heading */}
      <div>
        <h2 className="text-lg font-semibold text-fg">做种概览</h2>
        <p className="mt-0.5 text-sm text-fg-subtle">{accounts?.length ?? 0} 个账户</p>
      </div>

      {/* Summary stats */}
      <div className="grid grid-cols-3 gap-2 sm:gap-4">
        <SummaryCard icon={HardDrive} label="做种总数" value={totalSeedingCount.toString()} />
        <SummaryCard icon={Layers} label="做种体积" value={formatBytes(totalSeedingBytes)} />
        <SummaryCard icon={ArrowUpDown} label="下载中" value={totalLeechingCount.toString()} />
      </div>

      {/* Per-account table */}
      <div className="bento-card overflow-hidden p-0">
        <div className="border-b border-border px-4 py-3">
          <p className="text-xs font-medium uppercase tracking-wide text-fg-subtle">
            各账户做种详情
          </p>
        </div>
        <div className="divide-y divide-border">
          {accounts?.map((account) => <SeedingRow key={account.id} account={account} />)}
          {accounts?.length === 0 && (
            <p className="px-4 py-8 text-center text-sm text-fg-subtle">暂无账户数据</p>
          )}
        </div>
      </div>
    </div>
  );
}

function SummaryCard({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
}) {
  return (
    <div className="bento-card flex min-w-0 flex-col gap-1 p-3 sm:p-4">
      <div className="flex min-w-0 items-center gap-1.5 text-fg-subtle">
        <Icon className="h-3.5 w-3.5 shrink-0" />
        <span className="truncate text-xs font-medium">{label}</span>
      </div>
      <p className="truncate text-base font-bold tabular-nums text-fg sm:text-xl">{value}</p>
    </div>
  );
}

function SeedingRow({ account }: { account: Account }) {
  const s = account.stats;
  return (
    <div className="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium text-fg">{account.accountName}</p>
        <p className="truncate text-xs text-fg-subtle">{account.siteName}</p>
      </div>
      <div className="grid grid-cols-3 gap-2 sm:flex sm:shrink-0 sm:items-center sm:gap-6 sm:text-right">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium tabular-nums text-fg">
            {s?.seedingCount ?? 0}
          </p>
          <p className="truncate text-xs text-fg-subtle">做种中</p>
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-medium tabular-nums text-fg">
            {formatBytes(BigInt(s?.seedingBytes ?? '0'))}
          </p>
          <p className="truncate text-xs text-fg-subtle">做种量</p>
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-medium tabular-nums text-fg">
            {s?.leechingCount ?? 0}
          </p>
          <p className="truncate text-xs text-fg-subtle">下载中</p>
        </div>
      </div>
    </div>
  );
}

function SeedingSkeleton() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="h-10 w-40 animate-pulse rounded-lg bg-bg-card" />
      <div className="grid grid-cols-3 gap-2 sm:gap-4">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-20 animate-pulse rounded-xl bg-bg-card" />
        ))}
      </div>
      <div className="h-64 animate-pulse rounded-xl bg-bg-card" />
    </div>
  );
}
