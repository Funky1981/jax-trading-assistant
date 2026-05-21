import { Link } from 'react-router-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

export function TradingModulesPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-semibold">Trading Modules</h1>
        <p className="text-sm text-muted-foreground mt-2">
          Choose a module. ETF and Equity Alpha trading are isolated into separate page spaces with their own workflows and guidance.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>ETF Module</CardTitle>
            <CardDescription>
              ETF-only execution space for approved ETF symbols with policy gating, approvals, evidence review, and beginner guidance pages
              for universe, strategies, timeline, and trading modes.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              <Button asChild>
                <Link to="/etf/trading">Open ETF Module</Link>
              </Button>
              <Button asChild variant="outline">
                <Link to="/etf/guide">Open ETF Guide</Link>
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Equity Alpha Module</CardTitle>
            <CardDescription>
              Equity alpha strategy space for non-ETF workflows, including opening-range, earnings drift, and event-driven strategy execution,
              with beginner guidance pages matching the ETF module experience.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              <Button asChild variant="secondary">
                <Link to="/equity-alpha/trading">Open Equity Alpha Module</Link>
              </Button>
              <Button asChild variant="outline">
                <Link to="/equity-alpha/guide">Open Equity Alpha Guide</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
