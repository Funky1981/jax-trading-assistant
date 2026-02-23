import { Input } from '@/components/ui/input';
import { useId, type InputHTMLAttributes } from 'react';

interface TextInputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
}

export function TextInput({ label, className, ...props }: TextInputProps) {
  const generatedId = useId();
  const inputId = props.id ?? generatedId;

  return (
    <div className="space-y-2">
      {label && (
        <label htmlFor={inputId} className="text-sm font-medium">
          {label}
        </label>
      )}
      <Input id={inputId} className={className} {...props} />
    </div>
  );
}
