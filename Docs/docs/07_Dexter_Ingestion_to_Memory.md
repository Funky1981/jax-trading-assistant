# 07 — Dexter Ingestion -> Memory Retain (Events + Signals)

**Goal:** Convert Dexter outputs into canonical events/signals and retain them.

## 7.1 — Event normalization
Dexter may output:
- earnings detected
- unusual volume
- news headline
- price gap

Normalize into `MarketEvent` and `Signal` DTOs.

## 7.2 — TDD
- Table-driven tests for normalization:
  - input payload -> expected canonical event
  - tag extraction
  - summary generation

## 7.3 — Retention rules
- Retain only when:
  - event is significant (configurable threshold)
  - or user explicitly bookmarks it
- Include minimal structured fields:
  - `event_type`, `impact_estimate`, `confidence`

## 7.4 — Definition of Done
- Normalizer covered by unit tests
- A single integration test:
  - feed sample Dexter output
  - verify `memory.retain` called with correct bank and item
