import { TextField, type TextFieldProps } from '@mui/material';

export function TextInput({ sx, ...props }: TextFieldProps) {
  return (
    <TextField
      size="small"
      variant="outlined"
      fullWidth
      sx={sx}
      {...props}
    />
  );
}
