// Package providers: TrackerProvider 统一接口 + 注册表。
package providers

import (
	"fmt"

	"github.com/1443205008/ptmanager-go/internal/domain"
)

// TrackerProvider 所有 PT Provider 必须实现的统一接口（对齐 TS 版）。
// 规则：业务层只能依赖此接口，禁止直接依赖具体 Provider。
type TrackerProvider interface {
	TestConnection(accountID string) (domain.ConnectionResult, error)
	GetProfile(accountID string) (domain.TrackerProfile, error)
	GetStats(accountID string) (domain.TrackerStats, error)
	GetTorrents(accountID string) ([]domain.TrackerTorrent, error)
	GetSeedingTorrents(accountID string) ([]domain.TrackerTorrent, error)
	GetLeechingTorrents(accountID string) ([]domain.TrackerTorrent, error)
	GetBonus(accountID string) (domain.BonusStats, error)
	GetHitAndRuns(accountID string) ([]domain.HitAndRun, error)
	GetMessages(accountID string) (domain.MessageStats, error)
	GetSiteStatus(accountID string) (domain.TrackerSiteStatusResult, error)
}

// Registry Provider 注册表。
type Registry struct {
	providers map[string]TrackerProvider
}

func NewRegistry() *Registry {
	return &Registry{providers: map[string]TrackerProvider{}}
}

func (r *Registry) Register(siteCode string, p TrackerProvider) {
	r.providers[siteCode] = p
}

func (r *Registry) Get(siteCode string) (TrackerProvider, error) {
	p, ok := r.providers[siteCode]
	if !ok {
		return nil, fmt.Errorf("no TrackerProvider registered for site: %s", siteCode)
	}
	return p, nil
}

func (r *Registry) Has(siteCode string) bool {
	_, ok := r.providers[siteCode]
	return ok
}
