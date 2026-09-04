"use strict";
/**
 * 枚举定义 — 前后端共用
 * 所有枚举均来自领域模型，与任何 PT 站点 API 无关
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.TrackerErrorCode = exports.AlertSeverity = exports.AlertType = exports.HnrStatus = exports.SiteStatus = exports.SyncStatus = exports.SyncJobType = exports.TorrentStatus = exports.AccountStatus = exports.TrackerSiteCode = void 0;
// ─── Tracker / Account ───────────────────────────────────────────────────────
var TrackerSiteCode;
(function (TrackerSiteCode) {
    TrackerSiteCode["MTEAM"] = "MTEAM";
    // 未来扩展：SITE_B = 'SITE_B'
})(TrackerSiteCode || (exports.TrackerSiteCode = TrackerSiteCode = {}));
var AccountStatus;
(function (AccountStatus) {
    AccountStatus["ACTIVE"] = "ACTIVE";
    AccountStatus["INACTIVE"] = "INACTIVE";
    AccountStatus["CREDENTIAL_INVALID"] = "CREDENTIAL_INVALID";
    AccountStatus["SYNC_ERROR"] = "SYNC_ERROR";
})(AccountStatus || (exports.AccountStatus = AccountStatus = {}));
// ─── Torrent ─────────────────────────────────────────────────────────────────
var TorrentStatus;
(function (TorrentStatus) {
    TorrentStatus["SEEDING"] = "SEEDING";
    TorrentStatus["LEECHING"] = "LEECHING";
    TorrentStatus["COMPLETED"] = "COMPLETED";
    TorrentStatus["STOPPED"] = "STOPPED";
    TorrentStatus["UNKNOWN"] = "UNKNOWN";
})(TorrentStatus || (exports.TorrentStatus = TorrentStatus = {}));
// ─── Sync ─────────────────────────────────────────────────────────────────────
var SyncJobType;
(function (SyncJobType) {
    SyncJobType["PROFILE_SYNC"] = "PROFILE_SYNC";
    SyncJobType["STATS_SYNC"] = "STATS_SYNC";
    SyncJobType["TORRENT_SYNC"] = "TORRENT_SYNC";
    SyncJobType["HNR_SYNC"] = "HNR_SYNC";
    SyncJobType["BONUS_SYNC"] = "BONUS_SYNC";
    SyncJobType["SITE_HEALTH_CHECK"] = "SITE_HEALTH_CHECK";
    SyncJobType["DAILY_SNAPSHOT"] = "DAILY_SNAPSHOT";
})(SyncJobType || (exports.SyncJobType = SyncJobType = {}));
var SyncStatus;
(function (SyncStatus) {
    SyncStatus["PENDING"] = "PENDING";
    SyncStatus["RUNNING"] = "RUNNING";
    SyncStatus["SUCCESS"] = "SUCCESS";
    SyncStatus["FAILED"] = "FAILED";
    SyncStatus["PARTIAL"] = "PARTIAL";
})(SyncStatus || (exports.SyncStatus = SyncStatus = {}));
// ─── Site health ──────────────────────────────────────────────────────────────
var SiteStatus;
(function (SiteStatus) {
    SiteStatus["HEALTHY"] = "HEALTHY";
    SiteStatus["DEGRADED"] = "DEGRADED";
    SiteStatus["OFFLINE"] = "OFFLINE";
    SiteStatus["UNKNOWN"] = "UNKNOWN";
})(SiteStatus || (exports.SiteStatus = SiteStatus = {}));
// ─── H&R ──────────────────────────────────────────────────────────────────────
var HnrStatus;
(function (HnrStatus) {
    HnrStatus["ACTIVE"] = "ACTIVE";
    HnrStatus["RESOLVED"] = "RESOLVED";
    HnrStatus["EXPIRED"] = "EXPIRED";
})(HnrStatus || (exports.HnrStatus = HnrStatus = {}));
// ─── Alerts ───────────────────────────────────────────────────────────────────
var AlertType;
(function (AlertType) {
    AlertType["HNR_DETECTED"] = "HNR_DETECTED";
    AlertType["AUTH_FAILED"] = "AUTH_FAILED";
    AlertType["SYNC_FAILED"] = "SYNC_FAILED";
    AlertType["SITE_OFFLINE"] = "SITE_OFFLINE";
    AlertType["API_CHANGED"] = "API_CHANGED";
    AlertType["RATE_LIMITED"] = "RATE_LIMITED";
})(AlertType || (exports.AlertType = AlertType = {}));
var AlertSeverity;
(function (AlertSeverity) {
    AlertSeverity["INFO"] = "INFO";
    AlertSeverity["WARNING"] = "WARNING";
    AlertSeverity["ERROR"] = "ERROR";
    AlertSeverity["CRITICAL"] = "CRITICAL";
})(AlertSeverity || (exports.AlertSeverity = AlertSeverity = {}));
// ─── Tracker errors ───────────────────────────────────────────────────────────
var TrackerErrorCode;
(function (TrackerErrorCode) {
    TrackerErrorCode["AUTH_INVALID"] = "AUTH_INVALID";
    TrackerErrorCode["AUTH_EXPIRED"] = "AUTH_EXPIRED";
    TrackerErrorCode["RATE_LIMITED"] = "RATE_LIMITED";
    TrackerErrorCode["SITE_OFFLINE"] = "SITE_OFFLINE";
    TrackerErrorCode["NETWORK_ERROR"] = "NETWORK_ERROR";
    TrackerErrorCode["TIMEOUT"] = "TIMEOUT";
    TrackerErrorCode["API_CHANGED"] = "API_CHANGED";
    TrackerErrorCode["INVALID_RESPONSE"] = "INVALID_RESPONSE";
    TrackerErrorCode["UNKNOWN_ERROR"] = "UNKNOWN_ERROR";
})(TrackerErrorCode || (exports.TrackerErrorCode = TrackerErrorCode = {}));
//# sourceMappingURL=index.js.map