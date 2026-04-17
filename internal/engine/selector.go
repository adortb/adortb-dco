package engine

import (
	"math/rand/v2"
	"strings"

	"github.com/adortb/adortb-dco/internal/model"
)

// RenderRequest is the input for the render pipeline.
type RenderRequest struct {
	CampaignID  int64             `json:"campaign_id"`
	User        UserContext       `json:"user"`
	Context     AdContext         `json:"context"`
	DynamicData map[string]string `json:"dynamic_data"`
}

type UserContext struct {
	Geo  string   `json:"geo"`
	Tags []string `json:"tags"`
}

type AdContext struct {
	Hour   int    `json:"hour"`
	Domain string `json:"domain"`
}

// RenderResult is the output of the render pipeline.
type RenderResult struct {
	TemplateID     int64             `json:"template_id"`
	RenderedHTML   string            `json:"rendered_html,omitempty"`
	RenderedNative map[string]string `json:"rendered_native,omitempty"`
	AssetsUsed     []AssetUsed       `json:"assets_used"`
	RuleID         int64             `json:"rule_id"`
}

type AssetUsed struct {
	Slot    string `json:"slot"`
	AssetID int64  `json:"asset_id"`
}

// matchRule checks whether a rule's conditions match the request.
func matchRule(rule *model.Rule, req RenderRequest) bool {
	if rule.Conditions == nil {
		return true
	}
	c := rule.Conditions

	if len(c.Geo) > 0 && !containsStr(c.Geo, req.User.Geo) {
		return false
	}

	if len(c.UserTags) > 0 {
		for _, required := range c.UserTags {
			if !containsStr(req.User.Tags, required) {
				return false
			}
		}
	}

	if len(c.HourIn) == 2 {
		h := req.Context.Hour
		if h < c.HourIn[0] || h > c.HourIn[1] {
			return false
		}
	}

	return true
}

// selectAssets picks one asset per slot using weighted random selection,
// applying tag filters from the rule's asset_selection config.
func selectAssets(
	rule *model.Rule,
	assetsBySlot map[string][]*model.Asset,
) (map[string]*model.Asset, []AssetUsed) {
	chosen := make(map[string]*model.Asset)
	var used []AssetUsed

	for slot, candidates := range assetsBySlot {
		filtered := candidates
		if sf, ok := rule.AssetSelection[slot]; ok && len(sf.Filter) > 0 {
			filtered = filterAssets(candidates, sf.Filter)
		}
		if len(filtered) == 0 {
			filtered = candidates
		}
		if a := weightedRandom(filtered); a != nil {
			chosen[slot] = a
			used = append(used, AssetUsed{Slot: slot, AssetID: a.ID})
		}
	}
	return chosen, used
}

func filterAssets(assets []*model.Asset, filter map[string]string) []*model.Asset {
	var out []*model.Asset
	for _, a := range assets {
		tags, err := a.TagsMap()
		if err != nil || tags == nil {
			continue
		}
		match := true
		for k, v := range filter {
			// support dot notation: "tags.category" → "category"
			key := k
			if strings.HasPrefix(k, "tags.") {
				key = strings.TrimPrefix(k, "tags.")
			}
			if tags[key] != v {
				match = false
				break
			}
		}
		if match {
			out = append(out, a)
		}
	}
	return out
}

func weightedRandom(assets []*model.Asset) *model.Asset {
	if len(assets) == 0 {
		return nil
	}
	total := 0
	for _, a := range assets {
		w := a.Weight
		if w <= 0 {
			w = 1
		}
		total += w
	}
	r := rand.IntN(total)
	cum := 0
	for _, a := range assets {
		w := a.Weight
		if w <= 0 {
			w = 1
		}
		cum += w
		if r < cum {
			return a
		}
	}
	return assets[len(assets)-1]
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
