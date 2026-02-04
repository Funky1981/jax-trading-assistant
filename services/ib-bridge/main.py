"""
Interactive Brokers Bridge Service
FastAPI server that bridges Go backend to IB Gateway using ib_insync
"""
import asyncio
import logging
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import FastAPI, HTTPException, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
import uvicorn

from ib_client import IBClient
from config import settings
from models import (
    ConnectRequest,
    ConnectResponse,
    QuoteResponse,
    CandlesRequest,
    CandlesResponse,
    OrderRequest,
    OrderResponse,
    PositionsResponse,
    AccountResponse,
    HealthResponse,
)

# Configure logging
logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Global IB client instance
ib_client: Optional[IBClient] = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifecycle manager for the FastAPI app"""
    global ib_client
    
    # Startup
    logger.info("Starting IB Bridge service...")
    ib_client = IBClient(
        host=settings.IB_GATEWAY_HOST,
        port=settings.IB_GATEWAY_PORT,
        client_id=settings.IB_CLIENT_ID
    )
    
    # Auto-connect if configured
    if settings.AUTO_CONNECT:
        try:
            await ib_client.connect()
            logger.info("Auto-connected to IB Gateway")
        except Exception as e:
            logger.error(f"Auto-connect failed: {e}")
    
    yield
    
    # Shutdown
    logger.info("Shutting down IB Bridge service...")
    if ib_client:
        await ib_client.disconnect()


app = FastAPI(
    title="IB Bridge API",
    description="Python bridge service for Interactive Brokers integration",
    version="1.0.0",
    lifespan=lifespan
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    is_connected = ib_client.is_connected() if ib_client else False
    return HealthResponse(
        status="healthy" if is_connected else "degraded",
        connected=is_connected,
        version="1.0.0"
    )


@app.post("/connect", response_model=ConnectResponse)
async def connect(request: ConnectRequest):
    """Connect to IB Gateway"""
    try:
        if ib_client.is_connected():
            return ConnectResponse(
                success=True,
                message="Already connected to IB Gateway"
            )
        
        # Update connection settings if provided
        if request.host:
            ib_client.host = request.host
        if request.port:
            ib_client.port = request.port
        if request.client_id:
            ib_client.client_id = request.client_id
        
        await ib_client.connect()
        
        return ConnectResponse(
            success=True,
            message=f"Connected to IB Gateway at {ib_client.host}:{ib_client.port}"
        )
    except Exception as e:
        logger.error(f"Connection failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/disconnect")
async def disconnect():
    """Disconnect from IB Gateway"""
    try:
        if not ib_client.is_connected():
            return {"success": True, "message": "Already disconnected"}
        
        await ib_client.disconnect()
        return {"success": True, "message": "Disconnected from IB Gateway"}
    except Exception as e:
        logger.error(f"Disconnect failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/quotes/{symbol}", response_model=QuoteResponse)
async def get_quote(symbol: str):
    """Get real-time quote for a symbol"""
    try:
        if not ib_client.is_connected():
            raise HTTPException(status_code=503, detail="Not connected to IB Gateway")
        
        quote = await ib_client.get_quote(symbol)
        return quote
    except Exception as e:
        logger.error(f"Failed to get quote for {symbol}: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/candles/{symbol}", response_model=CandlesResponse)
async def get_candles(symbol: str, request: CandlesRequest):
    """Get historical candles for a symbol"""
    try:
        if not ib_client.is_connected():
            raise HTTPException(status_code=503, detail="Not connected to IB Gateway")
        
        candles = await ib_client.get_candles(
            symbol=symbol,
            duration=request.duration,
            bar_size=request.bar_size,
            what_to_show=request.what_to_show
        )
        
        return CandlesResponse(
            symbol=symbol,
            candles=candles,
            count=len(candles)
        )
    except Exception as e:
        logger.error(f"Failed to get candles for {symbol}: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/orders", response_model=OrderResponse)
async def place_order(request: OrderRequest):
    """Place an order"""
    try:
        if not ib_client.is_connected():
            raise HTTPException(status_code=503, detail="Not connected to IB Gateway")
        
        order_id = await ib_client.place_order(
            symbol=request.symbol,
            action=request.action,
            quantity=request.quantity,
            order_type=request.order_type,
            limit_price=request.limit_price,
            stop_price=request.stop_price
        )
        
        return OrderResponse(
            success=True,
            order_id=order_id,
            message=f"Order placed successfully"
        )
    except Exception as e:
        logger.error(f"Failed to place order: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/positions", response_model=PositionsResponse)
async def get_positions():
    """Get current positions"""
    try:
        if not ib_client.is_connected():
            raise HTTPException(status_code=503, detail="Not connected to IB Gateway")
        
        positions = await ib_client.get_positions()
        return PositionsResponse(
            positions=positions,
            count=len(positions)
        )
    except Exception as e:
        logger.error(f"Failed to get positions: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/account", response_model=AccountResponse)
async def get_account():
    """Get account information"""
    try:
        if not ib_client.is_connected():
            raise HTTPException(status_code=503, detail="Not connected to IB Gateway")
        
        account_info = await ib_client.get_account_info()
        return account_info
    except Exception as e:
        logger.error(f"Failed to get account info: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.websocket("/ws/quotes/{symbol}")
async def websocket_quotes(websocket: WebSocket, symbol: str):
    """WebSocket endpoint for streaming real-time quotes"""
    await websocket.accept()
    
    try:
        if not ib_client.is_connected():
            await websocket.send_json({
                "error": "Not connected to IB Gateway"
            })
            await websocket.close()
            return
        
        # Subscribe to quote stream
        quote_stream = ib_client.subscribe_quotes(symbol)
        
        async for quote in quote_stream:
            await websocket.send_json(quote.dict())
            
    except WebSocketDisconnect:
        logger.info(f"WebSocket disconnected for {symbol}")
    except Exception as e:
        logger.error(f"WebSocket error for {symbol}: {e}")
        await websocket.send_json({"error": str(e)})
    finally:
        await ib_client.unsubscribe_quotes(symbol)


if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        log_level=settings.LOG_LEVEL.lower()
    )
