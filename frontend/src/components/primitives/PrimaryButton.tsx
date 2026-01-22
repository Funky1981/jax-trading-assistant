import { Button, type ButtonProps } from '@mui/material';
import { tokens } from '../../styles/tokens';

export function PrimaryButton({ sx, ...props }: ButtonProps) {
  return (
    <Button
      variant="contained"
      color="primary"
      size="small"
      sx={{
        textTransform: 'none',
        fontWeight: tokens.typography.weight.semibold,
        paddingX: tokens.spacing.lg,
        paddingY: tokens.spacing.sm,
        ...sx,
      }}
      {...props}
    />
  );
}
