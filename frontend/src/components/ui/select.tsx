import { ReactNode } from 'react'

interface SelectProps {
  children: ReactNode
  value?: string
  onValueChange?: (value: string) => void
  className?: string
}

export function Select({ children, value, onValueChange, className = '' }: SelectProps) {
  return (
    <div className={`relative ${className}`}>
      {children}
    </div>
  )
}

export function SelectTrigger({ children, className = '' }: { children: ReactNode, className?: string }) {
  return (
    <button className={`flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${className}`}>
      {children}
    </button>
  )
}

export function SelectValue({ placeholder }: { placeholder?: string }) {
  return <span className="text-muted-foreground">{placeholder}</span>
}

export function SelectContent({ children, className = '' }: { children: ReactNode, className?: string }) {
  return (
    <div className={`relative z-50 min-w-[8rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md ${className}`}>
      {children}
    </div>
  )
}

export function SelectItem({ children, value, className = '' }: { children: ReactNode, value: string, className?: string }) {
  return (
    <div className={`relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none focus:bg-accent focus:text-accent-foreground ${className}`}>
      {children}
    </div>
  )
}