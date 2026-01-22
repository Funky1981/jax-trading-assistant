import { MenuItem, TextField, type TextFieldProps } from '@mui/material';

export interface SelectOption {
  label: string;
  value: string;
}

interface SelectInputProps extends Omit<TextFieldProps, 'select' | 'children'> {
  options: SelectOption[];
}

export function SelectInput({ options, sx, ...props }: SelectInputProps) {
  return (
    <TextField select size="small" variant="outlined" fullWidth sx={sx} {...props}>
      {options.map((option) => (
        <MenuItem key={option.value} value={option.value}>
          {option.label}
        </MenuItem>
      ))}
    </TextField>
  );
}
