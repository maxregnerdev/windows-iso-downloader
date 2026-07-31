# Spec: "Needs Warming" Community Help Page

**Status:** Proposed
**Date:** 2026-07-13
**Author:** Shekhar + Grok (original idea), refined against live codebase by Claude
**Related:** Crowdsourced cache warming (`/contribute`), Sentinel lockdown resilience, CLI contribution flow

---

## 1. Goal

Give visible, honest signal for products currently failing web users because Microsoft's Sentinel WAF (or a rate limit) has blocked the backend and no cached link is available to fall back on. Point people at the CLI as the fix, and let a successful fetch/contribution clear the item automatically.

This is not a new mechanism bolted onto the project — it's a public window into resilience state (`negCache` / `linkCache`) that already exists and already drives the per-product "blocked, use CLI" banner. This just makes that signal visible in aggregate, with a stronger call to action.

**Change from the original draft:** a dedicated page, not a homepage section. Original draft (written by Grok from the GitHub repo alone, without runtime visibility into cache state) proposed a parallel tracking system; this version derives the list from state that already exists, which is both less code and immune to drift between two copies of "is this broken."

---

## 2. What Actually Needs Tracking (verified against `backend/main.go`)

`handleProxy` (main.go:1189-1404) has exactly three outcomes when a product/SKU is under an active Sentinel or rate-limit block:

| Outcome | Code path | User experience | Needs warming? |
|---|---|---|---|
| `serving cached` | negCache active, `linkCache` entry still within its own TTL (e.g. just contributed) | Gets a working link | **No** |
| `serving stale` | negCache active, `linkCache` entry expired but still returned | Gets a (possibly near-expired) link | **No** |
| `no stale available` | negCache active, nothing in `linkCache` at all | Gets a 429, no link | **Yes** — this is the actual failure |

So "needs warming" = **an active `negCache` entry with no corresponding valid-or-stale `linkCache` entry** — computed on read, not tracked as separate state. No new manager struct, no `Record()`/`Remove()` calls to keep in sync, no new Redis schema.

**Known gap, scoped out of MVP:** a fetch that fails with a non-rate-limit error (network blip, malformed response) and has no stale cache to fall back on returns an error directly (main.go:1377) without ever touching `negCache`. In practice this hasn't been the dominant failure mode this month — Sentinel/rate-limit blocks have been — so the MVP doesn't cover it. If it turns out to matter, closing the gap is a two-line addition (write a `negCacheEntry` with `IsSentinel: false` at that call site) rather than a redesign.

---

## 3. Data Needed That Doesn't Exist Yet

Two small additions, both minimal:

1. **A request counter per locked product/SKU.** `negCache`/`linkCache` don't track "how many times did this get hit while broken" — add a small `map[string]int` (or extend `negCacheEntry` with a counter field) incremented at the existing `"no stale available"` log call sites.
2. **Consumer product display names on the backend.** `validContributeProducts` (main.go:476) is `map[string]bool` — just an ID allow-list, no names. Eval products already have names (`{Name, EvalURL}` map at main.go:207-211). Converting `validContributeProducts` to `map[string]string` (ID → Name) serves both its existing validation role and this new lookup need, with no behavior change to `/contribute`.

Language name hydration for consumer products reuses the existing `skuCache` (already populated per-product from the normal SKU-fetch flow) — look up the SKU by ID within that product's cached SKU list to get `Language`/`LocalizedLanguage`. No new structure needed; if the product isn't in `skuCache` yet, just omit the language field rather than blocking the response.

---

## 4. New Endpoint

### `GET /needs-warming`

Computed on each request by iterating `negCache`, filtering to entries where `IsSentinel` (or rate-limited) is still active **and** no valid `linkCache`/eval-cache entry exists for that key. No background job, no persistence — it's a live view.

**Query params:** `limit` (default 10, max 50)

**Response:**
```json
{
  "items": [
    {
      "product_id": "3262",
      "sku_id": "0x0409",
      "product_name": "Windows 11 25H2",
      "language": "English (United States)",
      "is_eval": false,
      "reason": "waf_blocked",
      "last_seen": "2026-07-13T17:42:00Z",
      "request_count": 47,
      "cli_command": "msdl --id 3262 --lang \"English (United States)\""
    }
  ],
  "total": 3
}
```

- Public, unauthenticated, rate-limited (30 req/min per IP — matches the existing pattern used by `/contribute` at 5/min and `/telemetry` at 10/min)
- Sorted by `request_count` descending (surfaces the highest-impact items first, more useful than recency alone)
- Eval products included (`is_eval: true`, `cli_command` uses `msdl --eval <slug>` instead of `--id`)

