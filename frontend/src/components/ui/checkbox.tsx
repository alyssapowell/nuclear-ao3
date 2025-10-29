"use client"

import * as React from "react"

export interface CheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange'> {
  label?: string
  checked?: boolean
  onCheckedChange?: (checked: boolean) => void
}

export function Checkbox({ label, className = "", checked, onCheckedChange, ...props }: CheckboxProps) {
  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (onCheckedChange) {
      onCheckedChange(event.target.checked)
    }
  }

  return (
    <div className="flex items-center space-x-2">
      <input 
        type="checkbox" 
        className={`h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500 ${className}`}
        checked={checked}
        onChange={handleChange}
        {...props}
      />
      {label && (
        <label className="text-sm font-medium text-gray-900">
          {label}
        </label>
      )}
    </div>
  )
}