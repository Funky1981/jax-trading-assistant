# Research Integration

The Dexter service provides research summaries and fundamental insights that complement technical signals. Currently, research integration is optional and minimally implemented.

## Why it matters

Combining technical signals with fundamental or sentiment analysis improves trade selection and decision quality. Automated research also saves analysts time and ensures each trade rationale is well documented.

## Tasks

1. **Define research questions**
   - Identify a core set of questions to ask for each trade. For example:
     - “Summarise the last four quarters of earnings.”
     - “Highlight key risks and catalysts for this trade idea.”
     - “Summarise recent news sentiment for this symbol.”
   - Make these configurable per strategy or symbol.

2. **Integrate Dexter client**
   - Flesh out the Dexter client in `services/jax-api/internal/app` or a new package. Handle authentication, request formatting and response parsing.
   - Implement batching if multiple questions are asked in one call. Handle rate limiting and timeouts gracefully.

3. **Attach research to trade setups**
   - When trade generation produces a setup, call Dexter with the configured questions and attach the returned `ResearchBundle` to the setup’s `Research` field.
   - If Dexter fails or times out, log the error and allow the trade to proceed without research (but mark it accordingly in the audit log).

4. **Caching and reuse**
   - Avoid redundant research calls by caching responses keyed by `(symbol, question)` and a freshness window (e.g. one day).
   - Invalidate cache entries when significant events occur (earnings releases, major news).

5. **Quality control**
   - Perform manual spot‑checks of Dexter outputs to ensure summaries are accurate and relevant.
   - Consider integrating with another research provider or building a fallback summariser for times when Dexter is unavailable.

6. **User experience**
   - Expose research results via the API and frontend, giving users the ability to drill into the details behind a trade.
   - Present research in a readable format (bullet points or paragraphs) and clearly separate it from technical signals.