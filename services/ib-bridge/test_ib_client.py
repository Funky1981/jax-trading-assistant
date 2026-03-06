import asyncio
import unittest

from ib_insync import Stock

from ib_client import IBClient


class FakeEvent:
    def __init__(self):
        self.emitted = False

    def emit(self):
        self.emitted = True


class FakeLowLevelClient:
    def __init__(self):
        self.calls = []
        self._accounts = ["DUH949450"]

    async def connectAsync(self, host, port, client_id, timeout):
        self.calls.append((host, port, client_id, timeout))

    def getAccounts(self):
        return list(self._accounts)

    def serverVersion(self):
        return 176

    def isReady(self):
        return True


class FakeIB:
    MaxSyncedSubAccounts = 10

    def __init__(self):
        self.client = FakeLowLevelClient()
        self.wrapper = type("Wrapper", (), {"clientId": None})()
        self.connectedEvent = FakeEvent()
        self.auto_open_orders_enabled = False

    def reqAutoOpenOrders(self, enabled):
        self.auto_open_orders_enabled = enabled

    async def reqPositionsAsync(self):
        await asyncio.sleep(0.02)

    async def reqOpenOrdersAsync(self):
        await asyncio.sleep(0.02)

    async def reqCompletedOrdersAsync(self, _include_api_only):
        await asyncio.sleep(0.02)

    async def reqAccountUpdatesAsync(self, _account):
        await asyncio.sleep(0.02)

    async def reqAccountUpdatesMultiAsync(self, _account):
        await asyncio.sleep(0.02)

    async def reqExecutionsAsync(self):
        await asyncio.sleep(0.02)


class IBClientConnectionTests(unittest.IsolatedAsyncioTestCase):
    async def test_connect_with_tolerant_sync_keeps_ready_session_on_timeouts(self):
        client = IBClient()
        client.ib = FakeIB()
        client.host = "host.docker.internal"
        client.port = 4002
        client.client_id = 7

        await client._connect_with_tolerant_sync(timeout=0.001)

        self.assertEqual(client.ib.wrapper.clientId, 7)
        self.assertEqual(
            client.ib.client.calls,
            [("host.docker.internal", 4002, 7, 0.001)],
        )
        self.assertTrue(client.ib.connectedEvent.emitted)

    async def test_qualify_stock_contract_falls_back_on_timeout(self):
        client = IBClient()

        class SlowQualifierIB(FakeIB):
            async def qualifyContractsAsync(self, _contract):
                await asyncio.sleep(5.1)
                return []

        client.ib = SlowQualifierIB()

        contract = await client._qualify_stock_contract("AAPL")

        self.assertIsInstance(contract, Stock)
        self.assertEqual(contract.symbol, "AAPL")
        self.assertEqual(contract.exchange, "SMART")
        self.assertEqual(contract.currency, "USD")


if __name__ == "__main__":
    unittest.main()
