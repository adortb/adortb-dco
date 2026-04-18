# adortb-dco

> adortb 平台动态创意优化（DCO）服务，基于规则匹配 + 加权随机选择素材，在毫秒级完成个性化广告渲染。

## 快速理解

- **本项目做什么**：接收广告渲染请求（含用户上下文），查找最优规则，从资产库加权随机选择素材，渲染为 HTML 或 Native JSON 广告
- **架构位置**：adortb-adx 竞拍胜出后调用 DCO 生成最终广告内容
- **核心入口**：
  - 服务启动：`cmd/dco/main.go`
  - HTTP 入口：`internal/api/handler.go:Handler.handleRender`
  - 渲染引擎：`internal/engine/engine.go:Engine.Render`

## 目录结构

```
adortb-dco/
├── cmd/dco/main.go             # 主程序：DB/Redis 初始化，demo 数据种植，端口 8090
└── internal/
    ├── api/handler.go          # HTTP Handler（路由注册、请求解析）
    ├── engine/
    │   ├── engine.go           # 核心渲染管线（Rules → Template → Assets → Render）
    │   ├── selector.go         # RenderRequest/RenderResult 类型定义 + 加权随机选择
    │   ├── renderer.go         # HTML/Native 渲染（Go text/template）
    │   └── cache.go            # Redis 缓存（规则 + 模板）
    ├── model/
    │   ├── template.go         # Template（HTMLTemplate/NativeTemplate/Slots）
    │   ├── asset.go            # Asset（SlotType/Value/Tags/Weight）
    │   └── rule.go             # Rule（Conditions/AssetSelection/Priority）
    └── repo/pg_repo.go         # PostgreSQL CRUD
```

## 核心概念

### 关键类型

| 类型 | 定义位置 | 说明 |
|------|---------|------|
| `RenderRequest` | `engine/selector.go` | 渲染请求（CampaignID/User/Context/DynamicData） |
| `RenderResult` | `engine/selector.go` | 渲染结果（RenderedHTML/RenderedNative/AssetsUsed） |
| `Repository` | `engine/engine.go:16` | 数据访问接口（便于测试时 Mock） |
| `Rule` | `model/rule.go` | 规则（含 Conditions/AssetSelection/Priority） |
| `Template` | `model/template.go` | 广告模板（ad_type: banner/native） |
| `Asset` | `model/asset.go` | 素材（含 Tags JSONB + Weight） |

### 渲染管线

```
Engine.Render(req)
    1. loadRules(campaignID)           → 优先 Redis Cache，按 Priority 降序
    2. matchRule(rule, req)            → 检查 Geo/UserTags/HourIn 条件，找第一个匹配
    3. loadTemplate(matched.TemplateID)→ 优先 Redis Cache
    4. loadAssets(rule, slots)         → 按 slot_type + advertiser_id 查库
    5. selectAssets(rule, assets)      → 加权随机（weightedRandom），支持 tags.category 过滤
    6. buildSlotValues(chosen, req.DynamicData) → dynamic_data 优先级最高
    7. renderHTML / renderNative       → Go text/template / JSON 字符串替换
```

### 规则条件（model/rule.go）

```json
{
  "conditions": {
    "geo": ["CN", "HK"],
    "user_tags": ["interest:electronics"],
    "hour_in": [8, 22]
  },
  "asset_selection": {
    "headline": {"filter": {"tags.category": "electronics"}},
    "image":    {"filter": {"tags.style": "promo"}}
  }
}
```

## 开发指南

### Go 版本

```bash
export PATH="$HOME/.goenv/versions/1.25.3/bin:$PATH"
```

### 本地运行

```bash
export DB_DSN="postgres://postgres:postgres@localhost:5432/adortb?sslmode=disable"
export REDIS_ADDR="localhost:6379"
go run cmd/dco/main.go

# 测试渲染
curl -X POST http://localhost:8090/v1/render \
  -H "Content-Type: application/json" \
  -d '{"campaign_id":123,"user":{"geo":"CN","tags":["interest:electronics"]},"context":{"hour":14}}'
```

### 测试

```bash
go test ./... -cover -race
go test ./internal/engine/... -v  # 引擎单元测试
```

### 代码约定

- `Engine` 通过 `Repository` 接口依赖注入，测试时可用 `MockRepo`
- 规则 JSON 通过 `rule.ParseJSON()` 在加载后解析（懒解析，只在缓存 miss 时执行）
- `DynamicData` 中的 key 会覆盖从资产库选出的同名 slot 值
- Cache 为 nil 时静默跳过（无 Redis 时正常工作）

## 依赖关系

- **上游**：adortb-adx（调用 DCO 获取个性化广告内容）
- **下游**（本服务依赖）：
  - PostgreSQL：模板/素材/规则数据
  - Redis（可选）：规则和模板缓存
- **依赖的库**：`lib/pq`（PostgreSQL），`go-redis/v9`，`prometheus`

## 深入阅读

- 加权随机算法：`internal/engine/selector.go:weightedRandom`
- Tag 过滤逻辑（`tags.category` 点语法）：`selector.go:filterAssets`
- Redis 缓存设计：`internal/engine/cache.go`
- Demo 数据（中文/英文电子产品广告示例）：`cmd/dco/main.go:seedDemoData`
