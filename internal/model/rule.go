package model

import (
	"encoding/json"
	"time"
)

// RuleConditions defines matching conditions for a rule.
type RuleConditions struct {
	Geo      []string `json:"geo,omitempty"`
	UserTags []string `json:"user_tags,omitempty"`
	HourIn   []int    `json:"hour_in,omitempty"`
}

// SlotFilter specifies how to select assets for a single slot.
type SlotFilter struct {
	Filter map[string]string `json:"filter,omitempty"`
}

type Rule struct {
	ID             int64           `db:"id" json:"id"`
	AdvertiserID   int64           `db:"advertiser_id" json:"advertiser_id"`
	CampaignID     int64           `db:"campaign_id" json:"campaign_id"`
	Name           string          `db:"name" json:"name"`
	TemplateID     int64           `db:"template_id" json:"template_id"`
	ConditionsRaw  json.RawMessage `db:"conditions" json:"-"`
	AssetSelRaw    json.RawMessage `db:"asset_selection" json:"-"`
	Conditions     *RuleConditions `db:"-" json:"conditions,omitempty"`
	AssetSelection map[string]SlotFilter `db:"-" json:"asset_selection,omitempty"`
	Priority       int             `db:"priority" json:"priority"`
	Status         string          `db:"status" json:"status"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

func (r *Rule) ParseJSON() error {
	if len(r.ConditionsRaw) > 0 {
		r.Conditions = &RuleConditions{}
		if err := json.Unmarshal(r.ConditionsRaw, r.Conditions); err != nil {
			return err
		}
	}
	if len(r.AssetSelRaw) > 0 {
		r.AssetSelection = make(map[string]SlotFilter)
		if err := json.Unmarshal(r.AssetSelRaw, &r.AssetSelection); err != nil {
			return err
		}
	}
	return nil
}
