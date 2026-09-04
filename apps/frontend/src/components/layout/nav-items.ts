import {
  LayoutDashboard,
  Globe,
  Files,
  HardDrive,
  Sparkles,
  BarChart3,
  Bell,
  Download,
  Settings,
  Search,
  Sprout,
  type LucideIcon,
} from 'lucide-react';

export interface NavItem {
  href: string;
  /** 侧边栏 / 抽屉里的完整标签 */
  label: string;
  /** 底部标签栏里的短标签（宽度只有 ~70px） */
  shortLabel: string;
  icon: LucideIcon;
  /** 是否常驻手机底部标签栏 */
  primary?: boolean;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { href: '/',            label: '仪表盘',   shortLabel: '仪表盘', icon: LayoutDashboard, primary: true },
  { href: '/search',      label: '种子搜索', shortLabel: '搜索',   icon: Search,          primary: true },
  { href: '/fake-seed',   label: '保种管理', shortLabel: '保种',   icon: Sprout,          primary: true },
  { href: '/sites',       label: '站点',     shortLabel: '站点',   icon: Globe,           primary: true },
  { href: '/torrents',    label: '种子',     shortLabel: '种子',   icon: Files },
  { href: '/seeding',     label: '做种',     shortLabel: '做种',   icon: HardDrive },
  { href: '/bonus',       label: '魔力值',   shortLabel: '魔力',   icon: Sparkles },
  { href: '/stats',       label: '统计',     shortLabel: '统计',   icon: BarChart3 },
  { href: '/alerts',      label: '告警',     shortLabel: '告警',   icon: Bell },
  { href: '/downloaders', label: '下载器',   shortLabel: '下载器', icon: Download },
  { href: '/settings',    label: '设置',     shortLabel: '设置',   icon: Settings },
] as const;

/** 手机底部标签栏常驻项（其余收进「更多」抽屉） */
export const PRIMARY_NAV_ITEMS = NAV_ITEMS.filter((i) => i.primary);

/** 根路径要精确匹配，否则每一页都会高亮 */
export function isNavActive(pathname: string, href: string): boolean {
  return href === '/' ? pathname === '/' : pathname.startsWith(href);
}

/** 当前路径对应的页面标题，用于手机顶栏 */
export function navTitleFor(pathname: string): string | undefined {
  return NAV_ITEMS.find((i) => isNavActive(pathname, i.href))?.label;
}
