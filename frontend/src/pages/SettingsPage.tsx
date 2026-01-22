import { Stack, Switch, Typography } from '@mui/material';
import { SelectInput, TextInput } from '../components';

export function SettingsPage() {
  return (
    <Stack spacing={2} maxWidth={480}>
      <Typography variant="h4">Settings</Typography>
      <Typography variant="body2" color="text.secondary">
        Customize your layout and preferences.
      </Typography>
      <SelectInput
        label="Theme"
        value="dark"
        options={[
          { label: 'Dark', value: 'dark' },
          { label: 'Light', value: 'light' },
        ]}
      />
      <TextInput label="Default Order Size" type="number" value={100} />
      <Stack direction="row" alignItems="center" spacing={1}>
        <Switch defaultChecked />
        <Typography variant="body2">Enable compact layout</Typography>
      </Stack>
    </Stack>
  );
}
