# Implementation Plan: Complete Agent0 Service Integration

**Generated:** February 4, 2026  
**Target:** 5 Phases, 8-10 days total  
**Status:** Ready to Execute

---

## Overview

This plan builds the complete AI-powered trading assistant by integrating Agent0 with the existing Jax Trading Assistant infrastructure. Each phase is independently testable, delivers user value, and takes 1-2 days maximum.

### Phase Summary

| Phase | Goal | Duration | Key Deliverable |
|-------|------|----------|-----------------|
| **Phase 1** | Agent0 HTTP Service | 2 days | AI suggestions via `/suggest` endpoint |
| **Phase 2** | Orchestrator HTTP API | 1 day | Frontend can trigger orchestration |
| **Phase 3** | Signal Generation & Storage | 2 days | Trading signals stored and displayed |
| **Phase 4** | Real Data Flow (IB → Agent0) | 2 days | Live market data drives AI suggestions |
| **Phase 5** | Memory & Reflection | 2 days | AI learns from outcomes |

**Prerequisites:**
- ✅ IB Gateway running (port 7497, paper trading)
- ✅ IB Bridge service operational
- ✅ PostgreSQL database running
- ✅ Hindsight + jax-memory services running
- ✅ Frontend UI built

---

## Phase 1: Agent0 HTTP Service (Core AI)

**Goal:** Create a FastAPI service that provides AI trading suggestions  
**Duration:** 2 days  
**Prerequisites:** OpenAI API key  

### Deliverables

1. **Agent0 HTTP Service** (`services/agent0-service/`)
   - FastAPI application with `/suggest` endpoint
   - Prompt engineering for trading suggestions
   - Integration with memory retrieval
   - Docker containerization

2. **API Endpoints:**
   - `POST /v1/suggest` - Get AI trading suggestion
   - `GET /health` - Health check
   - `GET /v1/models` - List available LLMs

3. **Docker Integration:**
   - Added to `docker-compose.yml`
   - Port: 8093

### Implementation Steps

#### Step 1.1: Create Agent0 Service Structure

Create the service directory and base files:

```powershell
# Create directory structure
New-Item -ItemType Directory -Force -Path "services\agent0-service"
New-Item -ItemType Directory -Force -Path "services\agent0-service\app"
New-Item -ItemType Directory -Force -Path "services\agent0-service\app\api"
New-Item -ItemType Directory -Force -Path "services\agent0-service\app\core"
New-Item -ItemType Directory -Force -Path "services\agent0-service\app\prompts"
```

**File 1: `services/agent0-service/requirements.txt`**

```txt
fastapi==0.109.0
uvicorn[standard]==0.27.0
pydantic==2.5.0
pydantic-settings==2.1.0
openai==1.12.0
httpx==0.26.0
python-dotenv==1.0.0
tenacity==8.2.3
```

**File 2: `services/agent0-service/app/core/config.py`**

```python
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    # Service
    HOST: str = "0.0.0.0"
    PORT: int = 8093
    LOG_LEVEL: str = "INFO"
    
    # OpenAI
    OPENAI_API_KEY: str
    OPENAI_MODEL: str = "gpt-4o-mini"
    OPENAI_TEMPERATURE: float = 0.7
    OPENAI_MAX_TOKENS: int = 2000
    
    # Memory Service
    MEMORY_SERVICE_URL: str = "http://jax-memory:8090"
    
    # IB Bridge
    IB_BRIDGE_URL: str = "http://ib-bridge:8092"
    
    class Config:
        env_file = ".env"
        case_sensitive = True

settings = Settings()
```

**File 3: `services/agent0-service/app/core/models.py`**

```python
from pydantic import BaseModel, Field
from typing import Optional, List, Dict, Any
from datetime import datetime

class SuggestRequest(BaseModel):
    """Request for AI trading suggestion"""
    symbol: str = Field(..., description="Stock symbol (e.g., AAPL)")
    context: Optional[Dict[str, Any]] = Field(default=None, description="Additional context (signals, events, etc.)")
    include_reasoning: bool = Field(default=True, description="Include AI reasoning in response")

class TradingSuggestion(BaseModel):
    """AI-generated trading suggestion"""
    action: str = Field(..., description="Recommended action: BUY, SELL, HOLD, or WATCH")
    symbol: str
    confidence: float = Field(..., ge=0.0, le=1.0, description="Confidence score 0-1")
    reasoning: Optional[str] = Field(None, description="AI reasoning/explanation")
    entry_price: Optional[float] = Field(None, description="Suggested entry price")
    stop_loss: Optional[float] = Field(None, description="Suggested stop loss")
    targets: Optional[List[float]] = Field(None, description="Price targets")
    risk_reward_ratio: Optional[float] = Field(None, description="Risk/reward ratio")
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    model_used: str = Field(default="gpt-4o-mini")

class SuggestResponse(BaseModel):
    """Response containing AI suggestion"""
    suggestion: TradingSuggestion
    memories_used: int = Field(default=0, description="Number of memories recalled")
    processing_time_ms: float = Field(default=0.0)

class HealthResponse(BaseModel):
    status: str
    version: str = "1.0.0"
    openai_configured: bool
    memory_service_available: bool
```

**File 4: `services/agent0-service/app/prompts/trading_planner.py`**

```python
"""
Prompt templates for trading AI
"""

SYSTEM_PROMPT = """You are Agent0, an expert AI trading assistant with deep knowledge of market analysis, risk management, and trading strategies.

Your role is to analyze market data, recall relevant historical patterns, and provide actionable trading suggestions with clear reasoning.

CRITICAL RULES:
1. Always prioritize risk management - suggest stop losses
2. Be conservative with confidence scores - only use >0.8 for strong setups
3. Explain your reasoning clearly and concisely
4. Consider both technical and fundamental factors
5. Account for current market conditions and volatility
6. If uncertain, recommend WATCH instead of forcing a trade

OUTPUT FORMAT:
- Action: BUY, SELL, HOLD, or WATCH
- Confidence: 0.0 to 1.0 (be realistic, not optimistic)
- Entry Price: Specific price level
- Stop Loss: Clear risk level
- Targets: 1-3 realistic price targets
- Risk/Reward: Calculated ratio
- Reasoning: Brief, clear explanation (2-3 sentences)
"""

def build_suggestion_prompt(
    symbol: str,
    current_price: float,
    memories: List[str],
    signals: List[Dict],
    market_data: Dict
) -> str:
    """Build the user prompt for trading suggestion"""
    
    prompt_parts = [
        f"# Trading Analysis Request for {symbol}",
        f"\n## Current Market Data",
        f"- Symbol: {symbol}",
        f"- Current Price: ${current_price:.2f}",
    ]
    
    if market_data:
        prompt_parts.append("\n### Market Context:")
        if "bid" in market_data:
            prompt_parts.append(f"- Bid: ${market_data['bid']:.2f}")
        if "ask" in market_data:
            prompt_parts.append(f"- Ask: ${market_data['ask']:.2f}")
        if "volume" in market_data:
            prompt_parts.append(f"- Volume: {market_data['volume']:,}")
    
    if signals:
        prompt_parts.append("\n## Active Signals:")
        for i, signal in enumerate(signals[:5], 1):
            signal_type = signal.get("type", "unknown")
            signal_strength = signal.get("strength", "N/A")
            prompt_parts.append(f"{i}. {signal_type} (strength: {signal_strength})")
    
    if memories:
        prompt_parts.append("\n## Relevant Historical Memories:")
        for i, memory in enumerate(memories[:3], 1):
            prompt_parts.append(f"{i}. {memory}")
    else:
        prompt_parts.append("\n## Historical Memories: None found (this is a new symbol or scenario)")
    
    prompt_parts.append("\n## Your Task:")
    prompt_parts.append(f"Analyze {symbol} and provide a trading suggestion with:")
    prompt_parts.append("1. Action (BUY/SELL/HOLD/WATCH)")
    prompt_parts.append("2. Confidence score (0.0-1.0)")
    prompt_parts.append("3. Entry price")
    prompt_parts.append("4. Stop loss")
    prompt_parts.append("5. Price targets (1-3)")
    prompt_parts.append("6. Risk/reward ratio")
    prompt_parts.append("7. Clear reasoning (2-3 sentences)")
    
    return "\n".join(prompt_parts)

def parse_llm_response(text: str) -> Dict:
    """Parse LLM response into structured data"""
    # This is a simple parser - you can make it more robust
    lines = [line.strip() for line in text.split('\n') if line.strip()]
    
    result = {
        "action": "WATCH",
        "confidence": 0.5,
        "entry_price": None,
        "stop_loss": None,
        "targets": [],
        "risk_reward_ratio": None,
        "reasoning": ""
    }
    
    reasoning_lines = []
    
    for line in lines:
        lower = line.lower()
        
        # Action
        if "action:" in lower or "recommendation:" in lower:
            for action in ["BUY", "SELL", "HOLD", "WATCH"]:
                if action in line.upper():
                    result["action"] = action
                    break
        
        # Confidence
        elif "confidence:" in lower:
            try:
                # Extract number between 0 and 1
                parts = line.split(":")
                if len(parts) > 1:
                    conf_str = parts[1].strip().rstrip('%')
                    conf = float(conf_str)
                    # Handle percentage vs decimal
                    result["confidence"] = conf if conf <= 1.0 else conf / 100.0
            except:
                pass
        
        # Entry price
        elif "entry" in lower and ("price" in lower or "$" in line):
            try:
                # Extract price
                import re
                prices = re.findall(r'\$?(\d+\.?\d*)', line)
                if prices:
                    result["entry_price"] = float(prices[0])
            except:
                pass
        
        # Stop loss
        elif "stop" in lower and ("loss" in lower or "$" in line):
            try:
                import re
                prices = re.findall(r'\$?(\d+\.?\d*)', line)
                if prices:
                    result["stop_loss"] = float(prices[0])
            except:
                pass
        
        # Targets
        elif "target" in lower and "$" in line:
            try:
                import re
                prices = re.findall(r'\$(\d+\.?\d*)', line)
                result["targets"].extend([float(p) for p in prices])
            except:
                pass
        
        # Risk/Reward
        elif "risk" in lower and "reward" in lower:
            try:
                import re
                ratios = re.findall(r'(\d+\.?\d*)', line)
                if ratios:
                    result["risk_reward_ratio"] = float(ratios[0])
            except:
                pass
        
        # Reasoning (collect all other lines)
        elif not any(kw in lower for kw in ["action", "confidence", "entry", "stop", "target", "risk"]):
            reasoning_lines.append(line)
    
    result["reasoning"] = " ".join(reasoning_lines).strip()
    
    return result
```

