'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { 
  HomeIcon, 
  MagnifyingGlassIcon, 
  BookmarkIcon, 
  UserIcon,
  Bars3Icon,
  XMarkIcon,
  ArrowDownTrayIcon,
  WifiIcon,
  SignalSlashIcon,
  CloudArrowDownIcon
} from '@heroicons/react/24/outline';
import { 
  HomeIcon as HomeSolid,
  MagnifyingGlassIcon as SearchSolid,
  BookmarkIcon as BookmarkSolid,
  UserIcon as UserSolid
} from '@heroicons/react/24/solid';
import { usePWAState } from '../hooks/usePWAState';
import { useOfflineReading } from '../utils/offlineReadingManager';
import OfflineReadingManager from './OfflineReadingManager';

interface MobileNavigationProps {
  user?: {
    id: string;
    username: string;
    avatar?: string;
  };
}

export default function MobileNavigation({ user }: MobileNavigationProps) {
  const [showMenu, setShowMenu] = useState(false);
  const [showOfflineManager, setShowOfflineManager] = useState(false);
  const [offlineWorksCount, setOfflineWorksCount] = useState(0);
  const pathname = usePathname();

  // PWA state management
  const [pwaState, pwaActions] = usePWAState();
  const offlineManager = useOfflineReading();

  // Load offline works count
  useEffect(() => {
    const loadOfflineWorksCount = async () => {
      try {
        const works = await offlineManager.getOfflineWorks();
        setOfflineWorksCount(works.filter(w => !w.isExpired).length);
      } catch (error) {
        console.error('Failed to load offline works count:', error);
      }
    };

    loadOfflineWorksCount();

    // Listen for offline works changes
    const handleWorkCached = () => loadOfflineWorksCount();
    const handleWorkDeleted = () => loadOfflineWorksCount();

    offlineManager.onMessage('WORK_CACHED_WITH_CONSENT', handleWorkCached);
    offlineManager.onMessage('WORK_DELETED', handleWorkDeleted);

    return () => {
      offlineManager.offMessage('WORK_CACHED_WITH_CONSENT');
      offlineManager.offMessage('WORK_DELETED');
    };
  }, [offlineManager]);

  const mainNavItems = [
    {
      name: 'Home',
      href: '/',
      icon: HomeIcon,
      iconSolid: HomeSolid,
      description: 'Dashboard and recent activity'
    },
    {
      name: 'Browse',
      href: '/search',
      icon: MagnifyingGlassIcon,
      iconSolid: SearchSolid,
      description: 'Search and discover works'
    },
    {
      name: 'Bookmarks',
      href: '/bookmarks',
      icon: BookmarkIcon,
      iconSolid: BookmarkSolid,
      description: 'Your saved works'
    },
    {
      name: 'Profile',
      href: user ? '/dashboard' : '/auth/login',
      icon: UserIcon,
      iconSolid: UserSolid,
      description: user ? 'Your works and settings' : 'Sign in to your account'
    }
  ];

  const menuItems = [
    { name: 'My Works', href: '/dashboard/works', requiresAuth: true },
    { name: 'Collections', href: '/collections', requiresAuth: false },
    { name: 'Series', href: '/series', requiresAuth: false },
    { name: 'Reading History', href: '/dashboard/history', requiresAuth: true },
    { name: 'Subscriptions', href: '/dashboard/subscriptions', requiresAuth: true },
    { name: 'Settings', href: '/dashboard/settings', requiresAuth: true },
    { name: 'Help', href: '/help', requiresAuth: false },
    { name: 'About', href: '/about', requiresAuth: false }
  ];

  const isActive = (href: string) => {
    if (href === '/') {
      return pathname === '/';
    }
    return pathname.startsWith(href);
  };

  return (
    <>
      {/* Top Status Bar */}
      <div className="fixed top-0 left-0 right-0 z-40 bg-white border-b border-gray-200 px-4 py-2">
        <div className="flex items-center justify-between">
          {/* Connection Status */}
          <div className="flex items-center space-x-2">
            {pwaState.isOnline ? (
              <WifiIcon className="w-4 h-4 text-green-600" />
            ) : (
              <SignalSlashIcon className="w-4 h-4 text-red-600" />
            )}
            <span className="text-xs text-gray-600">
              {pwaState.isOnline ? 'Online' : 'Offline'}
            </span>
            
            {offlineWorksCount > 0 && (
              <button
                onClick={() => setShowOfflineManager(true)}
                className="text-xs text-blue-600 ml-2 hover:text-blue-800"
              >
                {offlineWorksCount} work{offlineWorksCount !== 1 ? 's' : ''} offline
              </button>
            )}
          </div>

          {/* Logo */}
          <Link href="/" className="font-bold text-lg text-blue-600">
            Nuclear AO3
          </Link>

          {/* Menu Button */}
          <button
            onClick={() => setShowMenu(true)}
            className="p-2 rounded-lg hover:bg-gray-100"
            aria-label="Open menu"
          >
            <Bars3Icon className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Bottom Navigation */}
      <div className="fixed bottom-0 left-0 right-0 z-40 bg-white border-t border-gray-200">
        <div className="grid grid-cols-4">
          {mainNavItems.map((item) => {
            const Icon = isActive(item.href) ? item.iconSolid : item.icon;
            const active = isActive(item.href);
            
            return (
              <Link
                key={item.name}
                href={item.href}
                className={`flex flex-col items-center justify-center py-2 px-1 text-xs transition-colors ${
                  active 
                    ? 'text-blue-600 bg-blue-50' 
                    : 'text-gray-600 hover:text-blue-600'
                }`}
              >
                <Icon className="w-6 h-6 mb-1" />
                <span className="truncate">{item.name}</span>
              </Link>
            );
          })}
        </div>
      </div>

      {/* Slide-out Menu */}
      {showMenu && (
        <div className="fixed inset-0 z-50">
          {/* Backdrop */}
          <div 
            className="absolute inset-0 bg-black/50"
            onClick={() => setShowMenu(false)}
          />
          
          {/* Menu Panel */}
          <div className="absolute top-0 right-0 h-full w-80 bg-white shadow-xl overflow-y-auto">
            {/* Menu Header */}
            <div className="flex items-center justify-between p-4 border-b border-gray-200">
              <div>
                <h2 className="text-lg font-semibold">Menu</h2>
                {user && (
                  <p className="text-sm text-gray-600">Welcome, {user.username}</p>
                )}
              </div>
              <button
                onClick={() => setShowMenu(false)}
                className="p-2 rounded-lg hover:bg-gray-100"
              >
                <XMarkIcon className="w-5 h-5" />
              </button>
            </div>

            {/* PWA Install Banner */}
            {pwaState.isInstallable && !pwaState.isInstalled && (
              <div className="p-4 bg-blue-50 border-b border-blue-200">
                <div className="flex items-start space-x-3">
                  <ArrowDownTrayIcon className="w-5 h-5 text-blue-600 mt-0.5" />
                  <div className="flex-1">
                    <h3 className="text-sm font-medium text-blue-900">
                      Install Nuclear AO3
                    </h3>
                    <p className="text-xs text-blue-700 mt-1">
                      Install as an app for better offline reading and faster access
                    </p>
                    <div className="flex space-x-2 mt-2">
                      <button
                        onClick={async () => {
                          const success = await pwaActions.installPWA();
                          if (success) {
                            setShowMenu(false);
                          }
                        }}
                        className="px-3 py-1 bg-blue-600 text-white text-xs rounded-md hover:bg-blue-700"
                      >
                        Install
                      </button>
                      <button
                        onClick={pwaActions.dismissInstallPrompt}
                        className="px-3 py-1 text-blue-600 text-xs rounded-md hover:bg-blue-100"
                      >
                        Later
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* PWA Update Banner */}
            {pwaState.updateAvailable && (
              <div className="p-4 bg-green-50 border-b border-green-200">
                <div className="flex items-start space-x-3">
                  <ArrowDownTrayIcon className="w-5 h-5 text-green-600 mt-0.5" />
                  <div className="flex-1">
                    <h3 className="text-sm font-medium text-green-900">
                      Update Available
                    </h3>
                    <p className="text-xs text-green-700 mt-1">
                      A new version of Nuclear AO3 is ready to install
                    </p>
                    <button
                      onClick={() => {
                        pwaActions.skipWaiting();
                        setShowMenu(false);
                      }}
                      className="mt-2 px-3 py-1 bg-green-600 text-white text-xs rounded-md hover:bg-green-700"
                    >
                      Update Now
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* User Section */}
            {user ? (
              <div className="p-4 border-b border-gray-200">
                <div className="flex items-center space-x-3">
                  {user.avatar ? (
                    <img 
                      src={user.avatar} 
                      alt={user.username}
                      className="w-10 h-10 rounded-full"
                    />
                  ) : (
                    <div className="w-10 h-10 bg-gray-300 rounded-full flex items-center justify-center">
                      <UserIcon className="w-6 h-6 text-gray-600" />
                    </div>
                  )}
                  <div>
                    <div className="font-medium">{user.username}</div>
                    <div className="text-sm text-gray-600">Author & Reader</div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="p-4 border-b border-gray-200">
                <Link
                  href="/auth/login"
                  className="block w-full py-2 px-4 bg-blue-600 text-white text-center rounded-lg hover:bg-blue-700"
                  onClick={() => setShowMenu(false)}
                >
                  Sign In
                </Link>
                <Link
                  href="/auth/register"
                  className="block w-full py-2 px-4 mt-2 border border-gray-300 text-center rounded-lg hover:bg-gray-50"
                  onClick={() => setShowMenu(false)}
                >
                  Create Account
                </Link>
              </div>
            )}

            {/* Offline Works Section */}
            {offlineWorksCount > 0 && (
              <div className="p-4 border-b border-gray-200">
                <button
                  onClick={() => {
                    setShowOfflineManager(true);
                    setShowMenu(false);
                  }}
                  className="flex items-center justify-between p-3 bg-green-50 border border-green-200 rounded-lg w-full hover:bg-green-100"
                >
                  <div className="flex items-center space-x-3">
                    <CloudArrowDownIcon className="w-5 h-5 text-green-600" />
                    <div>
                      <div className="text-sm font-medium text-green-900">
                        Offline Works
                      </div>
                      <div className="text-xs text-green-700">
                        {offlineWorksCount} work{offlineWorksCount !== 1 ? 's' : ''} available
                      </div>
                    </div>
                  </div>
                  <div className="w-6 h-6 bg-green-600 text-white rounded-full flex items-center justify-center text-xs font-bold">
                    {offlineWorksCount}
                  </div>
                </button>
              </div>
            )}

            {/* Menu Items */}
            <div className="py-2">
              {menuItems.map((item) => {
                if (item.requiresAuth && !user) {
                  return null;
                }
                
                return (
                  <Link
                    key={item.name}
                    href={item.href}
                    className="block px-4 py-3 text-gray-700 hover:bg-gray-50 border-b border-gray-100"
                    onClick={() => setShowMenu(false)}
                  >
                    {item.name}
                  </Link>
                );
              })}
            </div>

            {/* Footer */}
            <div className="p-4 border-t border-gray-200 text-center text-xs text-gray-500">
              <div className="mb-2">Nuclear AO3 v2.0.0</div>
              <div className="flex justify-center space-x-4">
                <Link href="/privacy" className="hover:text-gray-700">Privacy</Link>
                <Link href="/terms" className="hover:text-gray-700">Terms</Link>
                <Link href="/contact" className="hover:text-gray-700">Contact</Link>
              </div>
            </div>

            {/* Sign Out */}
            {user && (
              <div className="p-4 border-t border-gray-200">
                <button
                  onClick={() => {
                    // Handle sign out
                    setShowMenu(false);
                  }}
                  className="w-full py-2 px-4 text-red-600 border border-red-300 rounded-lg hover:bg-red-50"
                >
                  Sign Out
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Offline Reading Manager */}
      <OfflineReadingManager 
        isOpen={showOfflineManager}
        onClose={() => setShowOfflineManager(false)}
      />

      {/* Spacers for fixed navigation */}
      <div className="h-14" /> {/* Top spacer */}
      <div className="h-16" /> {/* Bottom spacer */}
    </>
  );
}