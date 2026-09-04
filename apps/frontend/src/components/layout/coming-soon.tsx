import type { LucideIcon } from 'lucide-react';

interface ComingSoonProps {
  title: string;
  description: string;
  icon: LucideIcon;
  /** 计划落地的阶段，如 "Phase 4" */
  phase?: string;
}

/**
 * 通用「即将上线」占位页
 *
 * 用于侧边栏已列出但尚未实现的路由，避免 404，
 * 并作为 in-app 路线图告知用户该功能所属阶段。
 */
export function ComingSoon({ title, description, icon: Icon, phase }: ComingSoonProps) {
  return (
    <div className="flex min-h-[420px] flex-col items-center justify-center gap-4 text-center">
      <div className="flex h-14 w-14 items-center justify-center rounded-2xl border border-border bg-bg-card">
        <Icon className="h-7 w-7 text-fg-muted" />
      </div>
      <div className="max-w-[280px] sm:max-w-sm">
        <div className="flex flex-wrap items-center justify-center gap-2">
          <h2 className="text-base font-semibold text-fg">{title}</h2>
          {phase && (
            <span className="shrink-0 rounded-full border border-accent/30 bg-accent/10 px-2 py-0.5 text-xs font-medium text-accent">
              {phase}
            </span>
          )}
        </div>
        <p className="mt-1.5 text-sm text-fg-subtle">{description}</p>
      </div>
    </div>
  );
}
