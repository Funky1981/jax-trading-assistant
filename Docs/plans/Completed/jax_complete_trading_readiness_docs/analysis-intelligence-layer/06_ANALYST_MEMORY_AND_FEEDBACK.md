# 06 — Analyst Memory and Feedback

## Goal

Make Jax improve from outcomes.

Expertise comes from remembering:

```text
what happened
what Jax expected
what Jax did
what actually happened
why it worked or failed
```

## Store every case study

For each event:

```text
event facts
playbook used
technical snapshot
fundamental snapshot
risk decision
candidate decision
operator approval/rejection
paper outcome
post-trade reflection
lesson learned
```

## Data model

### analysis_case_studies

```sql
CREATE TABLE analysis_case_studies (
    id UUID PRIMARY KEY,
    macro_event_id UUID NULL,
    symbol TEXT NOT NULL,
    playbook_key TEXT NOT NULL,
    decision TEXT NOT NULL,
    expected_outcome TEXT NOT NULL,
    actual_outcome TEXT NULL,
    outcome_r NUMERIC NULL,
    what_worked TEXT[] NOT NULL DEFAULT '{}',
    what_failed TEXT[] NOT NULL DEFAULT '{}',
    lesson TEXT NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ NULL
);
```

### analyst_feedback

```sql
CREATE TABLE analyst_feedback (
    id UUID PRIMARY KEY,
    case_study_id UUID NOT NULL REFERENCES analysis_case_studies(id),
    feedback_source TEXT NOT NULL,
    rating TEXT NOT NULL,
    comment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Reflection questions

After each paper trade or rejected event:

```text
Was the event correctly classified?
Was the ETF mapping correct?
Did the chart confirmation work?
Was the move already priced in?
Were confounders missed?
Was the stop too tight/wide?
Was the target realistic?
Did Jax chase?
Did Jax reject a good trade?
Did Jax accept a weak trade?
```

## Memory retrieval

Before making a new decision, Jax should retrieve similar past cases:

```text
same event type
same ETF
same playbook
similar surprise size
similar technical setup
similar market regime
```

## Example lesson

```text
NFP beats only produced clean QQQ bearish continuation when TLT also sold off and QQQ failed VWAP for at least 15 minutes.
```

## Codex task

```text
Add analyst memory and feedback loop.

Store every analysed event as a case study.
Attach paper outcomes later.
Retrieve similar cases during evidence bundle generation.
Add reflection records after outcome.
```

## Tests

```text
case study created after analyst decision
operator rejection creates feedback record
paper outcome updates case study
similar case retrieval returns matching event/playbook/symbol
reflection cannot modify original immutable decision
```

## Acceptance criteria

```text
all decisions become case studies
outcomes can be attached
feedback is stored
similar cases are available to evidence bundle
memory informs but does not override hard rules
```
