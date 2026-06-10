# 11 — Folder Placement

Recommended repo placement:

```text
Docs/plans/macro-reaction-engine/
  README.md
  00_IMPLEMENTATION_ORDER.md
  01_MACRO_EVENT_MODEL_AND_CALENDAR_DATA.md
  02_CHART_REACTION_ENGINE.md
  03_ETF_MAPPING_AND_SCENARIO_PLAYBOOKS.md
  04_PRICED_IN_AND_CONFOUNDER_CHECKS.md
  05_EVIDENCE_BUNDLE_BUILDER.md
  06_CANDIDATE_TRADE_GENERATOR.md
  07_UI_AND_API_INTEGRATION.md
  08_BACKTESTING_AND_UAT.md
  09_CODEX_MASTER_PROMPT.md
  10_PHASED_CODEX_TICKETS.md
```

Update existing:

```text
Docs/plans/README.md
```

Add row:

```md
| `macro-reaction-engine/` | Macro calendar, chart reaction, ETF confirmation, evidence bundles, and paper-only candidate generation |
```

Optional cross-link:

```text
Docs/plans/world-monitor-jax-awareness/README.md
```

Add:

```md
After World Monitor triggers are safely ingested, continue with:
`Docs/plans/macro-reaction-engine/README.md`
```
