"use client"

import { ReactNode } from 'react'

interface AuthGuardProps {
  children: ReactNode
  redirectTo?: string
}

export function AuthGuard({ children, redirectTo = '/login' }: AuthGuardProps) {
  // Stub implementation - always return children
  // In real implementation, this would check authentication status
  return <>{children}</>
}