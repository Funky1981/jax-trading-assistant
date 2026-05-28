import { RadioTower, SlidersHorizontal } from 'lucide-react';
import type { ScannerSentimentMode, ScannerSettings, ScannerSourceTrustMode } from '@/data/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

const sentimentModeLabels: Record<ScannerSentimentMode, string> = {
  filter: 'Filter',
  rank_boost: 'Rank boost',
  required_feature: 'Required feature',
};

const sourceTrustLabels: Record<ScannerSourceTrustMode, string> = {
  equal: 'Equal source weighting',
  trust_weighted: 'Trust-weighted sources',
};

function percentLabel(value: number) {
  return `${Math.round(value * 100)}%`;
}

function Field({ id, label, value, help }: { id: string; label: string; value: string; help?: string }) {
  const helpId = help ? `${id}-help` : undefined;

  return (
    <div className="min-w-0 space-y-1">
      <label className="text-sm font-medium text-foreground" htmlFor={id}>
        {label}
      </label>
      <input
        aria-describedby={helpId}
        className="h-9 w-full min-w-0 rounded-md border border-input bg-muted px-3 text-sm text-muted-foreground"
        disabled
        id={id}
        readOnly
        value={value}
      />
      {help && (
        <p className="text-xs text-muted-foreground" id={helpId}>
          {help}
        </p>
      )}
    </div>
  );
}

export function ScannerSettingsCard({
  settings,
  onToggleScanner,
  isSaving,
}: {
  settings: ScannerSettings;
  onToggleScanner?: () => void;
  isSaving?: boolean;
}) {
  const sentiment = settings.sentiment;
  const controlsHelp = settings.connected
    ? 'Scanner settings are persisted and connected to the AI scanner API.'
    : 'Scanner settings are currently unavailable.';
  const sentimentHelp = sentiment.connected
    ? 'Sentiment settings are connected to scanner filtering.'
    : 'Sentiment settings are currently unavailable.';

  return (
    <Card>
      <CardHeader className="gap-3 md:flex-row md:items-start md:justify-between">
        <div className="space-y-1">
          <CardTitle className="flex items-center gap-2 text-xl">
            <SlidersHorizontal className="h-5 w-5" />
            Scanner settings
          </CardTitle>
          <CardDescription>What Jax is watching before an Opportunity reaches the queue.</CardDescription>
        </div>
        <div className="flex flex-wrap gap-2">
          <Badge variant={settings.enabled ? 'default' : 'secondary'}>{settings.enabled ? 'Watching' : 'Paused'}</Badge>
          <Badge variant={settings.connected ? 'default' : 'outline'}>{settings.connected ? 'Connected' : 'Offline'}</Badge>
          {onToggleScanner && (
            <Button disabled={!!isSaving} onClick={onToggleScanner} size="sm" type="button" variant="outline">
              {isSaving ? 'Saving...' : settings.enabled ? 'Pause scanner' : 'Resume scanner'}
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        <section className="space-y-3" aria-label="Scanner configuration">
          <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <RadioTower className="h-4 w-4" />
            Scan setup
          </div>
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            <Field id="scanner-asset-scope" label="Asset scope" value={settings.assetScope} />
            <Field id="scanner-symbols" label="Symbols" value={settings.symbols.join(', ')} />
            <Field id="scanner-universe" label="Universe" value={settings.universePreset} />
            <Field id="scanner-interval" label="Scan interval" value={`${settings.intervalSeconds} seconds`} />
            <Field id="scanner-confidence" label="Minimum confidence" value={percentLabel(settings.minimumConfidence)} />
          </div>
          <p className="rounded-md border border-border bg-muted px-3 py-2 text-sm text-muted-foreground">{controlsHelp}</p>
        </section>

        <section className="space-y-3" aria-label="Sentiment configuration">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-sm font-semibold text-foreground">Sentiment controls</p>
            <Badge variant={sentiment.enabled ? 'secondary' : 'outline'}>{sentiment.enabled ? 'Included' : 'Off'}</Badge>
            {!sentiment.supported && <Badge variant="outline">Unsupported</Badge>}
          </div>
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            <Field id="sentiment-source-scope" label="Sentiment source scope" value={sentiment.sourceScope} />
            <Field id="sentiment-window" label="Sentiment time window" value={sentiment.timeWindow} />
            <Field id="sentiment-threshold" label="Minimum sentiment threshold" value={sentiment.minimumThresholdLabel} />
            <Field id="sentiment-source-count" label="Minimum source count" value={`${sentiment.minimumSourceCount} sources`} />
            <Field id="sentiment-trust" label="Source trust weighting" value={sourceTrustLabels[sentiment.sourceTrustMode]} />
            <Field id="sentiment-mode" label="Sentiment mode" value={sentimentModeLabels[sentiment.mode]} />
          </div>
          <p className="rounded-md border border-border bg-muted px-3 py-2 text-sm text-muted-foreground">
            {sentiment.unsupportedReason ?? sentimentHelp}
          </p>
        </section>
      </CardContent>
    </Card>
  );
}