---

## 5. Backend Implementation Outline

No new package-level state beyond the two additions in §3. The handler:

```go
func handleNeedsWarming(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if !needsWarmingRL.allow(clientIP(r)) {
        respondJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
        return
    }
    limit := parseLimitParam(r, 10, 50)

    negCacheMu.RLock()
    defer negCacheMu.RUnlock()
    linkCacheMu.RLock()
    defer linkCacheMu.RUnlock()

    var items []needsWarmingItem
    for key, neg := range negCache {
        if time.Now().After(neg.ExpiresAt) {
            continue // lockdown/rate-limit window itself already expired
        }
        if _, hasValidLink := resolveValidLink(key, linkCache); hasValidLink {
            continue // already resolved via cached/stale link -- not a failure state
        }
        items = append(items, buildNeedsWarmingItem(key, neg))
    }
    sort.Slice(items, func(i, j int) bool { return items[i].RequestCount > items[j].RequestCount })
    if len(items) > limit {
        items = items[:limit]
    }
    json.NewEncoder(w).Encode(map[string]interface{}{"items": items, "total": len(items)})
}
```

Integration points:
- Increment the new request counter at the existing `"lockdown active, no stale available"` and first-time-Sentinel/rate-limit-with-no-stale log sites (main.go:1225, ~1367)
- No changes needed to `/contribute` or the fresh-fetch success path — an item stops appearing automatically once `linkCache` has a valid entry, which those paths already populate

---

## 6. Frontend: Dedicated Page

**Route:** `/needs-warming` (or `/community` — bikeshed-able, not load-bearing)

Linked from the footer and from the per-product "blocked" banner (`CliHandoff.tsx`'s `highlight` variant) — "See what else needs help →" — so people already primed to help via the CLI can see the fuller picture instead of just their one product.

**Empty state:** hide the page's list section entirely (or show a short "Everything's healthy right now" line) — no "0 items" placeholder box. Most visits will likely find this empty, which is a good sign, not a gap to fill with UI.

Each card:
- Product name + language (or eval product name)
- Request count as a small badge (social proof — "47 people hit this")
- One-click "Copy CLI command" button (same interaction pattern as the existing `CliHandoff` component — reuse its copy-button styling)
- No per-item "reason" jargon exposed in the UI (WAF/rate-limit distinction is backend detail); just "temporarily unavailable"

Tone: matches the project's existing calm, transparent voice (README already explains the caching layer and Sentinel lockdown openly) — helpful, not alarmist.

---

## 7. CLI Impact

None required for MVP. Optional future addition: `msdl --needs-warming` prints the same list from the terminal and offers to fetch+contribute the top item directly — natural fit for the homepage-screen work already shipped (`cli/homepage.go`), but a separate phase.

---

## 8. Privacy & Safety

- Only ever stores/exposes `product_id`, `sku_id`/slug, reason, counters, timestamps — no IPs, no user identifiers (matches the existing telemetry/contribute philosophy already documented in README)
- Rate-limited public endpoint, same pattern as existing public endpoints
- Nothing new to abuse: the list is derived from state Microsoft's own WAF already put the backend into, not something a client can inject

---

## 9. Implementation Phases

| Phase | Description | Effort |
|---|---|---|
| MVP | Request counter + product-name map conversion + `/needs-warming` handler + dedicated page | Small — most of the state already exists |
| 2 | Link from per-product blocked banner; polish empty/loading states | Small |
| 3 | Close the non-rate-limit fetch-error gap (§2) if it turns out to matter in practice | Small, only if needed |
| 4 | `msdl --needs-warming` CLI command | Small |

(Dropped the original draft's "Redis-backed persistence" phase — this is inherently live/transient state mirroring `negCache`'s own in-memory, reset-on-restart behavior; persisting it would add a Redis key namespace for no real benefit.)

---

## 10. Open Questions Resolved

1. **Eval products included?** Yes.
2. **Max age before auto-removal?** No separate TTL — inherits the underlying `negCache` entry's own expiry (60s rate-limit / 90min Sentinel), since the list is computed live rather than tracked separately.
3. **Empty state?** Hide the list, don't show a "0 items" box.
4. **Tone?** Helpful/neutral, matching existing project voice.

---

## 11. Success Metrics (Future)

- Contributions triggered from this page specifically (could tag `/contribute` calls with a `source=needs-warming` query param to measure this)
- Reduction in time-to-resolution for locked-down products
- CLI download/usage bump correlated with items appearing on the page
