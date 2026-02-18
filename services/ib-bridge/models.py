"""
Pydantic models for API request/response validation
"""
from typing import Optional, List
from pydantic import BaseModel, Field


class ConnectRequest(BaseModel):
    """Request to connect to IB Gateway"""
    host: Optional[str] = None
    port: Optional[int] = None
    client_id: Optional[int] = None


class ConnectResponse(BaseModel):
    """Response from connection attempt"""
    success: bool
    message: str


class QuoteResponse(BaseModel):
    """Real-time quote data"""
    symbol: str
    price: float
    bid: float
    ask: float
    bid_size: int
    ask_size: int
    volume: int
    timestamp: str
    exchange: str


class Candle(BaseModel):
    """OHLCV candle data"""
    timestamp: str
    open: float
    high: float
    low: float
    close: float
    volume: int
    vwap: float = 0.0


class CandlesRequest(BaseModel):
    """Request for historical candles"""
    duration: str = Field(default="1 D", description="Duration like '1 D', '1 W', '1 M'")
    bar_size: str = Field(default="1 min", description="Bar size like '1 min', '5 mins', '1 hour'")
    what_to_show: str = Field(default="TRADES", description="Data type: TRADES, MIDPOINT, BID, ASK")


class CandlesResponse(BaseModel):
    """Response with historical candles"""
    symbol: str
    candles: List[Candle]
    count: int


class OrderRequest(BaseModel):
    """Request to place an order"""
    symbol: str
    action: str = Field(..., description="BUY or SELL")
    quantity: int = Field(..., gt=0)
    order_type: str = Field(default="MKT", description="MKT, LMT, or STP")
    limit_price: Optional[float] = None
    stop_price: Optional[float] = None


class OrderResponse(BaseModel):
    """Response from order placement"""
    success: bool
    order_id: int
    message: str


class OrderStatusResponse(BaseModel):
    """Response with order status details"""
    order_id: int
    status: str
    filled_qty: int
    avg_fill_price: float
    last_update: str


class Position(BaseModel):
    """Position information with real IB market pricing"""
    symbol: str
    contract_id: int
    quantity: int
    avg_cost: float       # average cost per share (cost basis)
    market_price: float   # real-time (or delayed) market price per share
    market_value: float   # quantity * market_price
    unrealized_pnl: float # market_value - (avg_cost * quantity)
    realized_pnl: float
    account: str


class PositionsResponse(BaseModel):
    """Response with all positions"""
    positions: List[Position]
    count: int


class AccountResponse(BaseModel):
    """Account information"""
    account_id: str
    net_liquidation: float
    total_cash: float
    buying_power: float
    equity_with_loan: float
    currency: str


class HealthResponse(BaseModel):
    """Health check response"""
    status: str
    connected: bool
    version: str
