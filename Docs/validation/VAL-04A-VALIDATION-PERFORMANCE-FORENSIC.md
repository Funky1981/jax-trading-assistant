# VAL-04A Validation Performance Forensic

## Scope and safety

This forensic is infrastructure evidence only. The VAL-03R4B formal runner
was not executed, no historical result artifact was regenerated, and no market
data was reacquired. Measurements below use deterministic synthetic fixtures.

## Identified runtime source

The historical `timestampPlaceboSelections` implementation has the shape:

`replicate × instrument × year → scan every date → recompute actual-signal
eligibility and structural eligibility → rebuild a candidate slice → sort the
complete slice by SHA-256 score → take its first item`.

The dominant cost is the repeated eligibility path inside the replicate loop.
That path calls signal construction and structural-window checks for the same
instrument/date repeatedly, allocates candidate slices repeatedly, and sorts
all candidates when only the minimum score is required. Date and year ordering
also has to be reconstructed around the nested loops.

## Deterministic optimization

The reusable `internal/validation` package separates preparation from
selection. `PrepareTimestampPools` canonicalizes instruments, years, and dates,
then evaluates recovery/date eligibility, actual-signal exclusion,
active-interval exclusion, and structural eligibility once per in-range date.
It stores only eligible candidates in stable instrument/year strata.

`PreparedTimestampPools.Select` scans each prepared candidate pool once per
replicate and keeps the minimum SHA-256 score in memory. It does not sort the
candidate array merely to select a minimum. The selector preserves the frozen
SHA-256 score domain, replicate identity, instrument/year stratification,
stable selection order, selection JSON shape, digest shape, and
`future_matching=false` semantics.

## Non-formal benchmark methodology and measurements

The benchmarks use four synthetic instruments, two synthetic years, 500 dates
per instrument/year, 100 replicates, a fixed manifest string, and a deterministic
structural exclusion. The reference benchmark repeats the legacy scan and full
sort for every invocation. The optimized benchmark prepares once and measures
repeated selection, which represents the intended future-run lifecycle.

Environment: Windows amd64, Intel(R) Core(TM) i9-10980HK CPU @ 2.40GHz.

| Benchmark | ns/op | bytes/op | allocs/op |
|---|---:|---:|---:|
| `BenchmarkTimestampPlaceboReference` | 3,487,441,300 | 1,773,862,464 | 32,452,551 |
| `BenchmarkTimestampPlaceboOptimized` | 224,164,940 | 108,709,211 | 1,996,157 |

On this non-formal fixture, optimized selection was approximately 15.6× faster,
with approximately 93.9% fewer allocated bytes and 93.8% fewer allocations.
These figures are descriptive local measurements, not CI gates.

## Memory observations and remaining bottlenecks

Preparation retains one canonical string slice per eligible instrument/year
stratum, trading a bounded pool footprint for removal of repeated temporary
candidate slices. Selection still allocates its output slice and computes one
SHA-256 digest per candidate score; those are the remaining dominant costs in
the optimized selection stage. Future packages may pool output buffers or
cache score prefixes only if that preserves exact digest semantics. No such
change is needed for VAL-04A.

## Performance telemetry design

Future validation runs should collect non-scientific stage timings in a
separate diagnostic report or log with these independently measurable stages:

`data/preflight`, `primary evaluation`, `matched placebo`, `block bootstrap`,
`timestamp placebo`, `secondary diagnostics`, `robustness variants`, `other
falsifications`, and `serialization`.

Timing metadata must never enter scientific hashes, random seeds,
classification, result identity, or replay semantics. The VAL-03R4B artifact
contract is not retrofitted.
