import type { Meta, StoryObj } from '@storybook/react';
import { TextInput } from './TextInput';

const meta: Meta<typeof TextInput> = {
  title: 'Primitives/TextInput',
  component: TextInput,
  args: {
    label: 'Quantity',
    value: 100,
    type: 'number',
  },
};

export default meta;

type Story = StoryObj<typeof TextInput>;

export const Default: Story = {};
