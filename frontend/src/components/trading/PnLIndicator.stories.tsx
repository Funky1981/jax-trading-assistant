import type { Meta, StoryObj } from '@storybook/react';
import { PnLIndicator } from './PnLIndicator';

const meta: Meta<typeof PnLIndicator> = {
  title: 'Trading/PnLIndicator',
  component: PnLIndicator,
  args: {
    value: 12.34,
  },
};

export default meta;

type Story = StoryObj<typeof PnLIndicator>;

export const Positive: Story = {};
export const Negative: Story = { args: { value: -8.21 } };
