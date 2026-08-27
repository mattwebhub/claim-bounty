import { forwardRef, type HTMLAttributes } from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/shared/lib';

const alertVariants = cva('alert', {
  variants: {
    variant: {
      info: 'alert-info',
      success: 'alert-success',
      warning: 'alert-warning',
      destructive: 'alert-destructive',
    },
  },
  defaultVariants: { variant: 'info' },
});

export interface AlertProps
  extends HTMLAttributes<HTMLDivElement>, VariantProps<typeof alertVariants> {}

export const Alert = forwardRef<HTMLDivElement, AlertProps>(function Alert(
  { className, variant, ...props },
  ref,
) {
  return (
    <div
      ref={ref}
      role={variant === 'destructive' ? 'alert' : 'status'}
      className={cn(alertVariants({ className, variant }))}
      {...props}
    />
  );
});

export function AlertTitle({ className, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  const { children, ...headingProps } = props;
  return (
    <h2 className={cn('alert-title', className)} {...headingProps}>
      {children}
    </h2>
  );
}

export function AlertDescription({ className, ...props }: HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn('alert-description', className)} {...props} />;
}
