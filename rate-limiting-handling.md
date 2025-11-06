# Rate Limiting Handling Plan (Updated Final Strategy)

## Quick Start

For most use cases, start with these minimal settings:

**Environment (optional, uses defaults if not set):**
```bash
KANBANIZE_RETRY_MAX_ATTEMPTS=10        # 10 attempts per endpoint
KANBANIZE_RETRY_INITIAL_DELAY=5s       # Start with 5s backoff
KANBANIZE_RETRY_MAX_DELAY=300s         # Cap at 5 minutes
KANBANIZE_RETRY_TOTAL_CAP=1200s        # Global 20 min timeout
```

**MCP Tool Call:**
```json
{
  "tool": "read_card_with_retry",
  "arguments": {
    "card_id": "12345"
  }
}
```

This uses defaults: full jitter, exponential backoff (2x), Retry-After header respect, partial results enabled.

**For strict mode (fail if any endpoint fails):**
```json
{
  "tool": "read_card_with_retry",
  "arguments": {
    "card_id": "12345",
    "fail_on_partial": true
  }
}
```

## 0. Current State Recap

- `ReadCard` (client.go) performs 3 sequential GETs: card, comments, subtasks.
- No retry/backoff; HTTP 429 returns generic error.
- Rate limiting surfaced during bulk descendant fetches (multiple 429 responses).

## 1. Goals

| Goal | Description | Success Metric |
|------|-------------|----------------|
| Resilience | Automatically recover from transient 429s | <5% manual re-run rate |
| Fairness | Avoid hammering API under contention | Jitter prevents synchronized retries |
| Observability | Clear insight into retry behavior | Logs + counters present |
| Backward Compatibility | Preserve existing tool semantics | `read_card` unchanged |
| Partial Usefulness | Deliver partial data (card) even if comments/subtasks fail | Enabled via flag |

## 2. Architecture Overview

```
internal/kanbanize/
├── client.go                  (existing basic calls)
├── retry.go                   (NEW core backoff + helpers)
├── transport_retry.go         (Phase 3 optional: RoundTripper for global GET retry)
├── types.go                   (add RateLimitError)
└── client_test.go             (extend with retry tests)

main.go
└── register tools: read_card, read_card_with_retry
```

### New Tool: `read_card_with_retry`
Parameters:
| Name | Type | Default | Description |
|------|------|---------|-------------|
| card_id | string | required | Root card ID or URL |
| max_attempts | int | 10 | Upper bound attempts per endpoint |
| initial_delay_ms | int | 5000 | Initial backoff (ms) |
| max_delay_ms | int | 300000 | Max single delay (5 min) |
| multiplier | float | 2.0 | Exponential growth factor |
| full_jitter | bool | true | Use full jitter (0..delay) |
| respect_retry_after | bool | true | Honor server header if present |
| total_wait_cap_ms | int | 1200000 | Global cap (~20 min) |
| fail_on_partial | bool | false | If true, abort when secondary endpoints fail |

## 3. Composite Fetch Strategy

`ReadCardWithRetry` logic:
1. Fetch primary card with retry (required).
2. In parallel (fan-out): fetch comments & subtasks each with independent retry loops sharing a global time budget.
3. Assemble response; if `fail_on_partial=false`, include `partial_error` map.
4. Provide attempt counts and cumulative wait time.

## 4. Rate Limit Detection

```go
type RateLimitError struct {
    StatusCode  int
    RetryAfter  time.Duration // parsed from Retry-After header (seconds or HTTP-date)
    RawBody     string        // original body for diagnostics
}
```

Detection:
- HTTP 429 strictly.
- Optional future extension: treat specific 503 bodies containing known throttling phrases.
- Parse headers: `Retry-After`, optionally `X-RateLimit-Reset` (if available).

## 5. Backoff Algorithm (Full Jitter)

Formula per attempt `a`:
```
base = min(maxDelay, initialDelay * multiplier^a)
delay = rand(0, base)   // full jitter
if Retry-After present and > delay: delay = Retry-After
```
Advantages:
- Reduces synchronized retries more effectively than ±percentage jitter.
- Simple to reason about.

Global Constraints:
- Track cumulative waited time; stop if > total_wait_cap_ms.
- Abort immediately if context canceled.

## 6. Partial Results Semantics

Response Envelope:
```json
{
  "card_id": "24581",
  "attempts": { "card": 2, "comments": 3, "subtasks": 1 },
  "wait_seconds": 37.5,
  "rate_limit_hits": 4,
  "completed": { "card": true, "comments": true, "subtasks": false },
  "partial_error": { "subtasks": "max retries exceeded (429)" },
  "data": { "title": "...", "comments": [..], "subtasks": [] }
}
```
If `fail_on_partial=true`, any secondary failure aborts with final error.

