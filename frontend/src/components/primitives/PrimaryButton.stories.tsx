import type { Meta, StoryObj } from '@storybook/react';
import { PrimaryButton } from './PrimaryButton';

const meta: Meta<typeof PrimaryButton> = {
  title: 'Primitives/PrimaryButton',
  component: PrimaryButton,
  args: {
    children: 'Place Order',
  },
};

export default meta;

type Story = StoryObj<typeof PrimaryButton>;

export const Default: Story = {};
