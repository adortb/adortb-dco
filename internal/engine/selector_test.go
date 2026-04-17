package engine

import (
	"encoding/json"
	"testing"

	"github.com/adortb/adortb-dco/internal/model"
)

func TestMatchRule_GeoMatch(t *testing.T) {
	rule := &model.Rule{
		Conditions: &model.RuleConditions{Geo: []string{"CN", "JP"}},
	}
	req := RenderRequest{User: UserContext{Geo: "CN"}}
	if !matchRule(rule, req) {
		t.Fatal("expected match for geo=CN")
	}
}

func TestMatchRule_GeoNoMatch(t *testing.T) {
	rule := &model.Rule{
		Conditions: &model.RuleConditions{Geo: []string{"US"}},
	}
	req := RenderRequest{User: UserContext{Geo: "CN"}}
	if matchRule(rule, req) {
		t.Fatal("expected no match for geo=CN when rule requires US")
	}
}

func TestMatchRule_UserTags(t *testing.T) {
	rule := &model.Rule{
		Conditions: &model.RuleConditions{
			UserTags: []string{"interest:electronics", "high_value_user"},
		},
	}
	req := RenderRequest{
		User: UserContext{Tags: []string{"interest:electronics", "high_value_user", "new_user"}},
	}
	if !matchRule(rule, req) {
		t.Fatal("expected match when all required tags present")
	}

	req2 := RenderRequest{
		User: UserContext{Tags: []string{"interest:electronics"}},
	}
	if matchRule(rule, req2) {
		t.Fatal("expected no match when required tag missing")
	}
}

func TestMatchRule_HourIn(t *testing.T) {
	rule := &model.Rule{
		Conditions: &model.RuleConditions{HourIn: []int{9, 18}},
	}
	cases := []struct {
		hour    int
		wantHit bool
	}{
		{9, true},
		{14, true},
		{18, true},
		{8, false},
		{19, false},
	}
	for _, c := range cases {
		req := RenderRequest{Context: AdContext{Hour: c.hour}}
		got := matchRule(rule, req)
		if got != c.wantHit {
			t.Errorf("hour=%d: want %v got %v", c.hour, c.wantHit, got)
		}
	}
}

func TestMatchRule_NilConditions(t *testing.T) {
	rule := &model.Rule{Conditions: nil}
	if !matchRule(rule, RenderRequest{}) {
		t.Fatal("nil conditions should always match")
	}
}

func TestWeightedRandom_Distribution(t *testing.T) {
	assets := []*model.Asset{
		{ID: 1, Weight: 90},
		{ID: 2, Weight: 10},
	}
	counts := map[int64]int{}
	for range 10000 {
		a := weightedRandom(assets)
		if a == nil {
			t.Fatal("nil asset returned")
		}
		counts[a.ID]++
	}
	// ID=1 should appear ~90% of the time; allow 5% tolerance
	ratio := float64(counts[1]) / 10000
	if ratio < 0.85 || ratio > 0.95 {
		t.Errorf("unexpected distribution for weighted random: id1=%d id2=%d", counts[1], counts[2])
	}
}

func TestWeightedRandom_Empty(t *testing.T) {
	if weightedRandom(nil) != nil {
		t.Fatal("expected nil for empty slice")
	}
}

func TestFilterAssets_TagFilter(t *testing.T) {
	makeAsset := func(id int64, tags map[string]string) *model.Asset {
		raw, _ := json.Marshal(tags)
		return &model.Asset{ID: id, Tags: raw}
	}

	assets := []*model.Asset{
		makeAsset(1, map[string]string{"category": "electronics", "style": "promo"}),
		makeAsset(2, map[string]string{"category": "fashion"}),
		makeAsset(3, map[string]string{"category": "electronics", "style": "plain"}),
	}

	filter := map[string]string{"tags.category": "electronics"}
	out := filterAssets(assets, filter)
	if len(out) != 2 {
		t.Fatalf("expected 2 filtered assets, got %d", len(out))
	}

	filter2 := map[string]string{"tags.category": "electronics", "tags.style": "promo"}
	out2 := filterAssets(assets, filter2)
	if len(out2) != 1 || out2[0].ID != 1 {
		t.Fatalf("expected only asset 1, got %v", out2)
	}
}

func TestSelectAssets_FallbackNonFilter(t *testing.T) {
	rule := &model.Rule{
		AssetSelection: map[string]model.SlotFilter{
			"headline": {Filter: map[string]string{"tags.category": "nonexistent"}},
		},
	}
	raw, _ := json.Marshal(map[string]string{"category": "electronics"})
	assets := map[string][]*model.Asset{
		"headline": {{ID: 10, Weight: 100, Tags: raw}},
	}
	chosen, used := selectAssets(rule, assets)
	if len(chosen) != 1 || chosen["headline"] == nil {
		t.Fatal("expected fallback to full asset list when filter returns empty")
	}
	if len(used) != 1 {
		t.Fatalf("expected 1 asset used, got %d", len(used))
	}
}
