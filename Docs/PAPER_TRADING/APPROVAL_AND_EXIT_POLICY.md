# Exploratory Approval and Exit Policy

Human approval is required before a simulated PAPER-01 order is created. Approval records the reviewer, time, thesis version, assumptions, provenance, risk boundary, invalidation condition, and explicit statement that the record is limited to the isolated simulated paper venue.

Exit review is rule-based and records the first applicable reason: `STOP`, `TARGET`, `THESIS_INVALIDATED`, `RISK_KILL`, `TIME_LIMIT`, or `MANUAL_OPERATOR`. Missing or contradictory evidence is retained through thesis review, uncertainty, or unresolved state and does not invent an exit reason. Simulated paper exit orders/fills remain inside the isolated venue; no broker/live order or fill can be created, and exploratory observations cannot become formal evidence.
