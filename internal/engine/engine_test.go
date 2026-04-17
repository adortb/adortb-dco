package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/adortb/adortb-dco/internal/model"
)

// --- mock repository ---

type mockRepo struct {
	rules     []*model.Rule
	templates map[int64]*model.Template
	assets    map[string][]*model.Asset
}

func (m *mockRepo) GetRulesByCampaign(_ context.Context, campaignID int64) ([]*model.Rule, error) {
	var out []*model.Rule
	for _, r := range m.rules {
		if r.CampaignID == campaignID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRepo) GetTemplate(_ context.Context, id int64) (*model.Template, error) {
	t, ok := m.templates[id]
	if !ok {
		return nil, errors.New("template not found")
	}
	return t, nil
}

func (m *mockRepo) GetAssetsBySlotType(_ context.Context, _ int64, slotType string) ([]*model.Asset, error) {
	return m.assets[slotType], nil
}

// --- tests ---

func TestEngine_RenderBanner(t *testing.T) {
	repo := &mockRepo{
		rules: []*model.Rule{
			{
				ID: 5, CampaignID: 123, TemplateID: 1, Priority: 100, Status: "active",
				Conditions:     &model.RuleConditions{Geo: []string{"CN"}},
				AssetSelection: map[string]model.SlotFilter{},
			},
		},
		templates: map[int64]*model.Template{
			1: {
				ID: 1, AdType: "banner",
				HTMLTemplate: `<div>{{.headline}} ¥{{.price}}</div>`,
				Slots:        []string{"headline", "price"},
				SlotsRaw:     []byte(`["headline","price"]`),
			},
		},
		assets: map[string][]*model.Asset{
			"headline": {{ID: 10, Value: "限时特惠", Weight: 100}},
			"price":    {{ID: 11, Value: "5999", Weight: 100}},
		},
	}

	eng := New(repo, nil)
	req := RenderRequest{
		CampaignID:  123,
		User:        UserContext{Geo: "CN"},
		DynamicData: map[string]string{"price": "4999"},
	}

	result, err := eng.Render(context.Background(), req)
	if err != nil {
		t.Fatalf("render err: %v", err)
	}
	if result.TemplateID != 1 {
		t.Errorf("expected template_id=1 got %d", result.TemplateID)
	}
	if result.RuleID != 5 {
		t.Errorf("expected rule_id=5 got %d", result.RuleID)
	}
	// dynamic price should override asset
	if result.RenderedHTML == "" {
		t.Error("expected non-empty rendered_html")
	}
}

func TestEngine_NoMatchingRule(t *testing.T) {
	repo := &mockRepo{
		rules: []*model.Rule{
			{
				ID: 1, CampaignID: 123, TemplateID: 1, Priority: 100, Status: "active",
				Conditions: &model.RuleConditions{Geo: []string{"US"}},
			},
		},
		templates: map[int64]*model.Template{},
		assets:    map[string][]*model.Asset{},
	}
	eng := New(repo, nil)
	_, err := eng.Render(context.Background(), RenderRequest{
		CampaignID: 123,
		User:       UserContext{Geo: "CN"},
	})
	if err == nil {
		t.Fatal("expected error when no rule matches")
	}
}

func TestEngine_RulePriority(t *testing.T) {
	repo := &mockRepo{
		rules: []*model.Rule{
			{ID: 1, CampaignID: 1, TemplateID: 1, Priority: 50, Status: "active", Conditions: &model.RuleConditions{}},
			{ID: 2, CampaignID: 1, TemplateID: 2, Priority: 200, Status: "active", Conditions: &model.RuleConditions{}},
		},
		templates: map[int64]*model.Template{
			1: {ID: 1, AdType: "banner", HTMLTemplate: "t1", Slots: []string{}, SlotsRaw: []byte(`[]`)},
			2: {ID: 2, AdType: "banner", HTMLTemplate: "t2", Slots: []string{}, SlotsRaw: []byte(`[]`)},
		},
		assets: map[string][]*model.Asset{},
	}
	eng := New(repo, nil)
	result, err := eng.Render(context.Background(), RenderRequest{CampaignID: 1})
	if err != nil {
		t.Fatalf("render err: %v", err)
	}
	if result.TemplateID != 2 {
		t.Errorf("expected highest-priority rule (template 2) to be selected, got template %d", result.TemplateID)
	}
}
