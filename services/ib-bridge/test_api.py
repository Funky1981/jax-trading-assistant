import asyncio
import unittest

import main
from models import (
    BrokerOrder,
    BracketOrderRequest,
    ClosePositionRequest,
    ProtectPositionRequest,
)


class FakeBridge:
    def __init__(self):
        self.connected = True

    def is_connected(self):
        return self.connected

    async def list_orders(self):
        return [
            BrokerOrder(
                order_id=101,
                symbol="AAPL",
                action="BUY",
                order_type="LMT",
                quantity=10,
                limit_price=201.25,
                stop_price=None,
                status="Submitted",
                filled_qty=0,
                remaining_qty=10,
                avg_fill_price=0.0,
                can_cancel=True,
                created_at="2026-03-06T14:00:00Z",
                updated_at="2026-03-06T14:01:00Z",
            )
        ]

    async def cancel_order(self, order_id: int):
        return {
            "success": True,
            "order_id": order_id,
            "status": "PendingCancel",
            "message": f"Cancel requested for {order_id}",
        }

    async def close_position(self, symbol: str, quantity=None, order_type="MKT", limit_price=None):
        return 9001

    async def protect_position(self, symbol: str, stop_loss: float, take_profit=None, quantity=None, replace_existing=True):
        return {
            "success": True,
            "order_ids": [3001, 3002],
            "cancelled_order_ids": [2999],
            "message": f"Submitted protection for {symbol}",
        }

    async def place_bracket_order(
        self,
        symbol: str,
        action: str,
        quantity: int,
        entry_order_type="MKT",
        entry_limit_price=None,
        stop_loss=None,
        take_profit=None,
    ):
        return {
            "success": True,
            "parent_order_id": 4001,
            "child_order_ids": [4002, 4003],
            "message": f"Bracket order submitted for {symbol}",
        }


class IBBridgeApiTests(unittest.TestCase):
    def setUp(self):
        self.original_bridge = main.ib_client
        main.ib_client = FakeBridge()

    def tearDown(self):
        main.ib_client = self.original_bridge

    def test_list_orders(self):
        response = asyncio.run(main.list_orders())

        self.assertEqual(response.count, 1)
        self.assertTrue(response.orders[0].can_cancel)
        self.assertEqual(response.orders[0].symbol, "AAPL")

    def test_cancel_order(self):
        response = asyncio.run(main.cancel_order(101))

        self.assertTrue(response["success"])
        self.assertEqual(response["order_id"], 101)

    def test_close_position(self):
        response = asyncio.run(
            main.close_position(
                "SPY",
                ClosePositionRequest(quantity=5, order_type="MKT"),
            )
        )

        self.assertTrue(response.success)
        self.assertEqual(response.order_id, 9001)

    def test_protect_position(self):
        response = asyncio.run(
            main.protect_position(
                "SPY",
                ProtectPositionRequest(quantity=5, stop_loss=500.0, take_profit=530.0),
            )
        )

        self.assertEqual(response["order_ids"], [3001, 3002])
        self.assertEqual(response["cancelled_order_ids"], [2999])

    def test_place_bracket_order(self):
        response = asyncio.run(
            main.place_bracket_order(
                BracketOrderRequest(
                    symbol="AAPL",
                    action="BUY",
                    quantity=10,
                    entry_order_type="LMT",
                    entry_limit_price=200.0,
                    stop_loss=195.0,
                    take_profit=210.0,
                )
            )
        )

        self.assertEqual(response["parent_order_id"], 4001)
        self.assertEqual(response["child_order_ids"], [4002, 4003])


if __name__ == "__main__":
    unittest.main()
