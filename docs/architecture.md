# adortb-dco 内部架构

## 内部架构图

```
┌─────────────────────────────────────────────────────────┐
│                   adortb-dco 内部架构                    │
│                                                         │
│  HTTP 请求                                              │
│      │                                                  │
│      ▼                                                  │
│  ┌──────────────────────────────────────────────────┐   │
│  │  internal/api/handler.go                         │   │
│  │  Handler.handleRender()                          │   │
│  │  Handler.handleTemplates()                       │   │
│  │  Handler.handleAssets()                          │   │
│  │  Handler.handleRules()                           │   │
│  └────────────────────┬─────────────────────────────┘   │
│                       │ Engine.Render(RenderRequest)     │
│                       ▼                                  │
│  ┌──────────────────────────────────────────────────┐   │
│  │  internal/engine/engine.go                       │   │
│  │  Engine { repo Repository, cache *Cache }        │   │
│  │                                                  │   │
│  │  1. loadRules(campaignID)                        │   │
│  │  2. matchRule(rule, req)                         │   │
│  │  3. loadTemplate(templateID)                     │   │
│  │  4. loadAssets(rule, slots)                      │   │
│  │  5. selectAssets() → weightedRandom              │   │
│  │  6. buildSlotValues(chosen, dynamicData)         │   │
│  │  7. renderHTML / renderNative                    │   │
│  └──────┬─────────────────────────┬─────────────────┘   │
│         │ Repository              │ Cache                │
│         ▼                         ▼                      │
│  ┌─────────────┐         ┌────────────────┐             │
│  │  PGRepo     │         │  Redis Cache   │             │
│  │  pg_repo.go │         │  cache.go      │             │
│  │             │         │  GetRules()    │             │
│  │  templates  │         │  SetRules()    │             │
│  │  assets     │         │  GetTemplate() │             │
│  │  rules      │         │  SetTemplate() │             │
│  └──────┬──────┘         └────────────────┘             │
│         │                                               │
│  ┌──────▼──────┐                                        │
│  │  PostgreSQL  │                                        │
│  │             │                                        │
│  │  dco_templates                                       │
│  │  dco_assets                                          │
│  │  dco_rules                                           │
│  └─────────────┘                                        │
└─────────────────────────────────────────────────────────┘
```

## 数据流

### 渲染请求数据流

```
POST /v1/render
    │
    │ RenderRequest{CampaignID, User{Geo,Tags}, Context{Hour,Domain}, DynamicData}
    ▼
Handler.handleRender()
    │
    │ 校验 campaign_id 非零
    ▼
Engine.Render(ctx, req)
    │
    ├─[1]─► loadRules(campaignID)
    │        Redis HIT → []Rule（带 ParsedJSON）
    │        Redis MISS → repo.GetRulesByCampaign() → Redis.SetRules()
    │        sort by Priority DESC
    │
    ├─[2]─► matchRule(rule, req)     第一个匹配的规则
    │        检查 Conditions.Geo ∩ req.User.Geo
    │        检查 Conditions.UserTags ⊆ req.User.Tags
    │        检查 Conditions.HourIn[0] ≤ req.Context.Hour ≤ HourIn[1]
    │
    ├─[3]─► loadTemplate(matched.TemplateID)
    │        Redis HIT → *Template
    │        Redis MISS → repo.GetTemplate() → Redis.SetTemplate()
    │        tmpl.ParseSlots()  解析 {{slot}} 占位符列表
    │
    ├─[4]─► loadAssets(rule, slots)
    │        对每个 slot → repo.GetAssetsBySlotType(advertiserID, slot)
    │        返回 map[slot][]*Asset
    │
    ├─[5]─► selectAssets(rule, assetsBySlot)
    │        对每个 slot：filterAssets（tags.X 过滤）→ weightedRandom
    │        返回 map[slot]*Asset + []AssetUsed
    │
    ├─[6]─► buildSlotValues(chosen, req.DynamicData)
    │        合并 Asset.Value 和 DynamicData（DynamicData 覆盖）
    │
    └─[7]─► renderHTML(tmpl, slotVals)
              或 renderNative(tmpl, slotVals)
              返回 RenderResult
```

## 时序图

```
Client          Handler         Engine          Cache(Redis)    PGRepo
  │               │               │                 │             │
  │──POST /render─►               │                 │             │
  │               │──Render(req)─►│                 │             │
  │               │               │──GetRules()────►│             │
  │               │               │◄─cache miss─────│             │
  │               │               │──GetRulesByCampaign()────────►│
  │               │               │◄─[]Rule──────────────────────│
  │               │               │──SetRules()────►│             │
  │               │               │  matchRule()    │             │
  │               │               │──GetTemplate()─►│             │
  │               │               │◄─cache miss─────│             │
  │               │               │──GetTemplate()───────────────►│
  │               │               │◄─*Template───────────────────│
  │               │               │──SetTemplate()─►│             │
  │               │               │  ParseSlots()   │             │
  │               │               │──GetAssets()──────────────────►
  │               │               │◄─[]*Asset────────────────────│
  │               │               │  selectAssets() │             │
  │               │               │  renderHTML()   │             │
  │               │◄─RenderResult─│                 │             │
  │◄──200 JSON────│               │                 │             │
```

## 状态机

### Asset 状态

```
创建 → active ──► inactive（软删除）
                      │
                      ▼
                  （不参与选择）
```

### Template 状态

```
创建 → active ──► inactive
```

### Rule 状态

```
创建 → active ──► inactive
  │                   │
  │  priority 值影响匹配顺序
  └──（多条活跃规则按 priority 排序）
```

## 缓存策略

| 缓存对象 | Key 格式 | 说明 |
|---------|---------|------|
| 规则列表 | `dco:rules:camp:{campaignID}` | 按活动缓存，更新规则时需清除 |
| 模板 | `dco:template:{id}` | 按模板 ID 缓存 |

Redis 不可用时（`cache == nil`）自动降级为直接查库，不影响服务可用性。
