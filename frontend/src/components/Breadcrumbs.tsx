'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ChevronRight, Home } from 'lucide-react';

interface BreadcrumbItem {
  label: string;
  href: string;
  icon?: React.ReactNode;
}

interface BreadcrumbsProps {
  items?: BreadcrumbItem[];
  className?: string;
}

export default function Breadcrumbs({ items, className = '' }: BreadcrumbsProps) {
  const pathname = usePathname();

  // Auto-generate breadcrumbs from pathname if no items provided
  const generateBreadcrumbs = (): BreadcrumbItem[] => {
    const pathSegments = pathname.split('/').filter(Boolean);
    const breadcrumbs: BreadcrumbItem[] = [
      { label: 'Home', href: '/', icon: <Home className="w-4 h-4" /> }
    ];

    // Handle special routes
    const routeMap: Record<string, string> = {
      'works': 'Browse Works',
      'search': 'Search',
      'series': 'Series',
      'collections': 'Collections',
      'bookmarks': 'Bookmarks',
      'dashboard': 'Dashboard',
      'profile': 'Profile',
      'auth': 'Authentication',
      'login': 'Log In',
      'register': 'Sign Up',
      'new': 'Create New',
      'edit': 'Edit',
      'comments': 'Comments',
      'privacy': 'Privacy Settings',
      'pseudonyms': 'Pseudonyms',
      'manage': 'Manage',
      'moderate': 'Moderate'
    };

    let currentPath = '';
    
    pathSegments.forEach((segment, index) => {
      currentPath += `/${segment}`;
      
      // Skip dynamic segments like [id] in display
      if (segment.match(/^[0-9a-f-]{36}$|^\d+$/)) {
        // This looks like an ID, try to get a better label
        const parentSegment = pathSegments[index - 1];
        if (parentSegment === 'works') {
          breadcrumbs.push({ label: 'Work Details', href: currentPath });
        } else if (parentSegment === 'series') {
          breadcrumbs.push({ label: 'Series Details', href: currentPath });
        } else if (parentSegment === 'collections') {
          breadcrumbs.push({ label: 'Collection Details', href: currentPath });
        } else if (parentSegment === 'users') {
          breadcrumbs.push({ label: 'User Profile', href: currentPath });
        } else {
          breadcrumbs.push({ label: 'Details', href: currentPath });
        }
      } else {
        const label = routeMap[segment] || segment.charAt(0).toUpperCase() + segment.slice(1);
        breadcrumbs.push({ label, href: currentPath });
      }
    });

    return breadcrumbs;
  };

  const breadcrumbItems = items || generateBreadcrumbs();

  // Don't show breadcrumbs on home page or if only one item
  if (breadcrumbItems.length <= 1) {
    return null;
  }

  return (
    <nav 
      className={`flex items-center space-x-1 text-sm text-slate-600 mb-4 ${className}`}
      aria-label="Breadcrumb navigation"
    >
      <ol className="flex items-center space-x-1">
        {breadcrumbItems.map((item, index) => {
          const isLast = index === breadcrumbItems.length - 1;
          
          return (
            <li key={item.href} className="flex items-center space-x-1">
              {index > 0 && (
                <ChevronRight className="w-4 h-4 text-slate-400 mx-1" aria-hidden="true" />
              )}
              
              {isLast ? (
                <span 
                  className="flex items-center space-x-1 text-slate-900 font-medium"
                  aria-current="page"
                >
                  {item.icon}
                  <span>{item.label}</span>
                </span>
              ) : (
                <Link
                  href={item.href}
                  className="flex items-center space-x-1 hover:text-orange-600 transition-colors"
                >
                  {item.icon}
                  <span>{item.label}</span>
                </Link>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}