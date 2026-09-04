/**
 * 枚举定义 — 前后端共用
 * 所有枚举均来自领域模型，与任何 PT 站点 API 无关
 */
export declare enum TrackerSiteCode {
    MTEAM = "MTEAM"
}
export declare enum AccountStatus {
    ACTIVE = "ACTIVE",
    INACTIVE = "INACTIVE",
    CREDENTIAL_INVALID = "CREDENTIAL_INVALID",
    SYNC_ERROR = "SYNC_ERROR"
}
export declare enum TorrentStatus {
    SEEDING = "SEEDING",
    LEECHING = "LEECHING",
    COMPLETED = "COMPLETED",
    STOPPED = "STOPPED",
    UNKNOWN = "UNKNOWN"
}
export declare enum SyncJobType {
    PROFILE_SYNC = "PROFILE_SYNC",
    STATS_SYNC = "STATS_SYNC",
    TORRENT_SYNC = "TORRENT_SYNC",
    HNR_SYNC = "HNR_SYNC",
    BONUS_SYNC = "BONUS_SYNC",
    SITE_HEALTH_CHECK = "SITE_HEALTH_CHECK",
    DAILY_SNAPSHOT = "DAILY_SNAPSHOT"
}
export declare enum SyncStatus {
    PENDING = "PENDING",
    RUNNING = "RUNNING",
    SUCCESS = "SUCCESS",
    FAILED = "FAILED",
    PARTIAL = "PARTIAL"
}
export declare enum SiteStatus {
    HEALTHY = "HEALTHY",
    DEGRADED = "DEGRADED",
    OFFLINE = "OFFLINE",
    UNKNOWN = "UNKNOWN"
}
export declare enum HnrStatus {
    ACTIVE = "ACTIVE",
    RESOLVED = "RESOLVED",
    EXPIRED = "EXPIRED"
}
export declare enum AlertType {
    HNR_DETECTED = "HNR_DETECTED",
    AUTH_FAILED = "AUTH_FAILED",
    SYNC_FAILED = "SYNC_FAILED",
    SITE_OFFLINE = "SITE_OFFLINE",
    API_CHANGED = "API_CHANGED",
    RATE_LIMITED = "RATE_LIMITED"
}
export declare enum AlertSeverity {
    INFO = "INFO",
    WARNING = "WARNING",
    ERROR = "ERROR",
    CRITICAL = "CRITICAL"
}
export declare enum TrackerErrorCode {
    AUTH_INVALID = "AUTH_INVALID",
    AUTH_EXPIRED = "AUTH_EXPIRED",
    RATE_LIMITED = "RATE_LIMITED",
    SITE_OFFLINE = "SITE_OFFLINE",
    NETWORK_ERROR = "NETWORK_ERROR",
    TIMEOUT = "TIMEOUT",
    API_CHANGED = "API_CHANGED",
    INVALID_RESPONSE = "INVALID_RESPONSE",
    UNKNOWN_ERROR = "UNKNOWN_ERROR"
}
