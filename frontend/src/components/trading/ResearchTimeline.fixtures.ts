import type { TimelineEvent } from './ResearchTimeline.types';

export function generateExampleTimeline(): TimelineEvent[] {
  const baseTime = new Date();
  return [
    {
      timestamp: new Date(baseTime.getTime() - 120 * 60000).toLocaleTimeString(),
      type: 'news',
      title: 'Breaking: AI Chip Announcement',
      description:
        'Major AI company announces new chip design. Semiconductor ETFs expected to move.',
      etf: 'SMH',
      details: {
        source: 'Reuters',
        impact: 'Market +1.2%',
      },
      severity: 'info',
    },
    {
      timestamp: new Date(baseTime.getTime() - 100 * 60000).toLocaleTimeString(),
      type: 'analysis',
      title: 'Jax Analyzes News',
      description:
        'Priced-in check: SMH moved 0.8% on the news, but similar announcements typically move it 1.5%. Room to play.',
      etf: 'SMH',
      details: {
        'priced-in': '~50%',
        confidence: '72%',
      },
      severity: 'info',
    },
    {
      timestamp: new Date(baseTime.getTime() - 85 * 60000).toLocaleTimeString(),
      type: 'approval',
      title: 'Mobile Alert Sent',
      description:
        'Approved to approve: Enter SMH at $285. Stop at $283. Target $290. You have 15 minutes to decide.',
      etf: 'SMH',
      signal: 'BUY',
      details: {
        entry: '$285.00',
        'stop-loss': '$283.00',
        target: '$290.00',
      },
      severity: 'success',
    },
    {
      timestamp: new Date(baseTime.getTime() - 80 * 60000).toLocaleTimeString(),
      type: 'approval',
      title: 'You Approved',
      description: 'Clicked "Approve Trade" on your phone. Execution proceeding now.',
      etf: 'SMH',
      details: {
        action: 'User approved via Telegram',
      },
      severity: 'success',
    },
    {
      timestamp: new Date(baseTime.getTime() - 75 * 60000).toLocaleTimeString(),
      type: 'execution',
      title: 'Paper Order Submitted',
      description: 'Paper buy order sent to broker. SMH at $285 for 10 shares.',
      etf: 'SMH',
      signal: 'BUY',
      details: {
        quantity: '10 shares',
        price: '$285.00',
        total: '$2,850.00',
      },
      severity: 'success',
    },
    {
      timestamp: new Date(baseTime.getTime() - 70 * 60000).toLocaleTimeString(),
      type: 'execution',
      title: 'Order Filled',
      description:
        'Paper order filled. You now hold 10 shares of SMH at $285 average. Waiting for target price or stop-loss hit.',
      etf: 'SMH',
      details: {
        shares: '10',
        'avg-price': '$285.00',
        position: 'Open',
      },
      severity: 'success',
    },
    {
      timestamp: new Date(baseTime.getTime() - 15 * 60000).toLocaleTimeString(),
      type: 'exit',
      title: 'Position Exited',
      description:
        'SMH climbed to $289.50. Closed position to lock in gains. Sold 10 shares at $289.50.',
      etf: 'SMH',
      signal: 'SELL',
      details: {
        shares: '10',
        price: '$289.50',
        profit: '+$45 (+1.6%)',
      },
      severity: 'success',
    },
    {
      timestamp: new Date(baseTime.getTime()).toLocaleTimeString(),
      type: 'reflection',
      title: 'Trade Complete',
      description:
        'Trade closed profitably. Jax learned: News took longer to be fully priced in than expected. Wider window for trend-following next time.',
      etf: 'SMH',
      details: {
        'p&l': '+$45.00',
        return: '+1.58%',
        duration: '65 minutes',
      },
      severity: 'info',
    },
  ];
}
