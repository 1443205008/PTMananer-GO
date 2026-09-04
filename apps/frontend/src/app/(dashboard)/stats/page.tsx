'use client';

import React from 'react';
import { ArrowUpRight, ArrowDownRight, TrendingUp, HardDrive, Zap, Sprout } from 'lucide-react';
import { useDashboard } from '@/hooks/use-dashboard';
import { useAccounts } from '@/hooks/use-accounts';
import { TrendChart, type TrendChartPoint } from '@/components/dashboard/trend-chart';
import { formatBytes, formatRatio, formatBonus, formatRelativeTime } from '@/lib/formatters';
import type { TrendPoint, Account } from '@/types/api';

export default function StatsPage() {
  const { data: dashboard, isLoading } = useDashboard();
  const { data: accounts } = useAccounts();

  if (isLoading) return <StatsSkeleton />;

  const summary = dashboard?.summary;
  const trend: TrendPoint[] = dashboard?.trend ?? [];

  // 趋势数据 → TrendChartPoint（上传/下载转为 GB 便于图表展示）
  const uploadTrend: TrendChartPoint[] = trend.map((p) => ({
    date: p.date,
    value: Number(BigInt(p.uploadBytes) / BigInt(1024 * 1024 * 1024)),
  }));
  const downloadTrend: TrendChartPoint[] = trend.map((p) => ({
    date: p.date,
    value: Number(BigInt(p.downloadBytes) / BigInt(1024 * 1024 * 1024)),
  }));
  const bonusTrend: TrendChartPoint[] = trend.map((p) => ({ date: p.date, value: p.bonus }));
  const seedingTrend: TrendChartPoint[] = trend.map((p) => ({ date: p.date, value: p.seedingCount }));

  // 日增量（最近两条快照差值）
  const latest = trend.at(-1);
  const prev = trend.at(-2);
  const uploadDelta  = latest && prev ? Number(BigInt(latest.uploadBytes)  - BigInt(prev.uploadBytes))  : null;
  const downloadDelta= latest && prev ? Number(BigInt(latest.downloadBytes)- BigInt(prev.downloadBytes)): null;
  const bonusDelta   = latest && prev ? latest.bonus    - prev.bonus    : null;

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* 标题 */}
      <div className="min-w-0">
        <h2 className="text-lg font-semibold text-fg">统计分析</h2>
        <p className="mt-0.5 text-sm text-fg-subtle">
          基于每日快照 · {trend.length} 天数据
          {summary?.lastSyncAt && ` · 最后同步 ${formatRelativeTime(summary.lastSyncAt)}`}
        </p>
      </div>

      {/* 概览卡片 */}
      <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4">
        <StatCard
          label="总上传"
          value={formatBytes(BigInt(summary?.totalUploadBytes ?? '0'))}
          delta={uploadDelta !== null ? formatBytes(BigInt(Math.abs(uploadDelta))) : null}
          deltaPositive={uploadDelta !== null && uploadDelta >= 0}
          icon={<ArrowUpRight className="h-4 w-4" />}
          color="text-success"
          bgColor="bg-success/10"
        />
        <StatCard
          label="总下载"
          value={formatBytes(BigInt(summary?.totalDownloadBytes ?? '0'))}
          delta={downloadDelta !== null ? formatBytes(BigInt(Math.abs(downloadDelta))) : null}
          deltaPositive={downloadDelta !== null && downloadDelta >= 0}
          icon={<ArrowDownRight className="h-4 w-4" />}
          color="text-blue-400"
          bgColor="bg-blue-400/10"
        />
        <StatCard
          label="综合分享率"
          value={formatRatio(summary?.overallRatio ?? 0)}
          icon={<TrendingUp className="h-4 w-4" />}
          color="text-accent"
          bgColor="bg-accent/10"
        />
        <StatCard
          label="总魔力值"
          value={formatBonus(summary?.totalBonus ?? 0)}
          delta={bonusDelta !== null ? `+${formatBonus(bonusDelta)}` : null}
          deltaPositive={bonusDelta !== null && bonusDelta >= 0}
          icon={<Zap className="h-4 w-4" />}
          color="text-accent-violet"
          bgColor="bg-accent-violet/10"
        />
      </div>

      {/* 趋势图 2×2 */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ChartCard title="上传量趋势" subtitle="单位 GB">
          <TrendChart data={uploadTrend} gradientId="stats-upload" color="#22C55E"
            formatValue={(v) => `${v.toFixed(0)} GB`} />
        </ChartCard>
        <ChartCard title="下载量趋势" subtitle="单位 GB">
          <TrendChart data={downloadTrend} gradientId="stats-download" color="#60A5FA"
            formatValue={(v) => `${v.toFixed(0)} GB`} />
        </ChartCard>
        <ChartCard title="魔力值趋势" subtitle="累计总量">
          <TrendChart data={bonusTrend} gradientId="stats-bonus" color="#8B5CF6"
            formatValue={(v) => formatBonus(v)} />
        </ChartCard>
        <ChartCard title="做种数量趋势" subtitle="活跃种子数">
          <TrendChart data={seedingTrend} gradientId="stats-seeding" color="#F59E0B"
            formatValue={(v) => `${v} 个`} />
        </ChartCard>
      </div>

      {/* 账户明细 —— 窄屏卡片，md 起表格 */}
      {accounts && accounts.length > 0 && (
        <div className="space-y-2.5 md:hidden">
          <p className="text-xs font-medium uppercase tracking-wide text-fg-subtle">账户明细</p>
          {accounts.map((a) => (
            <AccountStatCard key={a.id} account={a} />
          ))}
          {accounts.length > 1 && summary && (
            <div className="data-card border-accent/25 bg-accent/5">
              <p className="text-xs font-semibold uppercase tracking-wide text-fg-muted">合计</p>
              <div className="mt-2.5 grid grid-cols-2 gap-2">
                <div className="metric">
                  <p className="metric-label">上传</p>
                  <p className="metric-value text-success">
                    {formatBytes(BigInt(summary.totalUploadBytes))}
                  </p>
                </div>
                <div className="metric">
                  <p className="metric-label">下载</p>
                  <p className="metric-value text-blue-400">
                    {formatBytes(BigInt(summary.totalDownloadBytes))}
                  </p>
                </div>
                <div className="metric">
                  <p className="metric-label">分享率</p>
                  <p className="metric-value text-accent">{formatRatio(summary.overallRatio)}</p>
                </div>
                <div className="metric">
                  <p className="metric-label">魔力值</p>
                  <p className="metric-value text-accent-violet">
                    {formatBonus(summary.totalBonus)}
                  </p>
                </div>
              </div>
              <p className="mt-2 text-xs text-fg-subtle">
                做种 <span className="tabular-nums text-fg-muted">{summary.totalSeedingCount}</span> 个
              </p>
            </div>
          )}
        </div>
      )}

      {/* 账户明细表（md 起） */}
      {accounts && accounts.length > 0 && (
        <div className="bento-card hidden overflow-hidden md:block">
          <p className="mb-4 text-xs font-medium uppercase tracking-wide text-fg-subtle">账户明细</p>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[560px] text-sm">
              <thead>
                <tr className="border-b border-border text-xs text-fg-subtle">
                  <th className="pb-2 text-left font-medium">账户</th>
                  <th className="pb-2 text-right font-medium">上传</th>
                  <th className="pb-2 text-right font-medium">下载</th>
                  <th className="pb-2 text-right font-medium">分享率</th>
                  <th className="pb-2 text-right font-medium">做种</th>
                  <th className="pb-2 text-right font-medium">魔力值</th>
                  <th className="pb-2 text-right font-medium">时魔值</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {accounts.map((a) => <AccountRow key={a.id} account={a} />)}
              </tbody>
              {accounts.length > 1 && summary && (
                <tfoot>
                  <tr className="border-t-2 border-border text-xs font-semibold">
                    <td className="pt-3 text-fg-subtle">合计</td>
                    <td className="pt-3 text-right tabular-nums text-success">{formatBytes(BigInt(summary.totalUploadBytes))}</td>
                    <td className="pt-3 text-right tabular-nums text-blue-400">{formatBytes(BigInt(summary.totalDownloadBytes))}</td>
                    <td className="pt-3 text-right tabular-nums text-accent">{formatRatio(summary.overallRatio)}</td>
                    <td className="pt-3 text-right tabular-nums text-fg">{summary.totalSeedingCount}</td>
                    <td className="pt-3 text-right tabular-nums text-accent-violet">{formatBonus(summary.totalBonus)}</td>
                    <td className="pt-3" />
                  </tr>
                </tfoot>
              )}
            </table>
          </div>
        </div>
      )}

      {/* 次要统计 —— MiniStat 是横向布局，2 列在 375px 上放不下图标+数值 */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 sm:gap-4 lg:grid-cols-4">
        <MiniStat label="账户数"   value={String(summary?.accountCount ?? 0)}  icon={<HardDrive className="h-3.5 w-3.5" />} />
        <MiniStat label="活跃账户" value={String(summary?.activeCount ?? 0)}   icon={<TrendingUp className="h-3.5 w-3.5" />} />
        <MiniStat label="做种总量" value={formatBytes(BigInt(summary?.totalSeedingBytes ?? '0'))} icon={<Sprout className="h-3.5 w-3.5" />} />
        <MiniStat label="H&R 数"  value={String(summary?.totalHitAndRun ?? 0)} icon={<Zap className="h-3.5 w-3.5" />} danger={(summary?.totalHitAndRun ?? 0) > 0} />
      </div>
    </div>
  );
}

