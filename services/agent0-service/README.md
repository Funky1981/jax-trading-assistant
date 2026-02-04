# Agent0 Service - AI Trading Assistant

AI-powered trading suggestions with multi-provider LLM support.

## Quick Start

### Option 1: Use Ollama (FREE, Local)

1. Install Ollama from https://ollama.ai
2. Start Ollama: `ollama serve`
3. Pull a model: `ollama pull llama3.2`
4. Run the service: `python main.py`

### Option 2: Use OpenAI (Paid)

```bash
export AGENT0_LLM_PROVIDER=openai
export AGENT0_OPENAI_API_KEY=sk-your-key-here
python main.py
```

### Option 3: Use Docker

```bash
docker build -t agent0-service .
docker run -p 8093:8093 -e AGENT0_OLLAMA_HOST=http://host.docker.internal:11434 agent0-service
```

## API Endpoints

### GET /health
Check service health and LLM configuration.

### POST /suggest
Get a trading suggestion.

```bash
curl -X POST http://localhost:8093/suggest \
     -H "Content-Type: application/json" \
     -d '{"symbol": "AAPL"}'
```

**Response:**
```json
{
    "symbol": "AAPL",
    "action": "BUY",
    "confidence": 78,
    "reasoning": "Strong momentum with earnings beat...",
    "entry_price": 185.50,
    "target_price": 195.00,
    "stop_loss": 180.00,
    "risk": {
        "risk_level": "medium",
        "stop_loss_pct": 3.0,
        "position_size_pct": 5.0
    }
}
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| AGENT0_LLM_PROVIDER | ollama | LLM provider: ollama, openai, anthropic |
| AGENT0_OLLAMA_HOST | http://localhost:11434 | Ollama server URL |
| AGENT0_OLLAMA_MODEL | llama3.2 | Ollama model name |
| AGENT0_OPENAI_API_KEY | - | OpenAI API key (required if using openai) |
| AGENT0_OPENAI_MODEL | gpt-4o-mini | OpenAI model |
| AGENT0_ANTHROPIC_API_KEY | - | Anthropic API key (required if using anthropic) |
| AGENT0_MEMORY_SERVICE_URL | http://jax-memory:8090 | Memory service URL |
| AGENT0_IB_BRIDGE_URL | http://ib-bridge:8092 | IB Bridge URL |

## Cost Comparison

| Provider | Cost per Request | Quality |
|----------|-----------------|---------|
| Ollama | **FREE** | Good |
| OpenAI GPT-4o-mini | ~$0.01 | Excellent |
| OpenAI GPT-4o | ~$0.03 | Best |
| Anthropic Claude Haiku | ~$0.01 | Excellent |
