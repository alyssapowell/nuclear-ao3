import { useState } from 'react';
import { useReaderPreferences } from '@/hooks/useReaderPreferences';
import { READER_PRESETS } from '@/types/reader';
import { Switch, Menu, Transition } from '@headlessui/react';
import { Fragment } from 'react';

interface ReaderControlsProps {
  className?: string;
}

export default function ReaderControls({ className = '' }: ReaderControlsProps) {
  const { preferences, updatePreferences, applyPreset, toggleReaderMode } = useReaderPreferences();
  const [showSettings, setShowSettings] = useState(false);

  return (
    <div className={`reader-controls ${className}`}>
      {/* Main Reader Toggle */}
      <div className="flex items-center gap-3 mb-4">
        <Switch
          checked={preferences.readerMode}
          onChange={toggleReaderMode}
          className={`${
            preferences.readerMode ? 'bg-orange-600' : 'bg-slate-200'
          } relative inline-flex h-9 w-16 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-500 focus-visible:ring-opacity-75`}
        >
          <span className="sr-only">Toggle reader mode</span>
          <span
            aria-hidden="true"
            className={`${
              preferences.readerMode ? 'translate-x-7' : 'translate-x-0'
            } pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out mt-1 ml-1`}
          />
        </Switch>
        
        <span className="text-sm font-medium text-slate-700">
          {preferences.readerMode ? 'Reader Mode ON' : 'Reader Mode OFF'}
        </span>

        <Menu as="div" className="relative">
          <Menu.Button className="btn btn-outline btn-sm">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            Settings
          </Menu.Button>
          
          <Transition
            as={Fragment}
            enter="transition ease-out duration-100"
            enterFrom="transform opacity-0 scale-95"
            enterTo="transform opacity-100 scale-100"
            leave="transition ease-in duration-75"
            leaveFrom="transform opacity-100 scale-100"
            leaveTo="transform opacity-0 scale-95"
          >
            <Menu.Items className="absolute right-0 mt-2 w-80 origin-top-right divide-y divide-slate-100 rounded-md bg-white shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none z-50">
              <div className="px-1 py-1">
                <Menu.Item>
                  <button
                    onClick={() => setShowSettings(!showSettings)}
                    className="group flex w-full items-center rounded-md px-2 py-2 text-sm text-slate-700 hover:bg-slate-100"
                  >
                    Advanced Settings
                  </button>
                </Menu.Item>
              </div>
            </Menu.Items>
          </Transition>
        </Menu>
      </div>

      {/* Quick Presets */}
      <div className="flex flex-wrap gap-2 mb-4">
        {Object.keys(READER_PRESETS).map((presetName) => (
          <button
            key={presetName}
            onClick={() => applyPreset(presetName)}
            className="btn btn-outline btn-xs"
            title={`Apply ${presetName} preset`}
          >
            {presetName.replace('-', ' ')}
          </button>
        ))}
      </div>

      {/* Detailed Settings Panel */}
      {showSettings && (
        <div className="reader-settings-panel">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            
            {/* Theme Settings */}
            <div>
              <h4 className="font-semibold text-slate-900 mb-3">Theme</h4>
              <div className="space-y-2">
                {['light', 'dark', 'sepia', 'high-contrast'].map((theme) => (
                  <label key={theme} className="flex items-center">
                    <input
                      type="radio"
                      name="theme"
                      value={theme}
                      checked={preferences.theme === theme}
                      onChange={(e) => updatePreferences({ theme: e.target.value as 'light' | 'dark' | 'sepia' | 'high-contrast' })}
                      className="form-radio mr-3"
                    />
                    <span className="text-sm text-slate-700 capitalize">
                      {theme.replace('-', ' ')}
                    </span>
                  </label>
                ))}
              </div>
            </div>

            {/* Typography Settings */}
            <div>
              <h4 className="font-semibold text-slate-900 mb-3">Typography</h4>
              
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Font Family
                  </label>
                  <select
                    value={preferences.fontFamily}
                    onChange={(e) => updatePreferences({ fontFamily: e.target.value as 'system' | 'serif' | 'sans-serif' | 'dyslexic' | 'hyperlegible' })}
                    className="form-select text-sm"
                  >
                    <option value="system">System Default</option>
                    <option value="serif">Serif (Georgia)</option>
                    <option value="sans-serif">Sans-serif (Arial)</option>
                    <option value="dyslexic">Dyslexia-friendly</option>
                    <option value="hyperlegible">High Legibility</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Font Size: {preferences.fontSize}px
                  </label>
                  <input
                    type="range"
                    min="12"
                    max="28"
                    value={preferences.fontSize}
                    onChange={(e) => updatePreferences({ fontSize: parseInt(e.target.value) })}
                    className="w-full"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Line Height: {preferences.lineHeight}
                  </label>
                  <input
                    type="range"
                    min="1.2"
                    max="2.5"
                    step="0.1"
                    value={preferences.lineHeight}
                    onChange={(e) => updatePreferences({ lineHeight: parseFloat(e.target.value) })}
                    className="w-full"
                  />
                </div>
              </div>
            </div>

            {/* Layout Settings */}
            <div>
              <h4 className="font-semibold text-slate-900 mb-3">Layout</h4>
              
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Content Width: {preferences.contentWidth}%
                  </label>
                  <input
                    type="range"
                    min="60"
                    max="100"
                    value={preferences.contentWidth}
                    onChange={(e) => updatePreferences({ contentWidth: parseInt(e.target.value) })}
                    className="w-full"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Margin
                  </label>
                  <select
                    value={preferences.margin}
                    onChange={(e) => updatePreferences({ margin: e.target.value as 'compact' | 'comfortable' | 'spacious' })}
                    className="form-select text-sm"
                  >
                    <option value="compact">Compact</option>
                    <option value="comfortable">Comfortable</option>
                    <option value="spacious">Spacious</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Text Alignment
                  </label>
                  <select
                    value={preferences.textAlign}
                    onChange={(e) => updatePreferences({ textAlign: e.target.value as 'left' | 'justify' })}
                    className="form-select text-sm"
                  >
                    <option value="left">Left Aligned</option>
                    <option value="justify">Justified</option>
                  </select>
                </div>
              </div>
            </div>

            {/* Accessibility Settings */}
            <div>
              <h4 className="font-semibold text-slate-900 mb-3">Accessibility</h4>
              
              <div className="space-y-3">
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={preferences.highContrast}
                    onChange={(e) => updatePreferences({ highContrast: e.target.checked })}
                    className="form-checkbox mr-3"
                  />
                  <span className="text-sm text-slate-700">High Contrast</span>
                </label>

                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={preferences.reduceMotion}
                    onChange={(e) => updatePreferences({ reduceMotion: e.target.checked })}
                    className="form-checkbox mr-3"
                  />
                  <span className="text-sm text-slate-700">Reduce Motion</span>
                </label>

                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={preferences.focusIndicators}
                    onChange={(e) => updatePreferences({ focusIndicators: e.target.checked })}
                    className="form-checkbox mr-3"
                  />
                  <span className="text-sm text-slate-700">Enhanced Focus Indicators</span>
                </label>

                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={preferences.autoNightMode}
                    onChange={(e) => updatePreferences({ autoNightMode: e.target.checked })}
                    className="form-checkbox mr-3"
                  />
                  <span className="text-sm text-slate-700">Auto Night Mode</span>
                </label>
              </div>
            </div>
          </div>

          {/* Reset Button */}
          <div className="mt-6 pt-6 border-t border-slate-200">
            <button
              onClick={() => {
                if (confirm('Reset all reader preferences to defaults?')) {
                  // Reset handled by the reader manager
                  window.location.reload();
                }
              }}
              className="btn btn-outline btn-sm text-red-600 border-red-300 hover:bg-red-50"
            >
              Reset to Defaults
            </button>
          </div>
        </div>
      )}
    </div>
  );
}