import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Link } from 'react-router-dom';

type GuideItem = {
  title: string;
  description: string;
};

type GuideAction = {
  label: string;
  to: string;
  variant?: 'default' | 'secondary' | 'outline';
};

type ModuleGuideLayoutProps = {
  title: string;
  subtitle: string;
  isBeginner: boolean;
  checklist: GuideItem[];
  glossary: GuideItem[];
  actions: GuideAction[];
};

export function ModuleGuideLayout({
  title,
  subtitle,
  isBeginner,
  checklist,
  glossary,
  actions,
}: ModuleGuideLayoutProps) {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-semibold">{title}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{subtitle}</p>
        <div className="mt-3">
          <Badge variant={isBeginner ? 'default' : 'secondary'}>
            {isBeginner ? 'Beginner mode active' : 'Advanced mode active'}
          </Badge>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Quick Start Checklist</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
          {checklist.map((item, idx) => (
            <div key={`${item.title}-${idx}`}>
              <p className="font-semibold text-foreground">{`${idx + 1}. ${item.title}`}</p>
              <p>{item.description}</p>
            </div>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Glossary</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
          {glossary.map((item, idx) => (
            <div key={`${item.title}-${idx}`}>
              <p className="font-semibold text-foreground">{item.title}</p>
              <p>{item.description}</p>
            </div>
          ))}
        </CardContent>
      </Card>

      <div className="flex flex-wrap gap-2">
        {actions.map((action, idx) => (
          <Button key={`${action.label}-${idx}`} asChild variant={action.variant ?? 'default'}>
            <Link to={action.to}>{action.label}</Link>
          </Button>
        ))}
      </div>
    </div>
  );
}