'use client';

import { useState } from 'react';
import { DevicePhoneMobileIcon, ArrowDownTrayIcon, XMarkIcon, Cog8ToothIcon } from '@heroicons/react/24/outline';

interface WorkOfflineSettingsProps {
  defaultUserSetting: 'files_and_pwa' | 'pwa_only' | 'none';
  currentWorkSetting: 'files_and_pwa' | 'pwa_only' | 'none' | 'use_default';
  onSettingChange: (setting: 'files_and_pwa' | 'pwa_only' | 'none' | 'use_default', reason?: string) => void;
  workTitle?: string;
  isEditing?: boolean;
}

const OVERRIDE_OPTIONS = [
  {
    value: 'use_default' as const,
    title: 'Use Profile Default',
    icon: Cog8ToothIcon,
    description: 'Use your profile\'s default offline reading setting',
    color: 'slate'
  },
  {
    value: 'files_and_pwa' as const,
    title: 'Downloads + Offline Reading',
    icon: ArrowDownTrayIcon,
    description: 'Allow downloads and offline reading for this work',
    implications: ['Maximum convenience', 'Files can be permanent'],
    color: 'orange'
  },
  {
    value: 'pwa_only' as const,
    title: 'App Offline Only',
    icon: DevicePhoneMobileIcon,
    description: 'Allow offline reading in app, but no downloads',
    implications: ['Respects deletions', 'Temporary offline access'],
    color: 'blue'
  },
  {
    value: 'none' as const,
    title: 'Online Only',
    icon: XMarkIcon,
    description: 'This work must be read online only',
    implications: ['Maximum control', 'No offline access'],
    color: 'red'
  }
];

export default function WorkOfflineSettings({ 
  defaultUserSetting, 
  currentWorkSetting, 
  onSettingChange,
  workTitle,
  isEditing = false
}: WorkOfflineSettingsProps) {
  const [selectedSetting, setSelectedSetting] = useState(currentWorkSetting);
  const [reason, setReason] = useState('');
  const [showReasonField, setShowReasonField] = useState(false);

  const handleSettingChange = (setting: typeof selectedSetting) => {
    setSelectedSetting(setting);
    
    // Show reason field for overrides that differ from default
    if (setting !== 'use_default' && setting !== defaultUserSetting) {
      setShowReasonField(true);
    } else {
      setShowReasonField(false);
      setReason('');
      onSettingChange(setting);
    }
  };

  const handleSaveWithReason = () => {
    onSettingChange(selectedSetting, reason);
    setShowReasonField(false);
  };

  const getDefaultSettingDisplay = () => {
    const defaultOption = OVERRIDE_OPTIONS.find(opt => opt.value === defaultUserSetting);
    return defaultOption ? defaultOption.title.replace(/Downloads \+ /, '').replace(/App /, '') : 'Unknown';
  };

  return (
    <div className="space-y-4">
      <div className="bg-slate-50 border border-slate-200 rounded-lg p-4">
        <h4 className="text-sm font-medium text-slate-900 mb-2">
          {isEditing ? 'Offline Reading Override' : 'Offline Reading Setting'}
        </h4>
        <p className="text-xs text-slate-600 mb-3">
          Your profile default: <span className="font-medium">{getDefaultSettingDisplay()}</span>
          {workTitle && ` • Override for "${workTitle}"`}
        </p>

        <div className="space-y-3">
          {OVERRIDE_OPTIONS.map((option) => {
            const IconComponent = option.icon;
            const isSelected = selectedSetting === option.value;
            const isDefault = option.value === 'use_default';
            
            return (
              <div
                key={option.value}
                className={`relative border rounded-lg p-3 cursor-pointer transition-all ${
                  isSelected 
                    ? `border-${option.color}-500 bg-${option.color}-50` 
                    : 'border-slate-200 hover:border-slate-300'
                }`}
                onClick={() => handleSettingChange(option.value)}
              >
                <div className="flex items-start space-x-3">
                  <div className={`flex-shrink-0 p-1.5 rounded ${
                    isSelected 
                      ? `bg-${option.color}-500 text-white` 
                      : 'bg-slate-100 text-slate-600'
                  }`}>
                    <IconComponent className="w-4 h-4" />
                  </div>
                  
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <h5 className="text-sm font-medium text-slate-900">
                        {option.title}
                        {isDefault && (
                          <span className={`ml-2 text-xs px-2 py-0.5 rounded-full bg-${option.color}-100 text-${option.color}-700`}>
                            Current: {getDefaultSettingDisplay()}
                          </span>
                        )}
                      </h5>
                      <input
                        type="radio"
                        name="work_offline_reading"
                        value={option.value}
                        checked={isSelected}
                        onChange={() => handleSettingChange(option.value)}
                        className={`w-4 h-4 text-${option.color}-600 border-slate-300 focus:ring-${option.color}-500`}
                      />
                    </div>
                    
                    <p className="text-xs text-slate-600 mt-1">
                      {option.description}
                    </p>
                    
                    {option.implications && (
                      <div className="flex flex-wrap gap-1 mt-2">
                        {option.implications.map((implication, index) => (
                          <span key={index} className="text-xs px-2 py-0.5 bg-slate-100 text-slate-600 rounded">
                            {implication}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {/* Reason Field */}
        {showReasonField && (
          <div className="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
            <label className="block text-sm font-medium text-yellow-800 mb-2">
              Why are you changing this setting for this work?
            </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="e.g., 'This work contains sensitive content I want to control access to' or 'This is a gift fic that should be widely accessible'"
              rows={2}
              className="w-full px-3 py-2 text-sm border border-yellow-300 rounded-md focus:outline-none focus:ring-2 focus:ring-yellow-500 focus:border-yellow-500"
            />
            <div className="flex justify-end space-x-2 mt-2">
              <button
                onClick={() => {
                  setShowReasonField(false);
                  setSelectedSetting(currentWorkSetting);
                  setReason('');
                }}
                className="px-3 py-1 text-xs text-yellow-700 border border-yellow-300 rounded hover:bg-yellow-100"
              >
                Cancel
              </button>
              <button
                onClick={handleSaveWithReason}
                disabled={!reason.trim()}
                className="px-3 py-1 text-xs text-white bg-yellow-600 rounded hover:bg-yellow-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Save Override
              </button>
            </div>
          </div>
        )}

        {/* Current Override Display */}
        {currentWorkSetting !== 'use_default' && currentWorkSetting !== selectedSetting && (
          <div className="mt-3 p-2 bg-blue-50 border border-blue-200 rounded text-xs text-blue-800">
            <strong>Current override:</strong> {OVERRIDE_OPTIONS.find(opt => opt.value === currentWorkSetting)?.title}
          </div>
        )}
      </div>
    </div>
  );
}