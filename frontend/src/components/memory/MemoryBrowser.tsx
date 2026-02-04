import { useState } from 'react';
import { Search, HardDrive } from 'lucide-react';
import { useMemorySearch, useMemoryBanks } from '../../hooks/useMemory';
import type { MemoryItem } from '../../data/types';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

export function MemoryBrowser() {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedBank, setSelectedBank] = useState<string>('all');
  const { data: banks, isLoading: banksLoading } = useMemoryBanks();
  const { data: results, isLoading: searchLoading, error: searchError } = useMemorySearch(
    searchQuery,
    selectedBank === 'all' ? undefined : selectedBank,
    20,
  );

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
  };

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="space-y-6">
          <div className="flex items-center gap-2">
            <HardDrive className="h-5 w-5 text-primary" />
            <h2 className="text-lg font-semibold">
              Memory Browser
            </h2>
          </div>

          <div className="flex gap-4">
            <Select
              value={selectedBank}
              onValueChange={setSelectedBank}
              disabled={banksLoading}
            >
              <SelectTrigger className="w-[200px]">
                <SelectValue placeholder="Memory Bank" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Banks</SelectItem>
                {banks?.map((bank: string) => (
                  <SelectItem key={bank} value={bank}>
                    {bank}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <form onSubmit={handleSearch} className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Search memories..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9"
                />
              </div>
            </form>
          </div>

          {searchLoading && (
            <div className="py-16 text-center">
              <div className="mx-auto h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
              <p className="text-sm text-muted-foreground mt-4">
                Searching memories...
              </p>
            </div>
          )}

          {searchError && (
            <div className="rounded-md border border-destructive bg-destructive/10 p-4">
              <p className="text-sm text-destructive">
                Failed to search memories. Please try again.
              </p>
            </div>
          )}

          {results && results.length > 0 && (
            <div className="space-y-3">
              {results.map((item: MemoryItem, idx: number) => (
                <div
                  key={idx}
                  className="border-2 border-border rounded-md p-4 bg-muted transition-all hover:bg-accent hover:border-primary"
                >
                  <div className="space-y-2">
                    <div className="flex items-center gap-2">
                      <HardDrive className="h-4 w-4 text-primary" />
                      <h3 className="text-sm font-medium">{item.key}</h3>
                      <Badge>{item.bank}</Badge>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">
                        {item.summary || 'No summary'}
                      </p>
                      {item.tags && item.tags.length > 0 && (
                        <div className="flex gap-1 mt-2">
                          {item.tags.map((tag: string) => (
                            <Badge key={tag} variant="outline">
                              {tag}
                            </Badge>
                          ))}
                        </div>
                      )}
                      <p className="text-xs text-muted-foreground mt-2">
                        {new Date(item.timestamp).toLocaleString()}
                      </p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {results && results.length === 0 && searchQuery && (
            <div className="py-24 text-center">
              <HardDrive className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-base text-muted-foreground mb-2">
                No memories found
              </h3>
              <p className="text-sm text-muted-foreground/70">
                Try adjusting your search query
              </p>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
