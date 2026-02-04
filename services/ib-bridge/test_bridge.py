"""
Example usage and testing script for IB Bridge
Run this to verify the bridge is working correctly
"""
import asyncio
import sys
from ib_client import IBClient

async def test_connection():
    """Test connection to IB Gateway"""
    client = IBClient(
        host="127.0.0.1",
        port=7497,  # Paper trading
        client_id=1
    )
    
    try:
        print("🔌 Connecting to IB Gateway...")
        await client.connect()
        print("✅ Connected successfully!")
        return client
    except Exception as e:
        print(f"❌ Connection failed: {e}")
        print("\nTroubleshooting:")
        print("1. Ensure IB Gateway is running")
        print("2. Check that API access is enabled in IB Gateway settings")
        print("3. Verify port 7497 is correct (7497=paper, 7496=live)")
        sys.exit(1)


async def test_quote(client, symbol="AAPL"):
    """Test getting a real-time quote"""
    try:
        print(f"\n📊 Fetching quote for {symbol}...")
        quote = await client.get_quote(symbol)
        
        print(f"✅ Quote received:")
        print(f"   Symbol: {quote.symbol}")
        print(f"   Price: ${quote.price:.2f}")
        print(f"   Bid: ${quote.bid:.2f} x {quote.bid_size}")
        print(f"   Ask: ${quote.ask:.2f} x {quote.ask_size}")
        print(f"   Volume: {quote.volume:,}")
        print(f"   Exchange: {quote.exchange}")
        print(f"   Time: {quote.timestamp}")
        
        return True
    except Exception as e:
        print(f"❌ Failed to get quote: {e}")
        return False


async def test_candles(client, symbol="AAPL"):
    """Test getting historical candles"""
    try:
        print(f"\n📈 Fetching candles for {symbol}...")
        candles = await client.get_candles(
            symbol=symbol,
            duration="1 D",
            bar_size="5 mins",
            what_to_show="TRADES"
        )
        
        print(f"✅ Received {len(candles)} candles")
        
        if candles:
            last = candles[-1]
            print(f"   Last candle:")
            print(f"   Time: {last.timestamp}")
            print(f"   O: ${last.open:.2f}")
            print(f"   H: ${last.high:.2f}")
            print(f"   L: ${last.low:.2f}")
            print(f"   C: ${last.close:.2f}")
            print(f"   V: {last.volume:,}")
            
        return True
    except Exception as e:
        print(f"❌ Failed to get candles: {e}")
        return False


async def test_account(client):
    """Test getting account information"""
    try:
        print("\n💰 Fetching account information...")
        account = await client.get_account_info()
        
        print(f"✅ Account info received:")
        print(f"   Account ID: {account.account_id}")
        print(f"   Net Liquidation: ${account.net_liquidation:,.2f}")
        print(f"   Total Cash: ${account.total_cash:,.2f}")
        print(f"   Buying Power: ${account.buying_power:,.2f}")
        print(f"   Currency: {account.currency}")
        
        return True
    except Exception as e:
        print(f"❌ Failed to get account info: {e}")
        return False


async def test_positions(client):
    """Test getting positions"""
    try:
        print("\n📍 Fetching positions...")
        positions = await client.get_positions()
        
        if positions:
            print(f"✅ Found {len(positions)} position(s):")
            for pos in positions:
                print(f"   {pos.symbol}: {pos.quantity} shares @ ${pos.avg_cost:.2f}")
                print(f"   Market Value: ${pos.market_value:,.2f}")
        else:
            print("✅ No positions found (account is flat)")
        
        return True
    except Exception as e:
        print(f"❌ Failed to get positions: {e}")
        return False


async def main():
    """Run all tests"""
    print("=" * 60)
    print("IB Bridge Connection Test")
    print("=" * 60)
    
    # Connect
    client = await test_connection()
    
    # Run tests
    tests = [
        ("Quote", lambda: test_quote(client, "AAPL")),
        ("Candles", lambda: test_candles(client, "AAPL")),
        ("Account", lambda: test_account(client)),
        ("Positions", lambda: test_positions(client)),
    ]
    
    results = []
    for name, test_func in tests:
        try:
            result = await test_func()
            results.append((name, result))
        except Exception as e:
            print(f"❌ Test '{name}' raised exception: {e}")
            results.append((name, False))
    
    # Disconnect
    print("\n🔌 Disconnecting...")
    await client.disconnect()
    print("✅ Disconnected")
    
    # Summary
    print("\n" + "=" * 60)
    print("Test Summary")
    print("=" * 60)
    
    passed = sum(1 for _, result in results if result)
    total = len(results)
    
    for name, result in results:
        status = "✅ PASS" if result else "❌ FAIL"
        print(f"{status} - {name}")
    
    print(f"\nTotal: {passed}/{total} tests passed")
    
    if passed == total:
        print("\n🎉 All tests passed! IB Bridge is working correctly.")
        return 0
    else:
        print("\n⚠️  Some tests failed. Check the output above for details.")
        return 1


if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
