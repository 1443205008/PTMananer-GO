/**
 * 工具函数 — 前后端共用
 * 后端只用 formatBytes / formatRatio（日志用）；前端 UI 展示用完整版本
 */

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'] as const;

/**
 * 将 bytes 数值格式化为可读字符串
 * @example formatBytes(35_280_000_000_000n) // "35.28 TB"
 */
export function formatBytes(bytes: bigint | number, decimals = 2): string {
  const n = typeof bytes === 'bigint' ? Number(bytes) : bytes;
  if (n === 0) return '0 B';
  if (n < 0) return `−${formatBytes(-n, decimals)}`;

  const k = 1024;
  const i = Math.min(Math.floor(Math.log(n) / Math.log(k)), UNITS.length - 1);
  const value = n / Math.pow(k, i);
  return `${value.toFixed(decimals)} ${UNITS[i]}`;
}

/**
 * 将分享率格式化为字符串
 * ratio = -1 表示无穷大（见 M-Team MemberCount.shareRate min:-1）
 */
export function formatRatio(ratio: number): string {
  if (ratio < 0) return '∞';
  return ratio.toFixed(2);
}

/**
 * 将魔力值格式化（带千分位）
 * @example formatBonus(852320) // "852,320"
 */
export function formatBonus(bonus: number): string {
  return bonus.toLocaleString('en-US', { maximumFractionDigits: 0 });
}

/**
 * 将秒数格式化为"Xd Xh"形式
 * @example formatSeedTime(86400 + 3600) // "1d 1h"
 */
export function formatSeedTime(seconds: number): string {
  if (seconds <= 0) return '0h';
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  if (d > 0) return `${d}d ${h}h`;
  const m = Math.floor((seconds % 3600) / 60);
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

/**
 * 计算两个 snapshot 之间的增量
 * @returns 正值（增长）或负值（减少），单位与输入一致
 */
export function calcGrowth(current: bigint, previous: bigint): bigint {
  return current - previous;
}
