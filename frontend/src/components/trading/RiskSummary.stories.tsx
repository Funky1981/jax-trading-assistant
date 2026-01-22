import type { Meta, StoryObj } from '@storybook/react';
import { RiskSummary } from './RiskSummary';

const meta: Meta<typeof RiskSummary> = {
  title: 'Trading/RiskSummary',
  component: RiskSummary,
  args: {
    exposure: 1_240_000,
    pnl: -12_500,
    limits: {
      maxPositionValue: 5_000_000,
      maxDailyLoss: 100_000,
    },
  },
};

export default meta;

type Story = StoryObj<typeof RiskSummary>;

export const Default: Story = {};
