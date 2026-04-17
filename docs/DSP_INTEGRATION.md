# DSP Integration Guide

## Overview

The DCO service assembles dynamic creatives at bid time. The DSP calls `POST /v1/render` and uses the returned `rendered_html` or `rendered_native` as the final `ad_markup` in the bid response.

## Endpoint

```
POST http://<dco-host>:8090/v1/render
Content-Type: application/json
```

## Request

```json
{
  "campaign_id": 123,
  "user": {
    "geo": "CN",
    "tags": ["interest:electronics", "high_value_user"]
  },
  "context": {
    "hour": 14,
    "domain": "news.com"
  },
  "dynamic_data": {
    "product_name": "iPhone 15",
    "price": "5999"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `campaign_id` | int64 | yes | Identifies which rules to evaluate |
| `user.geo` | string | no | ISO country code (e.g. "CN") |
| `user.tags` | []string | no | DMP segments / interest tags |
| `context.hour` | int | no | Request hour (0-23) for time-of-day targeting |
| `context.domain` | string | no | Publisher domain |
| `dynamic_data` | map | no | Product feed values — override matched asset values |

## Response

```json
{
  "template_id": 1,
  "rendered_html": "<div class=\"ad-300x250\">...</div>",
  "rendered_native": null,
  "assets_used": [
    {"slot": "headline", "asset_id": 10},
    {"slot": "image",    "asset_id": 20},
    {"slot": "cta",      "asset_id": 30}
  ],
  "rule_id": 5
}
```

For `ad_type=banner`, use `rendered_html` as the creative markup.
For `ad_type=native`, use fields from `rendered_native` to populate title, image_url, cta, etc.

## Integration Steps

1. **At bid time**, after winning impression eligibility, call `/v1/render` with campaign context.
2. **Replace** `BidResponse.seatbid[].bid[].adm` with `rendered_html`.
3. **Log** `rule_id` and `assets_used` for attribution and A/B reporting.
4. **Handle errors**: if DCO returns a non-200, fall back to the static creative.

## Performance

- P99 target: **< 20ms** (cold DB path) / **< 5ms** (Redis cached)
- The client SDK (`client/client.go`) has a 50ms timeout — set your circuit breaker accordingly.
- The Redis cache TTL for templates is **5 minutes** and for rules is **30 seconds**.

## Error Handling

| HTTP Status | Meaning | Action |
|-------------|---------|--------|
| 200 | Creative assembled | Use `rendered_html` / `rendered_native` |
| 400 | Missing `campaign_id` or bad JSON | Log and skip DCO |
| 422 | No matching rule | Use static creative fallback |
| 500 | Internal error | Use static creative fallback |
