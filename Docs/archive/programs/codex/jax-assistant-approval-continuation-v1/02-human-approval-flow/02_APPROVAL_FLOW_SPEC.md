# Approval Flow Spec

## Required flow
1. watcher creates candidate trade
2. candidate goes to `awaiting_approval`
3. frontend shows candidate card
4. user chooses:
   - approve
   - reject
   - snooze
   - request re-analysis
5. approval service records decision
6. only approved candidates become execution instructions
7. execution engine submits broker order
8. fills/status linked back to approval record

## Approval metadata to store
- `approved_by`
- `approved_at`
- `decision`
- `notes`
- `expiry_at`
- `reanalysis_requested`
- `source_candidate_id`
