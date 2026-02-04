import { Button, type ButtonProps } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export function PrimaryButton({ className, ...props }: ButtonProps) {
  return (
    <Button
      className={cn('font-semibold px-4 py-2', className)}
      {...props}
    />
  );
}
