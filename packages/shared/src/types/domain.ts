import { SiteStatus, TorrentStatus, HnrStatus, TrackerErrorCode } from '../enums/index.js';

// ─── 统一领域模型 — 所有 Provider 必须映射到这里 ─────────────────────────────
// 这些类型与任何 PT 站点的 API 字段名无关

export interface ConnectionResult {
  success: boolean;
  latencyMs?: number;
  /** 连接成功时返回基础 profile（用于验证 Token 有效性）*/
  profile?: Pick<TrackerProfile, 'externalUserId' | 'username'>;
  error?: TrackerError;
}

export interface TrackerProfile {
  siteCode: string;         // 'MTEAM'
  externalUserId: string;   // 站点侧 user ID
  username: string;
  email?: string;
  joinedAt: Date;
  roleId?: number;
  levelName?: string;       // 等级显示名（如 "Power User"）
  avatarUrl?: string;
  title?: string;           // 自定义头衔
  isParked: boolean;        // 账户是否暂停
  isEnabled: boolean;
}

export interface TrackerStats {
  uploadBytes: bigint;
  downloadBytes: bigint;
  ratio: number;            // -1 表示无穷大（未达到下载量下限）
  bonus: number;            // 魔力值
  seedingCount: number;
  seedingBytes: bigint;
  leechingCount: number;
  hitAndRunCount: number;
  isWarned: boolean;
  isVip: boolean;
  isDonor: boolean;
  lastLoginAt?: Date;
  lastTrackerAt?: Date;
  snapshotTime: Date;
  /** 时魔值（bonus/h），来自 M-Team /tracker/mybonus formulaParams.finalBs */
  bonusHourlyRate?: number;
}

export interface TrackerTorrent {
  siteTorrentId: string;
  name: string;
  sizeBytes: bigint;
  status: TorrentStatus;
  uploadedBytes: bigint;
  downloadedBytes: bigint;
  ratio: number;
  seedTimeSecs: number;
  leechTimeSecs: number;
  completedAt?: Date;
  lastActivityAt?: Date;
}

export interface BonusStats {
  current: number;
  /** 每小时产出速率（若站点提供）*/
  hourlyRate?: number;
  updatedAt: Date;
}

export interface HitAndRun {
  siteTorrentId: string;
  torrentName?: string;
  detectedAt: Date;
  status: HnrStatus;
  resolvedAt?: Date;
}

export interface MessageStats {
  unreadCount: number;
  checkedAt: Date;
}

export interface TrackerSiteStatusResult {
  status: SiteStatus;
  latencyMs?: number;
  checkedAt: Date;
  message?: string;
}

// ─── 错误 ─────────────────────────────────────────────────────────────────────

export interface TrackerError {
  code: TrackerErrorCode;
  message: string;
  /** 不得包含任何凭据信息 */
  details?: Record<string, unknown>;
}
