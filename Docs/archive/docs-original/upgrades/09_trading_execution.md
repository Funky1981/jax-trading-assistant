# Trading Execution

The ultimate goal of the Jax trading assistant is to place orders in live markets. To minimise financial risk, execution must be approached cautiously. Start with paper trading, add human approval gates and build adapters to reputable brokers.

## Why it matters

Even a well‑designed strategy can incur large losses if execution is flawed or if an order is sent without proper safeguards. Adapters must handle order placement, order tracking and error reporting reliably.

## Tasks

1. **Implement a broker adapter interface**
   - Define an interface (e.g. `OrderService`) with methods to place orders (`PlaceOrder`), check order status (`GetOrderStatus`) and cancel orders (`CancelOrder`).
   - Start by implementing this interface for a paper trading API (e.g. **Alpaca Paper Trading**, **OANDA practice accounts**, or **Interactive Brokers’ paper environment**).

2. **Approval workflow**
   - Require explicit human approval before sending an order to the broker. This could be via a UI action or a Slack/Teams integration.
   - Record approvals in the audit log with correlation IDs.

3. **Order validation and sizing**
   - Validate that the order quantity matches the calculated position size and that there is sufficient account balance.
   - Round sizes to the nearest lot size or minimum order increment supported by the broker.

4. **Error handling and retries**
   - Handle transient errors such as network issues or temporary broker outages with retries and exponential backoff.
   - Do not retry permanent errors (e.g. symbol not tradable) and surface them to the user.

5. **Order tracking**
   - Persist order IDs and statuses in storage. Update records when fills occur, and reconcile them with the broker’s API.
   - Expose an endpoint to query open orders and historical fills.

6. **Gradual rollout**
   - Start with paper trading for an extended period (weeks/months) to validate signals and execution reliability.
   - When moving to live trading, begin with very small position sizes and maintain a kill switch to halt trading on unusual behaviour.

7. **Regulatory considerations**
   - Ensure that order routing complies with regulatory requirements and broker agreements.
   - Document procedures for emergency liquidation and account risk limits.
