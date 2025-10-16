'use client';

import Link from 'next/link';
import { Search, BookOpen, PlusCircle, FileText, Users, Folder, Inbox, Filter } from 'lucide-react';

export type EmptyStateType = 
  | 'no-works' 
  | 'no-search-results' 
  | 'no-bookmarks' 
  | 'no-collections' 
  | 'no-series' 
  | 'no-comments' 
  | 'no-notifications'
  | 'no-dashboard-content'
  | 'no-user-works'
  | 'filtered-results';

interface EmptyStateAction {
  label: string;
  href?: string;
  onClick?: () => void;
  variant?: 'primary' | 'secondary';
}

interface EmptyStateProps {
  type: EmptyStateType;
  title?: string;
  description?: string;
  actions?: EmptyStateAction[];
  className?: string;
  searchQuery?: string;
  filterCount?: number;
}

const getEmptyStateConfig = (type: EmptyStateType, searchQuery?: string, filterCount?: number) => {
  const configs = {
    'no-works': {
      icon: BookOpen,
      title: 'No Works Available',
      description: 'There are no published works yet. Why not be the first to share your story?',
      actions: [
        { label: 'Post a Work', href: '/works/new', variant: 'primary' as const },
        { label: 'Browse Tags', href: '/tags', variant: 'secondary' as const }
      ]
    },
    'no-search-results': {
      icon: Search,
      title: searchQuery ? `No results for "${searchQuery}"` : 'No Search Results',
      description: searchQuery 
        ? 'Try adjusting your search terms or browse our collection of works.'
        : 'Start typing to search for works, authors, or tags.',
      actions: [
        { label: 'Browse All Works', href: '/works', variant: 'primary' as const },
        { label: 'Advanced Search', href: '/search', variant: 'secondary' as const }
      ]
    },
    'no-bookmarks': {
      icon: BookOpen,
      title: 'No Bookmarks Yet',
      description: 'Bookmark works you want to read later or revisit. Your bookmarks will appear here.',
      actions: [
        { label: 'Discover Works', href: '/works', variant: 'primary' as const },
        { label: 'Popular This Week', href: '/works?sort=popular', variant: 'secondary' as const }
      ]
    },
    'no-collections': {
      icon: Folder,
      title: 'No Folders Found',
      description: 'Folders help organize related works around themes, challenges, or events.',
      actions: [
        { label: 'Create Folder', href: '/collections/new', variant: 'primary' as const },
        { label: 'Browse Folders', href: '/collections', variant: 'secondary' as const }
      ]
    },
    'no-series': {
      icon: FileText,
      title: 'No Series Available',
      description: 'Series group related works together in a specific order.',
      actions: [
        { label: 'Create Series', href: '/series/new', variant: 'primary' as const },
        { label: 'Browse All Series', href: '/series', variant: 'secondary' as const }
      ]
    },
    'no-comments': {
      icon: Users,
      title: 'No Comments Yet',
      description: 'Be the first to leave a comment and start a conversation!',
      actions: []
    },
    'no-notifications': {
      icon: Inbox,
      title: 'All Caught Up!',
      description: 'No new notifications right now. Your updates will appear here.',
      actions: [
        { label: 'Manage Subscriptions', href: '/profile/subscriptions', variant: 'secondary' as const }
      ]
    },
    'no-dashboard-content': {
      icon: PlusCircle,
      title: 'Welcome to Nuclear AO3!',
      description: 'Start by posting your first work or exploring what others have shared.',
      actions: [
        { label: 'Post Your First Work', href: '/works/new', variant: 'primary' as const },
        { label: 'Browse Works', href: '/works', variant: 'secondary' as const }
      ]
    },
    'no-user-works': {
      icon: FileText,
      title: 'No Works Published',
      description: 'This user hasn\'t published any works yet.',
      actions: []
    },
    'filtered-results': {
      icon: Filter,
      title: filterCount ? `No results with ${filterCount} filter${filterCount === 1 ? '' : 's'}` : 'No Filtered Results',
      description: 'Try removing some filters or adjusting your search criteria.',
      actions: [
        { label: 'Clear All Filters', onClick: () => window.location.href = window.location.pathname, variant: 'secondary' as const }
      ]
    }
  };

  return configs[type];
};

export default function EmptyState({ 
  type, 
  title, 
  description, 
  actions, 
  className = '', 
  searchQuery,
  filterCount
}: EmptyStateProps) {
  const config = getEmptyStateConfig(type, searchQuery, filterCount);
  const Icon = config.icon;

  const finalTitle = title || config.title;
  const finalDescription = description || config.description;
  const finalActions = actions || config.actions;

  return (
    <div className={`text-center py-12 px-4 ${className}`}>
      <div className="max-w-md mx-auto">
        <div className="w-16 h-16 mx-auto mb-4 bg-slate-100 rounded-full flex items-center justify-center">
          <Icon className="w-8 h-8 text-slate-400" />
        </div>
        
        <h2 className="text-xl font-semibold text-slate-900 mb-2">
          {finalTitle}
        </h2>
        
        <p className="text-slate-600 mb-6">
          {finalDescription}
        </p>
        
        {finalActions && finalActions.length > 0 && (
          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            {finalActions.map((action, index) => {
              const baseClasses = "px-6 py-2 rounded-lg font-medium transition-colors text-center";
              const variantClasses = action.variant === 'primary' 
                ? "bg-orange-600 text-white hover:bg-orange-700" 
                : "border border-slate-300 text-slate-700 hover:bg-slate-50";
              
              if ('href' in action && action.href) {
                return (
                  <Link
                    key={index}
                    href={action.href}
                    className={`${baseClasses} ${variantClasses}`}
                  >
                    {action.label}
                  </Link>
                );
              } else if ('onClick' in action && action.onClick) {
                return (
                  <button
                    key={index}
                    onClick={action.onClick}
                    className={`${baseClasses} ${variantClasses}`}
                  >
                    {action.label}
                  </button>
                );
              }
              return null;
            })}
          </div>
        )}
      </div>
    </div>
  );
}

// Convenience components for common empty states
export const NoWorksFound = (props: Omit<EmptyStateProps, 'type'>) => (
  <EmptyState type="no-works" {...props} />
);

export const NoSearchResults = (props: Omit<EmptyStateProps, 'type'>) => (
  <EmptyState type="no-search-results" {...props} />
);

export const NoBookmarks = (props: Omit<EmptyStateProps, 'type'>) => (
  <EmptyState type="no-bookmarks" {...props} />
);

export const FilteredEmptyState = (props: Omit<EmptyStateProps, 'type'>) => (
  <EmptyState type="filtered-results" {...props} />
);