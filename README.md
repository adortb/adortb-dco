# adortb-dco

> adortb 平台的动态创意优化（DCO）服务，基于用户上下文、地域、时段等维度，实时选择最优模板与素材组合，渲染个性化广告。

## 架构定位

```
┌─────────────────────────────────────────────────────────────────┐
│                      adortb 平台整体架构                         │
│                                                                  │
│  adortb-adx ──竞拍胜出──► adortb-dco ──渲染结果──► SDK          │
│                           ★ DCO Service                         │
│                            ↓                                    │
│                       [Rule Engine]                             │
│                       用户上下文匹配最优规则                      │
│                            ↓                                    │
│                       [Selector]                                │
│                       加权随机选择素材                            │
│                            ↓                                    │
│                       [Renderer]                                │
│                       Go template / native JSON 渲染            │
│                                                                  │
│  PostgreSQL ←→ PGRepo ←→ Engine ←→ Redis Cache                 │
└─────────────────────────────────────────────────────────────────┘
```

DCO 服务是广告内容**个性化渲染层**，在毫秒级完成：规则匹配 → 素材选择 → 广告渲染。

## 目录结构

```
adortb-dco/
├── go.mod                          # Go 1.25.3，依赖 lib/pq、redis、prometheus
├── cmd/dco/
│   └── main.go                     # 主程序：DB/Redis 初始化、路由注册、demo 数据
├── client/                         # Go 客户端（供其他服务调用 DCO API）
├── migrations/                     # PostgreSQL 数据库迁移 SQL
└── internal/
    ├── api/
    │   └── handler.go              # HTTP 路由：/v1/render, /v1/templates, /v1/assets, /v1/rules
    ├── engine/
    │   ├── engine.go               # 核心引擎：Rule 匹配 → Asset 加载 → 渲染
    │   ├── selector.go             # 加权随机素材选择（RenderRequest/RenderResult 类型）
    │   ├── renderer.go             # HTML/Native 渲染器（Go text/template）
    │   └── cache.go                # Redis 缓存层（规则、模板）
    ├── model/
    │   ├── template.go             # 模板模型（HTMLTemplate/NativeTemplate/Slots）
    │   ├── asset.go                # 素材模型（SlotType/Value/Tags/Weight）
    │   └── rule.go                 # 规则模型（Conditions/AssetSelection/Priority）
    ├── repo/
    │   └── pg_repo.go              # PostgreSQL 数据访问层
    └── metrics/
        └── metrics.go              # Prometheus 指标
```

## 快速开始

### 环境要求

- Go 1.25.3
- PostgreSQL
- Redis（可选，用于缓存加速）

```bash
export PATH="$HOME/.goenv/versions/1.25.3/bin:$PATH"
```

### 运行服务

```bash
cd adortb-dco

# 配置环境变量
export DB_DSN="postgres://postgres:postgres@localhost:5432/adortb?sslmode=disable"
export REDIS_ADDR="localhost:6379"   # 可选
export ADDR=":8090"

# 启动（自动执行 DDL 迁移 + 写入 Demo 数据）
go run cmd/dco/main.go
```

### 运行测试

```bash
go test ./... -cover -race
```

## HTTP API

### POST /v1/render

执行 DCO 渲染，根据请求上下文返回个性化广告 HTML 或 Native JSON。

**请求体**：

```json
{
  "campaign_id": 123,
  "user": {
    "geo": "CN",
    "tags": ["interest:electronics"]
  },
  "context": {
    "hour": 14,
    "domain": "example.com"
  },
  "dynamic_data": {
    "price": "¥999"
  }
}
```

**成功响应**（HTTP 200）：

```json
{
  "template_id": 1,
  "rule_id": 1,
  "rendered_html": "<div class=\"ad-300x250\">...</div>",
  "assets_used": [
    {"slot": "headline", "asset_id": 2},
    {"slot": "image", "asset_id": 5}
  ]
}
```

### 模板管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/templates` | 列出所有模板 |
| POST | `/v1/templates` | 创建模板 |
| GET | `/v1/templates/{id}` | 查询模板 |
| PUT | `/v1/templates/{id}` | 更新模板 |

### 素材管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/assets` | 列出素材（可按 slot_type 过滤） |
| POST | `/v1/assets` | 创建素材 |
| GET | `/v1/assets/{id}` | 查询素材 |

### 规则管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/rules` | 列出规则（可按 campaign_id 过滤） |
| POST | `/v1/rules` | 创建规则 |
| GET | `/v1/rules/{id}` | 查询规则 |
| PUT | `/v1/rules/{id}` | 更新规则 |

### GET /health

```json
{"status": "ok"}
```

## 配置说明

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `DB_DSN` | `postgres://...@localhost/adortb` | PostgreSQL 连接串 |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址（不设则不缓存） |
| `ADDR` | `:8090` | 监听端口 |

## DCO 渲染流程

```
POST /v1/render
    │
    ▼
[Handler]          解析 RenderRequest，校验 campaign_id
    │
    ▼
[Engine.Render]
    │
    ├─► loadRules(campaignID)     优先从 Redis Cache 读取
    │   按 Priority 降序排序
    │   matchRule(rule, req)      检查 Geo/UserTags/HourIn 条件
    │
    ├─► loadTemplate(ruleID)      优先从 Redis Cache 读取
    │   ParseSlots()              解析 {{slot}} 占位符
    │
    ├─► loadAssets(rule, slots)   按 slot_type + advertiser_id 查库
    │
    ├─► selectAssets()            加权随机（weightedRandom）
    │   filterAssets()            支持 tags.category 过滤
    │
    └─► renderHTML / renderNative
        HTML: Go text/template.Execute
        Native: JSON 字符串替换 + Unmarshal
```

## 数据模型

### 模板（dco_templates）

| 字段 | 类型 | 说明 |
|------|------|------|
| `ad_type` | varchar | `banner` / `native` |
| `html_template` | text | Go text/template 语法，用 `{{.slot_name}}` 引用 slot |
| `native_template` | jsonb | `{"title": "{{headline}}", ...}` |
| `slots` | jsonb | slot 名称列表 |

### 规则（dco_rules）

| 字段 | 类型 | 说明 |
|------|------|------|
| `conditions` | jsonb | `{"geo": ["CN"], "user_tags": [...], "hour_in": [8,22]}` |
| `asset_selection` | jsonb | 各 slot 的 tag 过滤条件 |
| `priority` | int | 数值越大优先级越高 |

## 性能设计

| 机制 | 参数 | 说明 |
|------|------|------|
| Redis 规则缓存 | TTL 可配 | 避免每次请求查库 |
| Redis 模板缓存 | TTL 可配 | 模板渲染前检查缓存 |
| DB 连接池 | MaxOpen=20, MaxIdle=5 | 防止过多空闲连接 |
| 加权随机 | O(n) | 支持 AB 测试和流量分配 |

## 相关项目

| 项目 | 说明 |
|------|------|
| [adortb-adx](https://github.com/adortb/adortb-adx) | 竞拍引擎，DCO 的上游调用方 |
| [adortb-common](https://github.com/adortb/adortb-common) | 配置加载、日志工具 |
| [adortb-infra](https://github.com/adortb/adortb-infra) | 基础设施（Docker Compose / K8s） |
