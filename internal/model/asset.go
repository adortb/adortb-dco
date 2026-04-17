package model

import (
	"encoding/json"
	"time"
)

type Asset struct {
	ID           int64           `db:"id" json:"id"`
	AdvertiserID int64           `db:"advertiser_id" json:"advertiser_id"`
	SlotType     string          `db:"slot_type" json:"slot_type"`
	Value        string          `db:"value" json:"value"`
	Locale       string          `db:"locale" json:"locale"`
	Tags         json.RawMessage `db:"tags" json:"tags,omitempty"`
	Weight       int             `db:"weight" json:"weight"`
	Status       string          `db:"status" json:"status"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
}

// TagsMap parses the JSON tags into a string map.
func (a *Asset) TagsMap() (map[string]string, error) {
	if len(a.Tags) == 0 {
		return nil, nil
	}
	m := make(map[string]string)
	if err := json.Unmarshal(a.Tags, &m); err != nil {
		return nil, err
	}
	return m, nil
}
