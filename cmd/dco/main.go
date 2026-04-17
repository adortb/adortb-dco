package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/adortb/adortb-dco/internal/api"
	"github.com/adortb/adortb-dco/internal/engine"
	"github.com/adortb/adortb-dco/internal/metrics"
	"github.com/adortb/adortb-dco/internal/model"
	"github.com/adortb/adortb-dco/internal/repo"
)

func main() {
	dbDSN := envOrDefault("DB_DSN", "postgres://postgres:postgres@localhost:5432/adortb?sslmode=disable")
	redisAddr := envOrDefault("REDIS_ADDR", "localhost:6379")
	addr := envOrDefault("ADDR", ":8090")

	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("redis unavailable (%v), running without cache", err)
		rdb = nil
	}

	r := repo.New(db)

	if err := runMigrations(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := seedDemoData(ctx, r); err != nil {
		log.Printf("seed: %v", err)
	}

	var cache *engine.Cache
	if rdb != nil {
		cache = engine.NewCache(rdb)
	}

	eng := engine.New(r, cache)
	h := api.NewHandler(eng, r)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("DCO service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func runMigrations(db *sql.DB) error {
	migration := `
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
CREATE INDEX IF NOT EXISTS idx_rules_camp ON dco_rules(campaign_id, status) WHERE status = 'active';`
	_, err := db.Exec(migration)
	return err
}

func seedDemoData(ctx context.Context, r *repo.PGRepo) error {
	// Check if demo data already seeded
	rows, err := r.ListTemplates(ctx)
	if err == nil && len(rows) > 0 {
		return nil
	}

	// Template: electronics banner 300x250
	tmpl := &model.Template{
		Name:         "electronics_banner_300x250",
		AdType:       "banner",
		Size:         "300x250",
		HTMLTemplate: `<div class="ad-300x250"><img src="{{.image}}" alt="ad"/><h2>{{.headline}}</h2><p class="price">¥{{.price}}</p><a href="#" class="cta">{{.cta}}</a></div>`,
		Slots:        []string{"headline", "image", "cta", "price"},
		Status:       "active",
	}
	if err := r.CreateTemplate(ctx, tmpl); err != nil {
		return err
	}

	// Assets: headlines
	headlines := []struct{ val, locale string }{
		{"限时特惠，手机好价！", "zh-CN"},
		{"Best Deals on Electronics", "en-US"},
		{"爆款促销，低价抢购！", "zh-CN"},
	}
	for _, h := range headlines {
		tags, _ := json.Marshal(map[string]string{"category": "electronics", "style": "promo"})
		a := &model.Asset{
			SlotType: "headline", Value: h.val, Locale: h.locale,
			Tags: tags, Weight: 100, Status: "active",
		}
		if err := r.CreateAsset(ctx, a); err != nil {
			return err
		}
	}

	// Assets: images
	images := []string{
		"https://cdn.example.com/iphone15.jpg",
		"https://cdn.example.com/electronics-promo.jpg",
	}
	for _, img := range images {
		tags, _ := json.Marshal(map[string]string{"category": "electronics"})
		a := &model.Asset{
			SlotType: "image", Value: img,
			Tags: tags, Weight: 100, Status: "active",
		}
		if err := r.CreateAsset(ctx, a); err != nil {
			return err
		}
	}

	// Assets: CTAs
	ctas := []string{"立即购买", "查看详情"}
	for _, cta := range ctas {
		a := &model.Asset{
			SlotType: "cta", Value: cta, Locale: "zh-CN",
			Weight: 100, Status: "active",
		}
		if err := r.CreateAsset(ctx, a); err != nil {
			return err
		}
	}

	// Rule: geo=CN → template_1
	rule := &model.Rule{
		CampaignID: 123,
		Name:       "CN electronics rule",
		TemplateID: tmpl.ID,
		Conditions: &model.RuleConditions{
			Geo:      []string{"CN"},
			UserTags: []string{"interest:electronics"},
		},
		AssetSelection: map[string]model.SlotFilter{
			"headline": {Filter: map[string]string{"tags.category": "electronics"}},
			"image":    {Filter: map[string]string{"tags.category": "electronics"}},
		},
		Priority: 100,
		Status:   "active",
	}
	return r.CreateRule(ctx, rule)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
