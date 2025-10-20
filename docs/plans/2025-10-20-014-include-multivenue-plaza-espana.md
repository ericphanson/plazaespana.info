## Include multi-venue city events that explicitly mention Plaza de España (Option B)

Goal

- Surface from last weekend onwards any city events that explicitly include Plaza de España in their program copy, even if their canonical map point is elsewhere. Keep existing strict geo for plaza-core events.

Scope and constraints

- Apply only to city events (not cultural events).
- Time window: “last weekend onwards” = from Saturday 00:00, Europe/Madrid.
- Strict geo rule (radius) remains unchanged and continues to keep plaza-core items.
- This plan does not modify data sources; it adjusts filter logic and audit/reporting.

Why this change

- The audit shows plaza-core events are being kept by the radius filter.
- Some city-wide listings that clearly include Plaza de España are excluded because their canonical coordinates point to another area (e.g., Plaza Mayor, Sol, Recoletos).
- Example IDs observed (city_events):
  - 93553 Mercadillos navideños — description explicitly lists Plaza de España’s La Navideña (mercadillo + pista de hielo) but center is ~0.96 km away → excluded by radius.
  - 71133 Fiestas del Orgullo 2026 — description lists multiple plazas including Plaza de España; center ~1.11 km away → excluded by radius.

Definition of “explicitly mention Plaza de España”

- Accent-insensitive variants across Title, Venue, Address, Description:
  - plaza de españa, plaza de espana, pza. españa/espana, pl. españa/espana, plaza españa/espana
- Accept multi-venue phrasing that enumerates several locations and includes Plaza de España (e.g., “Plaza de Pedro Zerolo, Plaza del Rey, Puerta del Sol, Plaza de España”).
- Exclude generic/historical references not indicating programming there (e.g., “cómo llegar desde Plaza de España”, “historia de la Plaza de España”).

Proposed rules (Option B)

1) Keep plaza-core by geo (status quo)
   - If FilterResult.within_radius == true (e.g., gps_distance_km ≤ 0.35 km), keep.

2) Keep multi-venue city events that include Plaza de España in the program copy
   - Only for city_events (Category/collection = city, not cultural).
   - If any of Title, VenueName, Address, Description contains explicit Plaza de España variant matches (accent-insensitive), then keep, even if outside the strict radius.
   - Optional guardrails to reduce false positives (recommend enabling both):
     - Date proximity: event start_date must be within the horizon we publish (already filtered by the “last weekend onwards” window in pipeline).
     - Multi-venue cue: if the match appears in a sentence that also mentions other plazas/venues (Pedro Zerolo, Plaza Mayor, Colón, Sol, etc.), treat as multi-venue and keep; otherwise, still keep if phrasing expresses programming “en/acoge/junto a” Plaza de España.

3) Prefer geo when available
   - If both rules apply, mark reason as geo.
   - If only text applies, mark reason as multi-venue text.

Heuristics and matching details

- Normalization: Unicode NFKD + strip diacritics → lowercase → collapse whitespace.
- Variants list to check as substrings:
  - "plaza de espana", "plaza espana", "pza espana", "pza de espana", "pl espana", "pl de espana", "plz espana".
- Regex (broad fallback): \bpl(aza|\.?|z\.)?\s*(de\s*)?espa(n|ñ|ny)a\b
- Context filters to reduce noise:
  - Prefer lines with proximity tokens like: “en”, “se celebra en”, “acoge”, “programación”, “mercadillo”, “pista de hielo”, “concierto”, “evento”, “plaza”.
  - Exclude lines with “cómo llegar”, “historia de”, “cerca de” if no positive verbs nearby.

Filter pipeline insertion points

- internal/filter (text matching helper): add text matching util that:
  - Collects fields: Title, VenueName/Venue, Address, Description.
  - Applies normalization, checks variants, optional context heuristics.
  - Returns { text_matched: bool, text_reason: "plaza_espana_multivenue" | null }.
