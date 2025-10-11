'use client';

import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/Button';
import { updatePrivacySettings, getPrivacySettings, type PrivacySettingsRequest } from '@/lib/api';

interface PrivacySettings {
  // Content Filtering
  showExplicitContent: boolean;
  showMatureContent: boolean;
  hideUnratedContent: boolean;
  hideCreatorChoseNotToWarn: boolean;
  hideArchiveWarnings: string[];
  
  // Profile Privacy
  profileVisibility: 'public' | 'logged_in_only' | 'private';
  showStatsPublicly: boolean;
  showEmailToUsers: boolean;
  allowUserContact: boolean;
  
  // Reading & Interaction Privacy
  showReadingHistory: boolean;
  showBookmarksPublicly: boolean;
  enableCommentNotifications: boolean;
  enableKudosNotifications: boolean;
  
  // Work Posting Defaults
  defaultWorkPrivacy: 'public' | 'unrevealed' | 'anonymous';
  defaultCommentPolicy: 'open' | 'registered_users' | 'disabled';
  allowConcrit: boolean;
  
  // Email & Notifications
  emailOnComments: boolean;
  emailOnKudos: boolean;
  emailOnBookmarks: boolean;
  emailOnSubscriptions: boolean;
  emailDigestFrequency: 'none' | 'daily' | 'weekly';
}

interface PrivacyWizardProps {
  onComplete: (settings: PrivacySettings) => void;
  onSkip?: () => void;
  initialSettings?: Partial<PrivacySettings>;
  showSkipOption?: boolean;
  title?: string;
  subtitle?: string;
}

const ARCHIVE_WARNINGS = [
  'Graphic Depictions of Violence',
  'Major Character Death', 
  'Rape/Non-Con',
  'Underage'
];

const DEFAULT_SETTINGS: PrivacySettings = {
  // Content Filtering - Default to more restrictive
  showExplicitContent: false,
  showMatureContent: true,
  hideUnratedContent: false,
  hideCreatorChoseNotToWarn: false,
  hideArchiveWarnings: [],
  
  // Profile Privacy - Default to reasonable privacy
  profileVisibility: 'logged_in_only',
  showStatsPublicly: true,
  showEmailToUsers: false,
  allowUserContact: true,
  
  // Reading & Interaction Privacy - Default to private
  showReadingHistory: false,
  showBookmarksPublicly: false,
  enableCommentNotifications: true,
  enableKudosNotifications: true,
  
  // Work Posting Defaults - Default to public posting
  defaultWorkPrivacy: 'public',
  defaultCommentPolicy: 'registered_users',
  allowConcrit: false,
  
  // Email & Notifications - Default to moderate notifications
  emailOnComments: true,
  emailOnKudos: false,
  emailOnBookmarks: false,
  emailOnSubscriptions: true,
  emailDigestFrequency: 'weekly'
};