**File 5: `services/agent0-service/app/core/agent.py`**

```python
import time
import logging
from typing import Optional, List, Dict, Any
from openai import AsyncOpenAI
from tenacity import retry, stop_after_attempt, wait_exponential
import httpx

from .config import settings
from .models import SuggestRequest, TradingSuggestion, SuggestResponse
from ..prompts.trading_planner import (
    SYSTEM_PROMPT,
    build_suggestion_prompt,
    parse_llm_response
)

logger = logging.getLogger(__name__)

class Agent0:
    """Core AI agent for trading suggestions"""
    
    def __init__(self):
        self.client = AsyncOpenAI(api_key=settings.OPENAI_API_KEY)
        self.http_client = httpx.AsyncClient(timeout=10.0)
    
    async def close(self):
        """Cleanup resources"""
        await self.http_client.aclose()
    
    @retry(stop=stop_after_attempt(3), wait=wait_exponential(multiplier=1, min=2, max=10))
    async def recall_memories(self, symbol: str, limit: int = 5) -> List[str]:
        """Recall relevant memories from jax-memory service"""
        try:
            response = await self.http_client.post(
                f"{settings.MEMORY_SERVICE_URL}/tools",
                json={
                    "tool": "memory.recall",
                    "input": {
                        "query": f"trading decisions and outcomes for {symbol}",
                        "limit": limit,
                        "bank_id": "default"
                    }
                },
                timeout=5.0
            )
            
            if response.status_code == 200:
                data = response.json()
                memories = data.get("output", {}).get("memories", [])
                return [m.get("content", "") for m in memories if m.get("content")]
            
            logger.warning(f"Memory recall failed: {response.status_code}")
            return []
        
        except Exception as e:
            logger.error(f"Error recalling memories: {e}")
            return []
    
    async def get_current_quote(self, symbol: str) -> Optional[Dict]:
        """Get current quote from IB Bridge"""
        try:
            response = await self.http_client.get(
                f"{settings.IB_BRIDGE_URL}/quote/{symbol}",
                timeout=5.0
            )
            
            if response.status_code == 200:
                return response.json()
            
            logger.warning(f"Quote fetch failed: {response.status_code}")
            return None
        
        except Exception as e:
            logger.error(f"Error fetching quote: {e}")
            return None
    
    @retry(stop=stop_after_attempt(2), wait=wait_exponential(multiplier=1, min=1, max=5))
    async def call_llm(self, system_prompt: str, user_prompt: str) -> str:
        """Call OpenAI API with retry logic"""
        try:
            response = await self.client.chat.completions.create(
                model=settings.OPENAI_MODEL,
                messages=[
                    {"role": "system", "content": system_prompt},
                    {"role": "user", "content": user_prompt}
                ],
                temperature=settings.OPENAI_TEMPERATURE,
                max_tokens=settings.OPENAI_MAX_TOKENS
            )
            
            return response.choices[0].message.content or ""
        
        except Exception as e:
            logger.error(f"LLM call failed: {e}")
            raise
    
    async def suggest(self, request: SuggestRequest) -> SuggestResponse:
        """Generate trading suggestion for a symbol"""
        start_time = time.time()
        
        symbol = request.symbol.upper()
        logger.info(f"Generating suggestion for {symbol}")
        
        # Step 1: Get current market data
        quote = await self.get_current_quote(symbol)
        current_price = quote.get("price", 0.0) if quote else 0.0
        
        if current_price == 0.0:
            logger.warning(f"No price data for {symbol}, using placeholder")
            current_price = 100.0  # Fallback for testing
        
        # Step 2: Recall relevant memories
        memories = await self.recall_memories(symbol, limit=5)
        
        # Step 3: Extract context
        context = request.context or {}
        signals = context.get("signals", [])
        
        # Step 4: Build prompt
        user_prompt = build_suggestion_prompt(
            symbol=symbol,
            current_price=current_price,
            memories=memories,
            signals=signals,
            market_data=quote or {}
        )
        
        # Step 5: Call LLM
        llm_response = await self.call_llm(SYSTEM_PROMPT, user_prompt)
        
        logger.info(f"LLM response: {llm_response[:200]}...")
        
        # Step 6: Parse response
        parsed = parse_llm_response(llm_response)
        
        # Step 7: Build suggestion
        suggestion = TradingSuggestion(
            action=parsed["action"],
            symbol=symbol,
            confidence=parsed["confidence"],
            reasoning=llm_response if request.include_reasoning else parsed["reasoning"],
            entry_price=parsed["entry_price"] or current_price,
            stop_loss=parsed["stop_loss"],
            targets=parsed["targets"] or None,
            risk_reward_ratio=parsed["risk_reward_ratio"],
            model_used=settings.OPENAI_MODEL
        )
        
        processing_time = (time.time() - start_time) * 1000
        
        return SuggestResponse(
            suggestion=suggestion,
            memories_used=len(memories),
            processing_time_ms=processing_time
        )
```

**File 6: `services/agent0-service/app/api/routes.py`**

```python
from fastapi import APIRouter, HTTPException, Depends
import httpx
import logging

from ..core.models import (
    SuggestRequest,
    SuggestResponse,
    HealthResponse
)
from ..core.agent import Agent0
from ..core.config import settings

logger = logging.getLogger(__name__)
router = APIRouter()

# Global agent instance
_agent: Agent0 | None = None

def get_agent() -> Agent0:
    global _agent
    if _agent is None:
        _agent = Agent0()
    return _agent

@router.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    
    # Check if OpenAI key is configured
    openai_configured = bool(settings.OPENAI_API_KEY and settings.OPENAI_API_KEY != "your-api-key-here")
    
    # Check memory service
    memory_available = False
    try:
        async with httpx.AsyncClient(timeout=2.0) as client:
            response = await client.get(f"{settings.MEMORY_SERVICE_URL}/health")
            memory_available = response.status_code == 200
    except:
        pass
    
    status = "healthy" if (openai_configured and memory_available) else "degraded"
    
    return HealthResponse(
        status=status,
        openai_configured=openai_configured,
        memory_service_available=memory_available
    )

@router.post("/v1/suggest", response_model=SuggestResponse)
async def suggest_trade(
    request: SuggestRequest,
    agent: Agent0 = Depends(get_agent)
):
    """
    Generate AI trading suggestion for a symbol
    
    This endpoint analyzes market data, recalls relevant memories,
    and uses AI to generate actionable trading suggestions.
    """
    try:
        return await agent.suggest(request)
    except Exception as e:
        logger.error(f"Suggestion failed: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/v1/models")
async def list_models():
    """List available LLM models"""
    return {
        "current_model": settings.OPENAI_MODEL,
        "available_models": [
            "gpt-4o",
            "gpt-4o-mini",
            "gpt-4-turbo",
            "gpt-3.5-turbo"
        ]
    }
```

**File 7: `services/agent0-service/main.py`**

```python
import logging
import uvicorn
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api.routes import router
from app.core.config import settings

# Configure logging
logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifecycle manager"""
    logger.info("Starting Agent0 service...")
    logger.info(f"OpenAI Model: {settings.OPENAI_MODEL}")
    logger.info(f"Memory Service: {settings.MEMORY_SERVICE_URL}")
    
    yield
    
    logger.info("Shutting down Agent0 service...")

# Create FastAPI app
app = FastAPI(
    title="Agent0 Trading AI",
    description="AI-powered trading suggestion service using GPT-4",
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

# Include routes
app.include_router(router)

if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=True,
        log_level=settings.LOG_LEVEL.lower()
    )
```

**File 8: `services/agent0-service/Dockerfile`**

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Expose port
EXPOSE 8093

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8093/health')"

# Run the application
CMD ["python", "main.py"]
```

**File 9: `services/agent0-service/.env.example`**

```env
# Service Configuration
HOST=0.0.0.0
PORT=8093
LOG_LEVEL=INFO

# OpenAI Configuration
OPENAI_API_KEY=your-api-key-here
OPENAI_MODEL=gpt-4o-mini
OPENAI_TEMPERATURE=0.7
OPENAI_MAX_TOKENS=2000

