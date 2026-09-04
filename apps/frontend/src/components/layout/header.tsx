'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { LogOut, Moon, RefreshCw, Search, Sun, User } from 'lucide-react';
import { useTheme } from 'next-themes';
import { useQueryClient } from '@tanstack/react-query';
import { triggerSyncAll } from '@/lib/api/accounts';
import { queryKeys } from '@/lib/query-keys';
import { cn } from '@/lib/utils';
import { useSettings } from '@/hooks/use-settings';
import { navTitleFor } from './nav-items';
import { useAuth, useLogout } from '@/hooks/use-auth';

interface HeaderProps {
  title?: string;
}

export function Header({ title }: HeaderProps) {
  const { theme, setTheme } = useTheme();
  const qc = useQueryClient();
  const pathname = usePathname();
  const router = useRouter();
  const { data: settings } = useSettings();
  const { data: user } = useAuth();
  const logout = useLogout();
  // 避免服务端/客户端 hydration 不匹配：挂载前不渲染依赖 theme 的内容
  const [mounted, setMounted] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  useEffect(() => setMounted(true), []);

  const handleSyncAll = async () => {
    if (isSyncing) return;
    setIsSyncing(true);
    try {
      await triggerSyncAll();
      // 3秒后刷新数据（给后端处理时间）
      setTimeout(() => {
        void qc.invalidateQueries({ queryKey: queryKeys.accounts() });
        void qc.invalidateQueries({ queryKey: ['dashboard'] });
        setIsSyncing(false);
      }, 3000);
    } catch (err) {
      console.error('Sync all failed:', err);
      setIsSyncing(false);
    }
  };

  const systemName = settings?.systemName ?? 'PT Manager';
  // 手机上侧边栏不可见，顶栏兼任「我在哪一页」的指示
  const pageTitle = title ?? navTitleFor(pathname) ?? systemName;

  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-2 border-b border-border bg-bg-card px-3 sm:px-6">
      {/* Left: 手机显示 logo + 当前页名，lg 起只留页名（logo 在侧边栏） */}
      <div className="flex min-w-0 items-center gap-2.5">
        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-accent text-white lg:hidden">
          <span className="text-xs font-bold">PT</span>
        </div>
        <div className="min-w-0">
          <h1 className="truncate text-sm font-semibold text-fg lg:text-fg-muted">{pageTitle}</h1>
          <p className="truncate text-[10px] leading-none text-fg-subtle lg:hidden">{systemName}</p>
        </div>
      </div>

      {/* Right: actions */}
      <div className="flex shrink-0 items-center gap-0.5 sm:gap-2">
        {/* 搜索 —— 直接进搜索页，而不是一个不响应的占位按钮 */}
        <Link href="/search" className="icon-btn" aria-label="种子搜索">
          <Search className="h-4 w-4" />
        </Link>

        {/* Manual Sync All */}
        <button
          onClick={handleSyncAll}
          disabled={isSyncing}
          className="icon-btn"
          aria-label="同步所有账户"
        >
          <RefreshCw className={cn('h-4 w-4', isSyncing && 'animate-spin')} />
        </button>

        {/* Theme toggle: Sun = 当前深色（点击切浅），Moon = 当前浅色（点击切深） */}
        <button
          onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
          className="icon-btn"
          aria-label={mounted && theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'}
        >
          {mounted && theme === 'dark' ? (
            <Sun className="h-4 w-4" />
          ) : (
            <Moon className="h-4 w-4" />
          )}
        </button>

        {/* User —— 手机上省掉，底部抽屉里有设置入口 */}
        <button
          type="button"
          onClick={() => logout.mutate(undefined, { onSuccess: () => router.replace('/login') })}
          disabled={logout.isPending}
          className={cn(
            'hidden h-10 w-10 shrink-0 items-center justify-center rounded-full',
            'bg-accent/10 text-accent transition-colors hover:bg-accent/20',
            'sm:flex sm:h-8 sm:w-8',
          )}
          aria-label="退出登录"
          title={user?.email ? `当前用户：${user.email}，点击退出登录` : '退出登录'}
        >
          {logout.isPending ? <LogOut className="h-4 w-4 animate-pulse" /> : <User className="h-4 w-4" />}
        </button>
      </div>
    </header>
  );
}
