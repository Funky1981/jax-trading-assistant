/**
 * Beginner-friendly text helpers and glossary translations
 * Converts technical jargon into plain English explanations
 */

export type UXMode = 'simple' | 'detailed' | 'technical';

// ETF descriptions for beginner display
export const ETF_DESCRIPTIONS: Record<string, { name: string; simple: string; detailed: string; riskLevel: 'low' | 'medium' | 'high'; categories: string[] }> = {
  SPY: {
    name: 'S&P 500 ETF',
    simple: 'The 500 biggest US companies. Wide diversification across all sectors.',
    detailed: 'Tracks the S&P 500 index. Broad exposure to large-cap US equities across tech, finance, healthcare, and other major sectors.',
    riskLevel: 'medium',
    categories: ['macro', 'market_breadth', 'us_equities'],
  },
  QQQ: {
    name: 'Nasdaq 100 ETF',
    simple: 'Tech-heavy: mostly software, semiconductors, and internet companies.',
    detailed: 'Tracks 100 largest non-financial Nasdaq stocks. Heavy concentration in technology, which makes it more volatile than SPY.',
    riskLevel: 'high',
    categories: ['tech', 'semiconductor', 'internet'],
  },
  DIA: {
    name: 'Dow Jones 30 ETF',
    simple: '30 of the oldest, most established US companies (blue-chip stocks).',
    detailed: 'Tracks the Dow Jones Industrial Average. Includes mature industrial, financial, and consumer staple companies.',
    riskLevel: 'medium',
    categories: ['macro', 'blue_chip', 'industrial'],
  },
  IWM: {
    name: 'Russell 2000 Small-Cap ETF',
    simple: 'Small US companies. Often move faster than large companies in boom/bust cycles.',
    detailed: 'Tracks 2000 smallest US public companies. More volatile than large-cap indices, sensitive to economic cycles.',
    riskLevel: 'high',
    categories: ['small_cap', 'economic_cycle', 'domestic'],
  },
  XLK: {
    name: 'Technology Sector ETF',
    simple: 'Tech companies: software, hardware, semiconductors, IT services.',
    detailed: 'Sector ETF tracking tech companies. Sensitive to AI news, interest rates, and enterprise spending.',
    riskLevel: 'high',
    categories: ['tech', 'semiconductor', 'software'],
  },
  XLF: {
    name: 'Financials Sector ETF',
    simple: 'Banks, insurance, and investment firms. Reacts to interest rates and economic outlook.',
    detailed: 'Sector ETF tracking financial services. Sensitive to Fed rate changes, credit spreads, and economic health.',
    riskLevel: 'medium',
    categories: ['macro', 'rates', 'financial'],
  },
  XLE: {
    name: 'Energy Sector ETF',
    simple: 'Oil, gas, and coal companies. Moves with energy prices and geopolitical events.',
    detailed: 'Sector ETF tracking energy companies. Sensitive to oil prices, supply disruptions, and energy policy.',
    riskLevel: 'high',
    categories: ['energy', 'commodities', 'geopolitical'],
  },
  SMH: {
    name: 'Semiconductor ETF',
    simple: 'Chip makers and designers. Reacts to AI news, supply chain, and tech spending.',
    detailed: 'Specialized ETF tracking semiconductor companies. Highly sensitive to AI hype, enterprise capex, and chip shortages.',
    riskLevel: 'high',
    categories: ['semiconductor', 'ai', 'tech'],
  },
  SOXX: {
    name: 'Semiconductor Index ETF',
    simple: 'Another chip ETF. Similar to SMH but slightly different holdings.',
    detailed: 'Similar semiconductor exposure to SMH but with different index methodology and weightings.',
    riskLevel: 'high',
    categories: ['semiconductor', 'ai', 'tech'],
  },
  TLT: {
    name: 'Long-Term Treasury Bond ETF',
    simple: 'US government bonds (20+ years). Falls when interest rates rise, rises when rates fall.',
    detailed: 'Tracks long-duration Treasury bonds. Inverse relationship to interest rates; sensitive to Fed policy.',
    riskLevel: 'low',
    categories: ['bonds', 'rates', 'macro'],
  },
  GLD: {
    name: 'Gold ETF',
    simple: 'Gold prices. Safe haven asset that rises when stocks fall or inflation worries increase.',
    detailed: 'Tracks physical gold prices. Typically inverse to USD strength and real yields; used as portfolio hedge.',
    riskLevel: 'medium',
    categories: ['commodities', 'inflation_hedge', 'safe_haven'],
  },
};

