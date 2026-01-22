import type { Meta, StoryObj } from '@storybook/react';
import { SelectInput } from './SelectInput';

const meta: Meta<typeof SelectInput> = {
  title: 'Primitives/SelectInput',
  component: SelectInput,
  args: {
    label: 'Side',
    value: 'buy',
    options: [
      { label: 'Buy', value: 'buy' },
      { label: 'Sell', value: 'sell' },
    ],
  },
};

export default meta;

type Story = StoryObj<typeof SelectInput>;

export const Default: Story = {};
