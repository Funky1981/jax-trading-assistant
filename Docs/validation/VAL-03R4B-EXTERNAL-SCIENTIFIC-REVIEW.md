# VAL-03R4B External Scientific Review

Status: `COMPLETE / FAILED_VALIDATION / EXTERNALLY REVIEWED`

External decision: `NO-GO FORWARD PAPER / FAILED HISTORICAL VALIDATION`

The single frozen recovery execution of `ma_crossover_v1` is final. The
primary sample floors passed: 108 actionable long episodes, 80 matched placebo
pairs, 16 non-empty instrument-year blocks, 9 instruments, and the registered
calendar/regime and concentration floors. The mean base-cost net return was
positive (`0.008069672809008646`) and the primary block-bootstrap lower bound
was positive (`0.0011135579758434141`).

Those facts do not establish incremental edge. The matched actual-minus-placebo
mean was negative (`-0.010025923924926792`) and the paired bootstrap lower
bound was negative (`-0.021823754952016352`). The preregistered top-5%
exclusion was also a blocking `FAIL`. Positive raw or base-cost returns
therefore did not survive the comparative placebo and robustness gates.

`ma_crossover_v1` is closed under this experiment identity. No tuning, rerun,
threshold reduction, parameter change, favourable-instrument selection, or
other salvage operation is permitted against the exposed 2025–2026 recovery
data. The 2023–2024 contaminated run is not rerun, and the R4B artifacts are
immutable.

Any modified moving-average strategy is a `NEW` hypothesis. The exposed
recovery period may be used for exploratory diagnosis, but it can never be
represented as unseen confirmation for a modified strategy.

Forward paper remains `NO-GO`; trading edge remains `NOT DEMONSTRATED`; VAL-04B
must be separately authorized after this infrastructure package is reviewed.
