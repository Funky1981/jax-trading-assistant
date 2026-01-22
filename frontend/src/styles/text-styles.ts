import { TypographyProps } from '@mui/material';
import { tokens } from './tokens';

export const textStyles: Record<'muted' | 'accent', TypographyProps['sx']> = {
  muted: { color: tokens.colors.textMuted },
  accent: { color: tokens.colors.accent },
};
