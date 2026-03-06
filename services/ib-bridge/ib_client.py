"""
Interactive Brokers Client Wrapper
Handles connection and operations with IB Gateway using ib_insync
"""
import asyncio
import logging
import os
from datetime import datetime
from typing import AsyncGenerator, List, Optional

from ib_insync import IB, Stock, Order, MarketOrder, LimitOrder, StopOrder

from models import (
    BrokerOrder,
    QuoteResponse,
    Candle,
    Position,
    AccountResponse,
    OrderStatusResponse,
    CancelOrderResponse,
    ProtectPositionResponse,
    BracketOrderResponse,
)

logger = logging.getLogger(__name__)
OPEN_ORDER_STATUSES = {
    "ApiPending",
    "PendingSubmit",
    "PreSubmitted",
    "Submitted",
    "PendingCancel",
}


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
        self._market_data_mode = "unknown"
        self._seen_warning_keys: set[tuple[int, str]] = set()
        
        # Set up event handlers
        self.ib.disconnectedEvent += self._on_disconnected
        self.ib.errorEvent += self._on_error
    
    async def connect(self) -> None:
        """Connect to IB Gateway"""
        try:
            await self._connect_with_tolerant_sync(timeout=20)
            self._connected = True
            logger.info(f"Connected to IB Gateway at {self.host}:{self.port} (client_id={self.client_id})")
            
            # Request market data type (1=live, 2=frozen, 3=delayed, 4=delayed-frozen)
            mode_value, mode_label = self._resolve_market_data_type()
            self.ib.reqMarketDataType(mode_value)
            self._market_data_mode = mode_label
            self._seen_warning_keys.clear()
            # Wait for the Gateway to acknowledge the data-type switch before
            # any quote requests fire (reqMarketDataType is async on the IB side).
            await asyncio.sleep(1.5)
            
        except Exception as e:
            self._connected = False
            logger.error(f"Failed to connect to IB Gateway: {e}")
            raise

    async def _await_sync_request(self, name: str, awaitable, timeout: Optional[float]) -> None:
        """Run an initial IB sync request without dropping the whole session on timeout."""
        try:
            if timeout:
                await asyncio.wait_for(awaitable, timeout)
            else:
                await awaitable
        except asyncio.TimeoutError:
            logger.warning(
                "IB startup sync timed out for %s; continuing with partial readiness",
                name,
            )
        except Exception as exc:
            logger.warning(
                "IB startup sync failed for %s; continuing with partial readiness: %s",
                name,
                exc,
            )

    async def _connect_with_tolerant_sync(
        self,
        timeout: Optional[float] = 20,
        readonly: bool = False,
        account: str = "",
    ) -> None:
        """Connect and tolerate slow IB sync requests that should not invalidate the session."""
        client_id = int(self.client_id)
        self.ib.wrapper.clientId = client_id
        timeout = timeout or None

        await self.ib.client.connectAsync(self.host, self.port, client_id, timeout)

        if client_id == 0:
            self.ib.reqAutoOpenOrders(True)

        accounts = self.ib.client.getAccounts()
        if not account and len(accounts) == 1:
            account = accounts[0]

        requests = {"positions": self.ib.reqPositionsAsync()}
        if not readonly:
            requests["open orders"] = self.ib.reqOpenOrdersAsync()
        if not readonly and self.ib.client.serverVersion() >= 150:
            requests["completed orders"] = self.ib.reqCompletedOrdersAsync(False)
        if account:
            requests["account updates"] = self.ib.reqAccountUpdatesAsync(account)
        if len(accounts) <= self.ib.MaxSyncedSubAccounts:
            for subaccount in accounts:
                requests[f"account updates for {subaccount}"] = self.ib.reqAccountUpdatesMultiAsync(subaccount)

        await asyncio.gather(
            *(self._await_sync_request(name, request, timeout) for name, request in requests.items())
        )
        await self._await_sync_request("executions", self.ib.reqExecutionsAsync(), timeout)

        if not self.ib.client.isReady():
            raise ConnectionError("Socket connection broken while connecting")

        self.ib.connectedEvent.emit()
    
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

    def market_data_mode(self) -> str:
        """Configured IB market data mode for this bridge instance."""
        return self._market_data_mode

    def _resolve_market_data_type(self) -> tuple[int, str]:
        raw = os.getenv("IB_MARKET_DATA_TYPE", "delayed").strip().lower()
        mapping = {
            "live": (1, "live"),
            "frozen": (2, "frozen"),
            "delayed": (3, "delayed"),
            "delayed-frozen": (4, "delayed-frozen"),
            "delayed_frozen": (4, "delayed-frozen"),
        }
        return mapping.get(raw, (3, "delayed"))
    
    async def _reconnect(self) -> None:
        """Attempt to reconnect to IB Gateway"""
        logger.warning("IB bridge reconnect scheduled")
        retry_count = 0
        max_retries = 5
        
        while retry_count < max_retries:
            try:
                logger.warning(
                    "IB bridge reconnect attempt %s/%s",
                    retry_count + 1,
                    max_retries,
                )
                await asyncio.sleep(5 * (retry_count + 1))  # Exponential backoff
                await self.connect()
                logger.warning("IB bridge reconnect succeeded")
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
        warning_key = (errorCode, errorString)
        if errorCode == 10167:
            self._market_data_mode = "delayed"
            if warning_key not in self._seen_warning_keys:
                self._seen_warning_keys.add(warning_key)
                logger.info(
                    "IB market data subscription unavailable; bridge is using delayed data "
                    "(reqId=%s, code=%s)",
                    reqId,
                    errorCode,
                )
            return
        if errorCode in {2103, 2104, 2105, 2106, 2110, 2151, 2157, 2158}:
            if warning_key not in self._seen_warning_keys:
                self._seen_warning_keys.add(warning_key)
                logger.info(f"IB notice - ReqId: {reqId}, Code: {errorCode}, Msg: {errorString}")
            return
        logger.error(f"IB Error - ReqId: {reqId}, Code: {errorCode}, Msg: {errorString}")

    async def _qualify_stock_contract(self, symbol: str):
        contract = Stock(symbol.upper(), 'SMART', 'USD')
        try:
            contracts = await asyncio.wait_for(
                self.ib.qualifyContractsAsync(contract),
                timeout=5,
            )
        except asyncio.TimeoutError:
            logger.warning(
                "IB contract qualification timed out for %s; using SMART/USD fallback contract",
                symbol.upper(),
            )
            return contract
        if not contracts:
            raise ValueError(f"Could not find contract for symbol {symbol}")
        return contracts[0]

    def _build_order(
        self,
        action: str,
        quantity: int,
        order_type: str = "MKT",
        limit_price: Optional[float] = None,
        stop_price: Optional[float] = None,
    ) -> Order:
        normalized_action = action.upper().strip()
        normalized_type = order_type.upper().strip()

        if normalized_type == "MKT":
            return MarketOrder(normalized_action, quantity)
        if normalized_type == "LMT":
            if limit_price is None:
                raise ValueError("Limit price required for limit orders")
            return LimitOrder(normalized_action, quantity, limit_price)
        if normalized_type == "STP":
            if stop_price is None:
                raise ValueError("Stop price required for stop orders")
            return StopOrder(normalized_action, quantity, stop_price)
        raise ValueError(f"Unsupported order type: {order_type}")

    def _format_timestamp(self, value) -> str:
        if isinstance(value, datetime):
            return value.isoformat()
        if hasattr(value, "isoformat"):
            return value.isoformat()
        return datetime.utcnow().isoformat()

    def _optional_price(self, value) -> Optional[float]:
        if value is None:
            return None
        try:
            numeric = float(value)
        except (TypeError, ValueError):
            return None
        return numeric if numeric > 0 else None

    def _is_open_status(self, status: Optional[str]) -> bool:
        return (status or "").strip() in OPEN_ORDER_STATUSES

    def _opposite_action(self, action: str) -> str:
        return "SELL" if action.upper() == "BUY" else "BUY"

    def _trade_timestamps(self, trade) -> tuple[str, str]:
        log_entries = getattr(trade, "log", None) or []
        if log_entries:
            created_at = self._format_timestamp(getattr(log_entries[0], "time", None))
            updated_at = self._format_timestamp(getattr(log_entries[-1], "time", None))
            return created_at, updated_at
        now = datetime.utcnow().isoformat()
        return now, now

    def _find_trade(self, order_id: int):
        for trade in self.ib.trades():
            order = getattr(trade, "order", None)
            if order and getattr(order, "orderId", None) == order_id:
                return trade
        return None

    def _serialize_trade(self, trade) -> BrokerOrder:
        order = getattr(trade, "order", None)
        order_status = getattr(trade, "orderStatus", None)
        contract = getattr(trade, "contract", None)
        quantity = int(float(getattr(order, "totalQuantity", 0) or 0))
        filled_qty = int(float(getattr(order_status, "filled", 0) or 0))
        remaining_qty = int(float(getattr(order_status, "remaining", max(quantity - filled_qty, 0)) or 0))
        created_at, updated_at = self._trade_timestamps(trade)
        status = getattr(order_status, "status", "Unknown")
        order_type = getattr(order, "orderType", "MKT")

        return BrokerOrder(
            order_id=getattr(order, "orderId", 0),
            symbol=getattr(contract, "symbol", ""),
            action=getattr(order, "action", ""),
            order_type=order_type,
            quantity=quantity,
            limit_price=self._optional_price(getattr(order, "lmtPrice", None)),
            stop_price=self._optional_price(getattr(order, "auxPrice", None)),
            status=status,
            filled_qty=filled_qty,
            remaining_qty=remaining_qty,
            avg_fill_price=float(getattr(order_status, "avgFillPrice", 0.0) or 0.0),
            can_cancel=self._is_open_status(status),
            parent_id=getattr(order, "parentId", 0) or None,
            order_ref=getattr(order, "orderRef", None),
            created_at=created_at,
            updated_at=updated_at,
        )

    def _find_position(self, symbol: str):
        normalized_symbol = symbol.upper()
        for item in self.ib.portfolio():
            if getattr(item.contract, "symbol", "").upper() == normalized_symbol and int(item.position) != 0:
                return item
        raise ValueError(f"No open position found for {symbol}")

    async def _cancel_trade(self, trade) -> int:
        order = getattr(trade, "order", None)
        order_id = getattr(order, "orderId", 0)
        self.ib.cancelOrder(order)
        await asyncio.sleep(0.25)
        return order_id
    
    async def get_quote(self, symbol: str) -> QuoteResponse:
        """Get real-time quote for a symbol"""
        try:
            contract = await self._qualify_stock_contract(symbol)
            
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
                bid_size=int(ticker.bidSize) if ticker.bidSize and ticker.bidSize == ticker.bidSize else 0,
                ask_size=int(ticker.askSize) if ticker.askSize and ticker.askSize == ticker.askSize else 0,
                volume=int(ticker.volume) if ticker.volume and ticker.volume == ticker.volume else 0,
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
            contract = await self._qualify_stock_contract(symbol)

            intraday_request = "min" in bar_size.lower() or "hour" in bar_size.lower()
            timeout_seconds = 8 if intraday_request else 20

            # Historical intraday bars can hang indefinitely when the IB account
            # lacks market-data entitlements. Fail fast with an explicit error.
            try:
                bars = await asyncio.wait_for(
                    self.ib.reqHistoricalDataAsync(
                        contract,
                        endDateTime='',
                        durationStr=duration,
                        barSizeSetting=bar_size,
                        whatToShow=what_to_show,
                        useRTH=True,
                        formatDate=1
                    ),
                    timeout=timeout_seconds,
                )
            except asyncio.TimeoutError as exc:
                if intraday_request:
                    raise RuntimeError(
                        "IB intraday historical data timed out. "
                        "This usually means the account lacks Level 1 market-data "
                        "subscriptions required for API historical intraday bars."
                    ) from exc
                raise RuntimeError(
                    f"IB historical data timed out for {symbol} ({bar_size}, {duration})."
                ) from exc
            
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
            contract = await self._qualify_stock_contract(symbol)
            order = self._build_order(
                action=action,
                quantity=quantity,
                order_type=order_type,
                limit_price=limit_price,
                stop_price=stop_price,
            )
            order.orderRef = "manual-entry"
            trade = self.ib.placeOrder(contract, order)
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

    async def list_orders(self) -> List[BrokerOrder]:
        """List broker orders known to the current bridge session."""
        try:
            orders = [self._serialize_trade(trade) for trade in self.ib.trades()]
            orders.sort(key=lambda order: order.updated_at, reverse=True)
            return orders
        except Exception as e:
            logger.error(f"Error listing orders: {e}")
            raise

    async def cancel_order(self, order_id: int) -> CancelOrderResponse:
        """Cancel a working broker order."""
        try:
            trade = self._find_trade(order_id)
            if trade is None:
                raise ValueError(f"Order {order_id} not found")

            current_status = getattr(getattr(trade, "orderStatus", None), "status", "Unknown")
            if not self._is_open_status(current_status):
                raise ValueError(f"Order {order_id} is not cancellable from status {current_status}")

            await self._cancel_trade(trade)
            updated_status = getattr(getattr(trade, "orderStatus", None), "status", "Cancelled")

            return CancelOrderResponse(
                success=True,
                order_id=order_id,
                status=updated_status,
                message=f"Cancel requested for order {order_id}",
            )
        except Exception as e:
            logger.error(f"Error cancelling order {order_id}: {e}")
            raise

    async def close_position(
        self,
        symbol: str,
        quantity: Optional[int] = None,
        order_type: str = "MKT",
        limit_price: Optional[float] = None,
    ) -> int:
        """Close or reduce an open position."""
        try:
            position = self._find_position(symbol)
            absolute_quantity = abs(int(position.position))
            close_quantity = quantity or absolute_quantity
            if close_quantity <= 0 or close_quantity > absolute_quantity:
                raise ValueError(f"Close quantity must be between 1 and {absolute_quantity}")

            action = "SELL" if int(position.position) > 0 else "BUY"
            contract = await self._qualify_stock_contract(symbol)
            order = self._build_order(
                action=action,
                quantity=close_quantity,
                order_type=order_type,
                limit_price=limit_price,
            )
            order.orderRef = "manual-close"
            trade = self.ib.placeOrder(contract, order)
            await asyncio.sleep(0.5)
            return trade.order.orderId
        except Exception as e:
            logger.error(f"Error closing position for {symbol}: {e}")
            raise

    async def protect_position(
        self,
        symbol: str,
        stop_loss: float,
        take_profit: Optional[float] = None,
        quantity: Optional[int] = None,
        replace_existing: bool = True,
    ) -> ProtectPositionResponse:
        """Place or replace protective exit orders for an existing position."""
        try:
            position = self._find_position(symbol)
            absolute_quantity = abs(int(position.position))
            protect_quantity = quantity or absolute_quantity
            if protect_quantity <= 0 or protect_quantity > absolute_quantity:
                raise ValueError(f"Protection quantity must be between 1 and {absolute_quantity}")

            exit_action = "SELL" if int(position.position) > 0 else "BUY"
            contract = await self._qualify_stock_contract(symbol)
            cancelled_order_ids: List[int] = []

            if replace_existing:
                for trade in self.ib.trades():
                    trade_contract = getattr(trade, "contract", None)
                    order = getattr(trade, "order", None)
                    status = getattr(getattr(trade, "orderStatus", None), "status", None)
                    if getattr(trade_contract, "symbol", "").upper() != symbol.upper():
                        continue
                    if getattr(order, "action", "").upper() != exit_action:
                        continue
                    if getattr(order, "orderType", "").upper() not in {"LMT", "STP"}:
                        continue
                    if not self._is_open_status(status):
                        continue

                    order_ref = getattr(order, "orderRef", "") or ""
                    if order_ref in {"manual-protect", "manual-bracket-child"} or getattr(order, "parentId", 0):
                        cancelled_order_ids.append(await self._cancel_trade(trade))

            oca_group = f"protect-{symbol.upper()}-{int(datetime.utcnow().timestamp())}"
            order_ids: List[int] = []

            stop_order = self._build_order(
                action=exit_action,
                quantity=protect_quantity,
                order_type="STP",
                stop_price=stop_loss,
            )
            stop_order.orderRef = "manual-protect"
            stop_order.ocaGroup = oca_group
            stop_order.ocaType = 1
            stop_order.transmit = take_profit is None
            stop_trade = self.ib.placeOrder(contract, stop_order)
            order_ids.append(stop_trade.order.orderId)

            if take_profit is not None:
                take_profit_order = self._build_order(
                    action=exit_action,
                    quantity=protect_quantity,
                    order_type="LMT",
                    limit_price=take_profit,
                )
                take_profit_order.orderRef = "manual-protect"
                take_profit_order.ocaGroup = oca_group
                take_profit_order.ocaType = 1
                take_profit_order.transmit = True
                take_profit_trade = self.ib.placeOrder(contract, take_profit_order)
                order_ids.append(take_profit_trade.order.orderId)

            await asyncio.sleep(0.5)

            return ProtectPositionResponse(
                success=True,
                order_ids=order_ids,
                cancelled_order_ids=cancelled_order_ids,
                message=f"Submitted protection for {symbol.upper()}",
            )
        except Exception as e:
            logger.error(f"Error protecting position for {symbol}: {e}")
            raise

    async def place_bracket_order(
        self,
        symbol: str,
        action: str,
        quantity: int,
        entry_order_type: str = "MKT",
        entry_limit_price: Optional[float] = None,
        stop_loss: Optional[float] = None,
        take_profit: Optional[float] = None,
    ) -> BracketOrderResponse:
        """Place an entry order with attached stop-loss and optional take-profit."""
        try:
            if stop_loss is None and take_profit is None:
                raise ValueError("Bracket orders require at least a stop loss or take profit")

            contract = await self._qualify_stock_contract(symbol)
            parent_order = self._build_order(
                action=action,
                quantity=quantity,
                order_type=entry_order_type,
                limit_price=entry_limit_price,
            )
            parent_order.orderRef = "manual-entry"
            parent_order.transmit = False
            parent_trade = self.ib.placeOrder(contract, parent_order)
            await asyncio.sleep(0.2)

            parent_id = parent_trade.order.orderId
            child_order_ids: List[int] = []
            exit_action = self._opposite_action(action)
            child_specs = []
            if take_profit is not None:
                child_specs.append(("LMT", take_profit))
            if stop_loss is not None:
                child_specs.append(("STP", stop_loss))

            for index, (child_type, price) in enumerate(child_specs):
                child_order = self._build_order(
                    action=exit_action,
                    quantity=quantity,
                    order_type=child_type,
                    limit_price=price if child_type == "LMT" else None,
                    stop_price=price if child_type == "STP" else None,
                )
                child_order.parentId = parent_id
                child_order.orderRef = "manual-bracket-child"
                child_order.transmit = index == len(child_specs) - 1
                child_trade = self.ib.placeOrder(contract, child_order)
                child_order_ids.append(child_trade.order.orderId)

            await asyncio.sleep(0.5)

            return BracketOrderResponse(
                success=True,
                parent_order_id=parent_id,
                child_order_ids=child_order_ids,
                message=f"Bracket order submitted for {symbol.upper()}",
            )
        except Exception as e:
            logger.error(f"Error placing bracket order for {symbol}: {e}")
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
                    bid_size=int(ticker.bidSize) if ticker.bidSize and ticker.bidSize == ticker.bidSize else 0,
                    ask_size=int(ticker.askSize) if ticker.askSize and ticker.askSize == ticker.askSize else 0,
                    volume=int(ticker.volume) if ticker.volume and ticker.volume == ticker.volume else 0,
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
