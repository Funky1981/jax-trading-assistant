import type { Meta, StoryObj } from '@storybook/react';
import { OrderTicket } from './OrderTicket';

const meta: Meta<typeof OrderTicket> = {
  title: 'Trading/OrderTicket',
  component: OrderTicket,
  args: {
    symbol: 'AAPL',
  },
};

export default meta;

type Story = StoryObj<typeof OrderTicket>;

export const Default: Story = {};
