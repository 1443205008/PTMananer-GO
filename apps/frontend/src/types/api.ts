import type { AccountStatus, SiteStatus, TrackerErrorCode } from '@pt-manager/shared';

// ─── 系统登录用户 ────────────────────────────────────────────────────────────
export interface AuthUser {
  id: string;
  email: string;
  name?: string | null;
  createdAt: string;
}

/**
 * 前端 API 契约类型
 *
 * 与后端 DTO 一一对应（apps/backend/src/**\/dto）。
 * 约定：
 * - 字节数为 string（后端 BigInt → string，防溢出），前端用 BigInt() 解析后格式化
 * - 比率 / 魔力值为 number
 * - 绝不包含任何凭据字段（encryptedApiKey / iv / authTag / 明文 API Key）
 */

// ─── 账户统计 ────────────────────────────────────────────────────────────────
export interface AccountStats {
  uploadBytes: string;
  downloadBytes: string;
  ratio: number;
  bonus: number;
  seedingCount: number;
  seedingBytes: string;
  leechingCount: number;
  hitAndRunCount: number;
  roleId?: number | null;
  levelName?: string | null;
  isWarned: boolean;
  isVip: boolean;
  isDonor: boolean;
  lastLoginAt?: string | null;
  lastTrackerAt?: string | null;
  siteStatus: SiteStatus;
  syncedAt?: string | null;
  /** 时魔值：基于最近两日快照差值计算（bonus/h），无历史快照时为 null */
  bonusHourlyRate?: number | null;
}

// ─── 账户 ────────────────────────────────────────────────────────────────────
export interface Account {
  id: string;
  siteCode: string;
  siteName: string;
  accountName: string;
  remark?: string | null;
  externalUserId?: string | null;
  username?: string | null;
  avatarUrl?: string | null;
  /** M-Team 注册时间（首次同步后填入）*/
  joinedAt?: string | null;
  isEnabled: boolean;
  syncEnabled: boolean;
  status: AccountStatus;
  lastSyncAt?: string | null;
  createdAt: string;
  updatedAt: string;
  stats?: AccountStats | null;
}

// ─── 账户创建 / 更新入参 ──────────────────────────────────────────────────────
export interface CreateAccountInput {
  siteCode: string;
  accountName: string;
  /** ⚠️ 仅在提交时使用，绝不回显、绝不持久化到前端状态 */
  apiKey: string;
  remark?: string;
}

export interface UpdateAccountInput {
  accountName?: string;
  remark?: string;
  isEnabled?: boolean;
  syncEnabled?: boolean;
  /** 轮换 API Key（可选） */
  apiKey?: string;
}

// ─── 连接测试结果 ─────────────────────────────────────────────────────────────
export interface ConnectionTestResult {
  success: boolean;
  latencyMs?: number;
  profile?: { externalUserId: string; username: string };
  error?: { code: TrackerErrorCode; message: string };
}

// ─── 仪表盘 ──────────────────────────────────────────────────────────────────
export interface DashboardSummary {
  accountCount: number;
  siteCount: number;
  activeCount: number;
  problemCount: number;
  totalUploadBytes: string;
  totalDownloadBytes: string;
  overallRatio: number;
  totalBonus: number;
  totalSeedingCount: number;
  totalSeedingBytes: string;
  totalHitAndRun: number;
  lastSyncAt?: string | null;
}

export interface TrendPoint {
  /** YYYY-MM-DD（UTC） */
  date: string;
  uploadBytes: string;
  downloadBytes: string;
  bonus: number;
  seedingCount: number;
}

export interface Dashboard {
  summary: DashboardSummary;
  accounts: Account[];
  trend: TrendPoint[];
}

// ─── 告警 ────────────────────────────────────────────────────────────────────
export type AlertSeverity = 'INFO' | 'WARNING' | 'ERROR' | 'CRITICAL';
export type AlertType =
  | 'HNR_DETECTED'
  | 'AUTH_FAILED'
  | 'SYNC_FAILED'
  | 'SITE_OFFLINE'
  | 'API_CHANGED'
  | 'RATE_LIMITED';

export interface AlertDto {
  id: string;
  accountId?: string | null;
  accountName?: string | null;
  type: AlertType;
  severity: AlertSeverity;
  title: string;
  message: string;
  isRead: boolean;
  isDismissed: boolean;
  createdAt: string;
  updatedAt: string;
}

// ─── 系统设置 ────────────────────────────────────────────────────────────────
export interface SystemSettings {
  systemName: string;
  timezone: string;
  defaultPageSize: number;
  syncEnabled: boolean;
  syncIntervalMinutes: number;
  alertEnabled: boolean;
  alertRetentionDays: number;
  syncLogRetentionDays: number;
  snapshotRetentionDays: number;
  updatedAt?: string | null;
}

export type UpdateSystemSettingsInput = Omit<SystemSettings, 'updatedAt'>;

export interface SettingsStatus {
  version: string;
  environment: string;
  database: 'connected' | 'error';
  redis: 'connected' | 'error';
  checkedAt: string;
}

export interface SettingsCleanupResult {
  syncJobs: number;
  snapshots: number;
  alerts: number;
  cleanedAt: string;
}

// ─── 保种任务 ──────────────────────────────────────────────────────────────────
export type FakeSeedStatus = 'RUNNING' | 'STOPPED' | 'ERROR';

export interface FakeSeedJob {
  id: string;
  accountId: string;
  accountName?: string | null;
  siteCode?: string | null;
  siteName?: string | null;
  torrentId: string;
  torrentName: string;
  infoHash: string;
  totalSize: string;
  status: FakeSeedStatus;
  interval: number;
  lastReportAt?: string | null;
  nextReportAt?: string | null;
  errorMessage?: string | null;
  createdAt: string;
}

// ─── 站内搜索 ─────────────────────────────────────────────────────────────────
export interface TorrentSearchItem {
  id: string;
  name: string;
  smallDescr?: string | null;
  /** bytes 字符串 */
  size: string;
  category?: string | null;
  createdDate?: string | null;
  seeders: number;
  leechers: number;
  timesCompleted: number;
  discount?: string | null;
  discountEndTime?: string | null;
  imdb?: string | null;
  /** 当前默认账户是否已经在该种子上做种 */
  isSeeding: boolean;
}

export interface TorrentSearchResponse {
  data: TorrentSearchItem[];
  total: number;
  page: number;
  pageSize: number;
}

export type SearchMode = 'normal' | 'adult' | 'movie' | 'music' | 'tvshow' | 'anime' | 'all';
export type SearchDiscount = 'FREE' | '_2X_FREE' | 'PERCENT_50' | '_2X' | 'NORMAL';
export type SearchSortField = 'CREATED_DATE' | 'SIZE' | 'SEEDERS' | 'LEECHERS' | 'TIMES_COMPLETED' | 'NAME';

export interface SearchTeam {
  id: number;
  name: string;
}
export type TorrentStatus = 'SEEDING' | 'LEECHING' | 'COMPLETED' | 'STOPPED' | 'UNKNOWN';

export interface TorrentDto {
  id: string;
  accountId: string;
  accountName: string;
  siteTorrentId: string;
  name: string;
  /** bytes，string 防溢出 */
  sizeBytes: string;
  status: TorrentStatus;
  uploadedBytes: string;
  downloadedBytes: string;
  ratio: number;
  seedTimeSecs: number;
  leechTimeSecs: number;
  completedAt?: string | null;
  lastActivityAt?: string | null;
  syncedAt?: string | null;
  createdAt: string;
}

export interface TorrentListResponse {
  data: TorrentDto[];
  total: number;
  page: number;
  limit: number;
}
