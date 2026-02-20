"""
Data Providers for Agent0

Fetches:
  - Historical OHLCV bars        → IB Bridge /candles/{symbol}
  - Technical indicators (RSI, MACD, SMA, BB, Volume) → computed from bars
  - News headlines               → yfinance (free, no API key required)
"""

import logging
from dataclasses import dataclass, field
from typing import Optional, List
from datetime import datetime

import httpx
import pandas as pd

logger = logging.getLogger(__name__)


# ── Data classes ──────────────────────────────────────────────────────────────

@dataclass
class Bar:
    time: str
    open: float
    high: float
    low: float
    close: float
    volume: float


@dataclass
class Technicals:
    rsi_14: Optional[float] = None          # 0-100; >70 overbought, <30 oversold
    macd: Optional[float] = None            # MACD line (12,26)
    macd_signal: Optional[float] = None    # Signal line (9)
    macd_hist: Optional[float] = None      # Histogram (positive = bullish momentum)
    sma_20: Optional[float] = None
    sma_50: Optional[float] = None
    bb_upper: Optional[float] = None       # Bollinger Band upper (20,2)
    bb_lower: Optional[float] = None       # Bollinger Band lower (20,2)
    bb_pct: Optional[float] = None         # % position within bands (0=lower, 1=upper)
    volume_ratio: Optional[float] = None   # current vol / 20-day avg vol
    trend: Optional[str] = None            # "bullish" | "bearish" | "neutral"
    bars_used: int = 0


@dataclass
class NewsItem:
    title: str
    publisher: str
    published: str   # ISO timestamp string
    url: str
    summary: str = ""


# ── Historical OHLCV from IB Bridge ──────────────────────────────────────────

async def fetch_historical_bars(
    ib_bridge_url: str,
    symbol: str,
    limit: int = 60,
    timeframe: str = "1D",
) -> List[Bar]:
    """
    Fetch historical OHLCV bars from IB Bridge.
    Returns empty list if IB Bridge is not connected.
    """
    try:
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.get(
                f"{ib_bridge_url}/candles/{symbol}",
                params={"limit": limit, "timeframe": timeframe},
            )
            if resp.status_code != 200:
                logger.warning(f"IB Bridge /candles/{symbol} → {resp.status_code}")
                return []
            data = resp.json()
            candles = data.get("candles", [])
            bars = []
            for c in candles:
                bars.append(Bar(
                    time=c.get("time", ""),
                    open=float(c.get("open", 0)),
                    high=float(c.get("high", 0)),
                    low=float(c.get("low", 0)),
                    close=float(c.get("close", 0)),
                    volume=float(c.get("volume", 0)),
                ))
            logger.info(f"Fetched {len(bars)} bars for {symbol}")
            return bars
    except Exception as e:
        logger.warning(f"Failed to fetch historical bars for {symbol}: {e}")
        return []


# ── Technical Indicators ──────────────────────────────────────────────────────

def compute_technicals(bars: List[Bar]) -> Technicals:
    """
    Compute RSI-14, MACD(12,26,9), SMA-20/50, Bollinger Bands(20,2), volume ratio.
    Requires at least 26 bars for meaningful MACD; 50 for SMA-50.
    """
    tech = Technicals(bars_used=len(bars))
    if len(bars) < 14:
        return tech

    closes = pd.Series([b.close for b in bars], dtype=float)
    volumes = pd.Series([b.volume for b in bars], dtype=float)
    price = closes.iloc[-1]

    # RSI-14
    if len(closes) >= 14:
        delta = closes.diff()
        gain = delta.where(delta > 0, 0.0).rolling(14).mean()
        loss = (-delta.where(delta < 0, 0.0)).rolling(14).mean()
        rs = gain / loss.replace(0, float("nan"))
        rsi = (100 - (100 / (1 + rs))).iloc[-1]
        tech.rsi_14 = round(float(rsi), 2) if pd.notna(rsi) else None

    # MACD (12, 26, 9)
    if len(closes) >= 26:
        ema12 = closes.ewm(span=12, adjust=False).mean()
        ema26 = closes.ewm(span=26, adjust=False).mean()
        macd_line = ema12 - ema26
        signal_line = macd_line.ewm(span=9, adjust=False).mean()
        histogram = macd_line - signal_line
        tech.macd = round(float(macd_line.iloc[-1]), 4)
        tech.macd_signal = round(float(signal_line.iloc[-1]), 4)
        tech.macd_hist = round(float(histogram.iloc[-1]), 4)

    # SMA-20
    if len(closes) >= 20:
        tech.sma_20 = round(float(closes.rolling(20).mean().iloc[-1]), 2)

    # SMA-50
    if len(closes) >= 50:
        tech.sma_50 = round(float(closes.rolling(50).mean().iloc[-1]), 2)

    # Bollinger Bands (20, 2)
    if len(closes) >= 20:
        sma = closes.rolling(20).mean()
        std = closes.rolling(20).std()
        upper = (sma + 2 * std).iloc[-1]
        lower = (sma - 2 * std).iloc[-1]
        tech.bb_upper = round(float(upper), 2)
        tech.bb_lower = round(float(lower), 2)
        band_width = upper - lower
        if band_width > 0:
            tech.bb_pct = round(float((price - lower) / band_width), 3)

    # Volume ratio (current / 20-day avg)
    if len(volumes) >= 20 and volumes.rolling(20).mean().iloc[-1] > 0:
        avg_vol = volumes.rolling(20).mean().iloc[-1]
        tech.volume_ratio = round(float(volumes.iloc[-1] / avg_vol), 2)

    # Overall trend heuristic
    bullish_signals = 0
    bearish_signals = 0
    if tech.rsi_14 is not None:
        if tech.rsi_14 > 55:
            bullish_signals += 1
        elif tech.rsi_14 < 45:
            bearish_signals += 1
    if tech.macd_hist is not None:
        if tech.macd_hist > 0:
            bullish_signals += 1
        else:
            bearish_signals += 1
    if tech.sma_20 is not None and price > tech.sma_20:
        bullish_signals += 1
    elif tech.sma_20 is not None:
        bearish_signals += 1
    if tech.sma_50 is not None and price > tech.sma_50:
        bullish_signals += 1
    elif tech.sma_50 is not None:
        bearish_signals += 1

    if bullish_signals > bearish_signals + 1:
        tech.trend = "bullish"
    elif bearish_signals > bullish_signals + 1:
        tech.trend = "bearish"
    else:
        tech.trend = "neutral"

    return tech


