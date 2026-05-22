import type { Meta, StoryObj } from '@storybook/react';
import { tokens } from './tokens';

function TokenSwatch({ name, className }: { name: string; className: string }) {
  return (
    <div className="rounded-md border border-border bg-card p-3">
      <div className={`mb-3 h-12 rounded border border-border ${className}`} />
      <p className="font-mono text-xs text-muted-foreground">{name}</p>
    </div>
  );
}

function TokenShowcase() {
  return (
    <div className="max-w-5xl space-y-8 bg-background p-6 text-foreground">
      <section className="space-y-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-primary">Design tokens</p>
          <h1 className="text-2xl font-semibold">Runtime token source</h1>
          <p className="text-sm text-muted-foreground">
            CSS variables in <code>frontend/src/index.css</code> are the runtime source. Tailwind, TypeScript tokens,
            MUI theme, and Storybook consume these semantic variables.
          </p>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <TokenSwatch name="background" className="bg-background" />
          <TokenSwatch name="card" className="bg-card" />
          <TokenSwatch name="primary" className="bg-primary" />
          <TokenSwatch name="accent" className="bg-accent" />
          <TokenSwatch name="success" className="bg-success" />
          <TokenSwatch name="warning" className="bg-warning" />
          <TokenSwatch name="destructive" className="bg-destructive" />
          <TokenSwatch name="border" className="bg-border" />
        </div>
      </section>

      <section className="grid gap-6 md:grid-cols-2">
        <div className="space-y-3 rounded-md border border-border bg-card p-4">
          <h2 className="text-lg font-semibold">Typography</h2>
          <p className="text-xs text-muted-foreground">Extra small label</p>
          <p className="text-sm">Small operational body copy</p>
          <p className="text-base">Base interface text</p>
          <p className="text-xl font-semibold">Section heading</p>
          <p className="font-mono text-sm text-muted-foreground">Monospace identifiers</p>
        </div>
        <div className="space-y-3 rounded-md border border-border bg-card p-4">
          <h2 className="text-lg font-semibold">Shape and elevation</h2>
          <div className="rounded-sm border border-border p-3">Small radius</div>
          <div className="rounded-md border border-border p-3 shadow-sm">Medium radius, small elevation</div>
          <div className="rounded-lg border border-border p-3 shadow-md">Large radius, medium elevation</div>
          <p className="text-xs text-muted-foreground">Primary radius: {tokens.radius.md}px</p>
        </div>
      </section>
    </div>
  );
}

const meta: Meta<typeof TokenShowcase> = {
  title: 'Design System/Tokens',
  component: TokenShowcase,
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;

type Story = StoryObj<typeof TokenShowcase>;

export const Overview: Story = {};
