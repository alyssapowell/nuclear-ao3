"use client"

import * as React from "react"

export interface LabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {
  children: React.ReactNode
}

export function Label({ children, className = "", ...props }: LabelProps) {
  return (
    <label 
      className={`block text-sm font-medium text-gray-900 mb-1 ${className}`}
      {...props}
    >
      {children}
    </label>
  )
}