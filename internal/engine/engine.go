package engine

import (
	"context"
	"fmt"
	"sort"

	"github.com/adortb/adortb-dco/internal/model"
)

// Repository defines the data access contract needed by the engine.
type Repository interface {
	GetRulesByCampaign(ctx context.Context, campaignID int64) ([]*model.Rule, error)
	GetTemplate(ctx context.Context, id int64) (*model.Template, error)
	GetAssetsBySlotType(ctx context.Context, advertiserID int64, slotType string) ([]*model.Asset, error)
}

type Engine struct {
	repo  Repository
	cache *Cache
}

func New(repo Repository, cache *Cache) *Engine {
	return &Engine{repo: repo, cache: cache}
}

// Render executes the full DCO pipeline for a given request.
func (e *Engine) Render(ctx context.Context, req RenderRequest) (*RenderResult, error) {
	rules, err := e.loadRules(ctx, req.CampaignID)
	if err != nil {
		return nil, fmt.Errorf("load rules: %w", err)
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	var matched *model.Rule
	for _, r := range rules {
		if matchRule(r, req) {
			matched = r
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("no matching rule for campaign %d", req.CampaignID)
	}

	tmpl, err := e.loadTemplate(ctx, matched.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("load template: %w", err)
	}
	if err := tmpl.ParseSlots(); err != nil {
		return nil, fmt.Errorf("parse template slots: %w", err)
	}

	assetsBySlot, err := e.loadAssets(ctx, matched, tmpl.Slots)
	if err != nil {
		return nil, fmt.Errorf("load assets: %w", err)
	}

	chosen, used := selectAssets(matched, assetsBySlot)
	slotVals := buildSlotValues(chosen, req.DynamicData)

	result := &RenderResult{
		TemplateID: tmpl.ID,
		RuleID:     matched.ID,
		AssetsUsed: used,
	}

	switch tmpl.AdType {
	case "banner":
		html, err := renderHTML(tmpl, slotVals)
		if err != nil {
			return nil, err
		}
		result.RenderedHTML = html
	case "native":
		native, err := renderNative(tmpl, slotVals)
		if err != nil {
			return nil, err
		}
		result.RenderedNative = native
	default:
		return nil, fmt.Errorf("unknown ad_type %q", tmpl.AdType)
	}

	return result, nil
}

func (e *Engine) loadRules(ctx context.Context, campaignID int64) ([]*model.Rule, error) {
	if e.cache != nil {
		if rules, err := e.cache.GetRules(ctx, campaignID); err == nil {
			return rules, nil
		}
	}
	rules, err := e.repo.GetRulesByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		_ = r.ParseJSON()
	}
	if e.cache != nil {
		_ = e.cache.SetRules(ctx, campaignID, rules)
	}
	return rules, nil
}

func (e *Engine) loadTemplate(ctx context.Context, id int64) (*model.Template, error) {
	if e.cache != nil {
		if t, err := e.cache.GetTemplate(ctx, id); err == nil {
			return t, nil
		}
	}
	t, err := e.repo.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.cache != nil {
		_ = e.cache.SetTemplate(ctx, t)
	}
	return t, nil
}

func (e *Engine) loadAssets(ctx context.Context, rule *model.Rule, slots []string) (map[string][]*model.Asset, error) {
	result := make(map[string][]*model.Asset, len(slots))
	for _, slot := range slots {
		assets, err := e.repo.GetAssetsBySlotType(ctx, rule.AdvertiserID, slot)
		if err != nil {
			return nil, fmt.Errorf("get assets slot=%s: %w", slot, err)
		}
		result[slot] = assets
	}
	return result, nil
}