# Service URLs
MEMORY_SERVICE_URL=http://jax-memory:8090
IB_BRIDGE_URL=http://ib-bridge:8092
```

#### Step 1.2: Update Docker Compose

Add Agent0 service to `docker-compose.yml`:

```yaml
  agent0-service:
    build:
      context: ./services/agent0-service
      dockerfile: Dockerfile
    environment:
      HOST: 0.0.0.0
      PORT: 8093
      LOG_LEVEL: ${AGENT0_LOG_LEVEL:-INFO}
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      OPENAI_MODEL: ${AGENT0_MODEL:-gpt-4o-mini}
      OPENAI_TEMPERATURE: ${AGENT0_TEMPERATURE:-0.7}
      OPENAI_MAX_TOKENS: ${AGENT0_MAX_TOKENS:-2000}
      MEMORY_SERVICE_URL: http://jax-memory:8090
      IB_BRIDGE_URL: http://ib-bridge:8092
    ports:
      - "8093:8093"
    depends_on:
      - jax-memory
      - ib-bridge
    healthcheck:
      test: ["CMD-SHELL", "python -c \"import requests; requests.get('http://localhost:8093/health')\""]
      interval: 30s
      timeout: 10s
      start_period: 40s
      retries: 3
```

#### Step 1.3: Update Root .env

Add to root `.env` file (create if doesn't exist):

```env
# Agent0 Configuration
OPENAI_API_KEY=sk-your-actual-api-key
AGENT0_MODEL=gpt-4o-mini
AGENT0_TEMPERATURE=0.7
AGENT0_MAX_TOKENS=2000
AGENT0_LOG_LEVEL=INFO
```

### Testing Phase 1

```powershell
# 1. Build and start the service
cd "c:\Projects\jax-trading assistant"
docker-compose build agent0-service
docker-compose up -d agent0-service

# 2. Check health
curl http://localhost:8093/health

# Expected output:
# {
#   "status": "healthy",
#   "version": "1.0.0",
#   "openai_configured": true,
#   "memory_service_available": true
# }

# 3. Test suggestion endpoint
curl -X POST http://localhost:8093/v1/suggest `
  -H "Content-Type: application/json" `
  -d '{
    "symbol": "AAPL",
    "context": {
      "signals": [
        {"type": "momentum_surge", "strength": 0.8}
      ]
    }
  }'

# Expected output:
# {
#   "suggestion": {
#     "action": "BUY" | "SELL" | "HOLD" | "WATCH",
#     "symbol": "AAPL",
#     "confidence": 0.75,
#     "reasoning": "...",
#     "entry_price": 178.50,
#     "stop_loss": 175.00,
#     "targets": [182.00, 185.00],
#     "risk_reward_ratio": 2.5,
#     "timestamp": "2026-02-04T...",
#     "model_used": "gpt-4o-mini"
#   },
#   "memories_used": 2,
#   "processing_time_ms": 1250.5
# }

# 4. Check logs
docker-compose logs -f agent0-service
```

### User Value After Phase 1

✅ **Users can:**
- Get AI trading suggestions for any symbol
- See confidence scores and reasoning
- View suggested entry/exit prices
- Understand the AI's thought process

✅ **Backend developers can:**
- Call Agent0 service from Go/Python services
- Integrate AI into workflows
- Test prompts and responses

---

## Phase 2: Orchestrator HTTP API

**Goal:** Expose orchestrator as HTTP service for frontend integration  
**Duration:** 1 day  
**Prerequisites:** Phase 1 complete (Agent0 service running)

### Deliverables

1. **HTTP wrapper around jax-orchestrator**
2. **API Endpoints:**
   - `POST /api/v1/orchestrate` - Trigger orchestration
   - `GET /api/v1/orchestrate/runs/{runId}` - Get run status
   - `GET /api/v1/orchestrate/runs` - List recent runs
3. **Run status tracking** in memory (Phase 5 will persist to DB)

### Implementation Steps

#### Step 2.1: Create Orchestrator HTTP Server

**File 1: `services/jax-orchestrator/internal/httpserver/server.go`**

```go
package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/services/jax-orchestrator/internal/app"
)

type OrchestrateRequest struct {
	Symbol  string                 `json:"symbol"`
	Context map[string]interface{} `json:"context,omitempty"`
}

type OrchestrateResponse struct {
	RunID     string    `json:"run_id"`
	Symbol    string    `json:"symbol"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