- Decision layer (city_events only):
  - kept := within_radius || text_matched.
  - If kept by text only, set FilterResult.multi_venue_kept = true and filter_reason = "kept (multi-venue: Plaza de España)".

Audit and reporting

- Extend FilterResult with:
  - text_matched (bool), text_reason (string, e.g., "plaza_espana_multivenue").
  - multi_venue_kept (bool) when text caused inclusion.
- In audit summary, add counters:
  - city_events.kept_multi_venue_plaza_espana (count)
  - city_events.excluded_outside_radius (count) (existing)
  - city_events.kept_geo (count)

Edge cases and handling

- No coordinates (lat/lon missing): rely on text matching; if match positive, keep.
- Ambiguous “Plaza de España” in other cities: we’re scoped to Madrid data set; if external feeds add other cities, require either Madrid city context or apply to city_events feed we already trust is Madrid-only.
- Historical mentions: apply the global time window (start_date >= last weekend start). Optionally also scan for year/month cues near the mention; defer unless false positives appear.
- Over-inclusion risk: monitor the multi_venue_kept counter; if spikes, enable the multi-venue cue requirement (must mention at least two venues while including Plaza de España) or add a curated allow/deny list.

Acceptance criteria

- Events physically in/adjacent to Plaza de España continue to be kept via geo.
- City events that explicitly list Plaza de España in their program copy are kept even if radius check fails.
- The audit shows non-zero city_events.kept_multi_venue_plaza_espana with concrete examples (e.g., 93553 Mercadillos navideños, 71133 Orgullo 2026 when in window).
- No cultural events are included solely via this text rule.

Testing plan

- Unit tests (internal/filter):
  - Positive: descriptions that enumerate multiple plazas including Plaza de España → text_matched true.
  - Positive: titles like “Pista de hielo de Plaza de España” → text_matched true.
  - Negative: “cómo llegar desde Plaza de España” with no verbs of hosting → text_matched false.
  - Negative: “historia de la Plaza de España” (museum talk elsewhere) → false unless radius matches.
- Integration tests (pipeline):
  - Event with gps_distance_km ~0.96 km (e.g., 93553) + explicit Plaza de España mention → kept (multi_venue_kept true).
  - Event with gps_distance_km > 4 km and generic city copy without Plaza de España → excluded.
  - City-only application verified: cultural events with text match but outside radius remain excluded.

Metrics and monitoring

- Add to build audit:
  - Count of kept via text vs kept via geo.
  - List (top N) of newly included via text (ID, title, start_date) for spot checking.
- Target: < 30% of plaza feed are text-only keeps; if exceeding, review heuristics.

Rollout plan

1) Implement text matcher and decision tweak behind a feature flag (e.g., FILTER_MULTI_VENUE_PLAZA=on).
2) Run a dry-run build (audit only) comparing before/after counts; record in docs/logs with examples.
3) Enable in production build; monitor counters for 3–5 build cycles.
4) If false positives appear, tighten with multi-venue cue requirement and/or add curated allow/deny IDs.

Repro and ops notes (optional, no code change)

- Quick jq to list excluded city events that mention Plaza de España (diagnostic):
  - .city_events.events[] | select(.FilterResult.kept==false and (.FilterResult.filter_reason=="outside GPS radius")) | grep mention in Title/Description/Venue/Address.
- Python audit helper (used in investigation) can be added to scripts/ for ongoing audits; not required for rollout.

Follow-ups (post-Option B improvements)

- If future feeds add structured venue lists, prefer per-venue instances or sub-events with per-venue coordinates.
- Consider a modest geo buffer (≤ 1.2 km) when text mentions Plaza de España but copy is ambiguous; keep this disabled initially.

Owner and timeline

- Owner: filtering/pipeline maintainers.
- Effort: ~0.5–1 day coding + 0.5 day tests and audit wiring.
- Target date: this week, pending code freeze windows.
