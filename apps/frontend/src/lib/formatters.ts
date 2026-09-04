/**
 * 前端格式化工具
 * 复用 @pt-manager/shared 的基础实现，并添加前端专用逻辑
 */
export { formatBytes, formatRatio, formatBonus, formatSeedTime, calcGrowth } from '@pt-manager/shared';

import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';

/** 格式化相对时间，如 "2 分钟前" */
export function formatRelativeTime(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return formatDistanceToNow(d, { addSuffix: true, locale: zhCN });
}

/** 将 bytes 增量格式化为带符号的字符串，如 "+182 GB" / "-1.2 GB" */
export function formatGrowth(bytes: number | bigint): string {
  const n = typeof bytes === 'bigint' ? Number(bytes) : bytes;
  if (n === 0) return '—';
  const sign = n > 0 ? '+' : '−';
  const { formatBytes } = require('@pt-manager/shared');
  return `${sign}${formatBytes(Math.abs(n))}`;
}

/** 格式化魔力增量，如 "+12,328" */
export function formatBonusGrowth(delta: number): string {
  if (delta === 0) return '—';
  const sign = delta > 0 ? '+' : '−';
  return `${sign}${Math.abs(delta).toLocaleString('en-US', { maximumFractionDigits: 0 })}`;
}
