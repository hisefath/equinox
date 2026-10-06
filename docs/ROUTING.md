# Routing: how Equinox picks a venue, and how it explains itself

The code is [`internal/route/route.go`](../internal/route/route.go) (about 400 lines; imports only
`internal/market`). The diagram is [`diagrams/routing-decision.mmd`](diagrams/routing-decision.mmd).

## 1. Contract

```go
func Route(o Order, quotes []Quote, p Policy, now time.Time) Decision
```

- **Order:** a hypothetical taker **BUY** of `Qty` whole contracts of `Side` (`yes` or `no`), with an
  optional `Limit` (the worst acceptable price per contract, before fees).
- **Quote:** one venue's market for the equivalent proposition. It holds the canonical `Market` (fee
  curve, minimum size, tradability), its `Book`, and `Unhealthy`, a plain string explaining why the
  venue's last refresh failed.
- **Policy:** `MaxBookAge` (older books are excluded) and `Split` (whether one order may be split across
  venues).
- **now:** passed in, never read from the clock.
- **Decision:** status (`filled` / `partial` / `rejected`), allocations, totals, all-in price, an
  evaluation of every venue (including why it was excluded), plain-English explanation lines, and an
  `id` that hashes the inputs.

`Route` **never fails and never blocks**. An unroutable order returns a `rejected` decision that says
why. The function performs no I/O, reads no clock, uses no randomness and contains no venue names. The
build enforces the last point (`TestRouteIsVenueAgnostic`).

## 2. Algorithm

1. **Validate the order.** The side must be yes or no, the quantity 1..1,000,000, and the limit below
   $1. Otherwise the decision is rejected with the reason.
2. **Canonicalize.** Sort quotes by (venue, market id) and hash everything into the decision id. Input
   order can no longer matter.
3. **Check every quote for eligibility, recording the first failing reason:**
   - venue unhealthy: its last refresh failed;
   - market not tradable: closed, awaiting resolution, or an unknown fee schedule;
   - untrusted book: empty, crossed, or no timestamp;
   - stale book: `now − AsOf > MaxBookAge`;
   - no offers on the side being bought;
   - best ask above the limit;
   - fillable size below the venue's minimum order (Polymarket: 5 shares);
   - duplicate quote for the same market.
4. **Simulate each eligible venue on its own.** Walk the asks best-first up to the quantity and the
   limit. For each price level, `notional += price × qty` and `fee += FeeCurve.Fee(price, qty)`
   (exact to the micro-dollar). The summed fee is then rounded once per order the way the venue
   specifies: Kalshi to the cent, Polymarket to $0.00001. This gives
   `all-in per contract = (notional + fee) / qty`, rounded up.
5. **Pick the best single venue.** Rank by (a) most contracts filled, (b) lowest all-in cost per contract,
   compared exactly by cross-multiplication (`total_a × qty_b` vs `total_b × qty_a`, no division), then
   (c) venue id and (d) market id.
6. **Split, if allowed and more than one venue is eligible.** Build a consolidated book from every
   eligible level of every venue, each level keyed by its fee-inclusive price for a 1M-contract reference
   size. Fill greedily. If a venue's share falls below its minimum order size, drop that venue and fill
   again. Use the split only if it **fills more, or fills the same for strictly less**; otherwise keep
   one child order.
7. **Status.** `filled` if everything was allocated, `partial` if some was, `rejected` if none.

### Buying NO

A NO contract is the complement of YES: it pays when YES doesn't. Every book stores only YES bids and
asks, and `Book.AsksFor(No)` returns the YES bids at `$1 − p`. That is exactly how both venues work:
- **Kalshi** publishes NO bids, which are YES asks.
- **Polymarket**'s NO-token book mirrors its YES-token book (verified on 241 of 242 pairs).

The router needs no special case.

## 3. Fees

Fees decide routing more often than you might expect: near 50¢, Kalshi's taker fee is 1.75¢ per
contract. Both venues' published schedules fit one curve:

```
fee = q × rate × p^a × (1 − p)^b      (then the venue's per-order rounding)
```

