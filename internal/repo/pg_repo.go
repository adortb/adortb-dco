package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/adortb/adortb-dco/internal/model"
)

type PGRepo struct {
	db *sql.DB
}

func New(db *sql.DB) *PGRepo {
	return &PGRepo{db: db}
}

// --- Template ---

func (r *PGRepo) CreateTemplate(ctx context.Context, t *model.Template) error {
	slotsJSON, err := json.Marshal(t.Slots)
	if err != nil {
		return err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO dco_templates (name, ad_type, size, html_template, native_template, slots, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at`,
		t.Name, t.AdType, t.Size, t.HTMLTemplate, nullJSON(t.NativeTemplate), slotsJSON, orDefault(t.Status, "active"),
	)
	return row.Scan(&t.ID, &t.CreatedAt)
}

func (r *PGRepo) GetTemplate(ctx context.Context, id int64) (*model.Template, error) {
	t := &model.Template{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, ad_type, COALESCE(size,''), COALESCE(html_template,''),
		       native_template, slots, status, created_at
		FROM dco_templates WHERE id=$1`, id,
	).Scan(&t.ID, &t.Name, &t.AdType, &t.Size, &t.HTMLTemplate,
		&t.NativeTemplate, &t.SlotsRaw, &t.Status, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get template %d: %w", id, err)
	}
	return t, nil
}

func (r *PGRepo) ListTemplates(ctx context.Context) ([]*model.Template, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, ad_type, COALESCE(size,''), COALESCE(html_template,''),
		       native_template, slots, status, created_at
		FROM dco_templates WHERE status='active' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTemplates(rows)
}

func (r *PGRepo) UpdateTemplate(ctx context.Context, t *model.Template) error {
	slotsJSON, err := json.Marshal(t.Slots)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE dco_templates SET name=$1, ad_type=$2, size=$3, html_template=$4,
		native_template=$5, slots=$6, status=$7 WHERE id=$8`,
		t.Name, t.AdType, t.Size, t.HTMLTemplate, nullJSON(t.NativeTemplate), slotsJSON, t.Status, t.ID,
	)
	return err
}

// --- Asset ---

func (r *PGRepo) CreateAsset(ctx context.Context, a *model.Asset) error {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO dco_assets (advertiser_id, slot_type, value, locale, tags, weight, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at`,
		a.AdvertiserID, a.SlotType, a.Value, a.Locale, nullJSON(a.Tags), orDefaultInt(a.Weight, 100), orDefault(a.Status, "active"),
	)
	return row.Scan(&a.ID, &a.CreatedAt)
}

func (r *PGRepo) GetAssetsBySlotType(ctx context.Context, advertiserID int64, slotType string) ([]*model.Asset, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, advertiser_id, slot_type, value, COALESCE(locale,''),
		       tags, weight, status, created_at
		FROM dco_assets WHERE slot_type=$1 AND status='active'
		  AND (advertiser_id=0 OR advertiser_id=$2)
		ORDER BY id`, slotType, advertiserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssets(rows)
}

func (r *PGRepo) ListAssets(ctx context.Context) ([]*model.Asset, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, advertiser_id, slot_type, value, COALESCE(locale,''),
		       tags, weight, status, created_at
		FROM dco_assets WHERE status='active' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssets(rows)
}

func (r *PGRepo) UpdateAsset(ctx context.Context, a *model.Asset) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE dco_assets SET advertiser_id=$1, slot_type=$2, value=$3, locale=$4,
		tags=$5, weight=$6, status=$7 WHERE id=$8`,
		a.AdvertiserID, a.SlotType, a.Value, a.Locale, nullJSON(a.Tags), a.Weight, a.Status, a.ID,
	)
	return err
}

// --- Rule ---

func (r *PGRepo) CreateRule(ctx context.Context, rule *model.Rule) error {
	condJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return err
	}
	selJSON, err := json.Marshal(rule.AssetSelection)
	if err != nil {
		return err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO dco_rules (advertiser_id, campaign_id, name, template_id, conditions, asset_selection, priority, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at`,
		rule.AdvertiserID, rule.CampaignID, rule.Name, rule.TemplateID,
		condJSON, selJSON, orDefaultInt(rule.Priority, 100), orDefault(rule.Status, "active"),
	)
	return row.Scan(&rule.ID, &rule.CreatedAt)
}

func (r *PGRepo) GetRulesByCampaign(ctx context.Context, campaignID int64) ([]*model.Rule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, advertiser_id, campaign_id, COALESCE(name,''), template_id,
		       conditions, asset_selection, priority, status, created_at
		FROM dco_rules WHERE campaign_id=$1 AND status='active'
		ORDER BY priority DESC`, campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

func (r *PGRepo) ListRules(ctx context.Context) ([]*model.Rule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, advertiser_id, campaign_id, COALESCE(name,''), template_id,
		       conditions, asset_selection, priority, status, created_at
		FROM dco_rules WHERE status='active' ORDER BY priority DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

func (r *PGRepo) UpdateRule(ctx context.Context, rule *model.Rule) error {
	condJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return err
	}
	selJSON, err := json.Marshal(rule.AssetSelection)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE dco_rules SET advertiser_id=$1, campaign_id=$2, name=$3, template_id=$4,
		conditions=$5, asset_selection=$6, priority=$7, status=$8 WHERE id=$9`,
		rule.AdvertiserID, rule.CampaignID, rule.Name, rule.TemplateID,
		condJSON, selJSON, rule.Priority, rule.Status, rule.ID,
	)
	return err
}

// --- scan helpers ---

func scanTemplates(rows *sql.Rows) ([]*model.Template, error) {
	var out []*model.Template
	for rows.Next() {
		t := &model.Template{}
		if err := rows.Scan(&t.ID, &t.Name, &t.AdType, &t.Size, &t.HTMLTemplate,
			&t.NativeTemplate, &t.SlotsRaw, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanAssets(rows *sql.Rows) ([]*model.Asset, error) {
	var out []*model.Asset
	for rows.Next() {
		a := &model.Asset{}
		if err := rows.Scan(&a.ID, &a.AdvertiserID, &a.SlotType, &a.Value,
			&a.Locale, &a.Tags, &a.Weight, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanRules(rows *sql.Rows) ([]*model.Rule, error) {
	var out []*model.Rule
	for rows.Next() {
		rule := &model.Rule{}
		if err := rows.Scan(&rule.ID, &rule.AdvertiserID, &rule.CampaignID,
			&rule.Name, &rule.TemplateID, &rule.ConditionsRaw, &rule.AssetSelRaw,
			&rule.Priority, &rule.Status, &rule.CreatedAt); err != nil {
			return nil, err
		}
		_ = rule.ParseJSON()
		out = append(out, rule)
	}
	return out, rows.Err()
}

// --- util ---

func nullJSON(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return nil
	}
	return []byte(raw)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func orDefaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}
