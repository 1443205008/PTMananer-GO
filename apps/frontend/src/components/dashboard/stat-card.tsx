'use client';

import { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

interface StatCardProps {
  title: string;
  value: string;
  subtitle?: string;
  growth?: number | null;      // percentage change, e.g. 3.5 or -1.2
  growthLabel?: string;
  icon?: LucideIcon;
  accent?: boolean;
  className?: string;
}

export function StatCard({
  title,
  value,
  subtitle,
  growth,
  growthLabel,
  icon: Icon,
  accent = false,
  className,
}: StatCardProps) {
  const growthSign =
    growth == null ? 'neutral' : growth > 0 ? 'up' : growth < 0 ? 'down' : 'neutral';

  return (
    <div
      className={cn(
        'bento-card flex flex-col justify-between gap-3',
        accent && 'border-accent/20 bg-accent/5',
        className,
      )}
    >
      {/* Top row: title + icon */}
      <div className="flex items-start justify-between">
        <span className="text-xs font-medium uppercase tracking-wide text-fg-subtle">
          {title}
        </span>
        {Icon && (
          <span
            className={cn(
              'flex h-7 w-7 items-center justify-center rounded-lg',
              accent ? 'bg-accent/20 text-accent' : 'bg-bg-elevated text-fg-muted',
            )}
          >
            <Icon className="h-3.5 w-3.5" />
          </span>
        )}
      </div>

      {/* Value —— 手机上 2 列排布只有 ~160px，字号先收一档再放大 */}
      <div className="min-w-0">
        <p className="truncate text-xl font-bold tabular-nums tracking-tight text-fg sm:text-2xl">
          {value}
        </p>
        {subtitle && <p className="mt-0.5 truncate text-xs text-fg-subtle">{subtitle}</p>}
      </div>

      {/* Growth badge */}
      {growth != null && (
        <div className="flex items-center gap-1.5">
          <span className={`growth-badge-${growthSign}`}>
            {growthSign === 'up' ? '▲' : growthSign === 'down' ? '▼' : '─'}
            {' '}
            {Math.abs(growth).toFixed(1)}%
          </span>
          {growthLabel && (
            <span className="text-xs text-fg-subtle">{growthLabel}</span>
          )}
        </div>
      )}
    </div>
  );
}