def format_technicals(tech: Technicals, current_price: Optional[float] = None) -> str:
    """Format technicals as a clean string for the LLM prompt."""
    if tech.bars_used == 0:
        return "No historical data available (IB Gateway not connected)."

    lines = [f"Based on {tech.bars_used} daily bars:"]

    if tech.rsi_14 is not None:
        interp = "overbought" if tech.rsi_14 > 70 else "oversold" if tech.rsi_14 < 30 else "neutral"
        lines.append(f"- RSI(14): {tech.rsi_14} [{interp}]")

    if tech.macd is not None:
        direction = "bullish crossover" if tech.macd_hist and tech.macd_hist > 0 else "bearish crossover"
        lines.append(f"- MACD(12,26,9): {tech.macd} | Signal: {tech.macd_signal} | Hist: {tech.macd_hist} [{direction}]")

    if tech.sma_20 is not None:
        pos = "above" if (current_price or 0) > tech.sma_20 else "below"
        lines.append(f"- SMA-20: {tech.sma_20} [price {pos} SMA-20]")

    if tech.sma_50 is not None:
        pos = "above" if (current_price or 0) > tech.sma_50 else "below"
        lines.append(f"- SMA-50: {tech.sma_50} [price {pos} SMA-50]")

    if tech.bb_upper is not None:
        pct_str = f"{tech.bb_pct:.1%}" if tech.bb_pct is not None else "N/A"
        lines.append(f"- Bollinger Bands: Lower={tech.bb_lower} | Upper={tech.bb_upper} | Position={pct_str}")

    if tech.volume_ratio is not None:
        vol_interp = "high" if tech.volume_ratio > 1.5 else "low" if tech.volume_ratio < 0.5 else "average"
        lines.append(f"- Volume ratio (vs 20d avg): {tech.volume_ratio}x [{vol_interp} volume]")

    if tech.trend:
        lines.append(f"- Overall trend signal: {tech.trend.upper()}")

    return "\n".join(lines)


# ── News via yfinance ─────────────────────────────────────────────────────────

def fetch_news_sync(symbol: str, limit: int = 5) -> List[NewsItem]:
    """
    Fetch recent news headlines via yfinance (synchronous).
    Run in a thread executor to avoid blocking the async event loop.
    """
    try:
        import yfinance as yf  # imported here so startup is not blocked if unavailable
        ticker = yf.Ticker(symbol)
        raw_news = ticker.news or []
        items = []
        for article in raw_news[:limit]:
            # yfinance news structure varies; handle both old and new schemas
            content = article.get("content", {})
            title = (
                content.get("title")
                or article.get("title")
                or ""
            )
            publisher = (
                content.get("provider", {}).get("displayName")
                or article.get("publisher")
                or "Unknown"
            )
            pub_date = (
                content.get("pubDate")
                or article.get("providerPublishTime")
                or ""
            )
            # Convert UNIX timestamp to ISO string if needed
            if isinstance(pub_date, (int, float)):
                try:
                    pub_date = datetime.utcfromtimestamp(pub_date).strftime("%Y-%m-%d %H:%M UTC")
                except Exception:
                    pub_date = str(pub_date)
            url = (
                content.get("canonicalUrl", {}).get("url")
                or article.get("link")
                or ""
            )
            summary = content.get("summary") or article.get("summary") or ""
            if title:
                items.append(NewsItem(
                    title=title,
                    publisher=publisher,
                    published=pub_date,
                    url=url,
                    summary=summary[:300] if summary else "",
                ))
        logger.info(f"Fetched {len(items)} news items for {symbol}")
        return items
    except Exception as e:
        logger.warning(f"Failed to fetch news for {symbol}: {e}")
        return []


async def fetch_news(symbol: str, limit: int = 5) -> List[NewsItem]:
    """Async wrapper: runs yfinance fetch in a thread pool so it doesn't block."""
    import asyncio
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, fetch_news_sync, symbol, limit)


def format_news(items: List[NewsItem]) -> str:
    """Format news items for the LLM prompt."""
    if not items:
        return "No recent news found."
    lines = []
    for i, item in enumerate(items, 1):
        lines.append(f"{i}. [{item.publisher}] {item.title} ({item.published})")
        if item.summary:
            lines.append(f"   Summary: {item.summary}")
    return "\n".join(lines)
