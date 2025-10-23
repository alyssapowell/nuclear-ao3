'use client';

import { useState } from 'react';
import { DevicePhoneMobileIcon, ArrowDownTrayIcon, XMarkIcon, InformationCircleIcon } from '@heroicons/react/24/outline';

interface OfflineReadingSettingsProps {
  defaultSetting: 'files_and_pwa' | 'pwa_only' | 'none';
  onSettingChange: (setting: 'files_and_pwa' | 'pwa_only' | 'none') => void;
  showAsModal?: boolean;
  onClose?: () => void;
}

const OFFLINE_OPTIONS = [
  {
    value: 'files_and_pwa' as const,
    title: 'Downloads + Offline Reading',
    icon: ArrowDownTrayIcon,
    description: 'Allow readers to download files (EPUB/MOBI/PDF) and read offline in the app',
    implications: [
      '✅ Maximum reader convenience',
      '✅ Works on all devices and apps', 
      '⚠️ Files can be kept permanently',
      '⚠️ Cannot revoke access after download'
    ],
    color: 'orange'
  },
  {
    value: 'pwa_only' as const,
    title: 'App Offline Reading Only',
    icon: DevicePhoneMobileIcon,
    description: 'Allow offline reading in the Nuclear AO3 app, but no file downloads',
    implications: [
      '✅ Reader convenience for commutes/travel',
      '✅ Respects deletions when back online',
      '✅ Cannot be shared or copied easily',
      'ℹ️ Requires periodic internet connection'
    ],
    color: 'blue'
  },
  {
    value: 'none' as const,
    title: 'Online Only',
    icon: XMarkIcon,
    description: 'Works must be read online - no downloads or offline access',
    implications: [
      '✅ Maximum author control',
      '✅ Real-time respect for changes/deletions',
      '✅ Cannot be accessed offline',
      '⚠️ Less convenient for readers'
    ],
    color: 'slate'
  }
];

export default function OfflineReadingSettings({ 
  defaultSetting, 
  onSettingChange, 
  showAsModal = false,
  onClose 
}: OfflineReadingSettingsProps) {
  const [selectedSetting, setSelectedSetting] = useState(defaultSetting);

  const handleSettingChange = (setting: typeof selectedSetting) => {
    setSelectedSetting(setting);
    onSettingChange(setting);
  };

  const content = (
    <div className="space-y-6">
      <div className="text-center">
        <h3 className="text-lg font-semibold text-slate-900 mb-2">
          Offline Reading Preferences
        </h3>
        <p className="text-sm text-slate-600">
          Choose how readers can access your works offline. This applies to all your works by default.
        </p>
      </div>

      <div className="space-y-4">
        {OFFLINE_OPTIONS.map((option) => {
          const IconComponent = option.icon;
          const isSelected = selectedSetting === option.value;
          
          return (
            <div
              key={option.value}
              className={`relative border-2 rounded-lg p-4 cursor-pointer transition-all ${
                isSelected 
                  ? `border-${option.color}-500 bg-${option.color}-50` 
                  : 'border-slate-200 hover:border-slate-300'
              }`}
              onClick={() => handleSettingChange(option.value)}
            >
              <div className="flex items-start space-x-3">
                <div className={`flex-shrink-0 p-2 rounded-lg ${
                  isSelected 
                    ? `bg-${option.color}-500 text-white` 
                    : 'bg-slate-100 text-slate-600'
                }`}>
                  <IconComponent className="w-5 h-5" />
                </div>
                
                <div className="flex-1">
                  <div className="flex items-center space-x-2">
                    <h4 className="font-medium text-slate-900">{option.title}</h4>
                    {isSelected && (
                      <div className={`w-2 h-2 rounded-full bg-${option.color}-500`} />
                    )}
                  </div>
                  
                  <p className="text-sm text-slate-600 mt-1 mb-3">
                    {option.description}
                  </p>
                  
                  <div className="space-y-1">
                    {option.implications.map((implication, index) => (
                      <div key={index} className="text-xs text-slate-500 flex items-center">
                        <span className="mr-2">{implication}</span>
                      </div>
                    ))}
                  </div>
                </div>
                
                <input
                  type="radio"
                  name="offline_reading"
                  value={option.value}
                  checked={isSelected}
                  onChange={() => handleSettingChange(option.value)}
                  className={`w-4 h-4 text-${option.color}-600 border-slate-300 focus:ring-${option.color}-500`}
                />
              </div>
            </div>
          );
        })}
      </div>

      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <div className="flex items-start space-x-2">
          <InformationCircleIcon className="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" />
          <div className="text-sm text-blue-800">
            <p className="font-medium mb-1">Per-Work Overrides</p>
            <p>You can override this default setting for specific works when posting or editing them.</p>
          </div>
        </div>
      </div>
    </div>
  );

  if (showAsModal) {
    return (
      <>
        {/* Backdrop */}
        <div className="fixed inset-0 bg-black bg-opacity-50 z-50" onClick={onClose} />
        
        {/* Modal */}
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between p-6 border-b border-slate-200">
              <h2 className="text-xl font-semibold text-slate-900">
                Offline Reading Settings
              </h2>
              <button
                onClick={onClose}
                className="text-slate-400 hover:text-slate-600"
              >
                <XMarkIcon className="w-6 h-6" />
              </button>
            </div>
            
            <div className="p-6">
              {content}
            </div>
            
            <div className="flex justify-end p-6 border-t border-slate-200">
              <button
                onClick={onClose}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 transition-colors"
              >
                Save Settings
              </button>
            </div>
          </div>
        </div>
      </>
    );
  }

  return content;
}