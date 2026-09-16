# Exploratory Thesis Monitoring

PAPER-01 monitoring records the original thesis, checkpoint timestamps, observed state, provenance, invalidation condition, recovery condition, and operator commentary. A checkpoint can be `TOUCHED`, `NOT_TOUCHED`, `UNKNOWN`, or `AMBIGUOUS`; unknown and ambiguous states are evidence, not permission to infer success.

Monitoring is bounded to the package-defined five-session window. A missed observation remains missing. The original thesis and rule version remain immutable after approval.
