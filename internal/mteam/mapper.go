package mteam

import (
	"strconv"
	"strings"
	"time"

	"github.com/1443205008/ptmanager-go/internal/domain"
)

// ─── 类型转换工具（M-Team 数值全为字符串）────────────────────────────────

func toBigInt(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// 浮点字符串取整数部分
	if i := strings.IndexAny(s, ".eE"); i >= 0 {
		s = s[:i]
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func toFloat(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || s == "" {
		return 0
	}
	return f
}

func toInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || s == "" {
		return 0
	}
	return n
}

// parseMTeamDate 解析 'YYYY-MM-DD HH:MM:SS'（无时区，实际为 UTC+8）。
// 返回 nil 表示无有效日期。
func parseMTeamDate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	str := *s
	// 允许 'T' 分隔
	str = strings.Replace(str, "T", " ", 1)
	var y, mo, d, h, mi, sec int
	n, err := fmtSscanf(str, &y, &mo, &d, &h, &mi, &sec)
	if err != nil || n != 6 {
		return nil
	}
	// UTC+8 → 减 8 小时
	t := time.Date(y, time.Month(mo), d, h, mi, sec, 0, time.UTC).Add(-8 * time.Hour)
	return &t
}

// fmtSscanf 手动解析 YYYY-MM-DD HH:MM:SS（避免引入 fmt.Sscanf 的复杂错误处理）
func fmtSscanf(s string, y, mo, d, h, mi, sec *int) (int, error) {
	// 格式: 2006-01-02 15:04:05
	if len(s) < 19 {
		return 0, errBadDate
	}
	var err error
	if *y, err = strconv.Atoi(s[0:4]); err != nil {
		return 0, err
	}
	if *mo, err = strconv.Atoi(s[5:7]); err != nil {
		return 0, err
	}
	if *d, err = strconv.Atoi(s[8:10]); err != nil {
		return 0, err
	}
	if *h, err = strconv.Atoi(s[11:13]); err != nil {
		return 0, err
	}
	if *mi, err = strconv.Atoi(s[14:16]); err != nil {
		return 0, err
	}
	if *sec, err = strconv.Atoi(s[17:19]); err != nil {
		return 0, err
	}
	return 6, nil
}

type dumbError struct{}

func (dumbError) Error() string { return "bad date" }

var errBadDate = dumbError{}

// ─── Mapper ────────────────────────────────────────────────────────────────

// roleID → 等级名（馒头官方等级体系）
var roleMap = map[int]string{
	0: "封禁", 1: "小卒", 2: "捕头", 3: "知县", 4: "通判", 5: "知州",
	6: "府丞", 7: "府尹", 8: "总督", 9: "大臣", 10: "VIP", 11: "上传者",
	12: "版主", 13: "职员", 14: "管理员",
}

func mapRoleToLevelName(roleID int) string {
	if name, ok := roleMap[roleID]; ok {
		return name
	}
	return "Level " + strconv.Itoa(roleID)
}

// ToProfile Member → domain.TrackerProfile
func ToProfile(m *Member) domain.TrackerProfile {
	roleID := toInt(m.Role)
	joinedAt := time.Unix(0, 0).UTC()
	if t := parseMTeamDate(&m.CreatedDate); t != nil {
		joinedAt = *t
	}
	var title string
	if m.Title != nil {
		title = *m.Title
	}
	return domain.TrackerProfile{
		SiteCode:       domain.SiteMTeam,
		ExternalUserID: m.ID,
		Username:       m.Username,
		Email:          m.Email,
		JoinedAt:       joinedAt,
		RoleID:         roleID,
		LevelName:      mapRoleToLevelName(roleID),
		AvatarURL:      m.AvatarURL,
		Title:          title,
		IsParked:       m.Parked,
		IsEnabled:      m.Enabled,
	}
}

// ToStats Member + counts → domain.TrackerStats
func ToStats(m *Member, seedingCount, leechingCount, hitAndRunCount int, bonusHourlyRate *float64) domain.TrackerStats {
	return domain.TrackerStats{
		UploadBytes:    toBigInt(m.MemberCount.Uploaded),
		DownloadBytes:  toBigInt(m.MemberCount.Downloaded),
		Ratio:          toFloat(m.MemberCount.ShareRate),
		Bonus:          toFloat(m.MemberCount.Bonus),
		SeedingCount:   seedingCount,
		LeechingCount:  leechingCount,
		HitAndRunCount: hitAndRunCount,
		IsWarned:       m.MemberStatus.Warned,
		IsVIP:          m.MemberStatus.VIP,
		IsDonor:        m.MemberStatus.Donor,
		LastLoginAt:    parseMTeamDate(m.MemberStatus.LastLogin),
		LastTrackerAt:  parseMTeamDate(m.MemberStatus.LastTracker),
		SnapshotTime:   time.Now(),
		BonusHourlyRate: bonusHourlyRate,
	}
}

// ToTorrent TorrentItem → domain.TrackerTorrent
// 结构 { torrent, peer, snatched }：累计数据优先 snatched，其次 peer。
func ToTorrent(item *TorrentItem, queryType string) *domain.TrackerTorrent {
	if item.Torrent.ID == "" {
		return nil
	}
	var uploadedBytes, downloadedBytes int64
	var seedTimeSecs int
	var completedAt, lastActivityAt *time.Time

	if item.Snatched != nil {
		uploadedBytes = toBigInt(item.Snatched.Uploaded)
		downloadedBytes = toBigInt(item.Snatched.Downloaded)
		seedTimeSecs = toInt(item.Snatched.Seedtime)
		completedAt = parseMTeamDate(item.Snatched.FinishDate)
		lastActivityAt = parseMTeamDate(item.Snatched.LastAction)
	} else if item.Peer != nil {
		uploadedBytes = toBigInt(item.Peer.Uploaded)
		downloadedBytes = toBigInt(item.Peer.Downloaded)
		lastActivityAt = parseMTeamDate(item.Peer.LastAction)
		if start := parseMTeamDate(&item.Peer.CreatedDate); start != nil && lastActivityAt != nil {
			if delta := lastActivityAt.Sub(*start).Seconds(); delta > 0 {
				seedTimeSecs = int(delta)
			}
		}
	}

	var ratio float64
	switch {
	case downloadedBytes > 0:
		ratio = float64(uploadedBytes) / float64(downloadedBytes)
	case uploadedBytes > 0:
		ratio = -1 // 无下载但有上传 → 无限
	}

	return &domain.TrackerTorrent{
		SiteTorrentID:  item.Torrent.ID,
		Name:           item.Torrent.Name,
		SizeBytes:      toBigInt(item.Torrent.Size),
		Status:         mapQueryTypeToStatus(queryType),
		UploadedBytes:  uploadedBytes,
		DownloadedBytes: downloadedBytes,
		Ratio:          ratio,
		SeedTimeSecs:   seedTimeSecs,
		LastActivityAt: lastActivityAt,
		CompletedAt:    completedAt,
	}
}

func mapQueryTypeToStatus(queryType string) domain.TorrentStatus {
	switch queryType {
	case "SEEDING":
		return domain.TorrentSeeding
	case "LEECHING", "INCOMPLETE":
		return domain.TorrentLeeching
	case "COMPLETED":
		return domain.TorrentCompleted
	default:
		return domain.TorrentUnknown
	}
}
