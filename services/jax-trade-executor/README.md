# Trade Executor Service

**Port:** 8097  
**Purpose:** Execute approved trading signals via Interactive Brokers with risk-based position sizing

---

## Overview

The Trade Executor service is responsible for converting approved trading signals into actual trades. It handles position sizing based on risk management rules, submits orders to Interactive Brokers via the IB Bridge, and stores trade records in the database.

---

## Features

- ✅ **Risk-Based Position Sizing** - Calculates share quantity based on account risk % and stop loss
- ✅ **Account Validation** - Checks buying power before placing orders
- ✅ **Order Execution** - Submits limit/market orders to IB Bridge
- ✅ **Trade Recording** - Stores complete trade details in database
- ✅ **Signal Validation** - Validates signal parameters before execution
- ✅ **R:R Calculation** - Tracks risk/reward ratio for each trade
- ✅ **Health Checks** - Monitors database and IB Bridge connectivity

---

## API Endpoints

### `GET /health`
Health check endpoint

**Response:**
```json
{
  "status": "healthy",
  "service": "jax-trade-executor",
  "database": "connected",
  "ib_bridge": {
    "connected": true,
    "url": "http://ib-bridge:8092"
  },
  "timestamp": "2026-02-06T10:30:00Z"
}
```

### `POST /api/v1/execute`
Execute an approved trading signal

**Request:**
```json
{
  "signal_id": "123e4567-e89b-12d3-a456-426614174000",
  "approved_by": "user@example.com"
}
```

**Response (Success):**
```json
{
  "success": true,
  "trade_id": "trade-uuid",
  "order_id": "12345",
  "message": "Trade executed: BUY 133 shares of AAPL",
  "trade": {
    "id": "trade-uuid",
    "symbol": "AAPL",
    "direction": "BUY",
    "quantity": 133,
    "entry_price": 150.00,
    "stop_loss": 145.00,
    "take_profit": 160.00,
    "risk_amount": 665.00,
    "risk_percent": 0.00665,
    "position_value": 19950.00,
    "rr_ratio": 2.0,
    "status": "pending",
    "submitted_at": "2026-02-06T10:30:00Z"
  }
}
```

**Response (Error):**
```json
{
  "success": false,
  "message": "trade execution failed",
  "error": "insufficient buying power: need $30000.00, have $20000.00"
}
```

### `GET /api/v1/trades`
List all trades (last 100)

**Response:**
```json
{
  "trades": [
    {
      "id": "trade-uuid",
      "symbol": "AAPL",
      "direction": "BUY",
      "entry": 150.00,
      "stop": 145.00,
      "targets": [160.00],
      "strategy_id": "ma_crossover_v1",
      "created_at": "2026-02-06T10:30:00Z",
      "risk": {
        "amount": 665.00,
        "percent": 0.00665,
        "position_value": 19950.00,
        "quantity": 133,
        "order_id": "12345",
        "status": "pending"
      }
    }
  ],
  "count": 1
}
```

---

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `POSTGRES_DSN` | PostgreSQL connection string | - | ✅ |
| `IB_BRIDGE_URL` | IB Bridge service URL | `http://localhost:8092` | ✅ |
| `RISK_PER_TRADE` | Risk % per trade (0.01 = 1%) | `0.01` | ❌ |
| `MAX_POSITION_PCT` | Max position size % (0.2 = 20%) | `0.20` | ❌ |
| `ORDER_TYPE` | Order type (LMT, MKT, STP) | `LMT` | ❌ |
| `PORT` | HTTP server port | `8097` | ❌ |

### Example Configuration

```bash
POSTGRES_DSN=postgresql://jax:jax@localhost:5432/jax
IB_BRIDGE_URL=http://localhost:8092
RISK_PER_TRADE=0.01        # Risk 1% per trade
MAX_POSITION_PCT=0.20       # Max 20% in one position
ORDER_TYPE=LMT              # Use limit orders
PORT=8097
```

---

## Trade Execution Workflow

1. **Receive Execution Request**
   - Signal ID and approved by user

2. **Fetch Signal from Database**
   - Query `strategy_signals` table
   - Verify status is 'approved'

3. **Validate Signal**
   - Check required fields
   - Validate stop loss placement
   - Ensure entry/stop/target are valid

4. **Get Account Information**
   - Query IB Bridge for account details
   - Get net liquidation and buying power

5. **Calculate Position Size**
   ```
   riskAmount = accountBalance × riskPerTrade
   stopDistance = |entryPrice - stopLoss|
   shares = riskAmount / stopDistance
   
   Apply constraints:
   - Minimum position size
   - Maximum position size
   - Maximum position value %
   - Buying power validation
   ```

6. **Create Order Request**
   - Build IB-compatible order
   - Set order type (limit/market)
   - Set price if limit order

7. **Submit to IB Bridge**
   - POST to `/orders` endpoint
   - Get order ID back

8. **Store Trade Record**
   - Insert into `trades` table
   - Include all trade details
   - Store risk metrics

9. **Update Trade Approval**
   - Add IB order ID to approval record
   - Link approval to execution

---

