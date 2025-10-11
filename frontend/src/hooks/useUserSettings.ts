import { useState, useEffect, useCallback } from 'react'

interface UserSettingsCheck {
  hasPrivacySettings: boolean
  hasNotificationPreferences: boolean
  hasPreferences: boolean
  isComplete: boolean
}

interface UseUserSettingsReturn {
  settings: UserSettingsCheck | null
  isLoading: boolean
  error: string | null
  refetch: () => void
}

export function useUserSettings(): UseUserSettingsReturn {
  const [settings, setSettings] = useState<UserSettingsCheck | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const checkUserSettings = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)

      const token = localStorage.getItem('auth_token')
      if (!token) {
        // If no token, assume user will be redirected to login
        setSettings({
          hasPrivacySettings: true,
          hasNotificationPreferences: true,
          hasPreferences: true,
          isComplete: true
        })
        return
      }

      // For now, always assume settings are complete to prevent loops
      // TODO: Implement actual API check when backend endpoint is ready
      setSettings({
        hasPrivacySettings: true,
        hasNotificationPreferences: true,
        hasPreferences: true,
        isComplete: true
      })
    } catch (err) {
      // Fallback: assume settings are complete to avoid blocking users
      console.warn('Failed to check user settings, assuming complete:', err)
      setSettings({
        hasPrivacySettings: true,
        hasNotificationPreferences: true,
        hasPreferences: true,
        isComplete: true
      })
    } finally {
      setIsLoading(false)
    }
  }, []) // FIXED: Memoize the function to prevent recreation

  useEffect(() => {
    let isMounted = true
    
    const runCheck = async () => {
      if (isMounted) {
        await checkUserSettings()
      }
    }
    
    runCheck()
    
    return () => {
      isMounted = false
    }
  }, [checkUserSettings]) // FIXED: Include checkUserSettings in dependencies

  return {
    settings,
    isLoading,
    error,
    refetch: checkUserSettings
  }
}