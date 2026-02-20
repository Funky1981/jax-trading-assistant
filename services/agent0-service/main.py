"""
Agent0 Service - FastAPI Application

AI Trading Assistant with multi-provider LLM support.
Default: Ollama (FREE, local)
Optional: OpenAI, Anthropic (paid)
"""

import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from config import settings, get_llm_info
from models import (
    SuggestionRequest, SuggestionResponse,
    ChatRequest, ChatResponse,
    HealthResponse, ErrorResponse,
    PlanRequest, PlanResponse,
)
from agent import agent0

# Configure logging
logging.basicConfig(
    level=logging.DEBUG if settings.debug else logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan events."""
    logger.info(f"Starting {settings.service_name}...")
    llm_info = get_llm_info()
    logger.info(f"LLM Provider: {llm_info['provider']} ({llm_info['model']}) - {llm_info['cost']}")
    yield
    logger.info("Shutting down...")
    await agent0.close()


# Create FastAPI app
app = FastAPI(
    title="Agent0 - AI Trading Assistant",
    description="""
AI-powered trading suggestions with multi-provider LLM support.

## Features
- 🤖 Trading suggestions with confidence scores
- 📊 Market data integration
- 🧠 Memory-enhanced analysis
- 💰 FREE local inference with Ollama (default)

## LLM Providers
- **Ollama** (default): FREE, runs locally
- **OpenAI**: Paid, requires API key
- **Anthropic**: Paid, requires API key
""",
    version="1.0.0",
    lifespan=lifespan,
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# ==================== Error Handling ====================

@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    """Global exception handler."""
    logger.error(f"Unhandled exception: {exc}", exc_info=True)
    return JSONResponse(
        status_code=500,
        content={"error": "Internal server error", "detail": str(exc)},
    )


# ==================== Health Endpoints ====================

@app.get("/health", response_model=HealthResponse, tags=["Health"])
async def health_check():
    """
    Check service health and LLM configuration.
    
    Returns current LLM provider, model, and connectivity status.
    """
    return await agent0.check_health()


@app.get("/", tags=["Health"])
async def root():
    """Service info and quick start guide."""
    llm_info = get_llm_info()
    return {
        "service": "Agent0 - AI Trading Assistant",
        "version": "1.0.0",
        "llm": llm_info,
        "endpoints": {
            "suggest": "POST /suggest - Get trading suggestion",
            "chat": "POST /chat - Conversational AI",
            "health": "GET /health - Service health",
        },
        "quick_start": {
            "1": "POST /suggest with {'symbol': 'AAPL'}",
            "2": "Get back: action, confidence, reasoning, targets",
        },
    }


# ==================== Core AI Endpoints ====================

@app.post("/suggest", response_model=SuggestionResponse, tags=["AI"])
async def get_suggestion(request: SuggestionRequest):
    """
    Get an AI-powered trading suggestion for a symbol.
    
    ## Request
    - **symbol**: Stock symbol (e.g., AAPL)
    - **context**: Optional additional context
    - **include_memory**: Whether to include relevant memories
    - **include_market_data**: Whether to fetch current market data
    
    ## Response
    - **action**: BUY, SELL, HOLD, or WATCH
    - **confidence**: 0-100 confidence score
    - **reasoning**: Detailed explanation
    - **entry_price**, **target_price**, **stop_loss**: Trade levels
    - **risk**: Risk assessment with position sizing
    
    ## Example
    ```bash
    curl -X POST http://localhost:8093/suggest \\
         -H "Content-Type: application/json" \\
         -d '{"symbol": "AAPL"}'
    ```
    """
    try:
        return await agent0.get_suggestion(request)
    except RuntimeError as e:
        # Ollama not running
        raise HTTPException(status_code=503, detail=str(e))
    except Exception as e:
        logger.error(f"Suggestion failed: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"Failed to generate suggestion: {e}")


# ── /v1/plan ─────────────────────────────────────────────────────────────────
# This endpoint is called by the Go orchestration layer (libs/agent0/client.go).
# It adapts the Go PlanRequest format to the existing /suggest logic.

@app.post("/v1/plan", response_model=PlanResponse, tags=["AI"])
async def plan(request: PlanRequest):
    """
    Planning endpoint for the Go orchestration layer.
    Accepts PlanRequest and maps to /suggest internally.
    """
    symbol = request.symbol or ""
    # Attempt to extract symbol from task if not provided
    if not symbol and request.task:
        for word in request.task.split():
            if word.isupper() and 2 <= len(word) <= 5:
                symbol = word
                break

    suggest_req = SuggestionRequest(
        symbol=symbol or "UNKNOWN",
        context=f"{request.task}\n{request.context or ''}".strip(),
        include_memory=True,
        include_market_data=bool(symbol),
    )
    try:
        result = await agent0.get_suggestion(suggest_req)
        return PlanResponse(
            summary=result.reasoning[:300] if result.reasoning else "",
            steps=result.key_factors or [],
            action=result.action.value,
            confidence=result.confidence / 100.0,  # normalize 0-100 → 0-1
            reasoning_notes=result.reasoning or "",
        )
    except RuntimeError as e:
        raise HTTPException(status_code=503, detail=str(e))
    except Exception as e:
        logger.error(f"Plan failed: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"Failed to generate plan: {e}")


@app.post("/chat", response_model=ChatResponse, tags=["AI"])
async def chat(request: ChatRequest):
    """
    Have a conversational interaction with Agent0.
    
    Ask questions about markets, get suggestions, or discuss trading ideas.
    
    ## Example
    ```bash
    curl -X POST http://localhost:8093/chat \\
         -H "Content-Type: application/json" \\
         -d '{"message": "What do you think about NVDA right now?"}'
    ```
    """
    # For now, redirect to suggestion if it looks like a symbol request
    # Full chat implementation would need conversation history
    raise HTTPException(
        status_code=501, 
        detail="Chat endpoint coming soon. Use /suggest for now."
    )


@app.get("/config", tags=["Config"])
async def get_config():
    """
    Get current LLM configuration.
    
    Shows which provider is active and estimated costs.
    """
    return get_llm_info()


# ==================== Main ====================

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=settings.service_port,
        reload=settings.debug,
    )
