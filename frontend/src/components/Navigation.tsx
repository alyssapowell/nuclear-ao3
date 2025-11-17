'use client';

import { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { useRouter, usePathname } from 'next/navigation';
import { Search, Menu, X, User, Bell, LogOut, Settings, ChevronDown, BookOpen, Library, FolderOpen, Plus, Bookmark, Heart, LayoutDashboard } from 'lucide-react';
import { NotificationBell } from './notifications';

interface User {
  id: string;
  username: string;
  email: string;
}

interface NavDropdownProps {
  label: string;
  icon?: React.ReactNode;
  children: React.ReactNode;
  isActive: boolean;
}

// Dropdown component for desktop nav
function NavDropdown({ label, icon, children, isActive }: NavDropdownProps) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center space-x-1 text-sm font-medium transition-colors hover:text-orange-600 ${
          isActive 
            ? 'text-orange-600 border-b-2 border-orange-600 pb-4 -mb-4' 
            : 'text-gray-700'
        }`}
        aria-label={`${label} menu`}
        aria-expanded={isOpen}
      >
        {icon && <span className="w-4 h-4">{icon}</span>}
        <span>{label}</span>
        <ChevronDown className={`w-4 h-4 transition-transform ${isOpen ? 'rotate-180' : ''}`} aria-hidden="true" />
      </button>

      {isOpen && (
        <div className="absolute left-0 mt-6 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
          {children}
        </div>
      )}
    </div>
  );
}

export default function Navigation() {
  const [user, setUser] = useState<User | null>(null);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const router = useRouter();
  const pathname = usePathname();

  // Check for authentication and listen for changes
  useEffect(() => {
    // Check for authentication - Safe SSR access
    if (typeof window === 'undefined') return;
    
    const checkAuth = () => {
      try {
        const token = localStorage.getItem('auth_token');
        const userData = localStorage.getItem('user');
        
        if (token && userData) {
          try {
            setUser(JSON.parse(userData));
          } catch (error) {
            console.error('Error parsing user data:', error);
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            setUser(null);
          }
        } else {
          setUser(null);
        }
      } catch (error) {
        console.warn('Could not access localStorage in Navigation:', error);
      }
    };

    // Check auth on mount
    checkAuth();

    // Listen for auth changes (from login/register/logout)
    window.addEventListener('authChange', checkAuth);
    
    // Cleanup
    return () => {
      window.removeEventListener('authChange', checkAuth);
    };
  }, []);

  const handleSignOut = () => {
    // Safe localStorage cleanup
    if (typeof window !== 'undefined') {
      try {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        localStorage.removeItem('auth_token');
        document.cookie = 'auth_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';
      } catch (error) {
        console.warn('Could not clear localStorage during signout:', error);
      }
    }
    setUser(null);
    setIsUserMenuOpen(false);
    router.push('/');
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchQuery.trim()) {
      router.push(`/search?q=${encodeURIComponent(searchQuery.trim())}`);
      setSearchQuery('');
    }
  };

  const isActive = (path: string) => {
    return pathname === path || pathname.startsWith(path + '/');
  };

  const isAnyActive = (paths: string[]) => {
    return paths.some(path => isActive(path));
  };

  // Organized navigation structure
  const browseLinks = [
    { href: '/works', label: 'Works', icon: <BookOpen className="w-4 h-4" /> },
    { href: '/series', label: 'Series', icon: <Library className="w-4 h-4" /> },
    { href: '/collections', label: 'Collections', icon: <FolderOpen className="w-4 h-4" /> },
  ];

  const createLinks = user ? [
    { href: '/works/new', label: 'Post Work', icon: <Plus className="w-4 h-4" /> },
    { href: '/series/new', label: 'Create Series', icon: <Plus className="w-4 h-4" /> },
  ] : [];

  const libraryLinks = user ? [
    { href: '/dashboard', label: 'Dashboard', icon: <LayoutDashboard className="w-4 h-4" /> },
    { href: '/bookmarks', label: 'Bookmarks', icon: <Bookmark className="w-4 h-4" /> },
    { href: '/subscriptions', label: 'Subscriptions', icon: <Heart className="w-4 h-4" /> },
  ] : [];

  return (
    <header className="bg-white border-b border-gray-200 sticky top-0 z-50 shadow-sm">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <div className="flex items-center">
            <Link href="/" className="flex items-center space-x-2 group">
              <div className="text-2xl">🚀</div>
              <div className="hidden sm:block">
                <h1 className="text-xl font-bold text-gray-900 group-hover:text-orange-600 transition-colors">
                  Nuclear AO3
                </h1>
                <p className="text-xs text-gray-500 -mt-1">Fast & Modern Archive</p>
              </div>
              <div className="sm:hidden">
                <h1 className="text-lg font-bold text-gray-900 group-hover:text-orange-600 transition-colors">
                  Nuclear AO3
                </h1>
              </div>
            </Link>
          </div>

          {/* Desktop Search Bar */}
          <div className="hidden md:block flex-1 max-w-lg mx-8">
            <form onSubmit={handleSearch} className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Search className="h-4 w-4 text-gray-400" />
              </div>
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search works, fandoms, characters..."
                className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg bg-gray-50 focus:ring-2 focus:ring-orange-500 focus:border-orange-500 focus:bg-white transition-colors text-sm"
              />
            </form>
          </div>

          {/* Desktop Navigation */}
          <div className="hidden md:flex items-center space-x-6">
            {/* Search Link (standalone) */}
            <Link
              href="/search"
              className={`text-sm font-medium transition-colors hover:text-orange-600 ${
                isActive('/search') 
                  ? 'text-orange-600 border-b-2 border-orange-600 pb-4 -mb-4' 
                  : 'text-gray-700'
              }`}
            >
              Search
            </Link>

            {/* Browse Dropdown */}
            <NavDropdown 
              label="Browse" 
              icon={<BookOpen className="w-4 h-4" />}
              isActive={isAnyActive(browseLinks.map(l => l.href))}
            >
              {browseLinks.map((link) => (
                <Link
                  key={link.href}
                  href={link.href}
                  className={`flex items-center space-x-2 px-4 py-2 text-sm transition-colors ${
                    isActive(link.href)
                      ? 'text-orange-600 bg-orange-50'
                      : 'text-gray-700 hover:bg-gray-50 hover:text-orange-600'
                  }`}
                >
                  {link.icon}
                  <span>{link.label}</span>
                </Link>
              ))}
            </NavDropdown>

            {/* Create Dropdown (only when logged in) */}
            {user && createLinks.length > 0 && (
              <NavDropdown 
                label="Create" 
                icon={<Plus className="w-4 h-4" />}
                isActive={isAnyActive(createLinks.map(l => l.href))}
              >
                {createLinks.map((link) => (
                  <Link
                    key={link.href}
                    href={link.href}
                    className={`flex items-center space-x-2 px-4 py-2 text-sm transition-colors ${
                      isActive(link.href)
                        ? 'text-orange-600 bg-orange-50'
                        : 'text-gray-700 hover:bg-gray-50 hover:text-orange-600'
                    }`}
                  >
                    {link.icon}
                    <span>{link.label}</span>
                  </Link>
                ))}
              </NavDropdown>
            )}

            {/* My Library Dropdown (only when logged in) */}
            {user && libraryLinks.length > 0 && (
              <NavDropdown 
                label="My Library" 
                icon={<Library className="w-4 h-4" />}
                isActive={isAnyActive(libraryLinks.map(l => l.href))}
              >
                {libraryLinks.map((link) => (
                  <Link
                    key={link.href}
                    href={link.href}
                    className={`flex items-center space-x-2 px-4 py-2 text-sm transition-colors ${
                      isActive(link.href)
                        ? 'text-orange-600 bg-orange-50'
                        : 'text-gray-700 hover:bg-gray-50 hover:text-orange-600'
                    }`}
                  >
                    {link.icon}
                    <span>{link.label}</span>
                  </Link>
                ))}
              </NavDropdown>
            )}

            {/* User Menu or Login */}
            {user ? (
              <div className="relative ml-4">
                <div className="flex items-center space-x-3">
                  {/* <NotificationBell /> Temporarily disabled - notification service not running */}
                  <button
                    onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
                    className="flex items-center space-x-2 text-sm font-medium text-gray-700 hover:text-orange-600 transition-colors"
                    aria-label="User menu"
                    aria-expanded={isUserMenuOpen}
                  >
                    <div className="w-8 h-8 bg-orange-100 rounded-full flex items-center justify-center">
                      <User className="w-4 h-4 text-orange-600" aria-hidden="true" />
                    </div>
                    <span className="hidden lg:block">{user.username}</span>
                    <ChevronDown className="w-4 h-4" aria-hidden="true" />
                  </button>
                </div>

                {isUserMenuOpen && (
                  <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
                    <Link
                      href="/profile"
                      className="flex items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                      onClick={() => setIsUserMenuOpen(false)}
                    >
                      <Settings className="w-4 h-4 mr-2" />
                      Profile Settings
                    </Link>
                    <Link
                      href="/profile/privacy"
                      className="flex items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                      onClick={() => setIsUserMenuOpen(false)}
                    >
                      <Settings className="w-4 h-4 mr-2" />
                      Privacy Settings
                    </Link>
                    <hr className="my-1" />
                    <button
                      onClick={handleSignOut}
                      className="flex items-center w-full px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                      aria-label="Sign out"
                    >
                      <LogOut className="w-4 h-4 mr-2" aria-hidden="true" />
                      Sign Out
                    </button>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex items-center space-x-4">
                <Link
                  href="/auth/login"
                  className="text-sm font-medium text-gray-700 hover:text-orange-600 transition-colors"
                >
                  Log In
                </Link>
                <Link
                  href="/auth/register"
                  className="bg-orange-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-orange-700 transition-colors"
                >
                  Sign Up
                </Link>
              </div>
            )}
          </div>

          {/* Mobile Menu Button */}
          <div className="md:hidden flex items-center space-x-4">
            {/* <NotificationBell /> Temporarily disabled - notification service not running */}
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className="p-2 rounded-md text-gray-700 hover:text-orange-600 hover:bg-gray-100 transition-colors"
              aria-label={isMobileMenuOpen ? 'Close mobile menu' : 'Open mobile menu'}
              aria-expanded={isMobileMenuOpen}
            >
              {isMobileMenuOpen ? (
                <X className="h-6 w-6" aria-hidden="true" />
              ) : (
                <Menu className="h-6 w-6" aria-hidden="true" />
              )}
            </button>
          </div>
        </div>

        {/* Mobile Search Bar */}
        <div className="md:hidden pb-4">
          <form onSubmit={handleSearch} className="relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Search className="h-4 w-4 text-gray-400" />
            </div>
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search works, fandoms, characters..."
              className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg bg-gray-50 focus:ring-2 focus:ring-orange-500 focus:border-orange-500 focus:bg-white transition-colors text-sm"
            />
          </form>
        </div>
      </div>

      {/* Mobile Menu */}
      {isMobileMenuOpen && (
        <div className="md:hidden border-t border-gray-200 bg-white">
          <div className="px-4 py-2 space-y-1">
            {/* Search */}
            <Link
              href="/search"
              className={`flex items-center space-x-2 px-3 py-2 rounded-md text-base font-medium transition-colors ${
                isActive('/search')
                  ? 'text-orange-600 bg-orange-50'
                  : 'text-gray-700 hover:text-orange-600 hover:bg-gray-50'
              }`}
              onClick={() => setIsMobileMenuOpen(false)}
            >
              <Search className="w-5 h-5" />
              <span>Search</span>
            </Link>

            {/* Browse Section */}
            <div className="pt-2">
              <div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
                Browse
              </div>
              {browseLinks.map((link) => (
                <Link
                  key={link.href}
                  href={link.href}
                  className={`flex items-center space-x-2 px-3 py-2 rounded-md text-base font-medium transition-colors ${
                    isActive(link.href)
                      ? 'text-orange-600 bg-orange-50'
                      : 'text-gray-700 hover:text-orange-600 hover:bg-gray-50'
                  }`}
                  onClick={() => setIsMobileMenuOpen(false)}
                >
                  {link.icon}
                  <span>{link.label}</span>
                </Link>
              ))}
            </div>

            {user && (
              <>
                {/* Create Section */}
                {createLinks.length > 0 && (
                  <div className="pt-2">
                    <div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
                      Create
                    </div>
                    {createLinks.map((link) => (
                      <Link
                        key={link.href}
                        href={link.href}
                        className={`flex items-center space-x-2 px-3 py-2 rounded-md text-base font-medium transition-colors ${
                          isActive(link.href)
                            ? 'text-orange-600 bg-orange-50'
                            : 'text-gray-700 hover:text-orange-600 hover:bg-gray-50'
                        }`}
                        onClick={() => setIsMobileMenuOpen(false)}
                      >
                        {link.icon}
                        <span>{link.label}</span>
                      </Link>
                    ))}
                  </div>
                )}

                {/* My Library Section */}
                {libraryLinks.length > 0 && (
                  <div className="pt-2">
                    <div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
                      My Library
                    </div>
                    {libraryLinks.map((link) => (
                      <Link
                        key={link.href}
                        href={link.href}
                        className={`flex items-center space-x-2 px-3 py-2 rounded-md text-base font-medium transition-colors ${
                          isActive(link.href)
                            ? 'text-orange-600 bg-orange-50'
                            : 'text-gray-700 hover:text-orange-600 hover:bg-gray-50'
                        }`}
                        onClick={() => setIsMobileMenuOpen(false)}
                      >
                        {link.icon}
                        <span>{link.label}</span>
                      </Link>
                    ))}
                  </div>
                )}

                {/* User Profile Section */}
                <hr className="my-2" />
                <div className="px-3 py-2">
                  <div className="flex items-center space-x-3 mb-3">
                    <div className="w-8 h-8 bg-orange-100 rounded-full flex items-center justify-center">
                      <User className="w-4 h-4 text-orange-600" />
                    </div>
                    <span className="text-base font-medium text-gray-900">{user.username}</span>
                  </div>
                  <Link
                    href="/profile"
                    className="block px-3 py-2 rounded-md text-base font-medium text-gray-700 hover:text-orange-600 hover:bg-gray-50 transition-colors"
                    onClick={() => setIsMobileMenuOpen(false)}
                  >
                    Profile Settings
                  </Link>
                  <button
                    onClick={handleSignOut}
                    className="block w-full text-left px-3 py-2 rounded-md text-base font-medium text-gray-700 hover:text-orange-600 hover:bg-gray-50 transition-colors"
                  >
                    Sign Out
                  </button>
                </div>
              </>
            )}

            {!user && (
              <>
                <hr className="my-2" />
                <Link
                  href="/auth/login"
                  className="block px-3 py-2 rounded-md text-base font-medium text-gray-700 hover:text-orange-600 hover:bg-gray-50 transition-colors"
                  onClick={() => setIsMobileMenuOpen(false)}
                >
                  Log In
                </Link>
                <Link
                  href="/auth/register"
                  className="block px-3 py-2 rounded-md text-base font-medium bg-orange-600 text-white hover:bg-orange-700 transition-colors"
                  onClick={() => setIsMobileMenuOpen(false)}
                >
                  Sign Up
                </Link>
              </>
            )}
          </div>
        </div>
      )}
    </header>
  );
}