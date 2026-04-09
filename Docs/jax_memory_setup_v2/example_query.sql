SELECT *
FROM research
WHERE symbol = 'AAPL'
ORDER BY embedding <=> '[query_vector]'
LIMIT 5;