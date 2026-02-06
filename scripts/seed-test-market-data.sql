-- Seed test market data for signal generator testing
-- This script creates historical candles for testing purposes

-- Insert test quotes (current prices)
INSERT INTO quotes (symbol, price, bid, ask, volume, timestamp, exchange)
VALUES 
    ('AAPL', 185.50, 185.48, 185.52, 45000000, NOW(), 'NASDAQ'),
    ('MSFT', 420.25, 420.20, 420.30, 28000000, NOW(), 'NASDAQ'),
    ('GOOGL', 142.75, 142.70, 142.80, 22000000, NOW(), 'NASDAQ'),
    ('AMZN', 178.90, 178.85, 178.95, 35000000, NOW(), 'NASDAQ'),
    ('TSLA', 238.45, 238.40, 238.50, 95000000, NOW(), 'NASDAQ'),
    ('META', 528.30, 528.25, 528.35, 18000000, NOW(), 'NASDAQ'),
    ('NVDA', 880.75, 880.70, 880.80, 42000000, NOW(), 'NASDAQ'),
    ('AMD', 142.60, 142.55, 142.65, 38000000, NOW(), 'NASDAQ'),
    ('NFLX', 655.80, 655.75, 655.85, 8000000, NOW(), 'NASDAQ'),
    ('SPY', 510.25, 510.20, 510.30, 75000000, NOW(), 'NYSE')
ON CONFLICT (symbol) DO UPDATE 
SET 
    price = EXCLUDED.price,
    bid = EXCLUDED.bid,
    ask = EXCLUDED.ask,
    volume = EXCLUDED.volume,
    timestamp = EXCLUDED.timestamp,
    updated_at = NOW();

-- Generate 250 days of historical candles for AAPL (for SMA200 calculation)
DO $$
DECLARE
    base_date TIMESTAMP := NOW() - INTERVAL '250 days';
    base_price FLOAT := 150.0;
    current_price FLOAT;
    i INT;
BEGIN
    FOR i IN 0..249 LOOP
        -- Simulate price movement with some randomness
        current_price := base_price + (i * 0.14) + (RANDOM() * 4 - 2);
        
        INSERT INTO candles (symbol, timestamp, open, high, low, close, volume)
        VALUES (
            'AAPL',
            base_date + (i || ' days')::INTERVAL,
            current_price,
            current_price + (RANDOM() * 2),
            current_price - (RANDOM() * 2),
            current_price + (RANDOM() * 2 - 1),
            40000000 + FLOOR(RANDOM() * 20000000)::BIGINT
        )
        ON CONFLICT (symbol, timestamp) DO NOTHING;
    END LOOP;
END $$;

-- Generate historical candles for other symbols
DO $$
DECLARE
    symbols TEXT[] := ARRAY['MSFT', 'GOOGL', 'AMZN', 'TSLA', 'META', 'NVDA', 'AMD', 'NFLX', 'SPY'];
    base_prices FLOAT[] := ARRAY[350.0, 95.0, 130.0, 180.0, 380.0, 600.0, 100.0, 500.0, 450.0];
    base_date TIMESTAMP := NOW() - INTERVAL '250 days';
    sym TEXT;
    base_price FLOAT;
    current_price FLOAT;
    i INT;
    j INT;
BEGIN
    FOR j IN 1..array_length(symbols, 1) LOOP
        sym := symbols[j];
        base_price := base_prices[j];
        
        FOR i IN 0..249 LOOP
            -- Simulate price movement with some randomness
            current_price := base_price + (i * (base_price * 0.001)) + (RANDOM() * (base_price * 0.02) - (base_price * 0.01));
            
            INSERT INTO candles (symbol, timestamp, open, high, low, close, volume)
            VALUES (
                sym,
                base_date + (i || ' days')::INTERVAL,
                current_price,
                current_price + (RANDOM() * (base_price * 0.015)),
                current_price - (RANDOM() * (base_price * 0.015)),
                current_price + (RANDOM() * (base_price * 0.02) - (base_price * 0.01)),
                30000000 + FLOOR(RANDOM() * 40000000)::BIGINT
            )
            ON CONFLICT (symbol, timestamp) DO NOTHING;
        END LOOP;
    END LOOP;
END $$;

-- Verify data was inserted
SELECT 
    'quotes' as table_name,
    COUNT(*) as record_count
FROM quotes
UNION ALL
SELECT 
    'candles' as table_name,
    COUNT(*) as record_count
FROM candles
UNION ALL
SELECT 
    symbol,
    COUNT(*) as candle_count
FROM candles
GROUP BY symbol
ORDER BY table_name, record_count DESC;
