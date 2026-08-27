import { forwardRef, type InputHTMLAttributes } from 'react';
import { cn } from '@/shared/lib';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  invalid?: boolean;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, invalid = false, ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      aria-invalid={invalid || undefined}
      className={cn('input focus-ring', className)}
      {...props}
    />
  );
});
