'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/Button'

interface AuthGuardProps {
  children: React.ReactNode
  redirectTo?: string
  fallback?: React.ReactNode
}

export default function AuthGuard({ 
  children, 
  redirectTo = '/auth/login',
  fallback 
}: AuthGuardProps) {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    let isMounted = true
    
    const checkAuth = () => {
      if (!isMounted) return // Prevent state updates if component unmounted
      if (typeof window === 'undefined') return // Safe SSR guard
      
      try {
        const token = localStorage.getItem('auth_token')
        setIsAuthenticated(!!token)
        setIsLoading(false)

        // If not authenticated, redirect with current path as return URL
        // FIXED: Prevent redirect loops by checking if we're already on login page
        if (!token && !window.location.pathname.includes('/auth/')) {
          const currentPath = window.location.pathname + window.location.search
          const redirectUrl = `${redirectTo}?redirect=${encodeURIComponent(currentPath)}`
          router.replace(redirectUrl)
        }
      } catch (error) {
        console.warn('Could not access localStorage in AuthGuard:', error)
        setIsAuthenticated(false)
        setIsLoading(false)
      }
    }

    // Check auth immediately
    checkAuth()

    // Listen for storage changes (user logs in/out in another tab)
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'auth_token' && isMounted) {
        checkAuth()
      }
    }

    window.addEventListener('storage', handleStorageChange)
    
    return () => {
      isMounted = false
      window.removeEventListener('storage', handleStorageChange)
    }
  }, [router, redirectTo])

  // Show loading state
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Checking authentication...</p>
        </div>
      </div>
    )
  }

  // Show fallback if not authenticated
  if (!isAuthenticated) {
    if (fallback) {
      return <>{fallback}</>
    }

    return (
      <div className="max-w-4xl mx-auto p-4">
        <div className="text-center py-16">
          <div className="bg-blue-50 rounded-full w-16 h-16 flex items-center justify-center mx-auto mb-6">
            <span className="text-2xl">🔒</span>
          </div>
          <h1 className="text-3xl font-bold mb-4">Authentication Required</h1>
          <p className="text-gray-600 mb-6 max-w-md mx-auto">
            You need to be logged in to access this page.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Button onClick={() => router.push(`${redirectTo}?redirect=${encodeURIComponent(window.location.pathname)}`)}>
              Log In
            </Button>
            <Button 
              variant="outline" 
              onClick={() => router.push(`/auth/register?redirect=${encodeURIComponent(window.location.pathname)}`)}
            >
              Create Account
            </Button>
          </div>
        </div>
      </div>
    )
  }

  // User is authenticated, render children
  return <>{children}</>
}