/**
 * 工具函数 — 前后端共用
 * 后端只用 formatBytes / formatRatio（日志用）；前端 UI 展示用完整版本
 */
/**
 * 将 bytes 数值格式化为可读字符串
 * @example formatBytes(35_280_000_000_000n) // "35.28 TB"
 */
export declare function formatBytes(bytes: bigint | number, decimals?: number): string;
/**
 * 将分享率格式化为字符串
 * ratio = -1 表示无穷大（见 M-Team MemberCount.shareRate min:-1）
 */
export declare function formatRatio(ratio: number): string;
/**
 * 将魔力值格式化（带千分位）
 * @example formatBonus(852320) // "852,320"
 */
export declare function formatBonus(bonus: number): string;
/**
 * 将秒数格式化为"Xd Xh"形式
 * @example formatSeedTime(86400 + 3600) // "1d 1h"
 */
export declare function formatSeedTime(seconds: number): string;
/**
 * 计算两个 snapshot 之间的增量
 * @returns 正值（增长）或负值（减少），单位与输入一致
 */
export declare function calcGrowth(current: bigint, previous: bigint): bigint;
