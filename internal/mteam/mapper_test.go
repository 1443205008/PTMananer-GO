package mteam

import (
	"testing"
	"time"
)

func TestParseMTeamDate(t *testing.T) {
	cases := []struct {
		in   string
		want string // UTC
	}{
		{"2026-08-15 20:30:00", "2026-08-15T12:30:00Z"},
		{"2026-01-01 00:00:00", "2025-12-31T16:00:00Z"},
		{"2026-08-15T20:30:00", "2026-08-15T12:30:00Z"},
	}
	for _, c := range cases {
		in := c.in
		got := parseMTeamDate(&in)
		if got == nil {
			t.Fatalf("parse %q returned nil", c.in)
		}
		if got.UTC().Format(time.RFC3339) != c.want {
			t.Errorf("parse %q = %s, want %s", c.in, got.UTC().Format(time.RFC3339), c.want)
		}
	}
	// 无效输入
	for _, bad := range []string{"", "garbage", "2026"} {
		s := bad
		if parseMTeamDate(&s) != nil {
			t.Errorf("parse %q should return nil", bad)
		}
	}
}

func TestToBigIntFloatInt(t *testing.T) {
	if toBigInt("35280000000000") != 35280000000000 {
		t.Fatal("plain int64")
	}
	if toBigInt("733142.6") != 733142 {
		t.Fatal("float string truncates to int part")
	}
	if toBigInt("") != 0 || toBigInt("abc") != 0 {
		t.Fatal("bad input → 0")
	}
	if toFloat("733142.6") != 733142.6 {
		t.Fatal("float parse")
	}
	if toFloat("") != 0 {
		t.Fatal("empty → 0")
	}
	if toInt("369195") != 369195 {
		t.Fatal("int parse")
	}
}

func TestToProfile(t *testing.T) {
	m := &Member{
		ID: "369195", Username: "tester", Role: "3",
		CreatedDate: "2020-05-10 08:00:00",
		MemberCount:  MemberCount{Bonus: "100.5", Uploaded: "1000", Downloaded: "500", ShareRate: "2.0"},
		MemberStatus: MemberStatus{VIP: true},
	}
	p := ToProfile(m)
	if p.ExternalUserID != "369195" || p.Username != "tester" {
		t.Fatal("profile fields")
	}
	if p.LevelName != "知县" {
		t.Fatalf("role 3 = 知县, got %s", p.LevelName)
	}
	if p.JoinedAt.UTC().Format(time.RFC3339) != "2020-05-10T00:00:00Z" {
		t.Fatalf("joinedAt UTC+8 conversion: %s", p.JoinedAt.UTC().Format(time.RFC3339))
	}

	s := ToStats(m, 10, 2, 0, nil)
	if s.UploadBytes != 1000 || s.DownloadBytes != 500 || s.Bonus != 100.5 || s.SeedingCount != 10 {
		t.Fatalf("stats: %+v", s)
	}
	if !s.IsVIP {
		t.Fatal("vip flag")
	}
}

func TestToTorrentSnatchedPriority(t *testing.T) {
	item := &TorrentItem{
		Torrent: TorrentMeta{ID: "123", Name: "ubuntu.iso", Size: "3528000000000"},
		Snatched: &TorrentSnatched{Uploaded: "5000", Downloaded: "2500", Seedtime: "3600"},
		Peer:     &TorrentPeer{Uploaded: "10", Downloaded: "5"},
	}
	tt := ToTorrent(item, "SEEDING")
	if tt == nil {
		t.Fatal("nil torrent")
	}
	if tt.UploadedBytes != 5000 || tt.DownloadedBytes != 2500 {
		t.Fatal("snatched takes priority over peer")
	}
	if tt.SeedTimeSecs != 3600 {
		t.Fatal("seedtime from snatched")
	}
	if tt.Status != "SEEDING" || tt.Ratio != 2.0 {
		t.Fatalf("status/ratio: %+v", tt)
	}

	// peer 回退 + 做种时长推算
	created := "2026-09-01 00:00:00"
	last := "2026-09-01 02:00:00"
	item2 := &TorrentItem{
		Torrent: TorrentMeta{ID: "124", Name: "x", Size: "1"},
		Peer:    &TorrentPeer{CreatedDate: created, LastAction: &last, Uploaded: "100"},
	}
	tt2 := ToTorrent(item2, "LEECHING")
	if tt2.SeedTimeSecs != 7200 {
		t.Fatalf("seed time inferred = %d, want 7200", tt2.SeedTimeSecs)
	}
	if tt2.Ratio != -1 {
		t.Fatalf("upload without download → -1 (infinite), got %f", tt2.Ratio)
	}
}

func TestMapRole(t *testing.T) {
	if mapRoleToLevelName(10) != "VIP" || mapRoleToLevelName(0) != "封禁" {
		t.Fatal("role map")
	}
	if mapRoleToLevelName(99) != "Level 99" {
		t.Fatal("unknown role fallback")
	}
}
