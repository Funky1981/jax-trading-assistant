import { FormEvent, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { approvalsService } from '@/data/approvals-service';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';

export function MobileApprovalHarnessPage() {
  const [token, setToken] = useState('');
  const [actor, setActor] = useState('operator-ui');
  const [reason, setReason] = useState('');
  const [guardrailHash, setGuardrailHash] = useState('');
  const [action, setAction] = useState<'approved' | 'rejected'>('approved');

  const decisionMutation = useMutation({
    mutationFn: () =>
      approvalsService.submitTelegramDecision({
        token: token.trim(),
        action,
        actor: actor.trim() || undefined,
        reason: reason.trim() || undefined,
        guardrailHash: guardrailHash.trim() || undefined,
        runtimeMode: 'paper',
      }),
  });

  const onSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!token.trim()) {
      return;
    }
    decisionMutation.mutate();
  };

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">MOBILE APPROVAL</p>
        <h1 className="text-2xl font-bold md:text-3xl">Telegram Approval Harness</h1>
        <p className="mt-1 text-muted-foreground">
          Submit one-time mobile approval tokens through the same webhook contract used by operator notifications.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Submit Mobile Decision</CardTitle>
          <CardDescription>Paper-mode only. Use generated one-time token and choose approve or reject.</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="space-y-2">
              <label htmlFor="mobile-token" className="text-sm font-medium">
                One-time Token
              </label>
              <Input
                id="mobile-token"
                placeholder="Paste token"
                value={token}
                onChange={(e) => setToken(e.target.value)}
              />
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <label htmlFor="mobile-actor" className="text-sm font-medium">
                  Actor
                </label>
                <Input
                  id="mobile-actor"
                  placeholder="operator-ui"
                  value={actor}
                  onChange={(e) => setActor(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="mobile-guardrail" className="text-sm font-medium">
                  Guardrail Hash (optional)
                </label>
                <Input
                  id="mobile-guardrail"
                  placeholder="guardrails:v1:..."
                  value={guardrailHash}
                  onChange={(e) => setGuardrailHash(e.target.value)}
                />
              </div>
            </div>

            <div className="space-y-2">
              <label htmlFor="mobile-reason" className="text-sm font-medium">
                Reason (required for reject)
              </label>
              <textarea
                id="mobile-reason"
                className="min-h-[84px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                placeholder="Optional for approve, recommended for reject"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
              />
            </div>

            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant={action === 'approved' ? 'default' : 'outline'}
                onClick={() => setAction('approved')}
              >
                Approve
              </Button>
              <Button
                type="button"
                variant={action === 'rejected' ? 'destructive' : 'outline'}
                onClick={() => setAction('rejected')}
              >
                Reject
              </Button>
              <Button type="submit" disabled={decisionMutation.isPending || token.trim() === ''}>
                Submit Mobile Decision
              </Button>
              <Button asChild variant="outline">
                <Link to="/testing">Back to Testing</Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {decisionMutation.isError && (
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive">Webhook Error</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-destructive">{(decisionMutation.error as Error).message}</p>
          </CardContent>
        </Card>
      )}

      {decisionMutation.data && (
        <Card>
          <CardHeader>
            <CardTitle>Decision Accepted</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 text-sm">
            <p>
              <strong>Approval ID:</strong> {decisionMutation.data.approvalId}
            </p>
            <p>
              <strong>Candidate ID:</strong> {decisionMutation.data.candidateId}
            </p>
            <p>
              <strong>Decision:</strong> {decisionMutation.data.decision}
            </p>
            <p>
              <strong>Runtime:</strong> {decisionMutation.data.runtimeMode}
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
