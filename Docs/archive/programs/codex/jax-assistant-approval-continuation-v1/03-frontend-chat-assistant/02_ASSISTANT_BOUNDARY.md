# Assistant Boundary (Non-Negotiable)

## Allowed
The chat assistant may:
- explain a candidate trade
- summarize a signal/trade/run
- compare similar scenarios
- answer "what if" questions
- propose a paper-trade candidate
- request re-analysis
- retrieve research/RAG evidence

## Not allowed
The chat assistant must not:
- place orders
- approve trades on behalf of user
- bypass risk rules
- mutate strategy configs without explicit user action
- mark trust gates as passed
- submit execution instructions directly

## Enforcement
- chat endpoints are read-mostly
- any mutation requires separate explicit endpoint + auth + audit
- `cmd/trader` execution path never consumes raw chat output
