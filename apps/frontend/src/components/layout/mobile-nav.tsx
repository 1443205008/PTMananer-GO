'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useRouter } from 'next/navigation';
import * as Dialog from '@radix-ui/react-dialog';
import { LayoutGrid, LogOut, X } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useMediaQuery } from '@/hooks/use-media-query';
import { useAuth, useLogout } from '@/hooks/use-auth';
import { NAV_ITEMS, PRIMARY_NAV_ITEMS, isNavActive } from './nav-items';

/**
 * 手机端底部标签栏：4 个常驻高频项 + 「更多」抽屉（全部 11 项）。
 * 作为 flex 列的最后一项渲染（不是 fixed），所以内容区不需要预留补偿内边距。
 * lg 断点以上整体隐藏，改由左侧 Sidebar 承担导航。
 */
export function MobileNav() {
  const pathname = usePathname();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  // 顶栏的退出按钮是 sm:flex，手机上不可见，所以退出入口放在这个抽屉里
  const { data: user } = useAuth();
  const logout = useLogout();

  // 路由变化后收起抽屉（也覆盖系统返回手势）
  useEffect(() => {
    setOpen(false);
  }, [pathname]);

  // 抽屉开着时旋转/拉宽到桌面：内容被 lg:hidden 藏起来但焦点和滚动锁还在，必须收掉
  const isDesktop = useMediaQuery('(min-width: 1024px)');
  useEffect(() => {
    if (isDesktop) setOpen(false);
  }, [isDesktop]);

  // 当前页不在常驻项里时，把「更多」点亮，避免底栏看起来毫无选中态
  const moreActive = !PRIMARY_NAV_ITEMS.some((i) => isNavActive(pathname, i.href));

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <nav
        className="safe-b shrink-0 border-t border-border bg-bg-card/95 backdrop-blur lg:hidden"
        aria-label="主导航"
      >
        <ul className="flex items-stretch">
          {PRIMARY_NAV_ITEMS.map(({ href, shortLabel, icon: Icon }) => {
            const active = isNavActive(pathname, href);
            return (
              <li key={href} className="min-w-0 flex-1">
                <Link
                  href={href}
                  aria-current={active ? 'page' : undefined}
                  className={cn(
                    'flex min-h-[56px] flex-col items-center justify-center gap-1 px-1 py-1.5',
                    'transition-colors',
                    active ? 'text-accent' : 'text-fg-subtle active:text-fg',
                  )}
                >
                  <span
                    className={cn(
                      'flex h-7 w-12 items-center justify-center rounded-full transition-colors',
                      active && 'bg-accent/10',
                    )}
                  >
                    <Icon className="h-5 w-5" />
                  </span>
                  <span className="w-full truncate text-center text-[10px] font-medium leading-none">
                    {shortLabel}
                  </span>
                </Link>
              </li>
            );
          })}

          <li className="min-w-0 flex-1">
            <Dialog.Trigger
              className={cn(
                'flex w-full min-h-[56px] flex-col items-center justify-center gap-1 px-1 py-1.5',
                'transition-colors',
                moreActive ? 'text-accent' : 'text-fg-subtle active:text-fg',
              )}
            >
              <span
                className={cn(
                  'flex h-7 w-12 items-center justify-center rounded-full transition-colors',
                  moreActive && 'bg-accent/10',
                )}
              >
                <LayoutGrid className="h-5 w-5" />
              </span>
              <span className="text-[10px] font-medium leading-none">更多</span>
            </Dialog.Trigger>
          </li>
        </ul>
      </nav>

      <Dialog.Portal>
        <Dialog.Overlay
          className={cn(
            'fixed inset-0 z-40 bg-black/60 backdrop-blur-sm lg:hidden',
            'data-[state=open]:animate-fade-in data-[state=closed]:animate-fade-out',
          )}
        />
        <Dialog.Content
          className={cn(
            'fixed inset-x-0 bottom-0 z-50 max-h-[85dvh] overflow-y-auto lg:hidden',
            'rounded-t-2xl border-t border-border bg-bg-card shadow-2xl outline-none',
            'pb-[calc(1rem+env(safe-area-inset-bottom))]',
            'data-[state=open]:animate-sheet-up data-[state=closed]:animate-sheet-down',
          )}
          // 抽屉只是导航网格，没有描述性正文，避免 Radix 的 a11y 警告
          aria-describedby={undefined}
        >
          {/* 抓手条 —— 暗示这是可下拉的抽屉 */}
          <div className="flex justify-center pb-1 pt-2.5">
            <span className="h-1 w-9 rounded-full bg-fg-subtle/40" />
          </div>

          <div className="flex items-center justify-between px-4 pb-2 pt-1">
            <Dialog.Title className="text-sm font-semibold text-fg">全部功能</Dialog.Title>
            <Dialog.Close className="icon-btn" aria-label="关闭">
              <X className="h-4 w-4" />
            </Dialog.Close>
          </div>

          <ul className="grid grid-cols-3 gap-2 px-3 pb-2">
            {NAV_ITEMS.map(({ href, label, icon: Icon }) => {
              const active = isNavActive(pathname, href);
              return (
                <li key={href}>
                  <Link
                    href={href}
                    aria-current={active ? 'page' : undefined}
                    className={cn(
                      'flex min-h-[76px] flex-col items-center justify-center gap-2 rounded-xl',
                      'border px-2 py-3 text-center transition-colors',
                      active
                        ? 'border-accent/30 bg-accent/10 text-accent'
                        : 'border-border bg-bg-elevated/40 text-fg-muted active:bg-bg-elevated',
                    )}
                  >
                    <Icon className={cn('h-5 w-5 shrink-0', !active && 'text-fg-subtle')} />
                    <span className="w-full truncate text-xs font-medium">{label}</span>
                  </Link>
                </li>
              );
            })}
          </ul>

          {/* 账户区 —— 手机上唯一的退出登录入口 */}
          <div className="mt-1 flex items-center justify-between gap-3 border-t border-border px-4 pt-3">
            <div className="min-w-0">
              <p className="text-[10px] uppercase tracking-wide text-fg-subtle">当前账户</p>
              <p className="truncate text-xs font-medium text-fg-muted">
                {user?.email ?? '未登录'}
              </p>
            </div>
            <button
              type="button"
              onClick={() =>
                logout.mutate(undefined, {
                  onSuccess: () => {
                    setOpen(false);
                    router.replace('/login');
                  },
                })
              }
              disabled={logout.isPending}
              className="btn-toolbar shrink-0"
            >
              <LogOut className={cn('h-3.5 w-3.5', logout.isPending && 'animate-pulse')} />
              退出登录
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