| Venue | Source of parameters | rate | a, b | Rounding |
|---|---|---|---|---|
| Kalshi | series `fee_type` + `fee_multiplier`; an event's `fee_type_override` / `fee_multiplier_override` wins | 0.07 × multiplier (e.g. KXMLBGAME ×0.5; fee-free series ×0) | 1, 1 | up to the cent, per order (the fee accumulator makes a multi-fill order cost what one fill would) |
| Polymarket | market `feeSchedule {rate, exponent}`; zero when `feesEnabled` is false or there is no schedule | e.g. 0.04 politics, 0.05 sports/economics, 0.07 crypto, 0 geopolitics | exponent, exponent | up to $0.00001 (the direction isn't documented, so up is the conservative choice) |

These parameters arrive in the `Market` as a `FeeCurve`. The router evaluates the curve and never asks
which venue it belongs to. The tests reproduce the venues' own worked examples to the micro-dollar:
- Kalshi $1.75 for 100 @ $0.50, and its $0.00363825 → $0.01 rounding example.
- Polymarket's $1.00 for 100 @ $0.50 and $0.84 for 100 @ $0.30.

Fee rebates (maker rebates, Polymarket's taker-rebate tiers) are paid after the fact, so they are ignored.

## 4. Why these choices

| Choice | Reason |
|---|---|
| Rank by **all-in cost**, not headline price | A cheaper price can lose to fees, as in `TestFeesCanFlipTheDecision` and in the live Iowa Senate example below. Best-execution rules call this "total consideration" |
| Prefer a **full fill** over a cheaper partial | An order that leaves inventory unfilled is not the same trade |
| **Split only when it strictly helps** | Every child order adds execution risk. "Same cost, more orders" is not an improvement |
| **Exclude, don't guess** | Stale, crossed or unhealthy data is excluded with a reason, never repaired |
| **Integer money, sorted inputs, total-order tie-breaks** | Byte-identical decisions on any machine. Verified by running permutations and repetitions (`TestDeterministic`), and on real recorded books in the end-to-end test |
| **Decision id = hash of inputs** | An auditor can show two decisions saw identical inputs, or replay logged inputs |

## 5. Explanations and the decision log

Every decision explains itself in plain English. Here is a real decision on recorded live books
(2026-10-06 04:46 UTC), routing a 2,000-contract YES order for "Republicans win the Iowa Senate race":

```
Decision e90dca203225521a: FILLED
  - order: BUY 2000 yes at market, split allowed, max book age 30s
  - kalshi/SENATEIA-26-R: eligible; alone fills 2000 at all-in 0.58716/contract (fees 34.32)
  - polymarket/630734: eligible; alone fills 2000 at all-in 0.589744/contract (fees 19.488)
  - split considered; no improvement over best single venue, keeping one child order
  - route 2000 to kalshi/SENATEIA-26-R: notional 1140.00 + fees 34.32 = 1174.32 (all-in 0.58716/contract, worst price 0.57)
  - next best: polymarket/630734 at all-in 0.589744/contract for 2000
  - result: filled 2000/2000 contracts, total 1174.32, all-in 0.58716/contract
```

Kalshi's fee is almost twice Polymarket's here ($34.32 vs $19.49), but its price is a cent lower, and the
all-in comparison still favours Kalshi, by about 0.26¢ per contract. A second example on the same
recording goes the other way. Buying NO on "Fed hikes 25 bp in January 2027" with a $0.70 limit routes
to Polymarket at an all-in $0.65152, against Kalshi's $0.671867.

Each decision is appended to `decisions.jsonl` (CLI `-log`, default `data/decisions.jsonl`) as one JSON
line containing:

- `pair`, `tier`, `score`, `match_evidence`, `match_caveats`: **why we believed the venues were the same
  bet**;
- `decision`: order, policy, `at`, `id`, status, allocations, every venue's evaluation and exclusion
  reason, and the explanation lines: **why money went where it did**.

## 6. The routing gate (before `Route` is called)

Routing only makes sense between equivalent markets, so the caller (`cmd/equinox`) applies a gate first:

| Pair state | Routed? |
|---|---|
| `equivalent` (matcher) | yes |
| `review` (matcher) | only with `-min-tier review` |
| `rejected` (reviewed mapping table) | never |
| anything without a confirming review, when `-require-review` is set | no: the production setting |

## 7. Limitations (all deliberate for a simulation)

- Taker BUY only. A SELL would walk bids and maximize proceeds net of fees; it is symmetric and was left
  out for scope.
- Displayed size is assumed fillable. There is no market impact, queue position, latency or
  fill-probability model (cf. Cont & Kukanov's optimal order placement).
- Per-order fee rounding is applied after allocation, so the split ranking uses unrounded marginal fees
  (the error is under 1¢ per venue).
- No settlement or capital-lock-up penalty: Kalshi pays at certification, Polymarket on the race call.
  This is surfaced as a match caveat; a desk could turn it into a per-venue bps cost in `Policy`.
- No risk limits, position netting or venue credit limits.
