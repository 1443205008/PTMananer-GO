'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import { useSettings } from '@/hooks/use-settings';
import { NAV_ITEMS, isNavActive } from './nav-items';

/** 桌面侧边栏。lg 以下隐藏，导航交给 <MobileNav /> 的底部标签栏。 */
export function Sidebar() {
  const pathname = usePathname();
  const { data: settings } = useSettings();

  return (
    <aside className="hidden h-full w-56 shrink-0 flex-col border-r border-border bg-bg-card lg:flex">
      {/* Logo */}
      <div className="flex h-14 shrink-0 items-center gap-2.5 border-b border-border px-4">
        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-accent text-white">
          <span className="text-xs font-bold">PT</span>
        </div>
        <span className="truncate font-semibold tracking-tight text-fg">
          {settings?.systemName ?? 'PT Manager'}
        </span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto p-2 py-3">
        <ul className="space-y-0.5">
          {NAV_ITEMS.map(({ href, label, icon: Icon }) => {
            const isActive = isNavActive(pathname, href);
            return (
              <li key={href}>
                <Link
                  href={href}
                  aria-current={isActive ? 'page' : undefined}
                  className={cn(
                    'flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-accent/10 text-accent'
                      : 'text-fg-muted hover:bg-bg-elevated hover:text-fg',
                  )}
                >
                  <Icon
                    className={cn(
                      'h-4 w-4 shrink-0',
                      isActive ? 'text-accent' : 'text-fg-subtle',
                    )}
                  />
                  {label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Footer */}
      <div className="shrink-0 border-t border-border p-3">
        <p className="text-center text-xs text-fg-subtle">Phase 1 · M-Team</p>
      </div>
    </aside>
  );
}
