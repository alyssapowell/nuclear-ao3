'use client';

import { useState, useEffect, useCallback, useMemo, useRef, useId } from 'react';
import { 
  getMyProfile, 
  updateProfile, 
  getPseudonyms, 
  createPseudonym,
  UserProfile, 
  UpdateProfileRequest,
  UserPseudonym,
  CreatePseudonymRequest 
} from '@/lib/api';

interface UserProfileSettingsProps {
  authToken?: string;
}

export default function UserProfileSettings({ authToken }: UserProfileSettingsProps) {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [pseudonyms, setPseudonyms] = useState<UserPseudonym[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'profile' | 'privacy' | 'pseudonyms'>('profile');
  const [announcements, setAnnouncements] = useState<string>('');
  
  // Form states
  const [formData, setFormData] = useState<UpdateProfileRequest>({});
  const [newPseudonym, setNewPseudonym] = useState<CreatePseudonymRequest>({
    name: '',
    description: ''
  });
  const [showPseudonymForm, setShowPseudonymForm] = useState(false);

  // Accessibility refs and IDs
  const componentId = useId();
  const tabPanelId = `${componentId}-tabpanel`;
  const tabListId = `${componentId}-tablist`;
  const liveRegionId = `${componentId}-live`;
  
  // Refs for focus management
  const profileTabRef = useRef<HTMLButtonElement>(null);
  const privacyTabRef = useRef<HTMLButtonElement>(null);
  const pseudonymsTabRef = useRef<HTMLButtonElement>(null);
  const firstFieldRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (authToken) {
      loadProfile();
      loadPseudonyms();
    }
  }, [authToken]); // Dependencies managed by useCallback

  const loadProfile = useCallback(async () => {
    try {
      setLoading(true);
      const response = await getMyProfile(authToken);
      setProfile(response.user);
      setFormData({
        display_name: response.user.display_name || '',
        bio: response.user.bio || '',
        location: response.user.location || '',
        website: response.user.website || '',
        preferred_categories: response.user.preferred_categories || [],
        preferred_tags: response.user.preferred_tags || [],
        email_notifications: response.user.email_notifications ?? true,
        show_adult_content: response.user.show_adult_content ?? false,
        allow_guest_downloads: response.user.allow_guest_downloads ?? true,
        default_work_privacy: response.user.default_work_privacy || 'public',
        disable_work_skins: response.user.disable_work_skins ?? false,
        hide_warnings: response.user.hide_warnings ?? false,
        minimize_warnings: response.user.minimize_warnings ?? true,
        show_stats: response.user.show_stats ?? true,
        allow_friend_requests: response.user.allow_friend_requests ?? true,
        show_bookmarks: response.user.show_bookmarks ?? true,
        show_works: response.user.show_works ?? true,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load profile');
    } finally {
      setLoading(false);
    }
  }, [authToken]);

  const loadPseudonyms = useCallback(async () => {
    try {
      const response = await getPseudonyms(authToken);
      setPseudonyms(response.pseudonyms || []);
    } catch (err) {
      console.error('Failed to load pseudonyms:', err);
    }
  }, [authToken]);

  const handleSaveProfile = useCallback(async () => {
    try {
      setSaving(true);
      setError(null);
      await updateProfile(formData, authToken);
      setSuccessMessage('Profile updated successfully!');
      setAnnouncements('Profile updated successfully!');
      setTimeout(() => setSuccessMessage(null), 3000);
      await loadProfile(); // Reload to get updated data
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update profile');
    } finally {
      setSaving(false);
    }
  }, [formData, authToken, loadProfile]);

  const handleCreatePseudonym = useCallback(async () => {
    if (!newPseudonym.name.trim()) {
      setError('Pseudonym name is required');
      return;
    }

    try {
      setSaving(true);
      setError(null);
      await createPseudonym(newPseudonym, authToken);
      setSuccessMessage('Pseudonym created successfully!');
      setAnnouncements('Pseudonym created successfully!');
      setTimeout(() => setSuccessMessage(null), 3000);
      setNewPseudonym({ name: '', description: '' });
      setShowPseudonymForm(false);
      await loadPseudonyms();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create pseudonym');
    } finally {
      setSaving(false);
    }
  }, [newPseudonym, authToken, loadPseudonyms]);

  const updateFormData = useCallback((field: keyof UpdateProfileRequest, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  }, []);

  // Enhanced tab navigation with keyboard support
  const handleTabKeyDown = useCallback((e: React.KeyboardEvent, tabId: string) => {
    const tabs = ['profile', 'privacy', 'pseudonyms'];
    const currentIndex = tabs.indexOf(tabId);
    
    switch (e.key) {
      case 'ArrowRight':
      case 'ArrowDown':
        e.preventDefault();
        const nextIndex = (currentIndex + 1) % tabs.length;
        setActiveTab(tabs[nextIndex] as any);
        setAnnouncements(`Switched to ${tabs[nextIndex]} tab`);
        // Focus next tab
        setTimeout(() => {
          const nextTab = [profileTabRef, privacyTabRef, pseudonymsTabRef][nextIndex];
          nextTab.current?.focus();
        }, 0);
        break;
        
      case 'ArrowLeft':
      case 'ArrowUp':
        e.preventDefault();
        const prevIndex = currentIndex === 0 ? tabs.length - 1 : currentIndex - 1;
        setActiveTab(tabs[prevIndex] as any);
        setAnnouncements(`Switched to ${tabs[prevIndex]} tab`);
        // Focus previous tab
        setTimeout(() => {
          const prevTab = [profileTabRef, privacyTabRef, pseudonymsTabRef][prevIndex];
          prevTab.current?.focus();
        }, 0);
        break;
        
      case 'Home':
        e.preventDefault();
        setActiveTab('profile');
        setAnnouncements('Switched to profile tab');
        profileTabRef.current?.focus();
        break;
        
      case 'End':
        e.preventDefault();
        setActiveTab('pseudonyms');
        setAnnouncements('Switched to pseudonyms tab');
        pseudonymsTabRef.current?.focus();
        break;
    }
  }, []);

  const handleTabClick = useCallback((tabId: 'profile' | 'privacy' | 'pseudonyms') => {
    setActiveTab(tabId);
    setAnnouncements(`Switched to ${tabId} tab`);
    
    // Focus first field in new tab
    setTimeout(() => {
      if (tabId === 'profile' && firstFieldRef.current) {
        firstFieldRef.current.focus();
      }
    }, 100);
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="loading-spinner mr-3"></div>
        <span>Loading profile settings...</span>
      </div>
    );
  }

  if (!profile) {
    return (
      <div className="text-center py-8">
        <p className="text-slate-600">Unable to load profile. Please try again.</p>
      </div>
    );
  }

  return (
    <main className="max-w-4xl mx-auto" role="main" aria-labelledby={`${componentId}-heading`}>
      {/* Live Region for Announcements */}
      <div
        id={liveRegionId}
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      >
        {announcements}
      </div>
      
      <header className="mb-6">
        <h1 id={`${componentId}-heading`} className="text-2xl font-bold text-slate-900 mb-2">Profile Settings</h1>
        <p className="text-slate-600">Manage your account information and privacy settings</p>
      </header>

      {/* Tab Navigation */}
      <section className="border-b border-slate-200 mb-6" aria-labelledby={`${componentId}-tabs-label`}>
        <h2 id={`${componentId}-tabs-label`} className="sr-only">Settings Categories</h2>
        <nav className="-mb-px flex space-x-8" role="tablist" aria-labelledby={`${componentId}-tabs-label`}>
          {[
            { id: 'profile', label: 'Profile', icon: '👤' },
            { id: 'privacy', label: 'Privacy', icon: '🔒' },
            { id: 'pseudonyms', label: 'Pseudonyms', icon: '🎭' },
          ].map((tab) => (
            <button
              key={tab.id}
              ref={tab.id === 'profile' ? profileTabRef : tab.id === 'privacy' ? privacyTabRef : pseudonymsTabRef}
              role="tab"
              tabIndex={activeTab === tab.id ? 0 : -1}
              aria-selected={activeTab === tab.id}
              aria-controls={`${tabPanelId}-${tab.id}`}
              id={`${componentId}-tab-${tab.id}`}
              onClick={() => handleTabClick(tab.id as any)}
              onKeyDown={(e) => handleTabKeyDown(e, tab.id)}
              className={`py-2 px-1 border-b-2 font-medium text-sm focus:outline-none focus:ring-2 focus:ring-orange-500 focus:ring-offset-2 ${
                activeTab === tab.id
                  ? 'border-orange-500 text-orange-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
              }`}
            >
              <span className="mr-2" aria-hidden="true">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </nav>
      </section>

      {error && (
        <section
          role="alert"
          className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg"
          aria-labelledby={`${componentId}-error-heading`}
        >
          <h3 id={`${componentId}-error-heading`} className="sr-only">Error</h3>
          <p className="text-red-800">{error}</p>
          <button
            onClick={() => setError(null)}
            className="text-red-600 text-sm underline mt-1 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2"
            aria-label="Dismiss error message"
          >
            Dismiss
          </button>
        </section>
      )}

      {successMessage && (
        <section
          role="status"
          className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg"
          aria-labelledby={`${componentId}-success-heading`}
        >
          <h3 id={`${componentId}-success-heading`} className="sr-only">Success</h3>
          <p className="text-green-800">{successMessage}</p>
        </section>
      )}

      {/* Profile Tab */}
      {activeTab === 'profile' && (
        <section
          role="tabpanel"
          id={`${tabPanelId}-profile`}
          aria-labelledby={`${componentId}-tab-profile`}
          className="card"
        >
          <header className="card-header">
            <h3 className="text-lg font-semibold">Profile Information</h3>
          </header>
          <form className="card-body space-y-6" onSubmit={(e) => { e.preventDefault(); handleSaveProfile(); }}>
            <fieldset className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <legend className="sr-only">Basic profile information</legend>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Username
                </label>
                <input
                  type="text"
                  value={profile.username}
                  disabled
                  className="form-input bg-slate-50 text-slate-500"
                  aria-describedby={`${componentId}-username-help`}
                />
                <p id={`${componentId}-username-help`} className="text-sm text-slate-500 mt-1">Username cannot be changed</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Display Name
                </label>
                <input
                  ref={firstFieldRef}
                  type="text"
                  value={formData.display_name || ''}
                  onChange={(e) => updateFormData('display_name', e.target.value)}
                  className="form-input"
                  placeholder="How should others see your name?"
                  aria-describedby={`${componentId}-display-name-help`}
                />
                <p id={`${componentId}-display-name-help`} className="sr-only">
                  Your display name is how others will see your name across the site
                </p>
              </div>
            </fieldset>

            <fieldset>
              <legend className="sr-only">Extended profile information</legend>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                Bio
              </label>
              <textarea
                value={formData.bio || ''}
                onChange={(e) => updateFormData('bio', e.target.value)}
                className="form-textarea"
                rows={4}
                placeholder="Tell others about yourself..."
                aria-describedby={`${componentId}-bio-help`}
              />
              <p id={`${componentId}-bio-help`} className="sr-only">
                Share information about yourself with other users
              </p>
            </fieldset>

            <fieldset className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <legend className="sr-only">Contact information</legend>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Location
                </label>
                <input
                  type="text"
                  value={formData.location || ''}
                  onChange={(e) => updateFormData('location', e.target.value)}
                  className="form-input"
                  placeholder="City, Country"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Website
                </label>
                <input
                  type="url"
                  value={formData.website || ''}
                  onChange={(e) => updateFormData('website', e.target.value)}
                  className="form-input"
                  placeholder="https://yourwebsite.com"
                />
              </div>
            </fieldset>

            <footer className="flex justify-end">
              <button
                type="submit"
                disabled={saving}
                className="btn btn-primary"
                aria-describedby={saving ? `${componentId}-saving-status` : undefined}
              >
                {saving ? (
                  <>
                    <div className="loading-spinner mr-2"></div>
                    Saving...
                  </>
                ) : (
                  'Save Changes'
                )}
              </button>
              {saving && (
                <div id={`${componentId}-saving-status`} className="sr-only" aria-live="polite">
                  Saving profile changes...
                </div>
              )}
            </footer>
          </form>
        </section>
      )}

      {/* Privacy Tab */}
      {activeTab === 'privacy' && (
        <section
          role="tabpanel"
          id={`${tabPanelId}-privacy`}
          aria-labelledby={`${componentId}-tab-privacy`}
          className="space-y-6"
        >
          <form className="card" onSubmit={(e) => { e.preventDefault(); handleSaveProfile(); }}>
            <header className="card-header">
              <h3 className="text-lg font-semibold">Privacy Settings</h3>
            </header>
            <div className="card-body space-y-6">
              <fieldset className="space-y-4">
                <legend className="sr-only">Content and interaction preferences</legend>
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-medium">Show Adult Content</h3>
                    <p className="text-sm text-slate-600">Display works with mature ratings</p>
                  </div>
                  <label className="switch" aria-label="Show Adult Content toggle">
                    <input
                      type="checkbox"
                      checked={formData.show_adult_content || false}
                      onChange={(e) => updateFormData('show_adult_content', e.target.checked)}
                      aria-describedby={`${componentId}-adult-content-desc`}
                    />
                    <span className="slider" aria-hidden="true"></span>
                  </label>
                  <p id={`${componentId}-adult-content-desc`} className="sr-only">
                    When enabled, works with mature ratings will be displayed in search results and on your dashboard
                  </p>
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-medium">Allow Friend Requests</h3>
                    <p className="text-sm text-slate-600">Let other users send you friend requests</p>
                  </div>
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={formData.allow_friend_requests ?? true}
                      onChange={(e) => updateFormData('allow_friend_requests', e.target.checked)}
                    />
                    <span className="slider"></span>
                  </label>
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-medium">Show Statistics</h3>
                    <p className="text-sm text-slate-600">Display work statistics on your profile</p>
                  </div>
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={formData.show_stats ?? true}
                      onChange={(e) => updateFormData('show_stats', e.target.checked)}
                    />
                    <span className="slider"></span>
                  </label>
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-medium">Show Works</h3>
                    <p className="text-sm text-slate-600">Display your works on your profile</p>
                  </div>
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={formData.show_works ?? true}
                      onChange={(e) => updateFormData('show_works', e.target.checked)}
                    />
                    <span className="slider"></span>
                  </label>
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-medium">Show Bookmarks</h3>
                    <p className="text-sm text-slate-600">Display your public bookmarks on your profile</p>
                  </div>
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={formData.show_bookmarks ?? true}
                      onChange={(e) => updateFormData('show_bookmarks', e.target.checked)}
                    />
                    <span className="slider"></span>
                  </label>
                </div>
              </fieldset>

              <fieldset>
                <legend className="sr-only">Default work privacy setting</legend>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Default Work Privacy
                </label>
                <select
                  value={formData.default_work_privacy || 'public'}
                  onChange={(e) => updateFormData('default_work_privacy', e.target.value)}
                  className="form-select"
                  aria-describedby={`${componentId}-privacy-help`}
                >
                  <option value="public">Public - Everyone can see</option>
                  <option value="users_only">Registered Users Only</option>
                  <option value="private">Private - Only you can see</option>
                </select>
                <p id={`${componentId}-privacy-help`} className="sr-only">
                  This setting will be the default privacy level for new works you post
                </p>
              </fieldset>

              <footer className="flex justify-end">
                <button
                  type="submit"
                  disabled={saving}
                  className="btn btn-primary"
                >
                  {saving ? (
                    <>
                      <div className="loading-spinner mr-2"></div>
                      Saving...
                    </>
                  ) : (
                    'Save Privacy Settings'
                  )}
                </button>
              </footer>
            </div>
          </form>
        </section>
      )}

      {/* Pseudonyms Tab */}
      {activeTab === 'pseudonyms' && (
        <section
          role="tabpanel"
          id={`${tabPanelId}-pseudonyms`}
          aria-labelledby={`${componentId}-tab-pseudonyms`}
          className="space-y-6"
        >
          <article className="card">
            <header className="card-header flex items-center justify-between">
              <h3 className="text-lg font-semibold">Pseudonyms</h3>
              <button
                onClick={() => setShowPseudonymForm(!showPseudonymForm)}
                className="btn btn-outline btn-sm"
                aria-expanded={showPseudonymForm}
                aria-controls={`${componentId}-pseudonym-form`}
              >
                {showPseudonymForm ? 'Cancel' : 'Add Pseudonym'}
              </button>
            </header>
            <div className="card-body">
              {showPseudonymForm && (
                <section
                  id={`${componentId}-pseudonym-form`}
                  className="mb-6 p-4 bg-slate-50 rounded-lg"
                  aria-labelledby={`${componentId}-pseudonym-form-heading`}
                >
                  <h4 id={`${componentId}-pseudonym-form-heading`} className="font-medium mb-4">Create New Pseudonym</h4>
                  <form className="space-y-4" onSubmit={(e) => { e.preventDefault(); handleCreatePseudonym(); }}>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-2">
                        Name *
                      </label>
                      <input
                        type="text"
                        value={newPseudonym.name}
                        onChange={(e) => setNewPseudonym(prev => ({ ...prev, name: e.target.value }))}
                        className="form-input"
                        placeholder="Enter pseudonym name"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-2">
                        Description
                      </label>
                      <textarea
                        value={newPseudonym.description || ''}
                        onChange={(e) => setNewPseudonym(prev => ({ ...prev, description: e.target.value }))}
                        className="form-textarea"
                        rows={3}
                        placeholder="Optional description for this pseudonym"
                      />
                    </div>
                    <div className="flex gap-2">
                      <button
                        type="submit"
                        disabled={saving || !newPseudonym.name.trim()}
                        className="btn btn-primary btn-sm"
                      >
                        {saving ? 'Creating...' : 'Create Pseudonym'}
                      </button>
                      <button
                        type="button"
                        onClick={() => setShowPseudonymForm(false)}
                        className="btn btn-outline btn-sm"
                      >
                        Cancel
                      </button>
                    </div>
                  </form>
                </section>
              )}

              <section className="space-y-4" aria-labelledby={`${componentId}-pseudonyms-list-heading`}>
                <h4 id={`${componentId}-pseudonyms-list-heading`} className="sr-only">Your pseudonyms</h4>
                {pseudonyms.length === 0 ? (
                  <div className="text-center py-8">
                    <p className="text-slate-600 mb-4">No pseudonyms yet.</p>
                    <p className="text-sm text-slate-500">
                      Pseudonyms let you write under different names while keeping your works organized.
                    </p>
                  </div>
                ) : (
                  pseudonyms.map((pseudonym) => (
                    <article key={pseudonym.id} className="border border-slate-200 rounded-lg p-4">
                      <div className="flex items-start justify-between">
                        <div>
                          <h3 className="font-medium flex items-center gap-2">
                            {pseudonym.name}
                            {pseudonym.is_default && (
                              <span className="text-xs px-2 py-1 bg-orange-100 text-orange-800 rounded">
                                Default
                              </span>
                            )}
                          </h3>
                          {pseudonym.description && (
                            <p className="text-sm text-slate-600 mt-1">{pseudonym.description}</p>
                          )}
                          <p className="text-xs text-slate-500 mt-2">
                            Created {new Date(pseudonym.created_at).toLocaleDateString()}
                          </p>
                        </div>
                      </div>
                    </article>
                  ))
                )}
              </section>
            </div>
          </article>
        </section>
      )}
    </main>
  );
}