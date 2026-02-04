"""
Configuration for IB Bridge service
Uses environment variables with sensible defaults
"""
import os
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings"""
    
    # Server settings
    HOST: str = "0.0.0.0"
    PORT: int = 8092
    DEBUG: bool = False
    LOG_LEVEL: str = "INFO"
    
    # IB Gateway settings
    IB_GATEWAY_HOST: str = "127.0.0.1"
    IB_GATEWAY_PORT: int = 7497  # 7497 for paper, 7496 for live
    IB_CLIENT_ID: int = 1
    
    # Connection settings
    AUTO_CONNECT: bool = True
    RECONNECT_ENABLED: bool = True
    RECONNECT_MAX_ATTEMPTS: int = 5
    RECONNECT_DELAY: int = 5  # seconds
    
    # Trading mode
    PAPER_TRADING: bool = True  # Safety: default to paper trading
    
    class Config:
        env_file = ".env"
        case_sensitive = True


settings = Settings()


# Validate settings
if not settings.PAPER_TRADING and settings.IB_GATEWAY_PORT == 7497:
    raise ValueError(
        "SAFETY CHECK: PAPER_TRADING=False but IB_GATEWAY_PORT=7497 (paper port). "
        "Set IB_GATEWAY_PORT=7496 for live trading."
    )

if settings.PAPER_TRADING and settings.IB_GATEWAY_PORT == 7496:
    raise ValueError(
        "SAFETY CHECK: PAPER_TRADING=True but IB_GATEWAY_PORT=7496 (live port). "
        "Set IB_GATEWAY_PORT=7497 for paper trading."
    )
