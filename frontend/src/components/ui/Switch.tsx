import { Fragment } from 'react';
import { Switch as HeadlessSwitch } from '@headlessui/react';
import { clsx } from 'clsx';

export interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  description?: string;
  disabled?: boolean;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export default function Switch({
  checked,
  onChange,
  label,
  description,
  disabled = false,
  size = 'md',
  className,
}: SwitchProps) {
  const sizes = {
    sm: {
      switch: 'h-5 w-9',
      thumb: 'h-4 w-4',
      translate: checked ? 'translate-x-4' : 'translate-x-0',
    },
    md: {
      switch: 'h-6 w-11',
      thumb: 'h-5 w-5',
      translate: checked ? 'translate-x-5' : 'translate-x-0',
    },
    lg: {
      switch: 'h-7 w-12',
      thumb: 'h-6 w-6',
      translate: checked ? 'translate-x-5' : 'translate-x-0',
    },
  };

  const switchComponent = (
    <HeadlessSwitch
      checked={checked}
      onChange={onChange}
      disabled={disabled}
      className={clsx(
        'relative inline-flex flex-shrink-0 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500',
        checked ? 'bg-orange-600' : 'bg-slate-200',
        disabled && 'opacity-50 cursor-not-allowed',
        sizes[size].switch,
        className
      )}
    >
      <span className="sr-only">{label}</span>
      <span
        aria-hidden="true"
        className={clsx(
          'pointer-events-none inline-block rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200',
          sizes[size].thumb,
          sizes[size].translate
        )}
      />
    </HeadlessSwitch>
  );

  if (label || description) {
    return (
      <HeadlessSwitch.Group as="div" className="flex items-center justify-between">
        <span className="flex-grow flex flex-col">
          {label && (
            <HeadlessSwitch.Label
              as="span"
              className="text-sm font-medium text-slate-900 cursor-pointer"
              passive
            >
              {label}
            </HeadlessSwitch.Label>
          )}
          {description && (
            <HeadlessSwitch.Description as="span" className="text-sm text-slate-500">
              {description}
            </HeadlessSwitch.Description>
          )}
        </span>
        {switchComponent}
      </HeadlessSwitch.Group>
    );
  }

  return switchComponent;
}