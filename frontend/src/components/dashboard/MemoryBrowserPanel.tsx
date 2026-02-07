import { useState } from 'react';
import { Database, Search } from 'lucide-react';
import { useMemoryBanks, useMemoryEntries, useMemorySearch } from '@/hooks/useMemoryHook';
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
import { formatTime, formatDate } from '@/lib/utils';

interface MemoryBrowserPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

export function MemoryBrowserPanel({ isOpen, onToggle }: MemoryBrowserPanelProps) {
  const { data: banks, isLoading: banksLoading } = useMemoryBanks();
  const [selectedBank, setSelectedBank] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  
  const { data: entries, isLoading: entriesLoading } = useMemoryEntries(selectedBank);
  const { data: searchResults, isLoading: searchLoading } = useMemorySearch(searchQuery);

  const displayEntries = searchQuery.length >= 2 ? searchResults : entries;
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
        {/* Bank Selector */}
        <div className="flex gap-2">
          <Select
            value={selectedBank || ''}
            onValueChange={(v) => setSelectedBank(v || null)}
          >
            <SelectTrigger className="flex-1">
              <SelectValue placeholder="Select memory bank..." />
            </SelectTrigger>
            <SelectContent>
              {banks?.map((bank) => (
                <SelectItem key={bank.id} value={bank.id}>
                  <div className="flex items-center justify-between w-full">
                    <span>{bank.name}</span>
                    <Badge variant="secondary" className="ml-2 text-xs">
                      {bank.entryCount}
                    </Badge>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search memories..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        {/* Bank Info */}
        {selectedBank && banks && (
          <div className="rounded-md border border-border bg-muted/30 p-3">
            {(() => {
              const bank = banks.find((b) => b.id === selectedBank);
              if (!bank) return null;
              return (
                <>
                  <p className="font-semibold">{bank.name}</p>
                  <p className="text-xs text-muted-foreground">{bank.description}</p>
                  <div className="flex gap-4 mt-2 text-xs">
                    <span>{bank.entryCount} entries</span>
                    <span>Updated {formatDate(bank.lastUpdated)}</span>
                  </div>
                </>
              );
            })()}
          </div>
        )}

        {/* Entries */}
        <div className="space-y-2">
          {isLoadingEntries ? (
            <p className="text-sm text-muted-foreground text-center py-4">
              Loading...
            </p>
          ) : displayEntries && displayEntries.length > 0 ? (
            displayEntries.map((entry) => (
              <div
                key={entry.id}
                className="rounded-md border border-border p-3 text-sm"
              >
                <p>{entry.content}</p>
                <div className="flex items-center justify-between mt-2">
                  <div className="flex gap-2">
                    {Object.entries(entry.metadata).slice(0, 3).map(([key, value]) => (
                      <Badge key={key} variant="outline" className="text-xs">
                        {key}: {String(value)}
                      </Badge>
                    ))}
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {formatTime(entry.createdAt)}
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
