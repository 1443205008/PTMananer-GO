import { SiteStatus, TorrentStatus, HnrStatus, TrackerErrorCode } from '../enums/index.js';
export interface ConnectionResult {
    success: boolean;
    latencyMs?: number;
    /** 连接成功时返回基础 profile（用于验证 Token 有效性）*/
    profile?: Pick<TrackerProfile, 'externalUserId' | 'username'>;
    error?: TrackerError;
}
export interface TrackerProfile {
    siteCode: string;
    externalUserId: string;
    username: string;
    email?: string;
    joinedAt: Date;
    roleId?: number;
    levelName?: string;
    avatarUrl?: string;
    title?: string;
    isParked: boolean;
    isEnabled: boolean;
}
export interface TrackerStats {
    uploadBytes: bigint;
    downloadBytes: bigint;
    ratio: number;
    bonus: number;
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
export interface TrackerError {
    code: TrackerErrorCode;
    message: string;
    /** 不得包含任何凭据信息 */
    details?: Record<string, unknown>;
}
