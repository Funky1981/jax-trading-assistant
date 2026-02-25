# DB Schema Master Plan

## Core table groups
- Platform/governance: runtime state, incidents, incident_actions
- Event domain: event_raw, event_normalized, event_symbol_map, event_sources
- Strategy/config: strategy_instances, versions, artifact_evidence_runs
- Research: datasets, dataset_versions, runs, run_metrics, run_trades, research_projects
- Execution: signals, approvals, order_intents, broker_orders, fills, positions_snapshots, flatten_reports, reconciliations
- AI audit: ai_decisions, ai_decision_acceptance, prompt_templates, prompt_template_versions
- Testing/trust: test_runs, gate_status, proof_artifacts, shadow_parity_results

## Rules
- immutable run config and dataset snapshot references
- append reconciliation corrections rather than mutate history