type RunStatus struct {
	RunID      string                 `json:"run_id"`
	Symbol     string                 `json:"symbol"`
	Status     string                 `json:"status"` // "running", "completed", "failed"
	StartedAt  time.Time              `json:"started_at"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	Result     *contracts.Decision    `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

type Server struct {
	orchestrator *app.Orchestrator
	runs         map[string]*RunStatus
	runsMutex    sync.RWMutex
	port         int
}

func NewServer(orchestrator *app.Orchestrator, port int) *Server {
	return &Server{
		orchestrator: orchestrator,
		runs:         make(map[string]*RunStatus),
		port:         port,
	}
}

func (s *Server) handleOrchestrate(w http.ResponseWriter, r *http.Request) {
	var req OrchestrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	// Generate run ID
	runID := fmt.Sprintf("run-%s-%d", req.Symbol, time.Now().UnixNano())

	// Create run status
	status := &RunStatus{
		RunID:     runID,
		Symbol:    req.Symbol,
		Status:    "running",
		StartedAt: time.Now(),
	}

	s.runsMutex.Lock()
	s.runs[runID] = status
	s.runsMutex.Unlock()

	// Run orchestration in background
	go s.runOrchestration(runID, req.Symbol)

	// Return immediate response
	response := OrchestrateResponse{
		RunID:     runID,
		Symbol:    req.Symbol,
		Status:    "running",
		StartedAt: status.StartedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) runOrchestration(runID, symbol string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.runsMutex.Lock()
	status := s.runs[runID]
	s.runsMutex.Unlock()

	// Run orchestration
	result, err := s.orchestrator.Orchestrate(ctx, symbol)

	// Update status
	s.runsMutex.Lock()
	defer s.runsMutex.Unlock()

	completedAt := time.Now()
	status.CompletedAt = &completedAt

	if err != nil {
		status.Status = "failed"
		status.Error = err.Error()
		log.Printf("Orchestration failed for %s: %v", symbol, err)
	} else {
		status.Status = "completed"
		status.Result = result
		log.Printf("Orchestration completed for %s: action=%s, confidence=%.2f",
			symbol, result.Action, result.Confidence)
	}
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["runId"]

	s.runsMutex.RLock()
	status, exists := s.runs[runID]
	s.runsMutex.RUnlock()

	if !exists {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	limit := 20 // Default limit

	s.runsMutex.RLock()
	defer s.runsMutex.RUnlock()

	// Convert map to slice and sort by start time
	runs := make([]*RunStatus, 0, len(s.runs))
	for _, status := range s.runs {
		runs = append(runs, status)
	}

	// Simple sort by start time (newest first)
	for i := 0; i < len(runs)-1; i++ {
		for j := i + 1; j < len(runs); j++ {
			if runs[j].StartedAt.After(runs[i].StartedAt) {
				runs[i], runs[j] = runs[j], runs[i]
			}
		}
	}

	// Apply limit
	if len(runs) > limit {
		runs = runs[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "jax-orchestrator",
		"version": "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) Start() error {
	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/api/v1/orchestrate", s.handleOrchestrate).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/orchestrate/runs/{runId}", s.handleGetRun).Methods("GET")
	router.HandleFunc("/api/v1/orchestrate/runs", s.handleListRuns).Methods("GET")
	router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// CORS middleware
	router.Use(corsMiddleware)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Orchestrator HTTP server starting on %s", addr)

	return http.ListenAndServe(addr, router)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
```

**File 2: Update `services/jax-orchestrator/cmd/jax-orchestrator/main.go`**

Add HTTP server mode:

```go
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/utcp"
	"jax-trading-assistant/services/jax-orchestrator/internal/app"
	"jax-trading-assistant/services/jax-orchestrator/internal/config"
	"jax-trading-assistant/services/jax-orchestrator/internal/httpserver"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	// HTTP server mode
	if cfg.HTTPPort > 0 {
		runHTTPServer(cfg)
		return
	}

	// CLI mode (existing code)
	runCLI(cfg)
}

func runHTTPServer(cfg *config.Config) {
	log.Printf("Starting orchestrator in HTTP server mode on port %d", cfg.HTTPPort)

	client, err := utcp.NewUTCPClientFromFile(cfg.ProvidersPath)
	if err != nil {
		log.Fatal(err)
	}

	memorySvc := utcp.NewMemoryService(client)
	memory := memoryAdapter{svc: memorySvc}
	agent := agent0Adapter{baseURL: cfg.Agent0URL}
	tools := utcpToolRunner{client: client}

	orch := app.NewOrchestrator(memory, agent, tools)

	server := httpserver.NewServer(orch, cfg.HTTPPort)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}

func runCLI(cfg *config.Config) {
	// Existing CLI code...
	if strings.TrimSpace(cfg.Symbol) == "" {
		log.Fatal("symbol is required (use -symbol)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ... rest of existing CLI code
}

// Add agent0 adapter
type agent0Adapter struct {
	baseURL string
}

func (a agent0Adapter) Plan(ctx context.Context, symbol string, signals []contracts.Signal, memories []string) (*contracts.Decision, error) {
	// TODO: Call Agent0 HTTP service
	// For now, return stub
	return &contracts.Decision{
		Action:     "WATCH",
		Symbol:     symbol,
		Confidence: 0.5,
		Reasoning:  "Agent0 integration pending",
	}, nil
}
```

**File 3: Update `services/jax-orchestrator/internal/config/config.go`**

Add HTTP port configuration:

```go
package config

import (
	"flag"
	"os"
)

type Config struct {
	Symbol        string
	ProvidersPath string
	HTTPPort      int
	Agent0URL     string
}

func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("jax-orchestrator", flag.ExitOnError)

	cfg := &Config{}

	fs.StringVar(&cfg.Symbol, "symbol", "", "Symbol to orchestrate (CLI mode only)")
	fs.StringVar(&cfg.ProvidersPath, "providers", getEnv("PROVIDERS_PATH", "config/providers.json"), "Path to providers.json")
	fs.IntVar(&cfg.HTTPPort, "http-port", getEnvInt("HTTP_PORT", 0), "HTTP server port (0 = CLI mode)")
	fs.StringVar(&cfg.Agent0URL, "agent0-url", getEnv("AGENT0_URL", "http://agent0-service:8093"), "Agent0 service URL")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		// Simple conversion
		var result int
		fmt.Sscanf(value, "%d", &result)
		return result
	}
	return defaultValue
}
```

#### Step 2.2: Update Docker Compose

Replace the jax-orchestrator profile service with HTTP server:

```yaml
  jax-orchestrator:
    build:
      context: .
      dockerfile: services/jax-orchestrator/Dockerfile
    environment:
      HTTP_PORT: 8094
      PROVIDERS_PATH: /workspace/config/providers.json
      AGENT0_URL: http://agent0-service:8093
    ports:
      - "8094:8094"
    volumes:
      - ./config:/workspace/config:ro
    depends_on:
      - jax-memory
      - agent0-service
    command: ["-http-port", "8094"]
```

**File 4: `services/jax-orchestrator/Dockerfile`**

```dockerfile
FROM golang:1.22 AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /jax-orchestrator \
    ./services/jax-orchestrator/cmd/jax-orchestrator

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /jax-orchestrator .

EXPOSE 8094

CMD ["./jax-orchestrator", "-http-port", "8094"]
```

#### Step 2.3: Install Required Go Packages

```powershell
cd "c:\Projects\jax-trading assistant"
go get github.com/gorilla/mux@latest
go mod tidy
```

### Testing Phase 2

```powershell
# 1. Build and start
docker-compose build jax-orchestrator
docker-compose up -d jax-orchestrator

# 2. Check health
curl http://localhost:8094/health

# 3. Trigger orchestration
curl -X POST http://localhost:8094/api/v1/orchestrate `
  -H "Content-Type: application/json" `
  -d '{"symbol": "TSLA"}'

# Expected output:
# {
#   "run_id": "run-TSLA-1738656000000",
#   "symbol": "TSLA",
#   "status": "running",
#   "started_at": "2026-02-04T..."
# }

# 4. Check run status (use run_id from above)
curl http://localhost:8094/api/v1/orchestrate/runs/run-TSLA-1738656000000

# Expected output:
# {
#   "run_id": "run-TSLA-1738656000000",
#   "symbol": "TSLA",
#   "status": "completed",
#   "started_at": "2026-02-04T...",
#   "completed_at": "2026-02-04T...",
#   "result": {
#     "action": "WATCH",
#     "symbol": "TSLA",
#     "confidence": 0.5,
#     "reasoning": "..."
#   }
# }

# 5. List recent runs
curl http://localhost:8094/api/v1/orchestrate/runs
```

### User Value After Phase 2

✅ **Frontend can:**
- Trigger orchestration for any symbol
- Poll for run status
- Display AI decisions in real-time

✅ **Users can:**
- See orchestration history
- Track decision making process

---

## Phase 3: Signal Generation & Storage

**Goal:** Store trading signals in database and expose via API  
**Duration:** 2 days  
**Prerequisites:** Phase 1-2 complete

### Deliverables

1. **Database schema** for signals
2. **Signal generation** logic (strategy-based)
3. **API endpoints** in jax-api
4. **Frontend integration** (optional UI update)

### Implementation Steps

#### Step 3.1: Database Migration

**File: `db/postgres/migrations/000004_signals_table.up.sql`**

```sql
-- Trading signals table
CREATE TABLE IF NOT EXISTS signals (
  id TEXT PRIMARY KEY,
  symbol TEXT NOT NULL,
  strategy_id TEXT NOT NULL,
  signal_type TEXT NOT NULL, -- 'BUY', 'SELL', 'WATCH'
  strength DOUBLE PRECISION NOT NULL CHECK (strength >= 0 AND strength <= 1),
  price DOUBLE PRECISION NOT NULL,
  indicators JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'active', -- 'active', 'expired', 'triggered'
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_signals_symbol ON signals(symbol);
CREATE INDEX IF NOT EXISTS idx_signals_strategy_id ON signals(strategy_id);
CREATE INDEX IF NOT EXISTS idx_signals_generated_at ON signals(generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_signals_status ON signals(status);
CREATE INDEX IF NOT EXISTS idx_signals_symbol_status ON signals(symbol, status, generated_at DESC);

-- Signal outcomes table (track what happened after signal)
CREATE TABLE IF NOT EXISTS signal_outcomes (
  id TEXT PRIMARY KEY,
  signal_id TEXT NOT NULL REFERENCES signals(id),
  outcome TEXT NOT NULL, -- 'success', 'failure', 'neutral', 'expired'
  pnl DOUBLE PRECISION,
  pnl_percent DOUBLE PRECISION,
  entry_price DOUBLE PRECISION,
  exit_price DOUBLE PRECISION,
  notes TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_signal_outcomes_signal_id ON signal_outcomes(signal_id);
CREATE INDEX IF NOT EXISTS idx_signal_outcomes_outcome ON signal_outcomes(outcome);
CREATE INDEX IF NOT EXISTS idx_signal_outcomes_recorded_at ON signal_outcomes(recorded_at DESC);
```

**File: `db/postgres/migrations/000004_signals_table.down.sql`**

```sql
DROP TABLE IF EXISTS signal_outcomes;
DROP TABLE IF EXISTS signals;
```

#### Step 3.2: Run Migration

```powershell
cd "c:\Projects\jax-trading assistant"
.\scripts\migrate.ps1 up
```

#### Step 3.3: Create Signal Generator Service

**File: `services/jax-signal-generator/main.py`**

```python
"""
Signal Generator Service
Analyzes market data and generates trading signals based on strategies
"""
import asyncio
import logging
import uuid
from datetime import datetime, timedelta
from typing import List, Dict, Any, Optional

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import httpx
import asyncpg
from pydantic import BaseModel, Field
from pydantic_settings import BaseSettings

# Configuration
class Settings(BaseSettings):
    HOST: str = "0.0.0.0"
    PORT: int = 8095
    LOG_LEVEL: str = "INFO"
    
    # Database
    DATABASE_URL: str = "postgresql://jax:jax@postgres:5432/jax"
    
    # Services
    IB_BRIDGE_URL: str = "http://ib-bridge:8092"
    
    # Signal parameters
    SIGNAL_EXPIRY_HOURS: int = 24
    MIN_SIGNAL_STRENGTH: float = 0.6
    
    class Config:
        env_file = ".env"

settings = Settings()

# Configure logging
logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Models
class GenerateSignalRequest(BaseModel):
    symbol: str
    strategy_id: str = Field(default="momentum_v1")

class Signal(BaseModel):
    id: str
    symbol: str
    strategy_id: str
    signal_type: str  # BUY, SELL, WATCH
    strength: float
    price: float
    indicators: Dict[str, Any]
    metadata: Dict[str, Any]
    generated_at: datetime
    expires_at: Optional[datetime]
    status: str = "active"

class SignalResponse(BaseModel):
    signal: Signal
    processing_time_ms: float

# Database pool
db_pool: Optional[asyncpg.Pool] = None

# HTTP client
http_client = httpx.AsyncClient(timeout=10.0)

async def get_db_pool():
    global db_pool
    if db_pool is None:
        db_pool = await asyncpg.create_pool(settings.DATABASE_URL)
    return db_pool

async def get_quote(symbol: str) -> Optional[Dict]:
    """Fetch current quote from IB Bridge"""
    try:
        response = await http_client.get(f"{settings.IB_BRIDGE_URL}/quote/{symbol}")
        if response.status_code == 200:
            return response.json()
        return None
    except Exception as e:
        logger.error(f"Error fetching quote for {symbol}: {e}")
        return None

async def calculate_momentum_signal(symbol: str, price: float) -> Dict[str, Any]:
    """
    Simple momentum signal calculator
    In production, this would use real technical indicators
    """
    # TODO: Replace with real technical analysis
    # For now, generate a mock signal based on simple rules
    
    import random
    
    # Simulate momentum calculation
    momentum_score = random.uniform(0.4, 0.9)
    
    if momentum_score >= 0.75:
        signal_type = "BUY"
    elif momentum_score <= 0.45:
        signal_type = "SELL"
    else:
        signal_type = "WATCH"
    
    return {
        "signal_type": signal_type,
        "strength": momentum_score,
        "indicators": {
            "momentum_score": momentum_score,
            "rsi": random.uniform(30, 70),
            "macd": random.uniform(-2, 2),
        },
        "metadata": {
            "calculation_method": "simple_momentum",
            "data_points_used": 14
        }
    }

async def generate_signal(symbol: str, strategy_id: str) -> Signal:
    """Generate a trading signal for a symbol"""
    
    # Get current quote
    quote = await get_quote(symbol)
    if not quote:
        raise ValueError(f"Could not fetch quote for {symbol}")
    
    price = quote.get("price", 0.0)
    
    # Calculate signal based on strategy
    if strategy_id == "momentum_v1":
        signal_data = await calculate_momentum_signal(symbol, price)
    else:
        raise ValueError(f"Unknown strategy: {strategy_id}")
    
    # Create signal
    signal_id = f"sig-{uuid.uuid4().hex[:12]}"
    generated_at = datetime.utcnow()
    expires_at = generated_at + timedelta(hours=settings.SIGNAL_EXPIRY_HOURS)
    
    signal = Signal(
        id=signal_id,
        symbol=symbol,
        strategy_id=strategy_id,
        signal_type=signal_data["signal_type"],
        strength=signal_data["strength"],
        price=price,
        indicators=signal_data["indicators"],
        metadata=signal_data["metadata"],
        generated_at=generated_at,
        expires_at=expires_at,
        status="active"
    )
    
    # Save to database
    pool = await get_db_pool()
    async with pool.acquire() as conn:
        await conn.execute(
            """
            INSERT INTO signals (
                id, symbol, strategy_id, signal_type, strength, price,
                indicators, metadata, generated_at, expires_at, status
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            """,
            signal.id,
            signal.symbol,
            signal.strategy_id,
            signal.signal_type,
            signal.strength,
            signal.price,
            signal.indicators,
            signal.metadata,
            signal.generated_at,
            signal.expires_at,
            signal.status
        )
    
    logger.info(f"Generated {signal.signal_type} signal for {symbol} (strength: {signal.strength:.2f})")
    
    return signal

# FastAPI app
app = FastAPI(
    title="Signal Generator Service",
    version="1.0.0"
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.post("/v1/signals/generate", response_model=SignalResponse)
async def generate_signal_endpoint(request: GenerateSignalRequest):
    """Generate a trading signal for a symbol"""
    import time
    start = time.time()
    
    signal = await generate_signal(request.symbol, request.strategy_id)
    
    processing_time = (time.time() - start) * 1000
    
    return SignalResponse(
        signal=signal,
        processing_time_ms=processing_time
    )

@app.get("/health")
async def health():
    return {"status": "healthy", "service": "signal-generator"}

@app.on_event("startup")
async def startup():
    logger.info("Signal Generator service starting...")
    await get_db_pool()
    logger.info("Database pool initialized")

@app.on_event("shutdown")
async def shutdown():
    logger.info("Signal Generator service shutting down...")
    if db_pool:
        await db_pool.close()
    await http_client.aclose()

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host=settings.HOST, port=settings.PORT)
```

**File: `services/jax-signal-generator/requirements.txt`**

```txt
fastapi==0.109.0
uvicorn[standard]==0.27.0
pydantic==2.5.0
pydantic-settings==2.1.0
asyncpg==0.29.0
httpx==0.26.0
python-dotenv==1.0.0
```

**File: `services/jax-signal-generator/Dockerfile`**

```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8095

CMD ["python", "main.py"]
```

#### Step 3.4: Add Signal Endpoints to jax-api

**File: `services/jax-api/internal/infra/http/signals.go`** (create new)

```go
package http

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Signal struct {
	ID          string                 `json:"id"`
	Symbol      string                 `json:"symbol"`
	StrategyID  string                 `json:"strategy_id"`
	SignalType  string                 `json:"signal_type"`
	Strength    float64                `json:"strength"`
	Price       float64                `json:"price"`
	Indicators  map[string]interface{} `json:"indicators"`
	Metadata    map[string]interface{} `json:"metadata"`
	GeneratedAt string                 `json:"generated_at"`
	ExpiresAt   *string                `json:"expires_at,omitempty"`
	Status      string                 `json:"status"`
}

func (s *Server) handleGetSignals(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	strategyID := vars["strategyId"]

	// Parse limit
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	rows, err := s.db.Query(`
		SELECT id, symbol, strategy_id, signal_type, strength, price,
		       indicators, metadata, generated_at, expires_at, status
		FROM signals
		WHERE strategy_id = $1 AND status = 'active'
		ORDER BY generated_at DESC
		LIMIT $2
	`, strategyID, limit)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	signals := []Signal{}
	for rows.Next() {
		var s Signal
		var indicatorsJSON, metadataJSON []byte

		err := rows.Scan(
			&s.ID, &s.Symbol, &s.StrategyID, &s.SignalType, &s.Strength, &s.Price,
			&indicatorsJSON, &metadataJSON, &s.GeneratedAt, &s.ExpiresAt, &s.Status,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(indicatorsJSON, &s.Indicators)
		json.Unmarshal(metadataJSON, &s.Metadata)

		signals = append(signals, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signals)
}

func (s *Server) RegisterSignalRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/strategies/{strategyId}/signals", s.handleGetSignals).Methods("GET")
}
```

Then add to `services/jax-api/internal/infra/http/server.go`:

```go
// In setupRoutes() method
s.RegisterSignalRoutes(router)
```

#### Step 3.5: Update Docker Compose

```yaml
  jax-signal-generator:
    build:
      context: ./services/jax-signal-generator
      dockerfile: Dockerfile
    environment:
      HOST: 0.0.0.0
      PORT: 8095
      LOG_LEVEL: INFO
      DATABASE_URL: ${DATABASE_URL:-postgresql://jax:jax@postgres:5432/jax}
      IB_BRIDGE_URL: http://ib-bridge:8092
      SIGNAL_EXPIRY_HOURS: 24
      MIN_SIGNAL_STRENGTH: 0.6
    ports:
      - "8095:8095"
    depends_on:
      - postgres
      - ib-bridge
```

### Testing Phase 3

```powershell
# 1. Build and start
docker-compose build jax-signal-generator jax-api
docker-compose up -d jax-signal-generator jax-api

# 2. Generate a signal
curl -X POST http://localhost:8095/v1/signals/generate `
  -H "Content-Type: application/json" `
  -d '{"symbol": "AAPL", "strategy_id": "momentum_v1"}'

# 3. Fetch signals from jax-api
curl "http://localhost:8081/api/v1/strategies/momentum_v1/signals?limit=10"

# 4. Verify in database
docker-compose exec postgres psql -U jax -d jax -c "SELECT * FROM signals LIMIT 5;"
```

### User Value After Phase 3

✅ **Users can:**
- See trading signals in the frontend
- View signal strength and indicators
- Track signal performance over time

✅ **System can:**
- Store and query signals
- Build signal history
- Prepare for outcome tracking

---

## Phase 4: Real Data Flow (IB → Dexter → Agent0)

**Goal:** Connect real market data from IB Gateway through the entire pipeline  
**Duration:** 2 days  
**Prerequisites:** Phase 1-3 complete

### Deliverables

1. **Market data ingestion** service (polls IB Bridge)
2. **Event detection** in Dexter (production mode)
3. **End-to-end flow:** IB → Events → Signals → Agent0 → Decision
4. **Automated orchestration** trigger

### Implementation Steps

#### Step 4.1: Market Data Ingestion Service

**File: `services/jax-market-ingest/main.py`**

```python
"""
Market Data Ingestion Service
Polls IB Bridge for market data and stores in PostgreSQL
"""
import asyncio
import logging
from datetime import datetime
from typing import List, Optional

import asyncpg
import httpx
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    LOG_LEVEL: str = "INFO"
    DATABASE_URL: str = "postgresql://jax:jax@postgres:5432/jax"
    IB_BRIDGE_URL: str = "http://ib-bridge:8092"
    
    # Watchlist (comma-separated symbols)
    WATCHLIST: str = "AAPL,TSLA,MSFT,GOOGL,AMZN"
    
    # Polling interval in seconds
    POLL_INTERVAL: int = 60
    
    class Config:
        env_file = ".env"

settings = Settings()

logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class MarketDataIngestor:
    def __init__(self):
        self.http_client = httpx.AsyncClient(timeout=10.0)
        self.db_pool: Optional[asyncpg.Pool] = None
    
    async def connect_db(self):
        """Connect to database"""
        self.db_pool = await asyncpg.create_pool(settings.DATABASE_URL)
        logger.info("Database connected")
    
    async def close(self):
        """Cleanup resources"""
        if self.db_pool:
            await self.db_pool.close()
        await self.http_client.aclose()
    
    async def fetch_quote(self, symbol: str) -> Optional[dict]:
        """Fetch quote from IB Bridge"""
        try:
            response = await self.http_client.get(
                f"{settings.IB_BRIDGE_URL}/quote/{symbol}",
                timeout=5.0
            )
            
            if response.status_code == 200:
                return response.json()
            
            logger.warning(f"Quote fetch failed for {symbol}: {response.status_code}")
            return None
        
        except Exception as e:
            logger.error(f"Error fetching quote for {symbol}: {e}")
            return None
    
    async def store_quote(self, symbol: str, quote: dict):
        """Store quote in database"""
        async with self.db_pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO quotes (symbol, price, bid, ask, bid_size, ask_size, volume, timestamp, exchange)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
                ON CONFLICT (symbol) DO UPDATE SET
                    price = EXCLUDED.price,
                    bid = EXCLUDED.bid,
                    ask = EXCLUDED.ask,
                    bid_size = EXCLUDED.bid_size,
                    ask_size = EXCLUDED.ask_size,
                    volume = EXCLUDED.volume,
                    timestamp = EXCLUDED.timestamp,
                    exchange = EXCLUDED.exchange,
                    updated_at = now()
                """,
                symbol,
                quote.get("price"),
                quote.get("bid"),
                quote.get("ask"),
                quote.get("bid_size"),
                quote.get("ask_size"),
                quote.get("volume"),
                datetime.fromisoformat(quote.get("timestamp")) if quote.get("timestamp") else datetime.utcnow(),
                quote.get("exchange")
            )
    
    async def poll_symbol(self, symbol: str):
        """Poll and store quote for a symbol"""
        quote = await self.fetch_quote(symbol)
        if quote:
            await self.store_quote(symbol, quote)
            logger.info(f"{symbol}: ${quote.get('price', 0):.2f} (vol: {quote.get('volume', 0):,})")
    
    async def poll_all_symbols(self):
        """Poll all watchlist symbols"""
        symbols = [s.strip() for s in settings.WATCHLIST.split(",") if s.strip()]
        
        logger.info(f"Polling {len(symbols)} symbols: {', '.join(symbols)}")
        
        tasks = [self.poll_symbol(symbol) for symbol in symbols]
        await asyncio.gather(*tasks, return_exceptions=True)
    
    async def run(self):
        """Main polling loop"""
        await self.connect_db()
        
        logger.info(f"Market data ingestion started (interval: {settings.POLL_INTERVAL}s)")
        
        try:
            while True:
                await self.poll_all_symbols()
                await asyncio.sleep(settings.POLL_INTERVAL)
        
        except KeyboardInterrupt:
            logger.info("Shutting down...")
        
        finally:
            await self.close()

async def main():
    ingestor = MarketDataIngestor()
    await ingestor.run()

if __name__ == "__main__":
    asyncio.run(main())
```

**File: `services/jax-market-ingest/requirements.txt`**

```txt
asyncpg==0.29.0
httpx==0.26.0
pydantic-settings==2.1.0
python-dotenv==1.0.0
```

**File: `services/jax-market-ingest/Dockerfile`**

```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["python", "main.py"]
```

#### Step 4.2: Event Detection Service

**File: `services/jax-event-detector/main.py`**

```python
"""
Event Detection Service
Analyzes market data to detect trading events (gaps, volume spikes, etc.)
"""
import asyncio
import logging
import uuid
from datetime import datetime, timedelta
from typing import List, Optional

import asyncpg
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    LOG_LEVEL: str = "INFO"
    DATABASE_URL: str = "postgresql://jax:jax@postgres:5432/jax"
    
    # Detection parameters
    VOLUME_SPIKE_THRESHOLD: float = 2.0  # 2x average volume
    PRICE_GAP_THRESHOLD: float = 0.02    # 2% gap
    
    # Polling interval in seconds
    POLL_INTERVAL: int = 300  # 5 minutes
    
    class Config:
        env_file = ".env"

settings = Settings()

logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class EventDetector:
    def __init__(self):
        self.db_pool: Optional[asyncpg.Pool] = None
    
    async def connect_db(self):
        """Connect to database"""
        self.db_pool = await asyncpg.create_pool(settings.DATABASE_URL)
        logger.info("Database connected")
    
    async def close(self):
        """Cleanup"""
        if self.db_pool:
            await self.db_pool.close()
    
    async def detect_volume_spike(self, symbol: str) -> Optional[dict]:
        """Detect volume spike for a symbol"""
        async with self.db_pool.acquire() as conn:
            # Get current volume
            current = await conn.fetchrow(
                "SELECT volume, price FROM quotes WHERE symbol = $1",
                symbol
            )
            
            if not current or not current["volume"]:
                return None
            
            # Get average volume from recent candles
            avg_volume = await conn.fetchval(
                """
                SELECT AVG(volume) FROM candles
                WHERE symbol = $1 AND timestamp > now() - interval '7 days'
                """,
                symbol
            )
            
            if not avg_volume or avg_volume == 0:
                return None
            
            ratio = current["volume"] / avg_volume
            
            if ratio >= settings.VOLUME_SPIKE_THRESHOLD:
                return {
                    "type": "volume_spike",
                    "symbol": symbol,
                    "current_volume": current["volume"],
                    "avg_volume": avg_volume,
                    "ratio": ratio,
                    "price": current["price"]
                }
        
        return None
    
    async def store_event(self, event: dict):
        """Store detected event in database"""
        event_id = f"evt-{uuid.uuid4().hex[:12]}"
        
        async with self.db_pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO events (id, symbol, type, time, payload)
                VALUES ($1, $2, $3, $4, $5)
                """,
                event_id,
                event["symbol"],
                event["type"],
                datetime.utcnow(),
                event
            )
        
        logger.info(f"Stored event: {event['type']} for {event['symbol']}")
    
    async def run(self):
        """Main detection loop"""
        await self.connect_db()
        
        logger.info(f"Event detection started (interval: {settings.POLL_INTERVAL}s)")
        
        try:
            while True:
                # Get all symbols from quotes table
                async with self.db_pool.acquire() as conn:
                    symbols = await conn.fetch("SELECT DISTINCT symbol FROM quotes")
                
                for row in symbols:
                    symbol = row["symbol"]
                    
                    # Detect volume spike
                    event = await self.detect_volume_spike(symbol)
                    if event:
                        await self.store_event(event)
                
                await asyncio.sleep(settings.POLL_INTERVAL)
        
        except KeyboardInterrupt:
            logger.info("Shutting down...")
        
        finally:
            await self.close()

async def main():
    detector = EventDetector()
    await detector.run()

if __name__ == "__main__":
    asyncio.run(main())
```

**File: `services/jax-event-detector/requirements.txt`** and **Dockerfile** (same as jax-market-ingest)

#### Step 4.3: Update Docker Compose

```yaml
  jax-market-ingest:
    build:
      context: ./services/jax-market-ingest
      dockerfile: Dockerfile
    environment:
      LOG_LEVEL: INFO
      DATABASE_URL: ${DATABASE_URL:-postgresql://jax:jax@postgres:5432/jax}
      IB_BRIDGE_URL: http://ib-bridge:8092
      WATCHLIST: ${MARKET_WATCHLIST:-AAPL,TSLA,MSFT,GOOGL,AMZN}
      POLL_INTERVAL: ${MARKET_POLL_INTERVAL:-60}
    depends_on:
      - postgres
      - ib-bridge
  
  jax-event-detector:
    build:
      context: ./services/jax-event-detector
      dockerfile: Dockerfile
    environment:
      LOG_LEVEL: INFO
      DATABASE_URL: ${DATABASE_URL:-postgresql://jax:jax@postgres:5432/jax}
      VOLUME_SPIKE_THRESHOLD: 2.0
      PRICE_GAP_THRESHOLD: 0.02
      POLL_INTERVAL: 300
    depends_on:
      - postgres
      - jax-market-ingest
```

### Testing Phase 4

```powershell
# 1. Start all services
docker-compose up -d jax-market-ingest jax-event-detector

# 2. Watch market data ingestion
docker-compose logs -f jax-market-ingest

# 3. Check quotes table
docker-compose exec postgres psql -U jax -d jax -c "SELECT symbol, price, volume, timestamp FROM quotes ORDER BY updated_at DESC LIMIT 10;"

# 4. Watch event detection
docker-compose logs -f jax-event-detector

# 5. Check events table
docker-compose exec postgres psql -U jax -d jax -c "SELECT id, symbol, type, time FROM events ORDER BY time DESC LIMIT 5;"

# 6. Trigger end-to-end flow manually
# Trigger orchestration when event is detected
curl -X POST http://localhost:8094/api/v1/orchestrate `
  -H "Content-Type: application/json" `
  -d '{"symbol": "AAPL"}'
```

### User Value After Phase 4

✅ **System now:**
- Continuously ingests real market data
- Detects trading events automatically
- Stores market data history

✅ **Users can:**
- See real-time market updates
- View detected events
- Trust AI decisions based on real data

---

## Phase 5: Memory & Reflection

**Goal:** AI learns from trading outcomes through reflection  
**Duration:** 2 days  
**Prerequisites:** Phase 1-4 complete

### Deliverables

1. **Outcome tracking** system
2. **Reflection job** (periodic belief generation)
3. **Memory integration** in suggestions
4. **Performance metrics**

### Implementation Steps

#### Step 5.1: Outcome Tracking Endpoint

**File: Add to `services/jax-api/internal/infra/http/outcomes.go`**

```go
package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type RecordOutcomeRequest struct {
	SignalID    string  `json:"signal_id"`
	Outcome     string  `json:"outcome"` // success, failure, neutral, expired
	PnL         float64 `json:"pnl"`
	PnLPercent  float64 `json:"pnl_percent"`
	EntryPrice  float64 `json:"entry_price"`
	ExitPrice   float64 `json:"exit_price"`
	Notes       string  `json:"notes,omitempty"`
}

func (s *Server) handleRecordOutcome(w http.ResponseWriter, r *http.Request) {
	var req RecordOutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	outcomeID := uuid.New().String()

	_, err := s.db.Exec(`
		INSERT INTO signal_outcomes (
			id, signal_id, outcome, pnl, pnl_percent,
			entry_price, exit_price, notes, recorded_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		outcomeID,
		req.SignalID,
		req.Outcome,
		req.PnL,
		req.PnLPercent,
		req.EntryPrice,
		req.ExitPrice,
		req.Notes,
		time.Now(),
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"outcome_id": outcomeID,
		"status":     "recorded",
	})
}

func (s *Server) RegisterOutcomeRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/outcomes", s.handleRecordOutcome).Methods("POST")
}
```

Add to server setup.

#### Step 5.2: Reflection Job

**File: `services/jax-reflection/main.py`**

```python
"""
Reflection Job
Periodically generates beliefs from trading outcomes
"""
import asyncio
import logging
from datetime import datetime, timedelta
from typing import List, Dict, Any

import asyncpg
import httpx
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    LOG_LEVEL: str = "INFO"
    DATABASE_URL: str = "postgresql://jax:jax@postgres:5432/jax"
    MEMORY_SERVICE_URL: str = "http://jax-memory:8090"
    
    # Reflection interval in seconds
    REFLECTION_INTERVAL: int = 3600  # 1 hour
    
    # Min outcomes needed for reflection
    MIN_OUTCOMES_FOR_REFLECTION: int = 5
    
    class Config:
        env_file = ".env"

settings = Settings()

logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class ReflectionJob:
    def __init__(self):
        self.db_pool: asyncpg.Pool | None = None
        self.http_client = httpx.AsyncClient(timeout=30.0)
    
    async def connect_db(self):
        self.db_pool = await asyncpg.create_pool(settings.DATABASE_URL)
        logger.info("Database connected")
    
    async def close(self):
        if self.db_pool:
            await self.db_pool.close()
        await self.http_client.aclose()
    
    async def get_recent_outcomes(self, hours: int = 24) -> List[Dict[str, Any]]:
        """Get recent trading outcomes"""
        async with self.db_pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT 
                    so.id, so.signal_id, so.outcome, so.pnl, so.pnl_percent,
                    so.entry_price, so.exit_price, so.notes, so.recorded_at,
                    s.symbol, s.strategy_id, s.signal_type, s.strength
                FROM signal_outcomes so
                JOIN signals s ON s.id = so.signal_id
                WHERE so.recorded_at > now() - interval '1 hour' * $1
                ORDER BY so.recorded_at DESC
                """,
                hours
            )
            
            return [dict(row) for row in rows]
    
    async def generate_belief(self, outcomes: List[Dict[str, Any]]) -> str:
        """Generate a belief statement from outcomes"""
        if not outcomes:
            return ""
        
        # Simple pattern detection
        success_count = sum(1 for o in outcomes if o["outcome"] == "success")
        total = len(outcomes)
        success_rate = success_count / total if total > 0 else 0.0
        
        # Calculate average PnL
        avg_pnl = sum(o["pnl_percent"] or 0 for o in outcomes) / total if total > 0 else 0.0
        
        # Group by symbol
        symbols = {}
        for o in outcomes:
            symbol = o["symbol"]
            if symbol not in symbols:
                symbols[symbol] = {"success": 0, "total": 0, "pnl": 0.0}
            
            symbols[symbol]["total"] += 1
            if o["outcome"] == "success":
                symbols[symbol]["success"] += 1
            symbols[symbol]["pnl"] += o["pnl_percent"] or 0.0
        
        # Generate belief narrative
        belief_parts = [
            f"Recent performance analysis ({total} outcomes in last 24 hours):",
            f"- Overall success rate: {success_rate:.1%}",
            f"- Average PnL: {avg_pnl:+.2f}%",
        ]
        
        # Best performing symbol
        best_symbol = None
        best_performance = -999
        for symbol, data in symbols.items():
            perf = data["pnl"] / data["total"] if data["total"] > 0 else 0
            if perf > best_performance:
                best_performance = perf
                best_symbol = symbol
        
        if best_symbol:
            belief_parts.append(f"- Best performer: {best_symbol} ({best_performance:+.2f}%)")
        
        return "\n".join(belief_parts)
    
    async def store_belief(self, belief: str):
        """Store belief in memory service"""
        try:
            response = await self.http_client.post(
                f"{settings.MEMORY_SERVICE_URL}/tools",
                json={
                    "tool": "memory.retain",
                    "input": {
                        "content": belief,
                        "bank_id": "beliefs",
                        "metadata": {
                            "type": "reflection",
                            "generated_at": datetime.utcnow().isoformat()
                        }
                    }
                }
            )
            
            if response.status_code == 200:
                logger.info(f"Belief stored: {belief[:100]}...")
            else:
                logger.error(f"Failed to store belief: {response.status_code}")
        
        except Exception as e:
            logger.error(f"Error storing belief: {e}")
    
    async def run_reflection(self):
        """Run one reflection cycle"""
        logger.info("Running reflection cycle...")
        
        # Get recent outcomes
        outcomes = await self.get_recent_outcomes(hours=24)
        
        if len(outcomes) < settings.MIN_OUTCOMES_FOR_REFLECTION:
            logger.info(f"Not enough outcomes ({len(outcomes)}) for reflection, skipping")
            return
        
        # Generate belief
        belief = await self.generate_belief(outcomes)
        
        if belief:
            await self.store_belief(belief)
            logger.info("Reflection completed")
    
    async def run(self):
        """Main reflection loop"""
        await self.connect_db()
        
        logger.info(f"Reflection job started (interval: {settings.REFLECTION_INTERVAL}s)")
        
        try:
            while True:
                await self.run_reflection()
                await asyncio.sleep(settings.REFLECTION_INTERVAL)
        
        except KeyboardInterrupt:
            logger.info("Shutting down...")
        
        finally:
            await self.close()

async def main():
    job = ReflectionJob()
    await job.run()

if __name__ == "__main__":
    asyncio.run(main())
```

**File: `services/jax-reflection/requirements.txt`** (same as others)

**File: `services/jax-reflection/Dockerfile`** (same pattern)

#### Step 5.3: Update Docker Compose

```yaml
  jax-reflection:
    build:
      context: ./services/jax-reflection
      dockerfile: Dockerfile
    environment:
      LOG_LEVEL: INFO
      DATABASE_URL: ${DATABASE_URL:-postgresql://jax:jax@postgres:5432/jax}
      MEMORY_SERVICE_URL: http://jax-memory:8090
      REFLECTION_INTERVAL: 3600
      MIN_OUTCOMES_FOR_REFLECTION: 5
    depends_on:
      - postgres
      - jax-memory
```

### Testing Phase 5

```powershell
# 1. Record some outcomes
curl -X POST http://localhost:8081/api/v1/outcomes `
  -H "Content-Type: application/json" `
  -d '{
    "signal_id": "sig-abc123",
    "outcome": "success",
    "pnl": 250.50,
    "pnl_percent": 2.5,
    "entry_price": 175.00,
    "exit_price": 179.38,
    "notes": "Strong momentum follow-through"
  }'

# 2. Start reflection service
docker-compose up -d jax-reflection

# 3. Watch reflection logs
docker-compose logs -f jax-reflection

# 4. Query beliefs from memory
curl -X POST http://localhost:8090/tools `
  -H "Content-Type: application/json" `
  -d '{
    "tool": "memory.recall",
    "input": {
      "query": "recent trading performance",
      "bank_id": "beliefs",
      "limit": 5
    }
  }'

# 5. Test full cycle: Generate signal → Get suggestion → Record outcome
# (This proves memory influences future suggestions)
```

### User Value After Phase 5

✅ **System now:**
- Learns from trading outcomes
- Generates beliefs from patterns
- Improves suggestions over time

✅ **Users can:**
- See AI learning progress
- Trust suggestions improve with experience
- View reflection-based insights

---

## Dependencies Summary

### Python Packages

Create `requirements-all.txt` for reference:

```txt
# Core FastAPI
fastapi==0.109.0
uvicorn[standard]==0.27.0
pydantic==2.5.0
pydantic-settings==2.1.0

# HTTP & Async
httpx==0.26.0
websockets==12.0

# Database
asyncpg==0.29.0

# AI/ML
openai==1.12.0

# IB Integration
ib-insync==0.9.86

# Utilities
python-dotenv==1.0.0
tenacity==8.2.3
```

### Go Packages

```bash
go get github.com/gorilla/mux@latest
go get github.com/google/uuid@latest
```

### Node/Bun Packages

Already in `dexter/package.json` and `frontend/package.json`

---

## Environment Variables

Complete `.env` file for root directory:

```env
# Database
DATABASE_URL=postgresql://jax:jax@postgres:5432/jax
POSTGRES_USER=jax
POSTGRES_PASSWORD=jax
POSTGRES_DB=jax

# Agent0 Service
OPENAI_API_KEY=sk-your-actual-openai-api-key
AGENT0_MODEL=gpt-4o-mini
AGENT0_TEMPERATURE=0.7
AGENT0_MAX_TOKENS=2000
AGENT0_LOG_LEVEL=INFO

# IB Gateway
IB_GATEWAY_HOST=host.docker.internal
IB_GATEWAY_PORT=7497
IB_CLIENT_ID=1
IB_AUTO_CONNECT=true
IB_PAPER_TRADING=true
IB_LOG_LEVEL=INFO

# Market Data
MARKET_WATCHLIST=AAPL,TSLA,MSFT,GOOGL,AMZN
MARKET_POLL_INTERVAL=60

# jax-api
API_PORT=8081
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_EXPIRY=24h
JWT_REFRESH_EXPIRY=168h
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173,http://127.0.0.1:3000,http://127.0.0.1:5173
RATE_LIMIT_REQUESTS_PER_MINUTE=100
RATE_LIMIT_REQUESTS_PER_HOUR=1000

# Service URLs (for inter-service communication)
IB_BRIDGE_URL=http://ib-bridge:8092
AGENT0_URL=http://agent0-service:8093
ORCHESTRATOR_URL=http://jax-orchestrator:8094
SIGNAL_GENERATOR_URL=http://jax-signal-generator:8095
```

---

## Docker Compose Complete Update

Add all services to `docker-compose.yml`:

```yaml
version: '3.8'

services:
  # Existing services (hindsight, jax-memory, ib-bridge, jax-api, postgres)
  # ... (keep existing)

  # NEW SERVICES

  agent0-service:
    build:
      context: ./services/agent0-service
      dockerfile: Dockerfile
    environment:
      HOST: 0.0.0.0
      PORT: 8093
      LOG_LEVEL: ${AGENT0_LOG_LEVEL:-INFO}
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      OPENAI_MODEL: ${AGENT0_MODEL:-gpt-4o-mini}
      OPENAI_TEMPERATURE: ${AGENT0_TEMPERATURE:-0.7}
      OPENAI_MAX_TOKENS: ${AGENT0_MAX_TOKENS:-2000}
      MEMORY_SERVICE_URL: http://jax-memory:8090
      IB_BRIDGE_URL: http://ib-bridge:8092
    ports:
      - "8093:8093"
    depends_on:
      - jax-memory
      - ib-bridge
    healthcheck:
      test: ["CMD-SHELL", "python -c \"import requests; requests.get('http://localhost:8093/health')\""]
      interval: 30s
      timeout: 10s
      start_period: 40s
      retries: 3

  jax-orchestrator:
    build:
      context: .
      dockerfile: services/jax-orchestrator/Dockerfile
    environment:
      HTTP_PORT: 8094
      PROVIDERS_PATH: /workspace/config/providers.json
      AGENT0_URL: http://agent0-service:8093
    ports:
      - "8094:8094"
    volumes:
      - ./config:/workspace/config:ro
    depends_on:
      - jax-memory
      - agent0-service
    command: ["-http-port", "8094"]

  jax-signal-generator:
    build:
      context: ./services/jax-signal-generator
      dockerfile: Dockerfile
    environment:
      HOST: 0.0.0.0
      PORT: 8095
      LOG_LEVEL: INFO
      DATABASE_URL: ${DATABASE_URL:-postgresql://jax:jax@postgres:5432/jax}
      IB_BRIDGE_URL: http://ib-bridge:8092
      SIGNAL_EXPIRY_HOURS: 24
      MIN_SIGNAL_STRENGTH: 0.6
    ports:
      - "8095:8095"
    depends_on:
      - postgres
      - ib-bridge

  jax-market-ingest:
    build:
      context: ./services/jax-market-ingest
      dockerfile: Dockerfile
    environment:
      LOG_LEVEL: INFO
      DATABASE_URL: ${DATABASE_URL:-postgresql://jax:jax@postgres:5432/jax}
      IB_BRIDGE_URL: http://ib-bridge:8092
      WATCHLIST: ${MARKET_WATCHLIST:-AAPL,TSLA,MSFT,GOOGL,AMZN}
      POLL_INTERVAL: ${MARKET_POLL_INTERVAL:-60}
    depends_on:
      - postgres
      - ib-bridge

  jax-event-detector:
    build:
      context: ./services/jax-event-detector
      dockerfile: Dockerfile
    environment:
      LOG_LEVEL: INFO
      DATABASE_URL: ${DATABASE_URL:-postgresql://jax:jax@postgres:5432/jax}
      VOLUME_SPIKE_THRESHOLD: 2.0
      PRICE_GAP_THRESHOLD: 0.02
      POLL_INTERVAL: 300
    depends_on:
      - postgres
      - jax-market-ingest

  jax-reflection:
    build:
      context: ./services/jax-reflection
      dockerfile: Dockerfile
    environment:
      LOG_LEVEL: INFO
      DATABASE_URL: ${DATABASE_URL:-postgresql://jax:jax@postgres:5432/jax}
      MEMORY_SERVICE_URL: http://jax-memory:8090
      REFLECTION_INTERVAL: 3600
      MIN_OUTCOMES_FOR_REFLECTION: 5
    depends_on:
      - postgres
      - jax-memory

volumes:
  jax-postgres:
```

---

## Complete Service Map

After all phases:

| Service | Port | Purpose | Technology |
|---------|------|---------|------------|
| **postgres** | 5432 | Database | PostgreSQL 16 |
| **hindsight** | 8888 | Memory storage | Python (vendored) |
| **jax-memory** | 8090 | Memory facade | Go |
| **jax-api** | 8081 | Main API | Go |
| **ib-bridge** | 8092 | IB Gateway integration | Python FastAPI |
| **agent0-service** | 8093 | AI suggestions | Python FastAPI + OpenAI |
| **jax-orchestrator** | 8094 | Orchestration HTTP API | Go |
| **jax-signal-generator** | 8095 | Signal generation | Python FastAPI |
| **jax-market-ingest** | - | Market data polling | Python (daemon) |
| **jax-event-detector** | - | Event detection | Python (daemon) |
| **jax-reflection** | - | Belief generation | Python (daemon) |
| **frontend** | 5173 | UI | React + Vite |

---

## Testing the Complete System

### End-to-End Flow Test

```powershell
# 1. Start all services
docker-compose up -d

# 2. Wait for services to be healthy (2-3 minutes)
Start-Sleep -Seconds 120

# 3. Check all health endpoints
$services = @(
    "http://localhost:8092/health",  # IB Bridge
    "http://localhost:8093/health",  # Agent0
    "http://localhost:8094/health",  # Orchestrator
    "http://localhost:8095/health",  # Signal Generator
    "http://localhost:8081/health"   # jax-api
)

foreach ($url in $services) {
    Write-Host "Checking $url..."
    curl $url
}

# 4. Generate a signal
curl -X POST http://localhost:8095/v1/signals/generate `
  -H "Content-Type: application/json" `
  -d '{"symbol": "AAPL", "strategy_id": "momentum_v1"}'

# 5. Get AI suggestion
curl -X POST http://localhost:8093/v1/suggest `
  -H "Content-Type: application/json" `
  -d '{"symbol": "AAPL", "include_reasoning": true}'

# 6. Trigger orchestration
curl -X POST http://localhost:8094/api/v1/orchestrate `
  -H "Content-Type: application/json" `
  -d '{"symbol": "AAPL"}'

# 7. Record outcome
curl -X POST http://localhost:8081/api/v1/outcomes `
  -H "Content-Type: application/json" `
  -d '{
    "signal_id": "sig-abc123",
    "outcome": "success",
    "pnl": 150.00,
    "pnl_percent": 1.5,
    "entry_price": 178.00,
    "exit_price": 180.67
  }'

# 8. View logs
docker-compose logs -f jax-market-ingest
docker-compose logs -f jax-event-detector
docker-compose logs -f jax-reflection
```

---

## Success Criteria

### Phase 1
- ✅ Agent0 service responds to `/suggest` requests
- ✅ AI provides BUY/SELL/HOLD/WATCH recommendations
- ✅ Confidence scores and reasoning returned
- ✅ Health endpoint shows OpenAI configured

### Phase 2
- ✅ Orchestrator HTTP API accepts requests
- ✅ Can trigger orchestration via POST
- ✅ Run status tracking works
- ✅ Frontend can consume orchestration API

### Phase 3
- ✅ Signals stored in database
- ✅ Signal endpoints return data
- ✅ Signal strength/indicators visible
- ✅ Database migrations successful

### Phase 4
- ✅ Market data flows from IB → database
- ✅ Events detected and stored
- ✅ Quote table updates every minute
- ✅ Event detector triggers on volume spikes

### Phase 5
- ✅ Outcomes recorded successfully
- ✅ Reflection job generates beliefs
- ✅ Beliefs stored in memory
- ✅ AI suggestions improve over time

---

## Troubleshooting

### Common Issues

**Agent0 service fails to start:**
```powershell
# Check OpenAI API key
docker-compose logs agent0-service | Select-String "OPENAI"

# Verify .env file
Get-Content .env | Select-String "OPENAI_API_KEY"
```

**Database connection errors:**
```powershell
# Check PostgreSQL health
docker-compose exec postgres pg_isready -U jax

# View connection logs
docker-compose logs postgres | Select-String "connection"
```

**IB Bridge disconnected:**
```powershell
# Verify IB Gateway is running on port 7497
# Check IB TWS/Gateway logs
docker-compose logs ib-bridge | Select-String "connect"
```

**Memory service unavailable:**
```powershell
# Restart memory services
docker-compose restart hindsight jax-memory

# Check health
curl http://localhost:8888/health
curl http://localhost:8090/health
```

---

## Next Steps After Completion

1. **Frontend Integration:**
   - Connect AI panel to Agent0 service
   - Display signals in Strategy Monitor
   - Show orchestration runs in UI

2. **Performance Optimization:**
   - Cache frequently accessed data
   - Optimize database queries
   - Add rate limiting

3. **Production Hardening:**
   - Add proper error handling
   - Implement circuit breakers
   - Set up monitoring/alerts

4. **Advanced Features:**
   - Multi-model ensemble (GPT-4 + Claude)
   - Real technical indicators (TA-Lib)
   - Backtesting integration
   - Paper trading execution

---

## Conclusion

This implementation plan provides a complete, step-by-step roadmap to integrate Agent0 with your Jax Trading Assistant. Each phase builds incrementally, is independently testable, and delivers user value.

**Total Timeline:** 8-10 days  
**Total New Services:** 7  
**Total Lines of Code:** ~3,500  
**Total API Endpoints:** 15+

You now have everything needed to build a production-ready AI trading assistant that learns from experience and makes intelligent trading decisions based on real market data.

**Start with Phase 1 and work sequentially. Each phase should take 1-2 days maximum.**

Good luck! 🚀