export default function PrivacyWizard({
  onComplete,
  onSkip,
  initialSettings = {},
  showSkipOption = true,
  title = "Privacy Setup",
  subtitle = "Configure your privacy and safety preferences"
}: PrivacyWizardProps) {
  const [currentStep, setCurrentStep] = useState(1);
  const [settings, setSettings] = useState<PrivacySettings>({
    ...DEFAULT_SETTINGS,
    ...initialSettings
  });
  const [saving, setSaving] = useState(false);
  const [loaded, setLoaded] = useState(false);

  const totalSteps = 5;

  // Load existing settings on mount - FIXED: Added proper dependencies and memoization
  useEffect(() => {
    let isMounted = true; // Prevent state updates if component unmounts
    
    const loadExistingSettings = async () => {
      if (loaded || !isMounted) return; // Prevent multiple loads
      if (typeof window === 'undefined') return; // Safe SSR guard
      
      try {
        let authToken: string | null = null;
        try {
          authToken = localStorage.getItem('auth_token');
        } catch (storageError) {
          console.warn('Could not access localStorage in PrivacyWizard:', storageError);
          return;
        }
        
        if (authToken) {
          // Try to load from API first
          try {
            const apiSettings = await getPrivacySettings(authToken);
            if (!isMounted) return; // Component unmounted during API call
            
            // Map API response to our settings format with null checks
            const mappedSettings = {
              // Content Filtering
              showExplicitContent: apiSettings?.show_explicit_content ?? DEFAULT_SETTINGS.showExplicitContent,
              showMatureContent: apiSettings?.show_mature_content ?? DEFAULT_SETTINGS.showMatureContent,
              hideUnratedContent: apiSettings?.hide_unrated_content ?? DEFAULT_SETTINGS.hideUnratedContent,
              hideCreatorChoseNotToWarn: apiSettings?.hide_creator_chose_not_to_warn ?? DEFAULT_SETTINGS.hideCreatorChoseNotToWarn,
              hideArchiveWarnings: apiSettings?.hide_archive_warnings ?? DEFAULT_SETTINGS.hideArchiveWarnings,
              
              // Profile Privacy
              profileVisibility: apiSettings?.profile_visibility ?? DEFAULT_SETTINGS.profileVisibility,
              showStatsPublicly: apiSettings?.show_stats_publicly ?? DEFAULT_SETTINGS.showStatsPublicly,
              showEmailToUsers: apiSettings?.show_email_to_users ?? DEFAULT_SETTINGS.showEmailToUsers,
              allowUserContact: apiSettings?.allow_user_contact ?? DEFAULT_SETTINGS.allowUserContact,
              
              // Reading & Interaction Privacy
              showReadingHistory: apiSettings?.show_reading_history ?? DEFAULT_SETTINGS.showReadingHistory,
              showBookmarksPublicly: apiSettings?.show_bookmarks_publicly ?? DEFAULT_SETTINGS.showBookmarksPublicly,
              enableCommentNotifications: apiSettings?.enable_comment_notifications ?? DEFAULT_SETTINGS.enableCommentNotifications,
              enableKudosNotifications: apiSettings?.enable_kudos_notifications ?? DEFAULT_SETTINGS.enableKudosNotifications,
              
              // Work Posting Defaults
              defaultWorkPrivacy: apiSettings?.default_work_privacy ?? DEFAULT_SETTINGS.defaultWorkPrivacy,
              defaultCommentPolicy: apiSettings?.default_comment_policy ?? DEFAULT_SETTINGS.defaultCommentPolicy,
              allowConcrit: apiSettings?.allow_concrit ?? DEFAULT_SETTINGS.allowConcrit,
              
              // Email & Notifications
              emailOnComments: apiSettings?.email_on_comments ?? DEFAULT_SETTINGS.emailOnComments,
              emailOnKudos: apiSettings?.email_on_kudos ?? DEFAULT_SETTINGS.emailOnKudos,
              emailOnBookmarks: apiSettings?.email_on_bookmarks ?? DEFAULT_SETTINGS.emailOnBookmarks,
              emailOnSubscriptions: apiSettings?.email_on_subscriptions ?? DEFAULT_SETTINGS.emailOnSubscriptions,
              emailDigestFrequency: apiSettings?.email_digest_frequency ?? DEFAULT_SETTINGS.emailDigestFrequency,
            };
            
            if (isMounted) {
              setSettings(prev => ({ ...prev, ...mappedSettings }));
            }
          } catch (apiError) {
            console.warn('API privacy settings not available, trying localStorage:', apiError);
            
            // Fallback to localStorage only if API fails
            try {
              const savedSettings = localStorage.getItem('user_privacy_settings');
              if (savedSettings && isMounted) {
                const parsedSettings = JSON.parse(savedSettings);
                setSettings(prev => ({ ...prev, ...parsedSettings }));
              }
            } catch (parseError) {
              console.warn('Could not parse saved privacy settings:', parseError);
            }
          }
        } else {
          // No auth token, try localStorage
          try {
            const savedSettings = localStorage.getItem('user_privacy_settings');
            if (savedSettings && isMounted) {
              const parsedSettings = JSON.parse(savedSettings);
              setSettings(prev => ({ ...prev, ...parsedSettings }));
            }
          } catch (parseError) {
            console.warn('Could not parse saved privacy settings:', parseError);
          }
        }
      } catch (error) {
        console.warn('Could not load existing privacy settings:', error);
      } finally {
        if (isMounted) {
          setLoaded(true);
        }
      }
    };
    
    // Only load once when component mounts
    if (!loaded) {
      loadExistingSettings();
    }
    
    // Cleanup function to prevent memory leaks
    return () => {
      isMounted = false;
    };
  }, []); // FIXED: Empty dependency array to run only once on mount

  const updateSetting = <K extends keyof PrivacySettings>(
    key: K, 
    value: PrivacySettings[K]
  ) => {
    setSettings(prev => ({ ...prev, [key]: value }));
  };

  const toggleArrayItem = (key: keyof PrivacySettings, item: string) => {
    const currentArray = settings[key] as string[];
    const newArray = currentArray.includes(item)
      ? currentArray.filter(i => i !== item)
      : [...currentArray, item];
    updateSetting(key, newArray as PrivacySettings[keyof PrivacySettings]);
  };

  const goToStep = (step: number) => {
    if (step >= 1 && step <= totalSteps) {
      setCurrentStep(step);
    }
  };

  const nextStep = () => goToStep(currentStep + 1);
  const prevStep = () => goToStep(currentStep - 1);

  const handleComplete = async () => {
    setSaving(true);
    try {
      // Save settings via API - Safe localStorage access
      let authToken: string | null = null;
      if (typeof window !== 'undefined') {
        try {
          authToken = localStorage.getItem('auth_token');
        } catch (error) {
          console.warn('Could not access localStorage for auth token:', error);
        }
      }
      if (authToken) {
        try {
          // Try dedicated privacy API first
          const privacyUpdate: PrivacySettingsRequest = {
            // Content Filtering
            show_explicit_content: settings.showExplicitContent,
            show_mature_content: settings.showMatureContent,
            hide_unrated_content: settings.hideUnratedContent,
            hide_creator_chose_not_to_warn: settings.hideCreatorChoseNotToWarn,
            hide_archive_warnings: settings.hideArchiveWarnings,
            
            // Profile Privacy
            profile_visibility: settings.profileVisibility === 'logged_in_only' ? 'logged_in_only' : 
                               settings.profileVisibility === 'private' ? 'private' : 'public',
            show_stats_publicly: settings.showStatsPublicly,
            show_email_to_users: settings.showEmailToUsers,
            allow_user_contact: settings.allowUserContact,
            
            // Reading & Interaction Privacy
            show_reading_history: settings.showReadingHistory,
            show_bookmarks_publicly: settings.showBookmarksPublicly,
            enable_comment_notifications: settings.enableCommentNotifications,
            enable_kudos_notifications: settings.enableKudosNotifications,
            
            // Work Posting Defaults
            default_work_privacy: settings.defaultWorkPrivacy,
            default_comment_policy: settings.defaultCommentPolicy,
            allow_concrit: settings.allowConcrit,
            
            // Email & Notifications
            email_on_comments: settings.emailOnComments,
            email_on_kudos: settings.emailOnKudos,
            email_on_bookmarks: settings.emailOnBookmarks,
            email_on_subscriptions: settings.emailOnSubscriptions,
            email_digest_frequency: settings.emailDigestFrequency,
          };
          
          await updatePrivacySettings(privacyUpdate, authToken);
          console.log('Privacy settings saved successfully via dedicated API');
        } catch (apiError) {
          console.warn('Dedicated privacy API not available, saving to local storage:', apiError);
          // Fallback: Save to localStorage for demo purposes
          if (typeof window !== 'undefined') {
            try {
              localStorage.setItem('user_privacy_settings', JSON.stringify(settings));
            } catch (error) {
              console.warn('Could not save privacy settings to localStorage:', error);
            }
          }
        }
      } else {
        // No auth token, save to localStorage for demo purposes
        if (typeof window !== 'undefined') {
          try {
            localStorage.setItem('user_privacy_settings', JSON.stringify(settings));
          } catch (error) {
            console.warn('Could not save privacy settings to localStorage:', error);
          }
        }
      }
      
      onComplete(settings);
    } catch (error) {
      console.error('Failed to save privacy settings:', error);
      // Still call onComplete to not block the user
      onComplete(settings);
    } finally {
      setSaving(false);
    }
  };

  const renderProgressBar = () => (
    <div className="mb-8">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium text-slate-700">
          Step {currentStep} of {totalSteps}
        </span>
        <span className="text-sm text-slate-500">
          {Math.round((currentStep / totalSteps) * 100)}% Complete
        </span>
      </div>
      <div className="w-full bg-slate-200 rounded-full h-2">
        <div 
          className="bg-orange-600 h-2 rounded-full transition-all duration-300 ease-out"
          style={{ width: `${(currentStep / totalSteps) * 100}%` }}
        ></div>
      </div>
    </div>
  );

  const renderStepIndicator = () => (
    <div className="flex justify-center mb-8">
      <div className="flex items-center space-x-4">
        {Array.from({ length: totalSteps }, (_, i) => {
          const stepNum = i + 1;
          const isActive = stepNum === currentStep;
          const isCompleted = stepNum < currentStep;
          
          return (
            <div key={stepNum} className="flex items-center">
              <div
                className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors ${
                  isCompleted
                    ? 'bg-green-600 text-white'
                    : isActive
                    ? 'bg-orange-600 text-white'
                    : 'bg-slate-200 text-slate-600'
                }`}
              >
                {isCompleted ? '✓' : stepNum}
              </div>
              {stepNum < totalSteps && (
                <div
                  className={`w-8 h-0.5 ${
                    stepNum < currentStep ? 'bg-green-600' : 'bg-slate-200'
                  }`}
                />
              )}
            </div>
          );
        })}
      </div>
    </div>
  );

  const renderToggleSwitch = (
    checked: boolean,
    onChange: (checked: boolean) => void,
    label: string,
    description?: string
  ) => (
    <div className="flex items-center justify-between">
      <div>
        <label className="text-sm font-medium text-slate-700">{label}</label>
        {description && <p className="text-xs text-slate-500">{description}</p>}
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
      </label>
    </div>
  );

  const renderStep1 = () => (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold text-slate-900 mb-3">Content Filtering</h2>
        <p className="text-slate-600">
          Choose what content you want to see based on ratings and warnings.
        </p>
      </div>

      <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
        <h3 className="font-semibold text-blue-900 mb-4">Content Visibility</h3>
        
        <div className="space-y-4">
          {renderToggleSwitch(
            settings.showExplicitContent,
            (checked) => updateSetting('showExplicitContent', checked),
            'Show Explicit Content',
            'Content rated Explicit (graphic sexual content)'
          )}

          {renderToggleSwitch(
            settings.showMatureContent,
            (checked) => updateSetting('showMatureContent', checked),
            'Show Mature Content',
            'Content rated Mature (adult themes, some sexual content)'
          )}

          {renderToggleSwitch(
            settings.hideUnratedContent,
            (checked) => updateSetting('hideUnratedContent', checked),
            'Hide Unrated Content',
            'Content without an assigned rating'
          )}
        </div>
      </div>

      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6">
        <h4 className="font-semibold text-yellow-900 mb-3">Archive Warnings to Hide</h4>
        <p className="text-yellow-800 text-sm mb-4">
          Select warnings for content you prefer not to see.
        </p>
        
        <div className="space-y-3">
          {ARCHIVE_WARNINGS.map(warning => (
            <label key={warning} className="flex items-center">
              <input
                type="checkbox"
                checked={settings.hideArchiveWarnings.includes(warning)}
                onChange={() => toggleArrayItem('hideArchiveWarnings', warning)}
                className="mr-3 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
              />
              <span className="text-sm text-slate-700">{warning}</span>
            </label>
          ))}
        </div>
      </div>
    </div>
  );

  const renderStep2 = () => (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold text-slate-900 mb-3">Profile Privacy</h2>
        <p className="text-slate-600">
          Control who can see your profile and how much information is visible.
        </p>
      </div>

      <div className="bg-white border border-slate-200 rounded-lg p-6">
        <h3 className="font-semibold text-slate-900 mb-4">Profile Visibility</h3>
        
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium text-slate-700 mb-3 block">
              Who can view your profile?
            </label>
            <div className="space-y-3">
              {[
                { value: 'public', label: 'Everyone (including guests)', desc: 'Your profile is visible to anyone on the internet' },
                { value: 'logged_in_only', label: 'Logged-in users only', desc: 'Only users with accounts can see your profile' },
                { value: 'private', label: 'Private', desc: 'Only you can see your profile' }
              ].map(option => (
                <label key={option.value} className="flex items-start cursor-pointer">
                  <input
                    type="radio"
                    name="profileVisibility"
                    value={option.value}
                    checked={settings.profileVisibility === option.value}
                    onChange={(e) => updateSetting('profileVisibility', e.target.value as any)}
                    className="mt-1 mr-3 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300"
                  />
                  <div>
                    <div className="text-sm font-medium text-slate-700">{option.label}</div>
                    <div className="text-xs text-slate-500">{option.desc}</div>
                  </div>
                </label>
              ))}
            </div>
          </div>

          <div className="border-t border-slate-200 pt-4 space-y-4">
            {renderToggleSwitch(
              settings.showStatsPublicly,
              (checked) => updateSetting('showStatsPublicly', checked),
              'Show statistics publicly',
              'Work count, kudos received, etc.'
            )}

            {renderToggleSwitch(
              settings.allowUserContact,
              (checked) => updateSetting('allowUserContact', checked),
              'Allow user contact',
              'Let other users send you messages'
            )}
          </div>
        </div>
      </div>
    </div>
  );

  const renderStep3 = () => (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold text-slate-900 mb-3">Reading & Interaction Privacy</h2>
        <p className="text-slate-600">
          Choose what reading activity and interactions others can see.
        </p>
      </div>

      <div className="bg-white border border-slate-200 rounded-lg p-6">
        <div className="space-y-6">
          {renderToggleSwitch(
            settings.showReadingHistory,
            (checked) => updateSetting('showReadingHistory', checked),
            'Show reading history',
            'Let others see what works you\'ve recently visited'
          )}

          {renderToggleSwitch(
            settings.showBookmarksPublicly,
            (checked) => updateSetting('showBookmarksPublicly', checked),
            'Show bookmarks publicly',
            'Display your bookmarks on your public profile'
          )}

          <div className="border-t border-slate-200 pt-4">
            <h4 className="font-medium text-slate-700 mb-4">Interaction Notifications</h4>
            
            <div className="space-y-4">
              {renderToggleSwitch(
                settings.enableCommentNotifications,
                (checked) => updateSetting('enableCommentNotifications', checked),
                'Comment notifications',
                'Get notified when someone comments on your works'
              )}

              {renderToggleSwitch(
                settings.enableKudosNotifications,
                (checked) => updateSetting('enableKudosNotifications', checked),
                'Kudos notifications',
                'Get notified when someone gives your work kudos'
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  const renderStep4 = () => (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold text-slate-900 mb-3">Work Posting Defaults</h2>
        <p className="text-slate-600">
          Set your preferred defaults for when you post new works.
        </p>
      </div>

      <div className="bg-white border border-slate-200 rounded-lg p-6">
        <div className="space-y-6">
          <div>
            <label className="text-sm font-medium text-slate-700 mb-3 block">
              Default work privacy
            </label>
            <div className="space-y-3">
              {[
                { value: 'public', label: 'Public', desc: 'Anyone can find and read your work' },
                { value: 'unrevealed', label: 'Unrevealed', desc: 'Work is posted but hidden until you reveal it' },
                { value: 'anonymous', label: 'Anonymous', desc: 'Work is posted without your name attached' }
              ].map(option => (
                <label key={option.value} className="flex items-start cursor-pointer">
                  <input
                    type="radio"
                    name="defaultWorkPrivacy"
                    value={option.value}
                    checked={settings.defaultWorkPrivacy === option.value}
                    onChange={(e) => updateSetting('defaultWorkPrivacy', e.target.value as any)}
                    className="mt-1 mr-3 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300"
                  />
                  <div>
                    <div className="text-sm font-medium text-slate-700">{option.label}</div>
                    <div className="text-xs text-slate-500">{option.desc}</div>
                  </div>
                </label>
              ))}
            </div>
          </div>

          <div className="border-t border-slate-200 pt-4">
            <label className="text-sm font-medium text-slate-700 mb-3 block">
              Default comment policy
            </label>
            <div className="space-y-3">
              {[
                { value: 'open', label: 'Open to all', desc: 'Anyone can comment, including guests' },
                { value: 'registered_users', label: 'Registered users only', desc: 'Only users with accounts can comment' },
                { value: 'disabled', label: 'Comments disabled', desc: 'No one can comment on your works' }
              ].map(option => (
                <label key={option.value} className="flex items-start cursor-pointer">
                  <input
                    type="radio"
                    name="defaultCommentPolicy"
                    value={option.value}
                    checked={settings.defaultCommentPolicy === option.value}
                    onChange={(e) => updateSetting('defaultCommentPolicy', e.target.value as any)}
                    className="mt-1 mr-3 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300"
                  />
                  <div>
                    <div className="text-sm font-medium text-slate-700">{option.label}</div>
                    <div className="text-xs text-slate-500">{option.desc}</div>
                  </div>
                </label>
              ))}
            </div>
          </div>

          <div className="border-t border-slate-200 pt-4">
            {renderToggleSwitch(
              settings.allowConcrit,
              (checked) => updateSetting('allowConcrit', checked),
              'Allow constructive criticism',
              'Signal that you welcome helpful feedback on your writing'
            )}
          </div>
        </div>
      </div>
    </div>
  );

  const renderStep5 = () => (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold text-slate-900 mb-3">Email & Notifications</h2>
        <p className="text-slate-600">
          Choose how and when you want to receive email notifications.
        </p>
      </div>

      <div className="bg-white border border-slate-200 rounded-lg p-6">
        <div className="space-y-6">
          <div>
            <h4 className="font-medium text-slate-700 mb-4">Email me when:</h4>
            
            <div className="space-y-4">
              {renderToggleSwitch(
                settings.emailOnComments,
                (checked) => updateSetting('emailOnComments', checked),
                'Someone comments on my work'
              )}

              {renderToggleSwitch(
                settings.emailOnKudos,
                (checked) => updateSetting('emailOnKudos', checked),
                'Someone gives my work kudos'
              )}

              {renderToggleSwitch(
                settings.emailOnBookmarks,
                (checked) => updateSetting('emailOnBookmarks', checked),
                'Someone bookmarks my work'
              )}

              {renderToggleSwitch(
                settings.emailOnSubscriptions,
                (checked) => updateSetting('emailOnSubscriptions', checked),
                'A subscribed author posts new work'
              )}
            </div>
          </div>

          <div className="border-t border-slate-200 pt-4">
            <label className="text-sm font-medium text-slate-700 mb-3 block">
              Email digest frequency
            </label>
            <div className="space-y-3">
              {[
                { value: 'none', label: 'No digest emails', desc: 'Only individual notifications' },
                { value: 'daily', label: 'Daily digest', desc: 'Summary of activity once per day' },
                { value: 'weekly', label: 'Weekly digest', desc: 'Summary of activity once per week' }
              ].map(option => (
                <label key={option.value} className="flex items-start cursor-pointer">
                  <input
                    type="radio"
                    name="emailDigestFrequency"
                    value={option.value}
                    checked={settings.emailDigestFrequency === option.value}
                    onChange={(e) => updateSetting('emailDigestFrequency', e.target.value as any)}
                    className="mt-1 mr-3 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300"
                  />
                  <div>
                    <div className="text-sm font-medium text-slate-700">{option.label}</div>
                    <div className="text-xs text-slate-500">{option.desc}</div>
                  </div>
                </label>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className="bg-green-50 border border-green-200 rounded-lg p-6">
        <h3 className="font-semibold text-green-900 mb-3">🎉 You're all set!</h3>
        <p className="text-green-800 text-sm">
          Your privacy preferences will be saved. You can always change these settings later 
          from your profile page.
        </p>
      </div>
    </div>
  );

  const renderNavigationButtons = () => (
    <div className="flex justify-between items-center mt-8 pt-6 border-t border-slate-200">
      <div className="flex items-center gap-4">
        {currentStep > 1 && (
          <Button variant="outline" onClick={prevStep}>
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            Previous
          </Button>
        )}
        
        {showSkipOption && onSkip && (
          <Button 
            variant="outline" 
            onClick={onSkip}
            className="text-slate-600"
          >
            Skip for Now
          </Button>
        )}
      </div>

      <div className="flex items-center gap-4">
        {currentStep < totalSteps ? (
          <Button onClick={nextStep}>
            Continue
            <svg className="w-4 h-4 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </Button>
        ) : (
          <Button 
            onClick={handleComplete}
            disabled={saving}
            className="bg-green-600 hover:bg-green-700"
          >
            {saving ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                Saving...
              </>
            ) : (
              <>
                Complete Setup
                <svg className="w-4 h-4 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </>
            )}
          </Button>
        )}
      </div>
    </div>
  );

  const renderCurrentStep = () => {
    switch (currentStep) {
      case 1: return renderStep1();
      case 2: return renderStep2();
      case 3: return renderStep3();
      case 4: return renderStep4();
      case 5: return renderStep5();
      default: return renderStep1();
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4">
      {/* Header */}
      <div className="text-center mb-8">
        <h1 className="text-3xl font-bold text-slate-900 mb-2">{title}</h1>
        <p className="text-slate-600">{subtitle}</p>
      </div>

      {/* Progress Indicator */}
      {renderStepIndicator()}
      
      {/* Progress Bar */}
      {renderProgressBar()}

      {/* Main Content */}
      <div className="bg-white rounded-lg shadow-lg border border-slate-200 p-8">
        {renderCurrentStep()}
        {renderNavigationButtons()}
      </div>
    </div>
  );
}