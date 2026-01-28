import { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Stack,
  Chip,
  List,
  ListItem,
  ListItemText,
  CircularProgress,
  Alert,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import { Search, Storage } from '@mui/icons-material';
import { useMemorySearch, useMemoryBanks } from '../../hooks/useMemory';
import type { MemoryItem } from '../../data/types';

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
      <CardContent>
        <Stack spacing={2}>
          <Typography variant="h6" gutterBottom>
            Memory Browser
          </Typography>

          <Stack direction="row" spacing={2}>
            <FormControl size="small" sx={{ minWidth: 200 }}>
              <InputLabel>Memory Bank</InputLabel>
              <Select
                value={selectedBank}
                label="Memory Bank"
                onChange={(e) => setSelectedBank(e.target.value)}
                disabled={banksLoading}
              >
                <MenuItem value="all">All Banks</MenuItem>
                {banks?.map((bank: string) => (
                  <MenuItem key={bank} value={bank}>
                    {bank}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <Box component="form" onSubmit={handleSearch} sx={{ flex: 1 }}>
              <TextField
                fullWidth
                size="small"
                placeholder="Search memories..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                InputProps={{
                  startAdornment: <Search sx={{ mr: 1, color: 'text.secondary' }} />,
                }}
              />
            </Box>
          </Stack>

          {searchLoading && (
            <Stack alignItems="center" py={4}>
              <CircularProgress />
            </Stack>
          )}

          {searchError && (
            <Alert severity="error">Failed to search memories</Alert>
          )}

          {results && results.length > 0 && (
            <List>
              {results.map((item: MemoryItem, idx: number) => (
                <ListItem
                  key={idx}
                  sx={{
                    border: 1,
                    borderColor: 'divider',
                    borderRadius: 1,
                    mb: 1,
                  }}
                >
                  <ListItemText
                    primary={
                      <Stack direction="row" spacing={1} alignItems="center">
                        <Storage fontSize="small" color="primary" />
                        <Typography variant="subtitle2">{item.key}</Typography>
                        <Chip label={item.bank} size="small" />
                      </Stack>
                    }
                    secondary={
                      <Box>
                        <Typography variant="body2" color="text.secondary">
                          {item.summary || 'No summary'}
                        </Typography>
                        {item.tags && item.tags.length > 0 && (
                          <Stack direction="row" spacing={0.5} mt={1}>
                            {item.tags.map((tag: string) => (
                              <Chip key={tag} label={tag} size="small" variant="outlined" />
                            ))}
                          </Stack>
                        )}
                        <Typography variant="caption" color="text.secondary" display="block" mt={1}>
                          {new Date(item.timestamp).toLocaleString()}
                        </Typography>
                      </Box>
                    }
                  />
                </ListItem>
              ))}
            </List>
          )}

          {results && results.length === 0 && searchQuery && (
            <Typography color="text.secondary" textAlign="center" py={4}>
              No memories found
            </Typography>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}