## 7. Circuit Breaker (Phase 3+)

Maintain rolling window of recent 429 timestamps. If >N (e.g. 8) in last minute:
- Increase initialDelay baseline to e.g. 10s for subsequent calls until window cools down.
- Prevents persistent hot-looping under global throttling.

## 8. Metrics & Logging

Structured logging (key=value):
```
level=info event=rate_retry attempt=3 endpoint=comments delay_ms=10000 reason=429
level=info event=rate_success attempts=2 endpoint=card total_wait_ms=5200
level=warn event=rate_partial endpoint=subtasks error="max retries"
```

Internal counters (in-memory; future Prometheus integration):
- `retry_attempts_total`
- `rate_limit_hits_total`
- `retry_aborts_total`
- `retry_total_wait_ms`

## 9. Configuration Validation

On load:
- `MaxAttempts >= 1`
- `Multiplier >= 1.0`
- `InitialDelay > 0`, `MaxDelay >= InitialDelay`
- `total_wait_cap_ms >= InitialDelay`
- Clamp jitter to full-jitter if `full_jitter=true` ignoring `JitterPercent`.

## 10. Error Handling Matrix

| Scenario | Action |
|----------|--------|
| 429 with Retry-After | Honor header exactly (ceil to duration) |
| 429 without Retry-After | Compute backoff with full jitter |
| Non-429 error (4xx/5xx) | Fail fast (no retry) |
| Network timeout | Retry up to 2 attempts (separate light strategy) |
| Context canceled | Abort immediately, return partial if available |
| MaxAttempts exceeded | Return error / partial based on flag |

## 11. Testing Plan (Expanded)

| Test | Purpose |
|------|---------|
| Parse Retry-After seconds | Correct numeric parsing |
| Parse Retry-After HTTP-date | Adjust delay to time delta |
| Full jitter bounds | Ensure 0 <= delay <= base |
| Multiple 429 then success | Confirms recovery path |
| Max attempts exceeded | Returns structured error |
| Partial results enabled | Comments fail, card OK |
| Partial results disabled | Entire call fails |
| Context cancellation mid-sleep | Aborts promptly |
| Circuit breaker escalation | Raises initial delay after spike |
| Parallel secondary fetch | No race conditions |

## 12. Implementation Phases (Revised)

Phase 1 (Core): retry.go + RateLimitError + `ReadCardWithRetry` (sequential).
Phase 2 (Composite): parallel comments/subtasks, partial results envelope, tool params.
Phase 3 (Transport): `RateLimiterTransport` RoundTripper + circuit breaker.
Phase 4 (Metrics): counters + structured logging refinement.
Phase 5 (Docs): README section + usage examples.

## 13. Tool Documentation Snippet

`read_card_with_retry` description:
"Fetches a card and optionally comments/subtasks using exponential full-jitter backoff with respect for Retry-After headers. Returns structured envelope including attempts, wait time, and partial errors when enabled."

## 14. Example Minimal Go Snippets

```go
func backoff(cfg RetryConfig, attempt int, retryAfter time.Duration) time.Duration {
    if retryAfter > 0 && cfg.RespectRetryAfter { return retryAfter }
    base := cfg.InitialDelay
    if attempt > 0 {
        base = time.Duration(float64(base) * math.Pow(cfg.Multiplier, float64(attempt)))
    }
    if base > cfg.MaxDelay { base = cfg.MaxDelay }
    // full jitter
    max := base.Nanoseconds()
    return time.Duration(rand.Int63n(max + 1))
}
```

**Note:** This snippet calculates the delay per attempt. The actual retry loop must also track cumulative wait time and abort if `total_wait_cap_ms` is exceeded, ensuring the global time budget is respected.

## 15. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Excessive latency | User frustration | total_wait_cap_ms limit + progress output |
| Thundering herd | API instability | full jitter + circuit breaker |
| Partial data misused | Incorrect decisions | Explicit `partial_error` + documentation |
| Reviewer overload on change | Slowed adoption | Separate tool (opt-in) |

## 16. Open Questions (Tracked)

1. ~~Return progressive streaming updates?~~ **Resolved:** MCP protocol doesn't support streaming responses. Progressive updates will be provided via structured log output (stderr) that clients can monitor.
2. Include metric export now or later? (Phase 4 target.)
3. Add per-call override for total_wait_cap_ms? (If needed for long batch ops.)

## 17. Summary

This final plan modernizes rate limit handling with:
- Dedicated retry tool (`read_card_with_retry`).
- Full-jitter exponential backoff respecting server hints.
- Partial result support for improved usability.
- Future-proof transport layer & circuit breaker.
- Clear metrics & structured logging path.

Ready to implement Phase 1 immediately; later phases layer on without breaking prior functionality.

