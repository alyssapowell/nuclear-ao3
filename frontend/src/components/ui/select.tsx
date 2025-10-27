// Stub UI select component
import { forwardRef } from 'react';

interface SelectProps {
  children: React.ReactNode;
  value?: string;
  onValueChange?: (value: string) => void;
}

const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ children, value, onValueChange, ...props }, ref) => (
    <select
      ref={ref}
      value={value}
      onChange={(e) => onValueChange?.(e.target.value)}
      className="px-3 py-2 border rounded"
      {...props}
    >
      {children}
    </select>
  )
);
Select.displayName = 'Select';

const SelectContent = ({ children }: { children: React.ReactNode }) => (
  <>{children}</>
);

const SelectItem = ({ value, children }: { value: string; children: React.ReactNode }) => (
  <option value={value}>{children}</option>
);

const SelectTrigger = ({ children }: { children: React.ReactNode }) => (
  <>{children}</>
);

const SelectValue = ({ placeholder }: { placeholder?: string }) => (
  <span>{placeholder}</span>
);

export { Select, SelectContent, SelectItem, SelectTrigger, SelectValue };