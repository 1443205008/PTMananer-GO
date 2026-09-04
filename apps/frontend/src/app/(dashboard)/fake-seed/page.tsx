'use client';

import React, { useState } from 'react';
import { Sprout, Square, Trash2, Loader2, Clock, CheckCircle2, AlertCircle, RefreshCw } from 'lucide-react';
import { useFakeSeedJobs, useStopFakeSeed, useRemoveFakeSeed } from '@/hooks/use-fake-seed';
import { useAccounts } from '@/hooks/use-accounts';
import { formatBytes, formatRelativeTime } from '@/lib/formatters';
import { cn } from '@/lib/utils';
import type { FakeSeedJob } from '@/types/api';

const STATUS: Record<string, { label: string; icon: React.ReactNode; cls: string }> = {
  RUNNING: { label: '保种中', icon: <Sprout className="h-3.5 w-3.5" />, cls: 'text-success' },
  STOPPED: { label: '已停止', icon: <Square className="h-3.5 w-3.5" />,  cls: 'text-fg-subtle' },
  ERROR:   { label: '错误',   icon: <AlertCircle className="h-3.5 w-3.5" />, cls: 'text-danger' },
};

export default function FakeSeedPage() {
  const [accountId, setAccountId] = useState('');
  const { data: jobs, isLoading, refetch, isFetching } = useFakeSeedJobs(accountId || undefined);
  const { data: accounts } = useAccounts();

  const running = jobs?.filter((j) => j.status === 'RUNNING').length ?? 0;

  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="page-header">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold text-fg">保种管理</h2>
          <p className="mt-0.5 text-sm text-fg-subtle">
            {running} 个运行中 · 共 {jobs?.length ?? 0} 个任务
          </p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:shrink-0">
          <select
            value={accountId}
            onChange={(event) => setAccountId(event.target.value)}
            className="field sm:max-w-56"
          >
            <option value="">全部站点账号</option>
            {accounts?.map((account) => (
              <option key={account.id} value={account.id}>
                {account.siteName} · {account.accountName}
              </option>
            ))}
          </select>
          <button onClick={() => void refetch()} className="btn-toolbar">
            <RefreshCw className={cn('h-3.5 w-3.5', isFetching && 'animate-spin')} />
            刷新
          </button>
        </div>
      </div>

      {isLoading ? <FakeSeedSkeleton /> : !jobs?.length ? <EmptyState /> : (
        <div className="space-y-2">
          {jobs.map((job) => <JobRow key={job.id} job={job} />)}
        </div>
      )}
    </div>
  );
}

function JobRow({ job }: { job: FakeSeedJob }) {
  const stop = useStopFakeSeed();
  const remove = useRemoveFakeSeed();
  const cfg = STATUS[job.status] ?? STATUS.STOPPED;

  const nextIn = job.nextReportAt
    ? Math.max(0, Math.round((new Date(job.nextReportAt).getTime() - Date.now()) / 1000))
    : null;

  return (
    <div className="bento-card flex flex-col gap-2 p-3.5 sm:flex-row sm:items-center sm:gap-3 sm:p-4">
      <div className="flex min-w-0 flex-1 items-start gap-3 sm:items-center">
        <div className={cn('mt-0.5 shrink-0 sm:mt-0', cfg.cls)}>{cfg.icon}</div>
        <div className="flex-1 min-w-0">
          <p className="truncate text-sm font-medium text-fg">{job.torrentName}</p>
          <div className="flex flex-wrap gap-x-2 gap-y-0.5 text-xs text-fg-subtle mt-0.5 sm:gap-x-3">
            <span className={cn('whitespace-nowrap', cfg.cls)}>{cfg.label}</span>
            <span className="break-anywhere min-w-0 text-accent">
              {job.siteName ?? job.siteCode ?? '未知站点'} · {job.accountName ?? job.accountId}
            </span>
            <span className="whitespace-nowrap">{formatBytes(BigInt(job.totalSize))}</span>
            {job.lastReportAt && (
              <span className="whitespace-nowrap">上次上报 {formatRelativeTime(job.lastReportAt)}</span>
            )}
            {nextIn !== null && job.status === 'RUNNING' && (
              <span className="flex items-center gap-0.5 whitespace-nowrap">
                <Clock className="h-3 w-3 shrink-0" />{nextIn}s 后上报
              </span>
            )}
            {job.status === 'RUNNING' && (
              <span className="flex items-center gap-0.5 whitespace-nowrap text-success">
                <CheckCircle2 className="h-3 w-3 shrink-0" />每 {job.interval}s
              </span>
            )}
            {job.errorMessage && (
              <span className="break-anywhere line-clamp-2 w-full min-w-0 text-danger sm:w-auto">
                {job.errorMessage}
              </span>
            )}
          </div>
        </div>
      </div>
      <div className="-mr-1 flex shrink-0 items-center justify-end gap-0.5 sm:mr-0 sm:gap-1">
        {job.status === 'RUNNING' && (
          <button onClick={() => void stop.mutateAsync(job.id)}
            disabled={stop.isPending}
            title="停止保种"
            className="icon-btn hover:bg-warning/10 hover:text-warning">
            {stop.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Square className="h-4 w-4" />}
          </button>
        )}
        <button onClick={() => void remove.mutateAsync(job.id)}
          disabled={remove.isPending}
          title="删除任务"
          className="icon-btn hover:bg-danger/10 hover:text-danger">
          {remove.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
        </button>
      </div>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex min-h-[300px] flex-col items-center justify-center gap-3 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-success/10">
        <Sprout className="h-6 w-6 text-success" />
      </div>
      <p className="text-sm font-medium text-fg">暂无保种任务</p>
      <p className="text-xs text-fg-subtle">在搜索页面找到种子后点击「保种」按钮创建任务</p>
    </div>
  );
}

function FakeSeedSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="h-28 animate-pulse rounded-xl bg-bg-card sm:h-16" />
      ))}
    </div>
  );
}
