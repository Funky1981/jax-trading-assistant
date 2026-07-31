import { useEffect, useState } from 'react';
import { Database, Search } from 'lucide-react';
import { useMemoryBanks, useMemoryRecall, useMemorySearch } from '@/hooks/useMemory';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { formatTime } from '@/lib/utils';

interface MemoryBrowserPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

export function MemoryBrowserPanel({ isOpen, onToggle }: MemoryBrowserPanelProps) {
  const { data: banks, isLoading: banksLoading, isError: banksUnavailable } = useMemoryBanks();
  const [selectedBank, setSelectedBank] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  const { data: recallData, isLoading: entriesLoading } = useMemoryRecall(
    selectedBank ?? '',
    {},
  );
  const { data: searchItems, isLoading: searchLoading } = useMemorySearch(searchQuery, selectedBank ?? '');

  useEffect(() => {
    if (!selectedBank && banks && banks.length > 0) {
      setSelectedBank(banks[0]);
    }
  }, [banks, selectedBank]);

  const displayItems = searchQuery.length >= 2 ? searchItems : recallData?.items;
  const isLoadingEntries = searchQuery.length >= 2 ? searchLoading : entriesLoading;

  const summary = banks ? (
    <span>{banks.length} memory banks</span>
  ) : null;

  return (
    <CollapsiblePanel
      title="Memory Browser"
      icon={<Database className="h-4 w-4" />}
      summary={summary}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={banksLoading}
    >
      <div className="space-y-4">
        {banksUnavailable && (
          <p role="alert" className="text-sm text-destructive">
            Memory-bank diagnostics are unavailable.
          </p>
        )}
        {/* Bank Selector */}
        <div className="flex gap-2">
          <Select
            value={selectedBank || ''}
            onValueChange={(v) => setSelectedBank(v || null)}
          >
            <SelectTrigger className="flex-1" aria-label="Select memory bank">
              <SelectValue placeholder="Select memory bank..." />
            </SelectTrigger>
            <SelectContent>
              {banks?.map((bank) => (
                <SelectItem key={bank} value={bank}>
                  <span>{bank}</span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="memory-browser-search"
            name="memorySearch"
            aria-label="Search memories"
            placeholder="Search memories..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        {/* Entries */}
        <div className="space-y-2">
          {isLoadingEntries ? (
            <p className="text-sm text-muted-foreground text-center py-4">
              Loading...
            </p>
          ) : displayItems && displayItems.length > 0 ? (
            displayItems.map((item, idx) => (
              <div
                key={item.id ?? idx}
                className="rounded-md border border-border p-3 text-sm"
              >
                <p>{item.summary}</p>
                <div className="flex items-center justify-between mt-2">
                  <div className="flex gap-2 flex-wrap">
                    {item.type && (
                      <Badge variant="outline" className="text-xs">
                        {item.type}
                      </Badge>
                    )}
                    {item.symbol && (
                      <Badge variant="secondary" className="text-xs">
                        {item.symbol}
                      </Badge>
                    )}
                    {item.tags?.slice(0, 3).map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {formatTime(new Date(item.ts).getTime())}
                  </span>
                </div>
              </div>
            ))
          ) : (
            <p className="text-sm text-muted-foreground text-center py-4">
              {selectedBank || searchQuery
                ? 'No entries found.'
                : 'Select a memory bank to browse entries.'}
            </p>
          )}
        </div>
      </div>
    </CollapsiblePanel>
  );
}
