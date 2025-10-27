import { forwardRef } from 'react';
import { clsx } from 'clsx';

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: 'primary' | 'secondary' | 'success' | 'warning' | 'danger' | 'info' | 'neutral';
  size?: 'sm' | 'md' | 'lg';
  prominence?: 'primary' | 'secondary' | 'micro';
}

const Badge = forwardRef<HTMLSpanElement, BadgeProps>(({
  className,
  variant = 'neutral',
  size = 'md',
  prominence,
  children,
  ...props
}, ref) => {
  const baseClasses = 'inline-flex items-center font-medium rounded-full';
  
  // If prominence is specified, use prominence-specific styling
  const prominenceVariants = {
    primary: 'bg-green-100 text-green-800 border border-green-300',
    secondary: 'bg-blue-100 text-blue-800 border border-blue-300',
    micro: 'bg-slate-100 text-slate-600 border border-slate-300',
  };

  const variants = {
    primary: 'bg-orange-100 text-orange-800',
    secondary: 'bg-slate-100 text-slate-800',
    success: 'bg-green-100 text-green-800',
    warning: 'bg-yellow-100 text-yellow-800',
    danger: 'bg-red-100 text-red-800',
    info: 'bg-blue-100 text-blue-800',
    neutral: 'bg-slate-100 text-slate-700',
  };

  const sizes = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-0.5 text-sm',
    lg: 'px-3 py-1 text-sm',
  };

  const prominenceSymbols = {
    primary: 'P',
    secondary: 'S',
    micro: 'M',
  };

  return (
    <span
      ref={ref}
      className={clsx(
        baseClasses,
        prominence ? prominenceVariants[prominence] : variants[variant],
        sizes[size],
        className
      )}
      {...props}
    >
      {prominence && (
        <span className="mr-1 text-xs font-bold">
          {prominenceSymbols[prominence]}
        </span>
      )}
      {children}
    </span>
  );
});

Badge.displayName = 'Badge';

// Named export for compatibility
export { Badge };

export default Badge;