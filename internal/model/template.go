package model

import (
	"encoding/json"
	"time"
)

type Template struct {
	ID             int64           `db:"id" json:"id"`
	Name           string          `db:"name" json:"name"`
	AdType         string          `db:"ad_type" json:"ad_type"`
	Size           string          `db:"size" json:"size"`
	HTMLTemplate   string          `db:"html_template" json:"html_template"`
	NativeTemplate json.RawMessage `db:"native_template" json:"native_template,omitempty"`
	Slots          []string        `db:"-" json:"slots"`
	SlotsRaw       json.RawMessage `db:"slots" json:"-"`
	Status         string          `db:"status" json:"status"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

func (t *Template) ParseSlots() error {
	if len(t.SlotsRaw) == 0 {
		return nil
	}
	return json.Unmarshal(t.SlotsRaw, &t.Slots)
}
