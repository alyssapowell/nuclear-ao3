import { useState, useEffect, useCallback } from 'react';
import { readerManager } from '@/lib/reader';
import { ReaderPreferences } from '@/types/reader';

// React hook for using reader preferences
export function useReaderPreferences() {
  const [preferences, setPreferences] = useState(() => readerManager.getPreferences());

  useEffect(() => {
    return readerManager.subscribe(setPreferences);
  }, []);

  const updatePreferences = useCallback((updates: Partial<ReaderPreferences>) => {
    readerManager.updatePreferences(updates);
  }, []);

  const applyPreset = useCallback((presetName: string) => {
    readerManager.applyPreset(presetName);
  }, []);

  const toggleReaderMode = useCallback(() => {
    readerManager.updatePreferences({ readerMode: !preferences.readerMode });
  }, [preferences.readerMode]);

  return {
    preferences,
    updatePreferences,
    applyPreset,
    toggleReaderMode,
    generateCSS: () => readerManager.generateReaderCSS(),
  };
}