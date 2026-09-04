'use client';

import { useState } from 'react';
import {
  Loader2,
  RefreshCw,
  Plug,
  Pencil,
  Trash2,
  CheckCircle2,
  XCircle,
  ShieldAlert,
  Clock,
} from 'lucide-react';
import {
  useDeleteAccount,
  useTestConnection,
  useTriggerSync,
} from '@/hooks/use-accounts';
import { formatBytes, formatRatio, formatBonus, formatRelativeTime } from '@/lib/formatters';
import type { Account } from '@/types/api';

interface AccountCardProps {
  account: Account;
  onEdit: (account: Account) => void;
}

/** 账户状态 → 展示样式 */
const STATUS_META: Record<Account['status'], { label: string; dot: string; text: string }> = {
  ACTIVE: { label: '正常', dot: 'bg-success', text: 'text-success' },
  INACTIVE: { label: '未同步', dot: 'bg-fg-subtle', text: 'text-fg-subtle' },
  CREDENTIAL_INVALID: { label: 'Key 失效', dot: 'bg-danger', text: 'text-danger' },
  SYNC_ERROR: { label: '同步异常', dot: 'bg-warning', text: 'text-warning' },
};

export function AccountCard({ account, onEdit }: AccountCardProps) {
  const testMutation = useTestConnection();
  const syncMutation = useTriggerSync();
  const deleteMutation = useDeleteAccount();

  const [testResult, setTestResult] = useState<
    { ok: boolean; message: string } | null
  >(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const status = STATUS_META[account.status];
  const stats = account.stats;

  async function handleTest() {
    setTestResult(null);
    try {
      const result = await testMutation.mutateAsync(account.id);
      setTestResult(
        result.success
          ? { ok: true, message: `连接正常${result.latencyMs ? ` · ${result.latencyMs}ms` : ''}` }
          : { ok: false, message: result.error?.message ?? '连接失败' },
      );
    } catch (err) {
      setTestResult({ ok: false, message: (err as Error)?.message ?? '连接失败' });
    }
  }

  async function handleDelete() {
    try {
      await deleteMutation.mutateAsync(account.id);
    } catch {
      setConfirmDelete(false);
    }
  }

  return (
    <div className="bento-card flex flex-col gap-4">
      {/* ── Header: name + status ── */}
      <div className="flex items-start justify-between">
        <div className="flex min-w-0 items-center gap-3">
          {account.avatarUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={account.avatarUrl}
              alt=""
              className="h-9 w-9 shrink-0 rounded-full border border-border object-cover"
            />
          ) : (
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-accent/15 text-sm font-semibold text-accent">
              {account.accountName.slice(0, 1).toUpperCase()}
            </div>
          )}
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-fg">{account.accountName}</p>
            <p className="truncate text-xs text-fg-subtle">
              {account.username ? `@${account.username}` : account.siteName}
              {stats?.levelName && ` · ${stats.levelName}`}
            </p>
          </div>
        </div>
        <span className={`flex shrink-0 items-center gap-1.5 text-xs ${status.text}`}>
          <span className={`h-1.5 w-1.5 rounded-full ${status.dot}`} />
          {status.label}
        </span>
      </div>

      {/* ── Stats grid ── */}
      {stats ? (
        <div className="grid grid-cols-2 gap-x-2 gap-y-3 rounded-lg bg-bg-elevated/50 p-3 sm:grid-cols-4 sm:gap-2">
          <Metric label="上传" value={formatBytes(BigInt(stats.uploadBytes))} />
          <Metric label="下载" value={formatBytes(BigInt(stats.downloadBytes))} />
          <Metric label="分享率" value={formatRatio(stats.ratio)} />
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold tabular-nums text-fg">
              {formatBonus(stats.bonus)}
            </p>
            {stats.bonusHourlyRate != null ? (
              <p className="truncate text-[10px] tabular-nums text-accent">
                +{formatBonus(stats.bonusHourlyRate)}/h
              </p>
            ) : (
              <p className="text-xs text-fg-subtle">魔力</p>
            )}
          </div>
        </div>
      ) : (
        <div className="rounded-lg bg-bg-elevated/50 p-3 text-center text-xs text-fg-subtle">
          尚未同步，点击「同步」拉取数据
        </div>
      )}

      {/* ── 下一等级进度 ── */}
      {stats?.roleId != null && account.joinedAt && (
        <LevelProgress
          roleId={stats.roleId}
          downloadBytes={stats.downloadBytes}
          ratio={stats.ratio}
          joinedAt={account.joinedAt}
        />
      )}

      {/* ── Badges row ── */}
      {stats && (stats.isVip || stats.isDonor || stats.isWarned) && (
        <div className="flex flex-wrap gap-1.5">
          {stats.isVip && (
            <span className="rounded-full bg-accent-violet/15 px-2 py-0.5 text-xs text-accent-violet">
              VIP
            </span>
          )}
          {stats.isDonor && (
            <span className="rounded-full bg-warning/15 px-2 py-0.5 text-xs text-warning">
              捐赠者
            </span>
          )}
          {stats.isWarned && (
            <span className="flex items-center gap-1 rounded-full bg-danger/15 px-2 py-0.5 text-xs text-danger">
              <ShieldAlert className="h-3 w-3" />
              被警告
            </span>
          )}
        </div>
      )}

      {/* ── Test result banner ── */}
      {testResult && (
        <div
          className={`flex items-center gap-1.5 break-anywhere rounded-lg px-3 py-2 text-xs ${
            testResult.ok
              ? 'bg-success/10 text-success'
              : 'bg-danger/10 text-danger'
          }`}
        >
          {testResult.ok ? (
            <CheckCircle2 className="h-3.5 w-3.5 shrink-0" />
          ) : (
            <XCircle className="h-3.5 w-3.5 shrink-0" />
          )}
          {testResult.message}
        </div>
      )}

      {/* ── Footer: last sync + actions ── */}
      <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 border-t border-border pt-3">
        <span className="min-w-0 truncate text-xs text-fg-subtle">
          {account.lastSyncAt ? `同步于 ${formatRelativeTime(account.lastSyncAt)}` : '从未同步'}
        </span>

        {confirmDelete ? (
          <div className="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto">
            <span className="text-xs text-danger">确认删除？</span>
            <button
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
              className="btn-danger"
            >
              {deleteMutation.isPending ? '删除中' : '删除'}
            </button>
            <button
              onClick={() => setConfirmDelete(false)}
              className="btn-toolbar"
            >
              取消
            </button>
          </div>
        ) : (
          <div className="flex shrink-0 items-center gap-1">
            <IconButton
              title="测试连接"
              onClick={handleTest}
              loading={testMutation.isPending}
              icon={Plug}
            />
            <IconButton
              title="立即同步"
              onClick={() => syncMutation.mutate(account.id)}
              loading={syncMutation.isPending}
              icon={RefreshCw}
            />
            <IconButton title="编辑" onClick={() => onEdit(account)} icon={Pencil} />
            <IconButton
              title="删除"
              onClick={() => setConfirmDelete(true)}
              icon={Trash2}
              danger
            />
          </div>
        )}
      </div>
    </div>
  );
}

