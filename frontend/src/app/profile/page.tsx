'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { getMyProfile, getPseudonyms, type UserPseudonym } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import AuthGuard from '@/components/auth/AuthGuard';
import Link from 'next/link';

interface UserProfile {
  id: string;
  username: string;
  email: string;
  display_name?: string;
  bio?: string;
  profile_image_url?: string;
  header_image_url?: string;
  location?: string;
  website?: string;
  date_of_birth?: string;
  preferred_categories?: string[];
  preferred_tags?: string[];
  email_notifications?: boolean;
  created_at: string;
  updated_at: string;
}

export default function ProfilePage() {
  const router = useRouter();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [pseudonyms, setPseudonyms] = useState<UserPseudonym[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchProfileData();
  }, []);

  const fetchProfileData = async () => {
    try {
      const authToken = localStorage.getItem('auth_token') || localStorage.getItem('token');
      if (!authToken) {
        router.push('/auth/login?redirect=/profile');
        return;
      }

      // Fetch profile and pseudonyms in parallel
      const [profileData, pseudsData] = await Promise.all([
        getMyProfile(authToken),
        getPseudonyms(authToken).catch(() => ({ pseudonyms: [] })) // Don't fail if pseuds aren't available
      ]);

      setProfile(profileData.user || profileData.profile || profileData);
      setPseudonyms(pseudsData.pseudonyms || []);

    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load profile');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  const getInitials = (name: string) => {
    return name.split(' ').map(word => word.charAt(0).toUpperCase()).slice(0, 2).join('');
  };

  return (
    <AuthGuard>
      <div className="max-w-4xl mx-auto px-4 py-8">
        {loading ? (
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-orange-600 mx-auto mb-4"></div>
            <p className="text-slate-600">Loading profile...</p>
          </div>
        ) : error ? (
          <div className="bg-red-50 border border-red-200 text-red-800 p-4 rounded-lg mb-6">
            {error}
          </div>
        ) : profile ? (
          <>
            {/* Profile Header */}
            <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 mb-6">
              <div className="flex items-start justify-between mb-6">
                <div className="flex items-start space-x-6">
                  {/* Avatar */}
                  <div className="flex-shrink-0">
                    {profile.profile_image_url ? (
                      <img
                        src={profile.profile_image_url}
                        alt={`${profile.username}'s avatar`}
                        className="w-20 h-20 rounded-full object-cover"
                      />
                    ) : (
                      <div className="w-20 h-20 bg-gradient-to-br from-orange-500 to-purple-600 rounded-full flex items-center justify-center text-white text-2xl font-bold">
                        {getInitials(profile.display_name || profile.username)}
                      </div>
                    )}
                  </div>

                  {/* Profile Info */}
                  <div className="flex-1">
                    <h1 className="text-3xl font-bold text-slate-900 mb-2">
                      {profile.display_name || profile.username}
                    </h1>
                    {profile.display_name && (
                      <p className="text-slate-600 mb-2">@{profile.username}</p>
                    )}
                    
                    {profile.bio && (
                      <p className="text-slate-700 mb-4 leading-relaxed">{profile.bio}</p>
                    )}

                    <div className="flex flex-wrap gap-4 text-sm text-slate-500">
                      {profile.location && (
                        <span className="flex items-center">
                          📍 {profile.location}
                        </span>
                      )}
                      {profile.website && (
                        <a
                          href={profile.website}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex items-center text-orange-600 hover:text-orange-700 transition-colors"
                        >
                          🔗 Website
                        </a>
                      )}
                      <span className="flex items-center">
                        📅 Joined {formatDate(profile.created_at)}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Edit Button */}
                <Link href="/profile/edit">
                  <Button variant="outline">
                    <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                    </svg>
                    Edit Profile
                  </Button>
                </Link>
              </div>

              {/* Preferences */}
              {(profile.preferred_categories?.length || profile.preferred_tags?.length) && (
                <div className="border-t border-slate-200 pt-4">
                  <h3 className="text-sm font-medium text-slate-700 mb-3">Content Preferences</h3>
                  
                  {profile.preferred_categories && profile.preferred_categories.length > 0 && (
                    <div className="mb-3">
                      <span className="text-sm text-slate-600 mr-2">Categories:</span>
                      <div className="inline-flex flex-wrap gap-1">
                        {profile.preferred_categories.map(category => (
                          <span 
                            key={category}
                            className="px-2 py-1 bg-orange-100 text-orange-800 text-xs rounded"
                          >
                            {category}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {profile.preferred_tags && profile.preferred_tags.length > 0 && (
                    <div>
                      <span className="text-sm text-slate-600 mr-2">Preferred Tags:</span>
                      <div className="inline-flex flex-wrap gap-1">
                        {profile.preferred_tags.slice(0, 10).map(tag => (
                          <span 
                            key={tag}
                            className="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded"
                          >
                            {tag}
                          </span>
                        ))}
                        {profile.preferred_tags.length > 10 && (
                          <span className="px-2 py-1 bg-slate-100 text-slate-600 text-xs rounded">
                            +{profile.preferred_tags.length - 10} more
                          </span>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Pseudonyms Section */}
            <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 mb-6">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xl font-semibold text-slate-900">Pseudonyms</h2>
                <Link href="/profile/pseudonyms">
                  <Button variant="outline" size="sm">
                    <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                    </svg>
                    Manage Pseudonyms
                  </Button>
                </Link>
              </div>

              {pseudonyms.length > 0 ? (
                <div className="space-y-3">
                  {pseudonyms.map(pseud => (
                    <div key={pseud.id} className="flex items-center justify-between p-3 border border-slate-200 rounded-lg">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-slate-900">{pseud.name}</span>
                          {pseud.is_default && (
                            <span className="px-2 py-1 bg-green-100 text-green-800 text-xs rounded">
                              Default
                            </span>
                          )}
                        </div>
                        {pseud.description && (
                          <p className="text-sm text-slate-600 mt-1">{pseud.description}</p>
                        )}
                      </div>
                      <span className="text-xs text-slate-500">
                        Created {formatDate(pseud.created_at)}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8">
                  <div className="text-slate-400 mb-4">
                    <svg className="w-12 h-12 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                  </div>
                  <p className="text-slate-600 mb-4">No pseudonyms created yet.</p>
                  <Link href="/profile/pseudonyms">
                    <Button>Create Your First Pseudonym</Button>
                  </Link>
                </div>
              )}
            </div>

            {/* Quick Actions */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
              <Link href="/dashboard" className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 hover:border-orange-300 transition-colors">
                <div className="text-center">
                  <svg className="w-8 h-8 mx-auto mb-3 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2H5a2 2 0 00-2-2z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 5a2 2 0 012-2h4a2 2 0 012 2v6a2 2 0 01-2 2H10a2 2 0 01-2-2V5z" />
                  </svg>
                  <h3 className="font-semibold text-slate-900 mb-2">Dashboard</h3>
                  <p className="text-sm text-slate-600">View your works and statistics</p>
                </div>
              </Link>
              
              <Link href="/works/new" className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 hover:border-blue-300 transition-colors">
                <div className="text-center">
                  <svg className="w-8 h-8 mx-auto mb-3 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  <h3 className="font-semibold text-slate-900 mb-2">Post New Work</h3>
                  <p className="text-sm text-slate-600">Share your latest creation</p>
                </div>
              </Link>
              
              <Link href="/bookmarks" className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 hover:border-green-300 transition-colors">
                <div className="text-center">
                  <svg className="w-8 h-8 mx-auto mb-3 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
                  </svg>
                  <h3 className="font-semibold text-slate-900 mb-2">My Bookmarks</h3>
                  <p className="text-sm text-slate-600">View your saved works</p>
                </div>
              </Link>
              
              <Link href="/profile/privacy" className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 hover:border-purple-300 transition-colors">
                <div className="text-center">
                  <svg className="w-8 h-8 mx-auto mb-3 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                  <h3 className="font-semibold text-slate-900 mb-2">Privacy Settings</h3>
                  <p className="text-sm text-slate-600">Update your privacy preferences</p>
                </div>
              </Link>
            </div>
          </>
        ) : (
          <div className="text-center py-8">
            <p className="text-slate-600">Profile not found.</p>
          </div>
        )}
      </div>
    </AuthGuard>
  );
}