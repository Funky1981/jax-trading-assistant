import type { Meta, StoryObj } from '@storybook/react';
import { Settings } from 'lucide-react';
import { Badge } from './badge';
import { Button } from './button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from './card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './dialog';
import { HelpHint } from './help-hint';
import { Input } from './input';
import { Progress } from './progress';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './select';
import { Skeleton } from './skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from './table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './tabs';

function CoreUiShowcase() {
  return (
    <div className="max-w-5xl space-y-6 bg-background p-6 text-foreground">
      <Card>
        <CardHeader>
          <CardTitle>Actions and status</CardTitle>
          <CardDescription>Button and badge variants for operational trading workflows.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-2">
            <Button>Primary</Button>
            <Button variant="secondary">Secondary</Button>
            <Button variant="outline">Outline</Button>
            <Button variant="success">Approve</Button>
            <Button variant="destructive">Reject</Button>
            <Button variant="ghost" size="icon" aria-label="Settings">
              <Settings className="h-4 w-4" />
            </Button>
            <Button disabled>Disabled</Button>
            <Button size="sm">Compact</Button>
          </div>
          <div className="flex flex-wrap gap-2">
            <Badge>Default</Badge>
            <Badge variant="secondary">Secondary</Badge>
            <Badge variant="success">Success</Badge>
            <Badge variant="warning">Warning</Badge>
            <Badge variant="destructive">Destructive</Badge>
            <Badge variant="outline">Outline</Badge>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Form controls</CardTitle>
          <CardDescription>Shared inputs, selects, tabs, progress, and inline help.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-3">
            <Input placeholder="Symbol, for example SPY" />
            <Input placeholder="Disabled input" disabled />
            <Select defaultValue="boost">
              <SelectTrigger>
                <SelectValue placeholder="Sentiment mode" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="off">Disabled</SelectItem>
                <SelectItem value="filter">Filter</SelectItem>
                <SelectItem value="boost">Boost</SelectItem>
              </SelectContent>
            </Select>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>Scanner confidence</span>
              <HelpHint text="Minimum confidence before an opportunity is shown." />
            </div>
            <Progress value={62} />
          </div>
          <Tabs defaultValue="summary" className="space-y-3">
            <TabsList>
              <TabsTrigger value="summary">Summary</TabsTrigger>
              <TabsTrigger value="evidence">Evidence</TabsTrigger>
              <TabsTrigger value="policy" disabled>
                Policy
              </TabsTrigger>
            </TabsList>
            <TabsContent value="summary" className="rounded-md border border-border p-3">
              AI opportunity summary.
            </TabsContent>
            <TabsContent value="evidence" className="rounded-md border border-border p-3">
              Sentiment, news, chart, and policy evidence.
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Data surfaces</CardTitle>
          <CardDescription>Tables, skeletons, and modal detail entry points.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Symbol</TableHead>
                <TableHead>Route</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>SPY</TableCell>
                <TableCell>Approval required</TableCell>
                <TableCell>
                  <Badge variant="warning">Review</Badge>
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell>MSFT</TableCell>
                <TableCell>Manual allowed</TableCell>
                <TableCell>
                  <Badge variant="success">Ready</Badge>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
          <div className="grid gap-2 md:grid-cols-3">
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
          </div>
          <Dialog>
            <DialogTrigger asChild>
              <Button variant="outline">Open modal</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Opportunity detail</DialogTitle>
                <DialogDescription>
                  Dialog styling uses the same tokens as the app and Storybook preview.
                </DialogDescription>
              </DialogHeader>
            </DialogContent>
          </Dialog>
        </CardContent>
      </Card>
    </div>
  );
}

const meta: Meta<typeof CoreUiShowcase> = {
  title: 'Design System/Core UI',
  component: CoreUiShowcase,
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;

type Story = StoryObj<typeof CoreUiShowcase>;

export const AllStates: Story = {};
