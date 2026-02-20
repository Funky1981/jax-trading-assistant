"""
Agent0 Service Configuration

Supports multiple LLM providers:
- Ollama (FREE, local) - Default
- OpenAI (paid, cloud)
- Anthropic (paid, cloud)
"""

import os
from enum import Enum
from pydantic_settings import BaseSettings
from typing import Optional


class LLMProvider(str, Enum):
    OLLAMA = "ollama"
    OPENAI = "openai"
    ANTHROPIC = "anthropic"


class Settings(BaseSettings):
    """Configuration settings loaded from environment variables."""
    
    # Service settings
    service_name: str = "agent0-service"
    service_port: int = 8093
    debug: bool = False
    
    # LLM Provider (default: ollama for FREE local inference)
    llm_provider: LLMProvider = LLMProvider.OLLAMA
    
    # Ollama settings (FREE - runs locally)
    ollama_host: str = "http://localhost:11434"
    ollama_model: str = "llama3.2"  # or mistral, codellama, etc.
    
    # OpenAI settings (PAID - requires API key)
    openai_api_key: Optional[str] = None
    openai_model: str = "gpt-4o-mini"  # Cheapest good model
    
    # Anthropic settings (PAID - requires API key)
    anthropic_api_key: Optional[str] = None
    anthropic_model: str = "claude-3-haiku-20240307"  # Cheapest Claude
    
    # Memory service for context (ADR-0012 Phase 6: memory proxy in jax-research)
    memory_service_url: str = "http://jax-research:8091"
    
    # IB Bridge for market data
    ib_bridge_url: str = "http://ib-bridge:8092"
    
    # JAX Trader (replaces jax-api; ADR-0012 Phase 6)
    jax_api_url: str = "http://jax-trader:8081"
    
    # Request timeouts
    llm_timeout: int = 300  # LLM can be slow; 300s covers cold-load + generation on CPU
    api_timeout: int = 10
    
    # Caching
    cache_suggestions_seconds: int = 300  # Cache suggestions for 5 min
    
    class Config:
        env_prefix = "AGENT0_"
        env_file = ".env"


settings = Settings()


def get_llm_info() -> dict:
    """Return current LLM configuration info."""
    return {
        "provider": settings.llm_provider.value,
        "model": {
            LLMProvider.OLLAMA: settings.ollama_model,
            LLMProvider.OPENAI: settings.openai_model,
            LLMProvider.ANTHROPIC: settings.anthropic_model,
        }[settings.llm_provider],
        "cost": {
            LLMProvider.OLLAMA: "FREE (local)",
            LLMProvider.OPENAI: "~$0.01-0.03 per request",
            LLMProvider.ANTHROPIC: "~$0.01-0.02 per request",
        }[settings.llm_provider],
    }
