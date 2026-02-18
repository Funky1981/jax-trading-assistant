"""
Interactive Brokers Client Wrapper
Handles connection and operations with IB Gateway using ib_insync
"""
import asyncio
import logging
from datetime import datetime
from typing import List, Optional, AsyncGenerator
from decimal import Decimal

from ib_insync import IB, Stock, Order, MarketOrder, LimitOrder, StopOrder
from ib_insync import util

from models import (
    QuoteResponse,
    Candle,
    Position,
    AccountResponse,
    OrderStatusResponse
)

logger = logging.getLogger(__name__)


class IBClient:
    """Wrapper for IB connection and operations"""
    
    def __init__(self, host: str = "127.0.0.1", port: int = 7497, client_id: int = 1):
        self.host = host
        self.port = port
        self.client_id = client_id
        self.ib = IB()
        self._connected = False
        self._reconnect_task: Optional[asyncio.Task] = None
        self._quote_subscriptions = {}
        
        # Set up event handlers
        self.ib.disconnectedEvent += self._on_disconnected
        self.ib.errorEvent += self._on_error
    
    async def connect(self) -> None:
        """Connect to IB Gateway"""
        try:
            await self.ib.connectAsync(
                host=self.host,
                port=self.port,
                clientId=self.client_id,
                timeout=20
            )
            self._connected = True
            logger.info(f"Connected to IB Gateway at {self.host}:{self.port} (client_id={self.client_id})")
            
            # Request market data type (1=live, 2=frozen, 3=delayed, 4=delayed-frozen)
            self.ib.reqMarketDataType(3)  # Use delayed data by default for safety
            # Wait for the Gateway to acknowledge the data-type switch before
            # any quote requests fire (reqMarketDataType is async on the IB side).
            await asyncio.sleep(1.5)
            
        except Exception as e:
            self._connected = False
            logger.error(f"Failed to connect to IB Gateway: {e}")
            raise
    
    async def disconnect(self) -> None:
        """Disconnect from IB Gateway"""
        try:
            if self._reconnect_task:
                self._reconnect_task.cancel()
            
            if self.ib.isConnected():
                self.ib.disconnect()
            
            self._connected = False
            logger.info("Disconnected from IB Gateway")
        except Exception as e:
            logger.error(f"Error during disconnect: {e}")
    
    def is_connected(self) -> bool:
        """Check if connected to IB Gateway"""
        return self._connected and self.ib.isConnected()
    
    async def _reconnect(self) -> None:
        """Attempt to reconnect to IB Gateway"""
        logger.info("Attempting to reconnect to IB Gateway...")
        retry_count = 0
        max_retries = 5
        
        while retry_count < max_retries:
            try:
                await asyncio.sleep(5 * (retry_count + 1))  # Exponential backoff
                await self.connect()
                logger.info("Reconnected successfully")
                return
            except Exception as e:
                retry_count += 1
                logger.error(f"Reconnection attempt {retry_count}/{max_retries} failed: {e}")
        
        logger.error("Max reconnection attempts reached. Giving up.")
    
    def _on_disconnected(self):
        """Handle disconnection event"""
        self._connected = False
        logger.warning("Disconnected from IB Gateway")
        
        # Start reconnection task
        if not self._reconnect_task or self._reconnect_task.done():
            self._reconnect_task = asyncio.create_task(self._reconnect())
    
    def _on_error(self, reqId, errorCode, errorString, contract):
        """Handle error events"""
        logger.error(f"IB Error - ReqId: {reqId}, Code: {errorCode}, Msg: {errorString}")
    
    async def get_quote(self, symbol: str) -> QuoteResponse:
        """Get real-time quote for a symbol"""
        try:
            # Create contract
            contract = Stock(symbol, 'SMART', 'USD')
            
            # Qualify the contract
            contracts = await self.ib.qualifyContractsAsync(contract)
            if not contracts:
                raise ValueError(f"Could not find contract for symbol {symbol}")
            
            contract = contracts[0]
            
            # Request market data
            ticker = self.ib.reqMktData(contract, '', False, False)
            
            # Wait for data to populate
            # Delayed data (type 3) takes longer than live — use 2s
            await asyncio.sleep(2.0)
            
            # Extract quote data
            quote = QuoteResponse(
                symbol=symbol,
                price=float(ticker.last) if ticker.last and ticker.last == ticker.last else 0.0,  # Check for NaN
                bid=float(ticker.bid) if ticker.bid and ticker.bid == ticker.bid else 0.0,
                ask=float(ticker.ask) if ticker.ask and ticker.ask == ticker.ask else 0.0,
                bid_size=int(ticker.bidSize) if ticker.bidSize else 0,
                ask_size=int(ticker.askSize) if ticker.askSize else 0,
                volume=int(ticker.volume) if ticker.volume else 0,
                timestamp=datetime.utcnow().isoformat(),
                exchange=contract.exchange if hasattr(contract, 'exchange') else 'SMART'
            )
            
            # Cancel market data subscription
            self.ib.cancelMktData(contract)
            
            return quote
            
        except Exception as e:
            logger.error(f"Error getting quote for {symbol}: {e}")
            raise
    
    async def get_candles(
        self,
        symbol: str,
        duration: str = "1 D",
        bar_size: str = "1 min",
        what_to_show: str = "TRADES"
    ) -> List[Candle]:
        """Get historical candles for a symbol"""
        try:
            # Create contract
            contract = Stock(symbol, 'SMART', 'USD')
            
            # Qualify the contract
            contracts = await self.ib.qualifyContractsAsync(contract)
            if not contracts:
                raise ValueError(f"Could not find contract for symbol {symbol}")
            
            contract = contracts[0]
            
            # Request historical data
            bars = await self.ib.reqHistoricalDataAsync(
                contract,
                endDateTime='',
                durationStr=duration,
                barSizeSetting=bar_size,
                whatToShow=what_to_show,
                useRTH=True,
                formatDate=1
            )
            
            # Convert to Candle objects
            candles = []
            for bar in bars:
                candle = Candle(
                    timestamp=bar.date.isoformat() if hasattr(bar.date, 'isoformat') else str(bar.date),
                    open=float(bar.open),
                    high=float(bar.high),
                    low=float(bar.low),
                    close=float(bar.close),
                    volume=int(bar.volume),
                    vwap=float(bar.average) if bar.average > 0 else 0.0
                )
                candles.append(candle)
            
            return candles
            
        except Exception as e:
            logger.error(f"Error getting candles for {symbol}: {e}")
            raise
    
    async def place_order(
        self,
        symbol: str,
        action: str,
        quantity: int,
        order_type: str = "MKT",
        limit_price: Optional[float] = None,
        stop_price: Optional[float] = None
    ) -> int:
        """Place an order"""
        try:
            # Create contract
            contract = Stock(symbol, 'SMART', 'USD')
            
            # Qualify the contract
            contracts = await self.ib.qualifyContractsAsync(contract)
            if not contracts:
                raise ValueError(f"Could not find contract for symbol {symbol}")
            
            contract = contracts[0]
            
            # Create order based on type
            if order_type == "MKT":
                order = MarketOrder(action, quantity)
            elif order_type == "LMT":
                if limit_price is None:
                    raise ValueError("Limit price required for limit orders")
                order = LimitOrder(action, quantity, limit_price)
            elif order_type == "STP":
                if stop_price is None:
                    raise ValueError("Stop price required for stop orders")
                order = StopOrder(action, quantity, stop_price)
            else:
                raise ValueError(f"Unsupported order type: {order_type}")
            
            # Place the order
            trade = self.ib.placeOrder(contract, order)
            
            # Wait a moment for order to be submitted
            await asyncio.sleep(0.5)
            
            return trade.order.orderId
            
        except Exception as e:
            logger.error(f"Error placing order for {symbol}: {e}")
            raise

    async def get_order_status(self, order_id: int) -> OrderStatusResponse:
        """Get order status for a specific order ID"""
        try:
            trade = None
            for t in self.ib.trades():
                if t.order and t.order.orderId == order_id:
                    trade = t
                    break

            if trade is None:
                return OrderStatusResponse(
                    order_id=order_id,
                    status="Unknown",
                    filled_qty=0,
                    avg_fill_price=0.0,
                    last_update=datetime.utcnow().isoformat()
                )

            status = trade.orderStatus.status if trade.orderStatus else "Unknown"
            filled = int(trade.orderStatus.filled) if trade.orderStatus else 0
            avg_fill = float(trade.orderStatus.avgFillPrice) if trade.orderStatus else 0.0

            return OrderStatusResponse(
                order_id=order_id,
                status=status,
                filled_qty=filled,
                avg_fill_price=avg_fill,
                last_update=datetime.utcnow().isoformat()
            )
        except Exception as e:
            logger.error(f"Error getting order status for {order_id}: {e}")
            raise
    
    async def get_positions(self) -> List[Position]:
        """Get current positions with real market pricing via portfolioItems().

        ib-insync's portfolio() returns PortfolioItem objects which carry the
        live (or delayed) marketPrice, marketValue and unrealizedPNL that IB
        Gateway pushes automatically for every account position — no separate
        market data subscription is needed.
        """
        try:
            portfolio_items = self.ib.portfolio()

            result = []
            for item in portfolio_items:
                # Skip non-STK items (options, futures, etc.) for now
                sec_type = getattr(item.contract, 'secType', '')
                if sec_type and sec_type != 'STK':
                    logger.debug(f"Skipping non-STK position: {item.contract.symbol} ({sec_type})")
                    continue

                market_price = float(item.marketPrice) if item.marketPrice == item.marketPrice else 0.0  # NaN guard
                market_value = float(item.marketValue) if item.marketValue == item.marketValue else 0.0
                unrealized = float(item.unrealizedPNL) if item.unrealizedPNL == item.unrealizedPNL else 0.0
                realized   = float(item.realizedPNL)   if item.realizedPNL   == item.realizedPNL   else 0.0

                position = Position(
                    symbol=item.contract.symbol,
                    contract_id=int(item.contract.conId or 0),
                    quantity=int(item.position),
                    avg_cost=float(item.averageCost),
                    market_price=market_price,
                    market_value=market_value,
                    unrealized_pnl=unrealized,
                    realized_pnl=realized,
                    account=item.account
                )
                result.append(position)

            return result

        except Exception as e:
            logger.error(f"Error getting positions: {e}")
            raise
    
    async def get_account_info(self) -> AccountResponse:
        """Get account information"""
        try:
            account_values = self.ib.accountValues()
            
            # Extract key account values
            account_data = {}
            for av in account_values:
                account_data[av.tag] = av.value
            
            return AccountResponse(
                account_id=account_values[0].account if account_values else "Unknown",
                net_liquidation=float(account_data.get('NetLiquidation', 0)),
                total_cash=float(account_data.get('TotalCashValue', 0)),
                buying_power=float(account_data.get('BuyingPower', 0)),
                equity_with_loan=float(account_data.get('EquityWithLoanValue', 0)),
                currency=account_data.get('Currency', 'USD')
            )
            
        except Exception as e:
            logger.error(f"Error getting account info: {e}")
            raise
    
    async def subscribe_quotes(self, symbol: str) -> AsyncGenerator[QuoteResponse, None]:
        """Subscribe to real-time quote stream"""
        try:
            # Create contract
            contract = Stock(symbol, 'SMART', 'USD')
            
            # Qualify the contract
            contracts = await self.ib.qualifyContractsAsync(contract)
            if not contracts:
                raise ValueError(f"Could not find contract for symbol {symbol}")
            
            contract = contracts[0]
            
            # Request streaming market data
            ticker = self.ib.reqMktData(contract, '', False, False)
            self._quote_subscriptions[symbol] = (contract, ticker)
            
            # Stream updates
            while True:
                await asyncio.sleep(1)  # Update frequency
                
                quote = QuoteResponse(
                    symbol=symbol,
                    price=float(ticker.last) if ticker.last and ticker.last == ticker.last else 0.0,
                    bid=float(ticker.bid) if ticker.bid and ticker.bid == ticker.bid else 0.0,
                    ask=float(ticker.ask) if ticker.ask and ticker.ask == ticker.ask else 0.0,
                    bid_size=int(ticker.bidSize) if ticker.bidSize else 0,
                    ask_size=int(ticker.askSize) if ticker.askSize else 0,
                    volume=int(ticker.volume) if ticker.volume else 0,
                    timestamp=datetime.utcnow().isoformat(),
                    exchange=contract.exchange if hasattr(contract, 'exchange') else 'SMART'
                )
                
                yield quote
                
        except Exception as e:
            logger.error(f"Error in quote subscription for {symbol}: {e}")
            raise
    
    async def unsubscribe_quotes(self, symbol: str) -> None:
        """Unsubscribe from quote stream"""
        if symbol in self._quote_subscriptions:
            contract, _ = self._quote_subscriptions[symbol]
            self.ib.cancelMktData(contract)
            del self._quote_subscriptions[symbol]
            logger.info(f"Unsubscribed from quotes for {symbol}")
