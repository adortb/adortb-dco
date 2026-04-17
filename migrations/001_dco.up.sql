CREATE TABLE IF NOT EXISTS dco_templates (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL UNIQUE,
    ad_type VARCHAR(30) NOT NULL,
    size VARCHAR(30),
    html_template TEXT,
    native_template JSONB,
    slots JSONB,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dco_assets (
    id BIGSERIAL PRIMARY KEY,
    advertiser_id BIGINT,
    slot_type VARCHAR(30) NOT NULL,
    value TEXT NOT NULL,
    locale VARCHAR(10),
    tags JSONB,
    weight INT DEFAULT 100,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dco_rules (
    id BIGSERIAL PRIMARY KEY,
    advertiser_id BIGINT,
    campaign_id BIGINT,
    name VARCHAR(128),
    template_id BIGINT REFERENCES dco_templates(id),
    conditions JSONB,
    asset_selection JSONB,
    priority INT DEFAULT 100,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rules_camp ON dco_rules(campaign_id, status) WHERE status = 'active';