/**
 * Translate technical terms to beginner-friendly language based on UX mode
 */
export function translateTerm(term: string, mode: UXMode): string {
  const glossary: Record<string, { simple: string; detailed: string; technical: string }> = {
    confidence: {
      simple: 'How sure Jax is',
      detailed: 'Confidence score (0-100%) of the signal strength',
      technical: 'Normalized confidence metric',
    },
    pricedIn: {
      simple: 'Already expected',
      detailed: 'The market may have already reacted to this news',
      technical: 'Price discovery lag analysis',
    },
    abnormalReturn: {
      simple: 'Bigger than expected move',
      detailed: 'ETF moved more than the wider market, so the news may have had a real effect',
      technical: 'Excess return vs benchmark',
    },
    confounder: {
      simple: 'Other news that might explain the move',
      detailed: 'Other market events happening around the same time that could affect the price',
      technical: 'Covariate confounding factor',
    },
    entryPrice: {
      simple: 'Buy here',
      detailed: 'Suggested price to buy the ETF',
      technical: 'Entry point',
    },
    stopLoss: {
      simple: 'Sell if it drops this far',
      detailed: 'Exit position if loss reaches this level to limit risk',
      technical: 'Stop-loss threshold',
    },
    takeProfit: {
      simple: 'Sell if it rises this far',
      detailed: 'Exit position and lock in gains when price reaches this target',
      technical: 'Take-profit threshold',
    },
    riskScore: {
      simple: 'Risk level',
      detailed: 'How much of your account is at stake',
      technical: 'Position risk metric',
    },
    volatility: {
      simple: 'How much it moves',
      detailed: 'Historical price swings (higher = more jumpy)',
      technical: 'Historical volatility',
    },
    correlation: {
      simple: 'Tends to move together',
      detailed: 'When one goes up, the other tends to go up too',
      technical: 'Pearson correlation coefficient',
    },
    drawdown: {
      simple: 'Losing streak',
      detailed: 'Peak-to-trough decline in value during a losing period',
      technical: 'Maximum drawdown',
    },
    sharpeRatio: {
      simple: 'Risk-adjusted returns',
      detailed: 'How much profit per unit of risk',
      technical: 'Sharpe ratio',
    },
  };

  const entry = glossary[term];
  if (!entry) return term; // Return original if not in glossary
  return entry[mode];
}

/**
 * Format a confidence score for display with beginner-friendly language
 */
export function formatConfidenceForBeginners(confidence: number | undefined, mode: UXMode): string {
  if (confidence === undefined) return 'Unknown';
  
  const pct = Math.round(confidence * 100);
  
  if (mode === 'simple') {
    if (pct >= 80) return `Very confident (${pct}%)`;
    if (pct >= 60) return `Somewhat confident (${pct}%)`;
    if (pct >= 40) return `Uncertain (${pct}%)`;
    return `Not confident (${pct}%)`;
  }
  
  return `${pct}%`;
}

/**
 * Create a beginner-friendly summary of price movement
 */
export function describeMovement(entry: number | undefined, current: number | undefined, mode: UXMode): string {
  if (entry === undefined || current === undefined) return '';
  
  const change = current - entry;
  const pctChange = ((change / entry) * 100).toFixed(2);
  
  if (mode === 'simple') {
    if (change > 0) return `Up ${Math.abs(parseFloat(pctChange))}% since suggested entry`;
    if (change < 0) return `Down ${Math.abs(parseFloat(pctChange))}% since suggested entry`;
    return 'No change since suggested entry';
  }
  
  return `${Math.abs(parseFloat(pctChange))}% ${change > 0 ? '📈' : '📉'}`;
}

/**
 * Explain a risk level in beginner terms
 */
