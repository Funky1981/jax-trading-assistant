"""
Agent0 Core - Multi-Provider LLM Integration

Supports:
- Ollama (FREE, local) - Default
- OpenAI (paid)
- Anthropic (paid)
"""

import json
import uuid
import httpx
import logging
from datetime import datetime
from typing import Optional, List, Dict, Any

from config import settings, LLMProvider, get_llm_info
from models import (
    SuggestionRequest, SuggestionResponse, 
    Action, SignalStrength, TimeHorizon,
    MarketContext, MemoryContext, RiskAssessment,
    ChatRequest, ChatResponse
)
from prompts import SYSTEM_PROMPT, SUGGESTION_PROMPT, NO_DATA_RESPONSE

logger = logging.getLogger(__name__)


class Agent0:
    """Core Agent0 AI trading assistant."""
    
    def __init__(self):
        self.http_client = httpx.AsyncClient(timeout=settings.llm_timeout)
        self.start_time = datetime.utcnow()
        
    async def close(self):
        """Cleanup resources."""
        await self.http_client.aclose()
    
    # ==================== LLM Providers ====================
    
    async def _call_ollama(self, prompt: str, system: str = SYSTEM_PROMPT) -> str:
        """Call Ollama local LLM (FREE)."""
        try:
            response = await self.http_client.post(
                f"{settings.ollama_host}/api/generate",
                json={
                    "model": settings.ollama_model,
                    "prompt": prompt,
                    "system": system,
                    "stream": False,
                    "format": "json",
                },
                timeout=settings.llm_timeout,
            )
            response.raise_for_status()
            result = response.json()
            return result.get("response", "")
        except httpx.ConnectError:
            logger.error("Ollama not running. Install from https://ollama.ai and run: ollama serve")
            raise RuntimeError(
                "Ollama is not running. Please install Ollama from https://ollama.ai "
                "and run 'ollama serve' in a terminal. Then pull a model with 'ollama pull llama3.2'"
            )
        except Exception as e:
            logger.error(f"Ollama error: {e}")
            raise
    
    async def _call_openai(self, prompt: str, system: str = SYSTEM_PROMPT) -> str:
        """Call OpenAI API (PAID)."""
        if not settings.openai_api_key:
            raise ValueError("OpenAI API key not configured. Set AGENT0_OPENAI_API_KEY env var.")
        
        try:
            response = await self.http_client.post(
                "https://api.openai.com/v1/chat/completions",
                headers={"Authorization": f"Bearer {settings.openai_api_key}"},
                json={
                    "model": settings.openai_model,
                    "messages": [
                        {"role": "system", "content": system},
                        {"role": "user", "content": prompt},
                    ],
                    "response_format": {"type": "json_object"},
                    "temperature": 0.7,
                },
                timeout=settings.llm_timeout,
            )
            response.raise_for_status()
            result = response.json()
            return result["choices"][0]["message"]["content"]
        except Exception as e:
            logger.error(f"OpenAI error: {e}")
            raise
    
    async def _call_anthropic(self, prompt: str, system: str = SYSTEM_PROMPT) -> str:
        """Call Anthropic Claude API (PAID)."""
        if not settings.anthropic_api_key:
            raise ValueError("Anthropic API key not configured. Set AGENT0_ANTHROPIC_API_KEY env var.")
        
        try:
            response = await self.http_client.post(
                "https://api.anthropic.com/v1/messages",
                headers={
                    "x-api-key": settings.anthropic_api_key,
                    "anthropic-version": "2023-06-01",
                    "content-type": "application/json",
                },
                json={
                    "model": settings.anthropic_model,
                    "max_tokens": 2048,
                    "system": system,
                    "messages": [{"role": "user", "content": prompt}],
                },
                timeout=settings.llm_timeout,
            )
            response.raise_for_status()
            result = response.json()
            return result["content"][0]["text"]
        except Exception as e:
            logger.error(f"Anthropic error: {e}")
            raise
    
    async def _call_llm(self, prompt: str, system: str = SYSTEM_PROMPT) -> str:
        """Call the configured LLM provider."""
        provider = settings.llm_provider
        
        if provider == LLMProvider.OLLAMA:
            return await self._call_ollama(prompt, system)
        elif provider == LLMProvider.OPENAI:
            return await self._call_openai(prompt, system)
        elif provider == LLMProvider.ANTHROPIC:
            return await self._call_anthropic(prompt, system)
        else:
            raise ValueError(f"Unknown LLM provider: {provider}")
    
    # ==================== Data Fetching ====================
    
    async def _fetch_market_data(self, symbol: str) -> Optional[MarketContext]:
        """Fetch current market data from IB Bridge."""
        try:
            response = await self.http_client.get(
                f"{settings.ib_bridge_url}/quotes/{symbol}",
                timeout=settings.api_timeout,
            )
            if response.status_code == 200:
                data = response.json()
                return MarketContext(
                    price=data.get("last", data.get("close", 0)),
                    change=data.get("change", 0),
                    change_percent=data.get("change_percent", 0),
                    volume=data.get("volume"),
                    high=data.get("high"),
                    low=data.get("low"),
                )
        except Exception as e:
            logger.warning(f"Failed to fetch market data for {symbol}: {e}")
        return None
    
    async def _fetch_memories(self, symbol: str, limit: int = 5) -> List[MemoryContext]:
        """Fetch relevant memories from Hindsight."""
        memories = []
        try:
            response = await self.http_client.get(
                f"{settings.memory_service_url}/v1/memory/search",
                params={"query": symbol, "limit": limit},
                timeout=settings.api_timeout,
            )
            if response.status_code == 200:
                data = response.json()
                memories_data = data.get("memories", []) if isinstance(data, dict) else data
                for mem in memories_data or []:
                    memories.append(MemoryContext(
                        memory_id=mem.get("id", ""),
                        content=mem.get("content", ""),
                        relevance_score=mem.get("score", 0.5),
                        source=mem.get("source", "unknown"),
                        created_at=datetime.fromisoformat(mem.get("created_at", datetime.utcnow().isoformat())),
                    ))
        except Exception as e:
            logger.warning(f"Failed to fetch memories for {symbol}: {e}")
        return memories
    
    # ==================== Core Methods ====================
    
    async def get_suggestion(self, request: SuggestionRequest) -> SuggestionResponse:
        """Generate a trading suggestion for a symbol."""
        request_id = str(uuid.uuid4())[:8]
        symbol = request.symbol.upper()
        
        # Fetch context
        market_data = None
        memories = []
        
        if request.include_market_data:
            market_data = await self._fetch_market_data(symbol)
        
        if request.include_memory:
            memories = await self._fetch_memories(symbol)
        
        # Build prompt
        market_data_str = "No market data available"
        if market_data:
            def fmt_float(value, prefix="", suffix=""):
                if value is None:
                    return "N/A"
                return f"{prefix}{value:.2f}{suffix}"

            volume = market_data.volume if market_data.volume is not None else "N/A"
            market_data_str = f"""
Symbol: {symbol}
Current Price: {fmt_float(market_data.price, '$')}
Change: {fmt_float(market_data.change, '$')} ({fmt_float(market_data.change_percent, '', '%')})
Volume: {volume}
High: {fmt_float(market_data.high, '$')}
Low: {fmt_float(market_data.low, '$')}
"""
        
        memories_str = "No relevant memories found."
        if memories:
            memories_str = "\n".join([
                f"- [{m.source}] {m.content} (relevance: {m.relevance_score:.2f})"
                for m in memories
            ])
        
        context_str = request.context or "No additional context provided."
        
        prompt = SUGGESTION_PROMPT.format(
            symbol=symbol,
            market_data=market_data_str,
            memories=memories_str,
            context=context_str,
        )
        
        # Call LLM
        try:
            llm_response = await self._call_llm(prompt)
            suggestion_data = json.loads(llm_response)
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse LLM response: {llm_response}")
            # Return a safe default
            suggestion_data = {
                "action": "WATCH",
                "confidence": 30,
                "signal_strength": "weak",
                "reasoning": f"Unable to parse AI response. Raw output: {llm_response[:200]}",
                "key_factors": ["AI response parsing failed"],
                "time_horizon": "swing",
                "risk_level": "high",
                "stop_loss_pct": 5.0,
                "position_size_pct": 2.0,
            }
        except Exception as e:
            logger.error(f"LLM call failed: {e}")
            suggestion_data = {
                "action": "HOLD",
                "confidence": 10,
                "signal_strength": "weak",
                "reasoning": f"LLM unavailable: {e}. Returning safe HOLD.",
                "key_factors": ["LLM unavailable", "Fallback response"],
                "time_horizon": "swing",
                "risk_level": "high",
                "stop_loss_pct": 5.0,
                "position_size_pct": 1.0,
            }
        
        # Build response
        llm_info = get_llm_info()
        
        def safe_enum(enum_cls, value, default):
            try:
                return enum_cls(value)
            except Exception:
                return enum_cls(default)

        confidence = suggestion_data.get("confidence", 50)
        if isinstance(confidence, (int, float)):
            if confidence <= 1:
                confidence = confidence * 100
        else:
            confidence = 50

        return SuggestionResponse(
            symbol=symbol,
            action=safe_enum(Action, suggestion_data.get("action", "WATCH"), "WATCH"),
            confidence=confidence,
            signal_strength=safe_enum(SignalStrength, suggestion_data.get("signal_strength", "moderate"), "moderate"),
            reasoning=suggestion_data.get("reasoning", "No reasoning provided"),
            key_factors=suggestion_data.get("key_factors", []),
            time_horizon=safe_enum(TimeHorizon, suggestion_data.get("time_horizon", "swing"), "swing"),
            entry_price=suggestion_data.get("entry_price"),
            target_price=suggestion_data.get("target_price"),
            stop_loss=suggestion_data.get("stop_loss"),
            risk=RiskAssessment(
                risk_level=suggestion_data.get("risk_level", "medium"),
                stop_loss_pct=suggestion_data.get("stop_loss_pct", 5.0),
                position_size_pct=suggestion_data.get("position_size_pct", 5.0),
                max_loss_amount=None,  # Could calculate if we had portfolio size
            ),
            market_data=market_data,
            memories_used=memories,
            model_used=llm_info["model"],
            provider=llm_info["provider"],
            request_id=request_id,
        )
    
    async def check_health(self) -> Dict[str, Any]:
        """Check service health and connectivity."""
        # Check memory service
        memory_ok = False
        try:
            resp = await self.http_client.get(
                f"{settings.memory_service_url}/health",
                timeout=5,
            )
            memory_ok = resp.status_code == 200
        except:
            pass
        
        # Check IB Bridge
        ib_ok = False
        try:
            resp = await self.http_client.get(
                f"{settings.ib_bridge_url}/health",
                timeout=5,
            )
            ib_ok = resp.status_code == 200
        except:
            pass
        
        llm_info = get_llm_info()
        uptime = (datetime.utcnow() - self.start_time).total_seconds()
        
        return {
            "status": "healthy",
            "service": settings.service_name,
            "version": "1.0.0",
            "llm_provider": llm_info["provider"],
            "llm_model": llm_info["model"],
            "llm_cost": llm_info["cost"],
            "memory_connected": memory_ok,
            "ib_connected": ib_ok,
            "uptime_seconds": uptime,
        }


# Singleton instance
agent0 = Agent0()
