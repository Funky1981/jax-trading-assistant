import unittest
from unittest.mock import AsyncMock, MagicMock, patch
from datetime import timezone

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

    async def test_fetch_memories_requires_explicit_bank(self):
        agent = Agent0()

        with patch("agent.settings.memory_bank", None):
            with self.assertRaisesRegex(ValueError, "AGENT0_MEMORY_BANK must be set"):
                await agent._fetch_memories("AAPL", limit=3)

    async def test_fetch_memories_raises_on_service_failure(self):
        agent = Agent0()
        agent.http_client.post = AsyncMock(side_effect=RuntimeError("boom"))

        with patch("agent.settings.memory_service_url", "http://jax-research:8091"), patch(
            "agent.settings.memory_bank", "trades"
        ), patch("agent.settings.api_timeout", 5):
            with self.assertRaisesRegex(RuntimeError, "memory fetch failed for AAPL"):
                await agent._fetch_memories("AAPL", limit=3)

    async def test_fetch_memories_normalizes_legacy_tools_url(self):
        agent = Agent0()
        agent.http_client.post = AsyncMock()

        response = MagicMock()
        response.raise_for_status.return_value = None
        response.json.return_value = {"output": {"items": []}}
        agent.http_client.post.return_value = response

        with patch("agent.settings.memory_service_url", "http://jax-research:8091/tools"), patch(
            "agent.settings.memory_bank", "trades"
        ), patch("agent.settings.api_timeout", 5):
            await agent._fetch_memories("AAPL", limit=3)

        args, _ = agent.http_client.post.await_args
        self.assertEqual(args[0], "http://jax-research:8091/tools")

    async def test_fetch_memories_tolerates_invalid_timestamp(self):
        agent = Agent0()
        agent.http_client.post = AsyncMock()

        response = MagicMock()
        response.raise_for_status.return_value = None
        response.json.return_value = {
            "output": {
                "items": [
                    {
                        "id": "mem-1",
                        "summary": "Malformed timestamp should not break recall.",
                        "ts": "not-a-timestamp",
                        "source": {"system": "jax-research"},
                    }
                ]
            }
        }
        agent.http_client.post.return_value = response

        with patch("agent.settings.memory_service_url", "http://jax-research:8091"), patch(
            "agent.settings.memory_bank", "trades"
        ), patch("agent.settings.api_timeout", 5):
            memories = await agent._fetch_memories("AAPL", limit=1)

        self.assertEqual(len(memories), 1)
        self.assertIsNotNone(memories[0].created_at.tzinfo)
        self.assertEqual(memories[0].created_at.tzinfo, timezone.utc)

    async def test_check_health_uses_ready_endpoint_on_normalized_memory_base(self):
        agent = Agent0()
        agent.http_client.get = AsyncMock()

        memory_response = MagicMock(status_code=200)
        ib_response = MagicMock(status_code=200)
        agent.http_client.get.side_effect = [memory_response, ib_response]

        with patch("agent.settings.memory_service_url", "http://jax-research:8091/tools"), patch(
            "agent.settings.memory_bank", "trades"
        ), patch("agent.settings.ib_bridge_url", "http://ib-bridge:8092"):
            health = await agent.check_health()

        self.assertTrue(health["ready"])
        self.assertEqual(health["status"], "healthy")
        self.assertTrue(health["memory_connected"])
        self.assertEqual(health["memory_status"], "ready")
        self.assertTrue(health["ib_connected"])
        self.assertEqual(health["ib_status"], "ready")
        calls = agent.http_client.get.await_args_list
        self.assertEqual(calls[0].args[0], "http://jax-research:8091/ready")
        self.assertEqual(calls[1].args[0], "http://ib-bridge:8092/health")

    async def test_check_health_reports_memory_misconfiguration(self):
        agent = Agent0()
        agent.http_client.get = AsyncMock(return_value=MagicMock(status_code=200, text="ok"))

        with patch("agent.settings.memory_bank", None), patch(
            "agent.settings.ib_bridge_url", "http://ib-bridge:8092"
        ):
            health = await agent.check_health()

        self.assertFalse(health["ready"])
        self.assertEqual(health["status"], "degraded")
        self.assertEqual(health["memory_status"], "misconfigured")
        self.assertIn("AGENT0_MEMORY_BANK", health["memory_error"])
        self.assertTrue(health["ib_connected"])

    async def test_check_health_treats_degraded_ib_payload_as_not_ready(self):
        agent = Agent0()
        agent.http_client.get = AsyncMock()

        memory_response = MagicMock(status_code=200)
        memory_response.json.return_value = {"status": "healthy", "memory_ready": True}
        ib_response = MagicMock(status_code=200)
        ib_response.json.return_value = {"status": "degraded", "connected": False}
        agent.http_client.get.side_effect = [memory_response, ib_response]

        with patch("agent.settings.memory_service_url", "http://jax-research:8091"), patch(
            "agent.settings.memory_bank", "trades"
        ), patch("agent.settings.ib_bridge_url", "http://ib-bridge:8092"):
            health = await agent.check_health()

        self.assertFalse(health["ready"])
        self.assertEqual(health["status"], "degraded")
        self.assertFalse(health["ib_connected"])
        self.assertEqual(health["ib_status"], "not_ready")
        self.assertIn("connected=false", health["ib_error"])