export function explainRiskLevel(level: 'low' | 'medium' | 'high', mode: UXMode): string {
  const explanations = {
    low: {
      simple: 'Safer. Smaller moves. Good for learning.',
      detailed: 'Lower volatility. Smaller typical daily price swings.',
      technical: 'Low historical volatility, lower drawdown risk',
    },
    medium: {
      simple: 'Moderate risk. Normal for most traders.',
      detailed: 'Balanced volatility. Typical daily movement range.',
      technical: 'Moderate volatility, standard market risk profile',
    },
    high: {
      simple: 'Risky. Big moves up and down. Advanced traders only.',
      detailed: 'Higher volatility. Large daily price swings possible.',
      technical: 'High historical volatility, larger potential drawdowns',
    },
  };
  
  return explanations[level][mode];
}

/**
 * Generate a beginner-friendly explanation of why Jax rejected a candidate
 */
export function explainRejectReason(reason: string | undefined, mode: UXMode): string {
  if (!reason) return 'No specific reason provided';
  
  const reasonMap: Record<string, { simple: string; detailed: string; technical: string }> = {
    'etf_policy_violation': {
      simple: 'Not an approved ETF for this phase.',
      detailed: 'This ETF is not yet approved for paper trading in the current phase.',
      technical: 'ETF policy constraint violation',
    },
    'insufficient_confidence': {
      simple: 'Not sure enough to trade.',
      detailed: 'The signal strength is below the minimum threshold.',
      technical: 'Confidence below minimum threshold',
    },
    'news_already_priced_in': {
      simple: 'The market already knows about this news.',
      detailed: 'The price has already moved to reflect this news.',
      technical: 'News event appears to be priced in',
    },
    'conflicting_news': {
      simple: 'Other news suggests a different direction.',
      detailed: 'Other market events suggest a conflicting signal.',
      technical: 'Conflicting signals detected',
    },
    'market_closed': {
      simple: 'Market is closed. Can only trade during market hours.',
      detailed: 'Trading can only occur during regular market hours (9:30 AM - 4:00 PM ET).',
      technical: 'Outside regular trading hours',
    },
    'position_exists': {
      simple: 'Already have a position in this ETF.',
      detailed: 'Cannot open multiple positions in the same ETF simultaneously.',
      technical: 'Position already exists',
    },
    'stop_loss_requirement': {
      simple: 'Need a stop-loss order for protection.',
      detailed: 'A stop-loss order is required but was not provided.',
      technical: 'Stop-loss requirement not met',
    },
  };
  
  const entry = reasonMap[reason];
  if (!entry) return reason; // Return original if not in map
  return entry[mode];
}

/**
 * Create a beginner-friendly news impact explanation
 */
export function explainNewsImpact(newsType: string, etf: string, mode: UXMode): string {
  const impacts: Record<string, string> = {
    'ai_breakthrough': `AI news typically pushes ${etf} up because many companies in it are involved in AI.`,
    'rate_hike': `Interest rate increases often push bonds down. ${etf} tends to decline because bonds become less attractive.`,
    'earnings_beat': `Strong company earnings suggest health, pushing sector ETFs higher.`,
    'supply_shock': `Supply disruptions push commodity prices and related ETFs higher.`,
    'geopolitical': `Political events or conflicts can disrupt markets. Oil and commodity ETFs are especially sensitive.`,
    'inflation_report': `Inflation concerns push up commodity and gold prices, while bonds fall.`,
    'fed_decision': `Fed actions directly affect bonds and financial stocks. ${etf} typically reacts strongly.`,
  };
  
  if (mode === 'simple' || mode === 'detailed') {
    return impacts[newsType] || `${newsType} may affect ${etf}`;
  }
  
  return `${newsType} → ${etf}`;
}

/**
 * Format a price for display
 */
export function formatPrice(price: number | undefined, mode: UXMode): string {
  if (price === undefined) return '-';
  return mode === 'simple' ? `$${price.toFixed(2)}` : price.toFixed(2);
}

/**
 * Get UX mode from localStorage with default
 */
export function getBeginnerMode(): UXMode {
  const stored = localStorage.getItem('beginner-mode');
  if (stored === 'simple' || stored === 'detailed' || stored === 'technical') {
    return stored;
  }
  return 'simple'; // Default to simple mode for beginners
}

/**
 * Set UX mode in localStorage
 */
export function setBeginnerMode(mode: UXMode): void {
  localStorage.setItem('beginner-mode', mode);
}
