package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/adortb/adortb-dco/internal/model"
)

const (
	templateTTL = 5 * time.Minute
	rulesTTL    = 30 * time.Second
)

type Cache struct {
	rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

func (c *Cache) GetTemplate(ctx context.Context, id int64) (*model.Template, error) {
	key := fmt.Sprintf("dco:tpl:%d", id)
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var t model.Template
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *Cache) SetTemplate(ctx context.Context, t *model.Template) error {
	key := fmt.Sprintf("dco:tpl:%d", t.ID)
	raw, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, raw, templateTTL).Err()
}

func (c *Cache) GetRules(ctx context.Context, campaignID int64) ([]*model.Rule, error) {
	key := fmt.Sprintf("dco:rules:%d", campaignID)
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var rules []*model.Rule
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *Cache) SetRules(ctx context.Context, campaignID int64, rules []*model.Rule) error {
	key := fmt.Sprintf("dco:rules:%d", campaignID)
	raw, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, raw, rulesTTL).Err()
}