// ── 子组件 ────────────────────────────────────────────────────────────────────

function StatCard({ label, value, delta, deltaPositive, icon, color, bgColor }: {
  label: string; value: string; delta?: string | null; deltaPositive?: boolean;
  icon: React.ReactNode; color: string; bgColor: string;
}) {
  return (
    <div className="bento-card flex flex-col gap-3">
      <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${bgColor} ${color}`}>{icon}</div>
      <div className="min-w-0">
        <p className="truncate text-xs text-fg-subtle">{label}</p>
        <p className="mt-0.5 truncate text-lg font-bold tabular-nums text-fg sm:text-xl">{value}</p>
        {delta != null
          ? <p className={`mt-0.5 flex items-center gap-0.5 text-xs tabular-nums ${deltaPositive ? 'text-success' : 'text-danger'}`}>
              {deltaPositive ? <ArrowUpRight className="h-3 w-3 shrink-0" /> : <ArrowDownRight className="h-3 w-3 shrink-0" />}
              <span className="truncate">{delta} 日增</span>
            </p>
          : <p className="mt-0.5 truncate text-xs text-fg-subtle">暂无对比数据</p>
        }
      </div>
    </div>
  );
}

function ChartCard({ title, subtitle, children }: {
  title: string; subtitle?: string; children: React.ReactNode;
}) {
  return (
    <div className="bento-card flex min-h-[220px] flex-col">
      <div className="mb-3 flex items-baseline gap-2">
        <p className="text-xs font-medium uppercase tracking-wide text-fg-subtle">{title}</p>
        {subtitle && <span className="text-[10px] text-fg-subtle">· {subtitle}</span>}
      </div>
      {children}
    </div>
  );
}

function AccountRow({ account }: { account: Account }) {
  const s = account.stats;
  if (!s) return (
    <tr className="text-fg-subtle">
      <td className="py-2.5">
        <p className="font-medium">{account.accountName}</p>
        <p className="text-xs">{account.siteName}</p>
      </td>
      <td colSpan={6} className="py-2.5 text-right text-xs">尚未同步</td>
    </tr>
  );
  return (
    <tr>
      <td className="py-2.5">
        <p className="font-medium text-fg">{account.accountName}</p>
        <p className="text-xs text-fg-subtle">{account.siteName}{s.levelName && ` · ${s.levelName}`}</p>
      </td>
      <td className="py-2.5 text-right tabular-nums text-success">{formatBytes(BigInt(s.uploadBytes))}</td>
      <td className="py-2.5 text-right tabular-nums text-blue-400">{formatBytes(BigInt(s.downloadBytes))}</td>
      <td className="py-2.5 text-right tabular-nums text-accent">{formatRatio(s.ratio)}</td>
      <td className="py-2.5 text-right tabular-nums text-fg">{s.seedingCount}</td>
      <td className="py-2.5 text-right tabular-nums text-accent-violet">{formatBonus(s.bonus)}</td>
      <td className="py-2.5 text-right tabular-nums">
        {s.bonusHourlyRate != null
          ? <span className="text-accent">+{formatBonus(s.bonusHourlyRate)}/h</span>
          : <span className="text-fg-subtle">—</span>}
      </td>
    </tr>
  );
}

/** 窄屏账户明细卡片 —— 替代 7 列表格行 */
function AccountStatCard({ account }: { account: Account }) {
  const s = account.stats;

  return (
    <div className="data-card">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-fg">{account.accountName}</p>
          <p className="truncate text-xs text-fg-subtle">
            {account.siteName}
            {s?.levelName && ` · ${s.levelName}`}
          </p>
        </div>
        {!s && <span className="shrink-0 text-xs text-fg-subtle">尚未同步</span>}
      </div>

      {s && (
        <>
          <div className="mt-2.5 grid grid-cols-2 gap-2">
            <div className="metric">
              <p className="metric-label">上传</p>
              <p className="metric-value text-success">{formatBytes(BigInt(s.uploadBytes))}</p>
            </div>
            <div className="metric">
              <p className="metric-label">下载</p>
              <p className="metric-value text-blue-400">{formatBytes(BigInt(s.downloadBytes))}</p>
            </div>
            <div className="metric">
              <p className="metric-label">分享率</p>
              <p className="metric-value text-accent">{formatRatio(s.ratio)}</p>
            </div>
            <div className="metric">
              <p className="metric-label">魔力值</p>
              <p className="metric-value text-accent-violet">{formatBonus(s.bonus)}</p>
            </div>
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-fg-subtle">
            <span>
              做种 <span className="tabular-nums text-fg-muted">{s.seedingCount}</span> 个
            </span>
            <span>
              时魔{' '}
              {s.bonusHourlyRate != null ? (
                <span className="tabular-nums text-accent">+{formatBonus(s.bonusHourlyRate)}/h</span>
              ) : (
                <span className="text-fg-subtle">—</span>
              )}
            </span>
          </div>
        </>
      )}
    </div>
  );
}

function MiniStat({ label, value, icon, danger }: {
  label: string; value: string; icon: React.ReactNode; danger?: boolean;
}) {
  return (
    <div className="bento-card flex items-center gap-3">
      <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${danger ? 'bg-danger/10 text-danger' : 'bg-bg-elevated text-fg-subtle'}`}>
        {icon}
      </div>
      <div className="min-w-0">
        <p className="truncate text-xs text-fg-subtle">{label}</p>
        <p className={`truncate text-sm font-semibold tabular-nums ${danger ? 'text-danger' : 'text-fg'}`}>{value}</p>
      </div>
    </div>
  );
}

function StatsSkeleton() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="h-10 w-40 animate-pulse rounded-lg bg-bg-card sm:w-48" />
      <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => <div key={i} className="h-28 animate-pulse rounded-xl bg-bg-card" />)}
      </div>
      <div className="grid grid-cols-1 gap-3 sm:gap-4 lg:grid-cols-2">
        {Array.from({ length: 4 }).map((_, i) => <div key={i} className="h-44 animate-pulse rounded-xl bg-bg-card sm:h-52" />)}
      </div>
    </div>
  );
}