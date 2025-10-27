"use client"

import * as React from "react"

export interface CheckboxProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
}

export function Checkbox({ label, className = "", ...props }: CheckboxProps) {
  return (
    <div className="flex items-center space-x-2">
      <input 
        type="checkbox" 
        className={`h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500 ${className}`}
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