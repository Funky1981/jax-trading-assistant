import { Stack, Typography } from '@mui/material';
import { SelectInput } from '../components';
import { DashboardGrid } from '../features/dashboard/DashboardGrid';
import { presetOptions, useDashboardLayout } from '../features/dashboard/useDashboardLayout';
import { useDomain } from '../domain/store';
import { selectOrders, selectPositions, selectTicks } from '../domain/selectors';

export function DashboardPage() {
  const { layout, applyPreset } = useDashboardLayout();
  const { state, actions } = useDomain();
  const positions = selectPositions(state);
  const orders = selectOrders(state).sort((a, b) => b.createdAt - a.createdAt);
  const ticks = selectTicks(state);

  return (
    <Stack spacing={3}>
      <Stack spacing={1}>
        <Typography variant="overline" sx={{ letterSpacing: 2 }}>
          MARKET OVERVIEW
        </Typography>
        <Typography variant="h4" sx={{ fontWeight: 600 }}>
          Dashboard
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Drag-and-drop layout support will arrive next. Choose a preset to start.
        </Typography>
      </Stack>

      <SelectInput
        label="Preset"
        value={layout.presetId}
        options={presetOptions}
        onChange={(event) => applyPreset(event.target.value)}
        sx={{ maxWidth: 240 }}
      />

      <DashboardGrid
        layout={layout}
        positions={positions}
        orders={orders}
        ticks={ticks}
        riskLimits={state.riskLimits}
        onOrderSubmit={actions.placeOrder}
      />
    </Stack>
  );
}
