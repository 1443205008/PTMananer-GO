'use client';

import { useState } from 'react';
import {
  Bell,
  BellOff,
  CheckCheck,
  X,
  AlertTriangle,
  Info,
  Zap,
  ShieldAlert,
} from 'lucide-react';
import {
  useAlerts,
  useDismissAlert,
  useMarkAlertRead,
  useMarkAllAlertsRead,
} from '@/hooks/use-alerts';
import { formatRelativeTime } from '@/lib/formatters';
import { cn } from '@/lib/utils';
import type { AlertDto, AlertSeverity, AlertType } from '@/types/api';

// ── Severity 配置 ─────────────────────────────────────────────────────────────

const SEVERITY_CONFIG: Record<
  AlertSeverity,
  { label: string; icon: React.ElementType; dot: string; badge: string }
> = {
  CRITICAL: {
    label: '严重',
    icon: ShieldAlert,
    dot: 'bg-danger',
    badge: 'bg-danger/10 text-danger border-danger/20',
  },
  ERROR: {
    label: '错误',
    icon: AlertTriangle,
    dot: 'bg-danger',
    badge: 'bg-danger/10 text-danger border-danger/20',
  },
  WARNING: {
    label: '警告',
    icon: AlertTriangle,
    dot: 'bg-warning',
    badge: 'bg-warning/10 text-warning border-warning/20',
  },
  INFO: {
    label: '信息',
    icon: Info,
    dot: 'bg-info',
    badge: 'bg-info/10 text-info border-info/20',
  },
};

const TYPE_LABEL: Record<AlertType, string> = {
  HNR_DETECTED: 'H&R 检测',
  AUTH_FAILED: 'API Key 失效',
  SYNC_FAILED: '同步失败',
  SITE_OFFLINE: '站点离线',
  API_CHANGED: 'API 变更',
  RATE_LIMITED: '请求频率限制',
};

// ── Page ─────────────────────────────────────────────────────────────────────

export default function AlertsPage() {
  const [unreadOnly, setUnreadOnly] = useState(false);
  const { data: alerts, isLoading, isError, refetch } = useAlerts(unreadOnly);
  const markRead = useMarkAlertRead();
  const markAll = useMarkAllAlertsRead();
  const dismiss = useDismissAlert();

  const unreadCount = alerts?.filter((a) => !a.isRead).length ?? 0;

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* Heading */}
      <div className="page-header">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold text-fg">告警中心</h2>
          <p className="mt-0.5 text-sm text-fg-subtle">
            {unreadCount > 0 ? `${unreadCount} 条未读` : '全部已读'}
            {alerts && ` · 共 ${alerts.length} 条`}
          </p>
        </div>
        <div className="flex items-center gap-2 sm:shrink-0">
          {/* Unread filter toggle */}
          <button
            onClick={() => setUnreadOnly((v) => !v)}
            className={cn(
              'btn-toolbar',
              unreadOnly &&
                'border-accent/30 bg-accent/10 text-accent hover:bg-accent/10 hover:text-accent',
            )}
          >
            <BellOff className="h-3.5 w-3.5" />
            仅未读
          </button>
          {/* Mark all read */}
          <button
            onClick={() => void markAll.mutateAsync()}
            disabled={markAll.isPending || unreadCount === 0}
            className="btn-toolbar"
          >
            <CheckCheck className="h-3.5 w-3.5" />
            全部已读
          </button>
        </div>
      </div>

      {/* Content */}
      {isLoading ? (
        <AlertsSkeleton />
      ) : isError ? (
        <div className="flex min-h-[300px] flex-col items-center justify-center gap-3">
          <AlertTriangle className="h-8 w-8 text-danger" />
          <p className="text-sm text-fg-subtle">加载失败</p>
          <button onClick={() => void refetch()} className="btn-toolbar">
            重试
          </button>
        </div>
      ) : !alerts || alerts.length === 0 ? (
        <div className="flex min-h-[300px] flex-col items-center justify-center gap-3 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-success/10">
            <Bell className="h-6 w-6 text-success" />
          </div>
          <p className="text-sm font-medium text-fg">暂无告警</p>
          <p className="text-xs text-fg-subtle">{unreadOnly ? '切换到「全部」查看历史记录' : '一切正常 🎉'}</p>
        </div>
      ) : (
        <div className="space-y-2">
          {alerts.map((alert) => (
            <AlertRow
              key={alert.id}
              alert={alert}
              onMarkRead={() => void markRead.mutateAsync(alert.id)}
              onDismiss={() => void dismiss.mutateAsync(alert.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ── AlertRow ─────────────────────────────────────────────────────────────────

function AlertRow({
  alert,
  onMarkRead,
  onDismiss,
}: {
  alert: AlertDto;
  onMarkRead: () => void;
  onDismiss: () => void;
}) {
  const cfg = SEVERITY_CONFIG[alert.severity];
  const Icon = cfg.icon;

  return (
    <div
      className={cn(
        'bento-card flex items-start gap-3 p-3',
        !alert.isRead && 'border-l-2 border-l-accent',
      )}
    >
      {/* Severity icon */}
      <div
        className={cn(
          'mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg',
          alert.severity === 'CRITICAL' || alert.severity === 'ERROR'
            ? 'bg-danger/10'
            : alert.severity === 'WARNING'
              ? 'bg-warning/10'
              : 'bg-info/10',
        )}
      >
        <Icon
          className={cn(
            'h-3.5 w-3.5',
            alert.severity === 'CRITICAL' || alert.severity === 'ERROR'
              ? 'text-danger'
              : alert.severity === 'WARNING'
                ? 'text-warning'
                : 'text-info',
          )}
        />
      </div>

      {/* Content */}
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 flex-wrap items-center gap-1.5">
          <span className="min-w-0 max-w-full truncate text-sm font-medium text-fg">
            {alert.title}
          </span>
          <span
            className={cn(
              'rounded-full border px-1.5 py-0.5 text-[10px] font-medium',
              cfg.badge,
            )}
          >
            {TYPE_LABEL[alert.type] ?? alert.type}
          </span>
          {!alert.isRead && (
            <span className="h-1.5 w-1.5 rounded-full bg-accent" />
          )}
        </div>
        <p className="break-anywhere mt-0.5 text-xs text-fg-muted">{alert.message}</p>
        <div className="mt-1 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-fg-subtle">
          {alert.accountName && (
            <span className="break-anywhere">{alert.accountName}</span>
          )}
          {alert.accountName && <span aria-hidden="true">·</span>}
          <span className="whitespace-nowrap">{formatRelativeTime(alert.createdAt)}</span>
        </div>
      </div>

      {/* Actions */}
      <div className="-mr-1 flex shrink-0 items-center gap-0.5 sm:mr-0 sm:gap-1">
        {!alert.isRead && (
          <button onClick={onMarkRead} title="标记已读" className="icon-btn">
            <Zap className="h-4 w-4" />
          </button>
        )}
        <button
          onClick={onDismiss}
          title="忽略"
          className="icon-btn hover:text-danger"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

// ── Skeleton ──────────────────────────────────────────────────────────────────

function AlertsSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="h-20 animate-pulse rounded-xl bg-bg-card" />
      ))}
    </div>
  );
}
