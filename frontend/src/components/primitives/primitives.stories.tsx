import type { Meta, StoryObj } from '@storybook/react';
import { Bell, Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { EmptyState } from './EmptyState';
import { LoadingCard } from './LoadingCard';
import { PrimaryButton } from './PrimaryButton';
import { SelectInput } from './SelectInput';
import { StatusCard } from './StatusCard';
import { TextInput } from './TextInput';

function PrimitiveShowcase() {
  return (
    <div className="max-w-5xl space-y-6 bg-background p-6 text-foreground">
      <section className="grid gap-4 md:grid-cols-2">
        <StatusCard
          status="success"
          title="Scanner online"
          description="Last scan completed 20 seconds ago."
        />
        <StatusCard
          status="warning"
          title="Sparse sentiment"
          description="Only two trusted sources are available for this symbol."
        />
        <StatusCard
          status="error"
          title="Notification channel down"
          description="Desktop notifications need attention."
          compact
        />
        <StatusCard
          status="info"
          title="Research setup saved"
          statusLabel="Saved"
          description="The configuration is ready for paper testing."
          compact
        />
      </section>

      <section className="grid gap-4 md:grid-cols-2">
        <EmptyState
          icon={<Bell className="h-10 w-10" />}
          title="No alerts yet"
          description="Sentiment alerts will appear here when configured rules are met."
          action={<Button variant="outline">Configure alerts</Button>}
        />
        <EmptyState
          icon={<Search className="h-8 w-8" />}
          title="No matching opportunities"
          description="Try a broader universe or lower confidence threshold."
          compact
        />
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        <LoadingCard />
        <LoadingCard variant="metric" />
        <LoadingCard variant="table" rows={2} />
      </section>

      <section className="grid gap-4 rounded-md border border-border bg-card p-4 md:grid-cols-3">
        <TextInput label="Symbol" placeholder="SPY" />
        <SelectInput
          placeholder="Sentiment mode"
          options={[
            { label: 'Disabled', value: 'off' },
            { label: 'Filter', value: 'filter' },
            { label: 'Boost', value: 'boost' },
          ]}
          value="boost"
        />
        <div className="flex items-end">
          <PrimaryButton className="w-full">Run scan</PrimaryButton>
        </div>
      </section>
    </div>
  );
}

const meta: Meta<typeof PrimitiveShowcase> = {
  title: 'Design System/Primitives',
  component: PrimitiveShowcase,
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;

type Story = StoryObj<typeof PrimitiveShowcase>;

export const OperationalStates: Story = {};