// ── Sub-components ─────────────────────────────────────────────────────────────

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="truncate text-sm font-semibold tabular-nums text-fg">{value}</p>
      <p className="text-xs text-fg-subtle">{label}</p>
    </div>
  );
}

function IconButton({
  title,
  onClick,
  icon: Icon,
  loading = false,
  danger = false,
}: {
  title: string;
  onClick: () => void;
  icon: typeof Plug;
  loading?: boolean;
  danger?: boolean;
}) {
  return (
    <button
      title={title}
      onClick={onClick}
      disabled={loading}
      className={`icon-btn ${
        danger ? 'hover:bg-danger/10 hover:text-danger' : ''
      }`}
    >
      {loading ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : (
        <Icon className="h-4 w-4" />
      )}
    </button>
  );
}

// ── 下一等级进度 ───────────────────────────────────────────────────────────────

const LEVEL_NAMES: Record<number, string> = {
  1: '小卒', 2: '捕头', 3: '知县', 4: '通判', 5: '知州',
  6: '府丞', 7: '府尹', 8: '总督', 9: '大臣',
};

/** 达到该等级所需的门槛（key = 目标等级 roleId）*/
const LEVEL_REQS: Record<number, { weeks: number; downloadGB: number; ratio: number }> = {
  2: { weeks: 4,  downloadGB: 200,  ratio: 2.0 },
  3: { weeks: 8,  downloadGB: 400,  ratio: 3.0 },
  4: { weeks: 12, downloadGB: 500,  ratio: 4.0 },
  5: { weeks: 16, downloadGB: 800,  ratio: 5.0 },
  6: { weeks: 20, downloadGB: 1000, ratio: 6.0 },
  7: { weeks: 24, downloadGB: 2000, ratio: 7.0 },
  8: { weeks: 28, downloadGB: 2500, ratio: 8.0 },
  9: { weeks: 32, downloadGB: 3000, ratio: 9.0 },
};

function LevelProgress({
  roleId, downloadBytes, ratio, joinedAt,
}: {
  roleId: number;
  downloadBytes: string;
  ratio: number;
  joinedAt: string;
}) {
  const nextRoleId = roleId + 1;
  const nextReq = LEVEL_REQS[nextRoleId];
  if (!nextReq) return null; // 已是最高等级

  const nextName = LEVEL_NAMES[nextRoleId] ?? `等级${nextRoleId}`;
  const currentGB = Number(BigInt(downloadBytes) / BigInt(1024 * 1024 * 1024));
  const weeksSinceJoin = Math.floor(
    (Date.now() - new Date(joinedAt).getTime()) / (7 * 24 * 3600 * 1000),
  );

  const timeOk  = weeksSinceJoin >= nextReq.weeks;
  const dlOk    = currentGB >= nextReq.downloadGB;
  const ratioOk = ratio >= nextReq.ratio;

  const timeGap  = Math.max(0, nextReq.weeks - weeksSinceJoin);
  const dlGap    = Math.max(0, nextReq.downloadGB - currentGB);
  const ratioGap = parseFloat(Math.max(0, nextReq.ratio - ratio).toFixed(2));

  return (
    <div className="rounded-lg bg-bg-elevated/50 px-3 py-2">
      <p className="mb-1.5 text-[10px] font-medium uppercase tracking-wide text-fg-subtle">
        距{nextName}
      </p>
      <div className="flex flex-wrap gap-x-4 gap-y-1">
        <LevelCond ok={timeOk}
          label={timeOk ? '注册时间 ✓' : `注册还差 ${timeGap} 周`} />
        <LevelCond ok={dlOk}
          label={dlOk ? '下载量 ✓' : `下载还差 ${dlGap.toFixed(0)} GB`} />
        <LevelCond ok={ratioOk}
          label={ratioOk ? '分享率 ✓' : `分享率差 ${ratioGap}`} />
      </div>
    </div>
  );
}

function LevelCond({ ok, label }: { ok: boolean; label: string }) {
  return (
    <span className={`flex items-center gap-1 text-xs ${ok ? 'text-success' : 'text-warning'}`}>
      {ok
        ? <CheckCircle2 className="h-3 w-3 shrink-0" />
        : <Clock className="h-3 w-3 shrink-0" />}
      {label}
    </span>
  );
}
