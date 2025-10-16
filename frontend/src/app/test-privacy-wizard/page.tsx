'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/Button';

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

const ARCHIVE_WARNINGS = [
  'Graphic Depictions of Violence',
  'Major Character Death', 
  'Rape/Non-Con',
  'Underage'
];

export default function TestPrivacyWizardPage() {
  const [currentStep, setCurrentStep] = useState(1);
  const [settings, setSettings] = useState<PrivacySettings>({
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
  });

  const totalSteps = 5;

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

  const renderStep1 = () => (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold text-slate-900 mb-3">Welcome to Nuclear AO3!</h2>
        <p className="text-slate-600 text-lg">
          Let's set up your privacy preferences to create a safe and comfortable experience.
        </p>
        <p className="text-slate-500 mt-2">
          You can always change these settings later in your profile.
        </p>
      </div>

      <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
        <h3 className="font-semibold text-blue-900 mb-3">Content Filtering</h3>
        <p className="text-blue-800 text-sm mb-4">
          Choose what content you want to see based on ratings and warnings.
        </p>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium text-slate-700">Show Explicit Content</label>
              <p className="text-xs text-slate-500">Content rated Explicit (graphic sexual content)</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.showExplicitContent}
                onChange={(e) => updateSetting('showExplicitContent', e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium text-slate-700">Show Mature Content</label>
              <p className="text-xs text-slate-500">Content rated Mature (adult themes, some sexual content)</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.showMatureContent}
                onChange={(e) => updateSetting('showMatureContent', e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium text-slate-700">Hide Unrated Content</label>
              <p className="text-xs text-slate-500">Content without an assigned rating</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.hideUnratedContent}
                onChange={(e) => updateSetting('hideUnratedContent', e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
            </label>
          </div>
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

          <div className="border-t border-slate-200 pt-4">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium text-slate-700">Show statistics publicly</label>
                  <p className="text-xs text-slate-500">Work count, kudos received, etc.</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.showStatsPublicly}
                    onChange={(e) => updateSetting('showStatsPublicly', e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
                </label>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium text-slate-700">Allow user contact</label>
                  <p className="text-xs text-slate-500">Let other users send you messages</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.allowUserContact}
                    onChange={(e) => updateSetting('allowUserContact', e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
                </label>
              </div>
            </div>
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
          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium text-slate-700">Show reading history</label>
              <p className="text-xs text-slate-500">Let others see what works you've recently visited</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.showReadingHistory}
                onChange={(e) => updateSetting('showReadingHistory', e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium text-slate-700">Show bookmarks publicly</label>
              <p className="text-xs text-slate-500">Display your bookmarks on your public profile</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={settings.showBookmarksPublicly}
                onChange={(e) => updateSetting('showBookmarksPublicly', e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
            </label>
          </div>

          <div className="border-t border-slate-200 pt-4">
            <h4 className="font-medium text-slate-700 mb-4">Interaction Notifications</h4>
            
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium text-slate-700">Comment notifications</label>
                  <p className="text-xs text-slate-500">Get notified when someone comments on your works</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.enableCommentNotifications}
                    onChange={(e) => updateSetting('enableCommentNotifications', e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
                </label>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium text-slate-700">Kudos notifications</label>
                  <p className="text-xs text-slate-500">Get notified when someone gives your work kudos</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.enableKudosNotifications}
                    onChange={(e) => updateSetting('enableKudosNotifications', e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
                </label>
              </div>
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
            <div className="flex items-center justify-between">
              <div>
                <label className="text-sm font-medium text-slate-700">Allow constructive criticism</label>
                <p className="text-xs text-slate-500">Signal that you welcome helpful feedback on your writing</p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={settings.allowConcrit}
                  onChange={(e) => updateSetting('allowConcrit', e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
              </label>
            </div>
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
              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium text-slate-700">Someone comments on my work</label>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.emailOnComments}
                    onChange={(e) => updateSetting('emailOnComments', e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
                </label>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium text-slate-700">Someone gives my work kudos</label>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.emailOnKudos}
                    onChange={(e) => updateSetting('emailOnKudos', e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
                </label>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium text-slate-700">Someone bookmarks my work</label>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.emailOnBookmarks}
                    onChange={(e) => updateSetting('emailOnBookmarks', e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
                </label>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium text-slate-700">A subscribed author posts new work</label>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.emailOnSubscriptions}
                    onChange={(e) => updateSetting('emailOnSubscriptions', e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
                </label>
              </div>
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
          Your privacy preferences have been configured. You can always change these settings later 
          from your profile page. Welcome to Nuclear AO3!
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
        
        <Button 
          variant="outline" 
          onClick={() => alert('Settings saved as draft!')}
          className="text-slate-600"
        >
          Save for Later
        </Button>
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
            onClick={() => alert('Privacy settings completed! Redirecting to dashboard...')}
            className="bg-green-600 hover:bg-green-700"
          >
            Complete Setup
            <svg className="w-4 h-4 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
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
    <div className="min-h-screen bg-slate-50 py-8">
      <div className="max-w-4xl mx-auto px-4">
        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-slate-900 mb-2">Privacy Setup Wizard</h1>
          <p className="text-slate-600">Preview of the new user onboarding experience</p>
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

        {/* Debug Info */}
        <div className="mt-8 p-4 bg-slate-100 rounded-lg">
          <details>
            <summary className="text-sm font-medium text-slate-700 cursor-pointer mb-2">
              Debug: Current Settings (Click to expand)
            </summary>
            <pre className="text-xs text-slate-600 bg-white p-3 rounded border overflow-auto">
              {JSON.stringify(settings, null, 2)}
            </pre>
          </details>
        </div>
      </div>
    </div>
  );
}