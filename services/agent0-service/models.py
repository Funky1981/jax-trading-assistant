"""
Pydantic models for Agent0 service API.
"""

from datetime import datetime
from enum import Enum
from typing import Optional, List, Dict, Any
from pydantic import BaseModel, Field


class Action(str, Enum):
    """Recommended trading action."""
    BUY = "BUY"
    SELL = "SELL"
    HOLD = "HOLD"
    WATCH = "WATCH"  # Interesting but wait for better entry


class SignalStrength(str, Enum):
    """Signal strength classification."""
    STRONG = "strong"
    MODERATE = "moderate"
    WEAK = "weak"


class TimeHorizon(str, Enum):
    """Suggested time horizon for the trade."""
    SCALP = "scalp"        # Minutes to hours
    SWING = "swing"        # Days to weeks
    POSITION = "position"  # Weeks to months


# ============ Request Models ============

class SuggestionRequest(BaseModel):
    """Request for trading suggestion."""
    symbol: str = Field(..., description="Stock symbol (e.g., AAPL)")
    context: Optional[str] = Field(None, description="Additional context for the AI")
    include_memory: bool = Field(True, description="Include relevant memories in analysis")
    include_market_data: bool = Field(True, description="Fetch current market data")
    
    class Config:
        json_schema_extra = {
            "example": {
                "symbol": "AAPL",
                "context": "Earnings report coming up next week",
                "include_memory": True,
                "include_market_data": True,
            }
        }


class AnalyzeRequest(BaseModel):
    """Request for detailed market analysis."""
    symbols: List[str] = Field(..., description="List of symbols to analyze")
    strategy: Optional[str] = Field(None, description="Specific strategy to apply")
    

class ChatRequest(BaseModel):
    """Request for conversational AI interaction."""
    message: str = Field(..., description="User message")
    conversation_id: Optional[str] = Field(None, description="ID for conversation continuity")


# ============ Response Models ============

class MarketContext(BaseModel):
    """Current market data context."""
    price: float
    change: float
    change_percent: float
    volume: Optional[int] = None
    high: Optional[float] = None
    low: Optional[float] = None
    timestamp: datetime = Field(default_factory=datetime.utcnow)


class MemoryContext(BaseModel):
    """Relevant memories retrieved for context."""
    memory_id: str
    content: str
    relevance_score: float
    source: str
    created_at: datetime


class RiskAssessment(BaseModel):
    """Risk assessment for the suggestion."""
    risk_level: str = Field(..., description="low, medium, high")
    stop_loss_pct: Optional[float] = Field(None, description="Suggested stop loss percentage")
    position_size_pct: Optional[float] = Field(None, description="Suggested position size as % of portfolio")
    max_loss_amount: Optional[float] = Field(None, description="Maximum loss if stop hit")


class SuggestionResponse(BaseModel):
    """AI trading suggestion response."""
    # Core suggestion
    symbol: str
    action: Action
    confidence: float = Field(..., ge=0, le=100, description="Confidence score 0-100")
    signal_strength: SignalStrength
    
    # Reasoning
    reasoning: str = Field(..., description="Detailed explanation of the suggestion")
    key_factors: List[str] = Field(default_factory=list, description="Key factors driving this suggestion")
    
    # Trade parameters
    time_horizon: TimeHorizon
    entry_price: Optional[float] = Field(None, description="Suggested entry price")
    target_price: Optional[float] = Field(None, description="Target exit price")
    stop_loss: Optional[float] = Field(None, description="Stop loss price")
    
    # Risk
    risk: RiskAssessment
    
    # Context used
    market_data: Optional[MarketContext] = None
    memories_used: List[MemoryContext] = Field(default_factory=list)
    
    # Metadata
    model_used: str
    provider: str
    generated_at: datetime = Field(default_factory=datetime.utcnow)
    request_id: str
    
    class Config:
        json_schema_extra = {
            "example": {
                "symbol": "AAPL",
                "action": "BUY",
                "confidence": 78,
                "signal_strength": "moderate",
                "reasoning": "AAPL showing strong momentum with earnings beat...",
                "key_factors": [
                    "Earnings beat by 12%",
                    "Positive analyst revisions",
                    "Strong volume on breakout"
                ],
                "time_horizon": "swing",
                "entry_price": 185.50,
                "target_price": 195.00,
                "stop_loss": 180.00,
                "risk": {
                    "risk_level": "medium",
                    "stop_loss_pct": 3.0,
                    "position_size_pct": 5.0,
                    "max_loss_amount": 500.0
                },
                "model_used": "llama3.2",
                "provider": "ollama",
                "generated_at": "2026-02-04T16:30:00Z",
                "request_id": "abc123"
            }
        }


class ChatResponse(BaseModel):
    """Conversational AI response."""
    message: str
    conversation_id: str
    suggestions: List[SuggestionResponse] = Field(default_factory=list)
    generated_at: datetime = Field(default_factory=datetime.utcnow)


class HealthResponse(BaseModel):
    """Health check response."""
    status: str
    service: str
    version: str
    llm_provider: str
    llm_model: str
    llm_cost: str
    memory_connected: bool
    ib_connected: bool
    uptime_seconds: float


class ErrorResponse(BaseModel):
    """Error response."""
    error: str
    detail: Optional[str] = None
    request_id: Optional[str] = None
