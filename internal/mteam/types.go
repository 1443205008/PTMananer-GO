// Package mteam: M-Team API 类型定义。
// ⚠️ 关键陷阱（与 TS 版一致，真实响应确认）：
//  1. M-Team 把几乎所有数值都序列化为「字符串」——uploaded/downloaded/bonus/shareRate/role/id/size 全是 string
//  2. code 字段类型不一致：/member/profile 返回 number 0，getUserTorrentList 返回 string "0"
//  3. 日期格式 'YYYY-MM-DD HH:MM:SS'（无时区，实际 UTC+8）
package mteam

import "encoding/json"

// Result 统一 Response 包装
type Result struct {
	Code    json.Number     `json:"code"` // 兼容 number 0 与 string "0"
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Member 用户资料（/api/member/profile）
type Member struct {
	ID           string       `json:"id"`
	Username     string       `json:"username"`
	Email        string       `json:"email"`
	Status       string       `json:"status"`
	Enabled      bool         `json:"enabled"`
	Parked       bool         `json:"parked"`
	Role         string       `json:"role"`
	AvatarURL    string       `json:"avatarUrl"`
	Title        *string      `json:"title"`
	CreatedDate  string       `json:"createdDate"`
	MemberCount  MemberCount  `json:"memberCount"`
	MemberStatus MemberStatus `json:"memberStatus"`
}

// MemberCount 数量/流量统计（全部为字符串数字）
type MemberCount struct {
	Bonus      string `json:"bonus"`
	Uploaded   string `json:"uploaded"`
	Downloaded string `json:"downloaded"`
	ShareRate  string `json:"shareRate"`
}

// MemberStatus 账户状态标志
type MemberStatus struct {
	VIP         bool    `json:"vip"`
	Donor       bool    `json:"donor"`
	Warned      bool    `json:"warned"`
	LeechWarn   bool    `json:"leechWarn"`
	LastLogin   *string `json:"lastLogin"`
	LastTracker *string `json:"lastTracker"`
}

// PageResult getUserTorrentList / search 分页包装（全部字符串数字）
type PageResult struct {
	PageNumber json.Number     `json:"pageNumber"`
	PageSize   json.Number     `json:"pageSize"`
	Total      json.Number     `json:"total"`
	TotalPages json.Number     `json:"totalPages"`
	Data       json.RawMessage `json:"data"`
}

// TorrentItem getUserTorrentList 单条：{ torrent, peer, snatched }
type TorrentItem struct {
	Torrent  TorrentMeta      `json:"torrent"`
	Peer     *TorrentPeer     `json:"peer"`
	Snatched *TorrentSnatched `json:"snatched"`
}

// TorrentMeta 种子元数据
type TorrentMeta struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Size   string             `json:"size"`
	Status *TorrentMetaStatus `json:"status"`
}

// TorrentMetaStatus 种子级状态
type TorrentMetaStatus struct {
	Seeders        string `json:"seeders"`
	Leechers       string `json:"leechers"`
	TimesCompleted string `json:"timesCompleted"`
}

// TorrentPeer 用户个人 peer 会话
type TorrentPeer struct {
	CreatedDate string  `json:"createdDate"`
	LastAction  *string `json:"lastAction"`
	Uploaded    string  `json:"uploaded"`
	Downloaded  string  `json:"downloaded"`
}

// TorrentSnatched 完成下载记录
type TorrentSnatched struct {
	Uploaded   string  `json:"uploaded"`
	Downloaded string  `json:"downloaded"`
	Seedtime   string  `json:"seedtime"`
	Leechtime  string  `json:"leechtime"`
	FinishDate *string `json:"finishDate"`
	LastAction *string `json:"lastAction"`
}

// Team 制作组
type Team struct {
	ID   json.Number `json:"id"`
	Name string      `json:"name"`
}

// SearchTorrentItem 搜索结果单条
type SearchTorrentItem struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	SmallDescr  *string              `json:"smallDescr"`
	Size        string               `json:"size"`
	Category    *string              `json:"category"`
	CreatedDate *string              `json:"createdDate"`
	Status      *SearchTorrentStatus `json:"status"`
	Imdb        *string              `json:"imdb"`
}

type SearchTorrentStatus struct {
	Seeders         string  `json:"seeders"`
	Leechers        string  `json:"leechers"`
	TimesCompleted  string  `json:"timesCompleted"`
	Discount        *string `json:"discount"`
	DiscountEndTime *string `json:"discountEndTime"`
}

// MyBonusResponse /api/tracker/mybonus
type MyBonusResponse struct {
	FormulaParams *struct {
		FinalBs *string `json:"finalBs"`
	} `json:"formulaParams"`
}

// SearchRequest 站内搜索请求
type SearchRequest struct {
	Keyword       string `json:"keyword,omitempty"`
	PageNumber    int    `json:"pageNumber"`
	PageSize      int    `json:"pageSize"`
	Mode          string `json:"mode"`
	Discount      string `json:"discount,omitempty"`
	SortField     string `json:"sortField,omitempty"`
	SortDirection string `json:"sortDirection,omitempty"`
	Teams         []int  `json:"teams,omitempty"`
}

// ─── 端点（全部 POST，来自官方 Swagger）───────────────────────────────────

const (
	EndpointMemberProfile      = "/api/member/profile"
	EndpointMemberTorrentList  = "/api/member/getUserTorrentList"
	EndpointMemberCrimeRecords = "/api/member/getCrimeRecords"
	EndpointMyBonus            = "/api/tracker/mybonus"
	EndpointTorrentSearch      = "/api/torrent/search"
	EndpointGenDlToken         = "/api/torrent/genDlToken"
	EndpointTeamList           = "/api/torrent/teamList"
)

const AuthHeader = "x-api-key"

const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
