# Research

Background research for Equinox, done 2026-10-05/06 against the live public APIs. Each report says which
claims were verified live and which come from documentation.

| File | Contents |
|---|---|
| [`kalshi_api.md`](kalshi_api.md) | Kalshi Trade API v2: endpoints, pagination, fixed-point fields, the bids-only order book, fee schedule and overrides, rate limits |
| [`polymarket_api.md`](polymarket_api.md) | Polymarket Gamma + CLOB: keyset paging, JSON-in-string fields, outcome tokens, worst-first books, the `feeSchedule` curve, rate limits |
| [`prior_art.md`](prior_art.md) | Aggregators, academic matching work, markets that resolved differently across venues, smart-order-routing concepts |
| [`overlap_census.md`](overlap_census.md), [`labelled_pairs.json`](labelled_pairs.json) | A live census of overlapping markets and the 55 hand-labelled pairs used as the matcher's CI gate |
| [`audit/`](audit/), [`audit2/`](audit2/), [`audit3/`](audit3/) | Live precision audits: the pairs shown to the judges (`chunk*.json`) and every verdict with its reason (`labels.json`). See `docs/TEST_RESULTS.md` §4 |
| `raw/` | The raw API captures that the reports cite (37 files, about 3 MB). Two large cited captures are not committed: `raw/series_list_all.json` (18.6 MB; the same `/series` response is in `testdata/snapshot/`) and `raw/events_open_limit200_nested_page1.json` (3.9 MB; an `/events` page of the same shape as those in `testdata/snapshot/`) |

**Correction to `polymarket_api.md`:** it estimates 70–85k open order-book markets. A later verification crawl
estimated about 257k. The design doesn't depend on the exact figure: ingestion is scoped to the 3,000 most-traded
markets either way.
