import { SelectInput, TextInput } from '../components';
import { HelpHint } from '@/components/ui/help-hint';

export function SettingsPage() {
  return (
    <div className="space-y-4 w-full max-w-lg">
      <h1 className="flex items-center gap-2 text-3xl font-semibold">
        Settings
        <HelpHint text="Customize UI defaults and preferences." />
      </h1>
      <p className="text-sm text-muted-foreground">
        Adjust basic preferences for the UI.
      </p>
      <SelectInput
        value="dark"
        options={[
          { label: 'Dark', value: 'dark' },
          { label: 'Light', value: 'light' },
        ]}
        placeholder="Select theme"
      />
      <TextInput label="Default Order Size" type="number" value={100} />
      <div className="flex items-center gap-2">
        <input type="checkbox" defaultChecked className="h-4 w-4" id="compact" />
        <label htmlFor="compact" className="text-sm">Enable compact layout</label>
      </div>
    </div>
  );
}
