import unittest
from unittest.mock import AsyncMock, MagicMock, patch

try:
    from agent import Agent0
    _IMPORT_ERROR = None
except ModuleNotFoundError as exc:  # pragma: no cover - environment-dependent
    Agent0 = None
    _IMPORT_ERROR = exc


@unittest.skipIf(Agent0 is None, f"agent0 test dependencies unavailable: {_IMPORT_ERROR}")
class Agent0MemoryFetchTests(unittest.IsolatedAsyncioTestCase):
    async def test_fetch_memories_uses_memory_tool_contract(self):
        agent = Agent0()
        agent.http_client.post = AsyncMock()

        response = MagicMock()
        response.raise_for_status.return_value = None
        response.json.return_value = {
            "output": {
                "items": [
                    {
                        "id": "mem-1",
                        "summary": "Prior AAPL trade worked on earnings follow-through.",
                        "ts": "2026-04-09T12:00:00Z",
                        "source": {"system": "jax-research"},
                    }
                ]
            }
        }
        agent.http_client.post.return_value = response

        with patch("agent.settings.memory_service_url", "http://jax-research:8091"), patch(
            "agent.settings.memory_bank", "trades"
        ), patch("agent.settings.api_timeout", 5):
            memories = await agent._fetch_memories("AAPL", limit=3)

        agent.http_client.post.assert_awaited_once()
        _, kwargs = agent.http_client.post.await_args
        self.assertEqual(kwargs["json"]["tool"], "memory.recall")
        self.assertEqual(kwargs["json"]["input"]["bank"], "trades")
        self.assertEqual(kwargs["json"]["input"]["query"]["symbol"], "AAPL")
        self.assertEqual(kwargs["json"]["input"]["query"]["limit"], 3)
        self.assertEqual(len(memories), 1)
        self.assertEqual(memories[0].content, "Prior AAPL trade worked on earnings follow-through.")
        self.assertEqual(memories[0].source, "jax-research")