## Position Sizing Examples

### Example 1: Normal Trade
- **Account**: $100,000
- **Risk**: 1% = $1,000
- **Signal**: AAPL BUY @ $150, Stop @ $145
- **Stop Distance**: $5
- **Shares**: $1,000 / $5 = **200 shares**
- **Position Value**: $30,000 (30%)
- **Adjusted**: Within limits ✅

### Example 2: Position Value Limit
- **Account**: $100,000
- **Risk**: 1% = $1,000
- **Signal**: TSLA BUY @ $800, Stop @ $795
- **Stop Distance**: $5
- **Shares**: $1,000 / $5 = 200 shares
- **Position Value**: $160,000 (160%) ❌
- **Max Position**: 20% = $20,000
- **Adjusted Shares**: $20,000 / $800 = **25 shares** ✅

### Example 3: Insufficient Buying Power
- **Account**: $100,000
- **Buying Power**: $10,000
- **Signal**: AAPL BUY @ $150, Stop @ $145
- **Shares**: 200 shares
- **Required**: $30,000
- **Result**: **Error - Insufficient buying power** ❌

---

## Risk Management Rules

### Position Size Constraints
1. **Risk per trade**: 0.5% - 2% of account (configurable)
2. **Max position value**: 10% - 30% of account (configurable)
3. **Minimum shares**: 1 (prevents fractional shares)
4. **Maximum shares**: Optional hard limit

### Validation Rules
1. **BUY signals**: Stop loss must be below entry
2. **SELL signals**: Stop loss must be above entry
3. **Take profit**: Must be beyond entry (profitable direction)
4. **Prices**: All prices must be > 0

### Safety Checks
- ✅ Buying power validation before order
- ✅ Signal status must be 'approved'
- ✅ Account connection verified
- ✅ IB Bridge health check

---

## Database Schema

### trades Table (Used)
```sql
CREATE TABLE trades (
  id TEXT PRIMARY KEY,
  symbol TEXT NOT NULL,
  direction TEXT NOT NULL,        -- BUY or SELL
  entry DOUBLE PRECISION NOT NULL,
  stop DOUBLE PRECISION NOT NULL,
  targets JSONB NOT NULL,         -- Array of take profit levels
  event_id TEXT NULL,
  strategy_id TEXT NOT NULL,
  notes TEXT NULL,                -- Execution notes
  risk JSONB NULL,                -- Risk metrics
  created_at TIMESTAMPTZ DEFAULT now()
);
```

### trade_approvals Table (Updated)
```sql
CREATE TABLE trade_approvals (
  id UUID PRIMARY KEY,
  signal_id UUID REFERENCES strategy_signals(id),
  approved BOOLEAN NOT NULL,
  approved_by VARCHAR(100),
  modification_notes TEXT,
  order_id VARCHAR(100),          -- IB order ID (added by executor)
  approved_at TIMESTAMPTZ DEFAULT now()
);
```

---

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `signal not found or not approved` | Signal doesn't exist or not approved | Approve signal first |
| `invalid stop loss` | Stop on wrong side of entry | Fix signal parameters |
| `insufficient buying power` | Not enough capital | Reduce position or add funds |
| `failed to place order` | IB Bridge error | Check IB Gateway connection |
| `position size below minimum` | Risk too small for price | Increase risk % or choose different symbol |

### Error Response Format
```json
{
  "success": false,
  "message": "Descriptive error message",
  "error": "Detailed technical error"
}
```

---

## Testing

### Unit Tests
```bash
cd libs/trading/executor
go test -v
```

### Integration Test
```bash
# 1. Start services
docker compose up -d jax-trade-executor

# 2. Execute a trade
curl -X POST http://localhost:8097/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "signal_id": "your-signal-uuid",
    "approved_by": "test@example.com"
  }'

# 3. Check results
curl http://localhost:8097/api/v1/trades
```

---

## Monitoring

### Health Check
```bash
curl http://localhost:8097/health
```

### Docker Health Check
```bash
docker compose ps jax-trade-executor
```

### Logs
```bash
docker compose logs -f jax-trade-executor
```

---

## Deployment

### Docker
```bash
docker compose up -d jax-trade-executor
```

### Standalone
```bash
cd services/jax-trade-executor
go build -o jax-trade-executor ./cmd/jax-trade-executor
POSTGRES_DSN="..." IB_BRIDGE_URL="..." ./jax-trade-executor
```

---

## Dependencies

- **PostgreSQL**: Trade storage
- **IB Bridge**: Order execution
- **Go 1.23+**: Runtime

---

## Future Enhancements

- [ ] Order status monitoring (track fills/cancels)
- [ ] Partial fill handling
- [ ] Bracket orders (entry + stop + target in one)
- [ ] Position management endpoints
- [ ] P&L calculation
- [ ] Trade analytics and reporting
- [ ] Risk limit checks (max daily loss, max open positions)
- [ ] Notification on execution (webhook/email)

---

## Support

For issues or questions:
- Check health endpoint: `/health`
- Review logs: `docker compose logs jax-trade-executor`
- Verify IB Bridge connectivity
- Check database connection
