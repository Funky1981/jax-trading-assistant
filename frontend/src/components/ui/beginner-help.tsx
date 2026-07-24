import type { ReactNode } from 'react';
import { beginnerGlossary, type BeginnerGlossaryTerm } from '@/data/beginner-glossary';
import { cn } from '@/lib/utils';

export function PageIntro({
  eyebrow,
  title,
  description,
  children,
}: {
  eyebrow?: string;
  title: string;
  description: string;
  children?: ReactNode;
}) {
  return (
    <header className="space-y-2">
      {eyebrow && (
        <p className="text-xs font-semibold uppercase tracking-widest text-primary">{eyebrow}</p>
      )}
      <h1 className="text-2xl font-bold md:text-3xl">{title}</h1>
      <p className="max-w-3xl text-muted-foreground">{description}</p>
      {children}
    </header>
  );
}

export function SafetyBanner({ safe, loading = false }: { safe: boolean; loading?: boolean }) {
  const warning = loading || !safe;
  return (
    <section
      role="status"
      aria-live="polite"
      className={cn(
        'rounded-lg border p-4 sm:p-5',
        warning ? 'border-destructive/60 bg-destructive/5' : 'border-success/60 bg-success/5',
      )}
    >
      <h2 className="font-semibold">
        {loading
          ? 'Checking system safety'
          : safe
            ? 'Paper-safe mode is on'
            : 'Safety needs attention'}
      </h2>
      <p className="mt-1 text-sm text-muted-foreground">
        {loading
          ? 'Jax is loading the current runtime state. Do not assume it is safe yet.'
          : safe
            ? 'Paper-safe mode is on. Jax can collect and review evidence, but it cannot place live orders.'
            : 'Jax cannot confirm a paper-safe state. Open System Safety before relying on this information.'}
      </p>
    </section>
  );
}

export function TechnicalDetailsDisclosure({ children }: { children: ReactNode }) {
  return (
    <details className="rounded-lg border border-border bg-card">
      <summary className="cursor-pointer px-4 py-3 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
        Technical detail
      </summary>
      <div className="border-t border-border p-4 text-sm text-muted-foreground">{children}</div>
    </details>
  );
}

export function GlossaryTerm({ term }: { term: BeginnerGlossaryTerm }) {
  return (
    <span>
      <strong>{term}</strong>: {beginnerGlossary[term]}
    </span>
  );
}
