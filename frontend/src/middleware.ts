import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

// Routes that require authentication
const protectedRoutes = [
  '/works/new',
  '/works/edit',
  '/dashboard',
  '/bookmarks',
  '/series/new',
  '/collections/new',
  '/profile',
  '/profile/edit',
  '/profile/pseudonyms',
  '/profile/privacy',
  '/onboarding'
]

// Routes that should redirect to dashboard if user is already logged in
const authRoutes = [
  '/auth/login',
  '/auth/register'
]

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl
  const token = request.cookies.get('auth_token')?.value
  
  // Only check for valid tokens (not empty strings)
  const hasValidToken = token && token.trim() !== ''

  // Check if the current route requires authentication
  const isProtectedRoute = protectedRoutes.some(route => 
    pathname.startsWith(route)
  )

  // Check if the current route is an auth route
  const isAuthRoute = authRoutes.includes(pathname)

  // If accessing a protected route without authentication
  if (isProtectedRoute && !hasValidToken) {
    const loginUrl = new URL('/auth/login', request.url)
    loginUrl.searchParams.set('redirect', pathname)
    return NextResponse.redirect(loginUrl)
  }

  // If accessing auth routes while already authenticated
  if (isAuthRoute && hasValidToken) {
    const redirectParam = request.nextUrl.searchParams.get('redirect')
    const dashboardUrl = new URL(redirectParam || '/dashboard', request.url)
    return NextResponse.redirect(dashboardUrl)
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    '/((?!api|_next/static|_next/image|favicon.ico).*)',
  ],
}