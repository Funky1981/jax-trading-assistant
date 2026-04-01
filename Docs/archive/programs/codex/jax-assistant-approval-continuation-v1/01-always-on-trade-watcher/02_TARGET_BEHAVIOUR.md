# Target Behaviour

Jax should:
1. boot
2. load enabled strategy instances
3. continuously scan market/event conditions
4. create **candidate trades**
5. run hard checks before they enter approval queue
6. publish updates to frontend
7. never require browser presence to keep scanning

## Candidate trade lifecycle
- `detected`
- `qualified`
- `blocked`
- `awaiting_approval`
- `approved`
- `rejected`
- `expired`
- `submitted`
- `filled`
- `cancelled`

## Hard checks before approval queue
- strategy instance enabled
- session valid
- flatten deadline still valid
- no duplicate open candidate for same symbol/instance
- no global kill switch active
- no max daily loss breach
- no fake/synthetic data provenance breach
