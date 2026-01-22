import type { Meta, StoryObj } from '@storybook/react';
import { PositionCard } from './PositionCard';

const meta: Meta<typeof PositionCard> = {
  title: 'Trading/PositionCard',
  component: PositionCard,
  args: {
    position: {
      symbol: 'AAPL',
      quantity: 250,
      avgPrice: 231.12,
      marketPrice: 249.42,
    },
  },
};

export default meta;

type Story = StoryObj<typeof PositionCard>;

export const Default: Story = {};
