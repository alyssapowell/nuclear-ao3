'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/Button'

interface SettingsWizardProps {
  onComplete: () => void
  onSkip?: () => void
}

export default function SettingsWizard({ onComplete, onSkip }: SettingsWizardProps) {
  const [currentStep, setCurrentStep] = useState(1)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSkip = () => {
    if (onSkip) {
      onSkip()
    } else {
      // Default: just mark as complete
      onComplete()
    }
  }

  const handleNext = () => {
    if (currentStep < 3) {
      setCurrentStep(currentStep + 1)
    } else {
      // For now, just complete without API calls to prevent infinite loops
      onComplete()
    }
  }

  const handleBack = () => {
    if (currentStep > 1) {
      setCurrentStep(currentStep - 1)
    }
  }

  const renderStep1 = () => (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold mb-2">Privacy Settings</h2>
        <p className="text-gray-600">Configure your default privacy preferences</p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-2">Default Work Visibility</label>
          <select className="w-full border border-gray-300 rounded-md px-3 py-2">
            <option value="public">Public (anyone can read)</option>
            <option value="restricted">Restricted (logged-in users only)</option>
            <option value="private">Private (invited users only)</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium mb-2">Default Comment Policy</label>
          <select className="w-full border border-gray-300 rounded-md px-3 py-2">
            <option value="open">Open (anyone can comment)</option>
            <option value="moderated">Moderated (comments need approval)</option>
            <option value="closed">Closed (no comments allowed)</option>
          </select>
        </div>

        <div className="space-y-3">
          <label className="flex items-center">
            <input type="checkbox" defaultChecked className="mr-2" />
            Allow guest comments
          </label>

          <label className="flex items-center">
            <input type="checkbox" defaultChecked className="mr-2" />
            Allow anonymous kudos
          </label>

          <label className="flex items-center">
            <input type="checkbox" defaultChecked className="mr-2" />
            Hide email address from other users
          </label>
        </div>
      </div>
    </div>
  )

  const renderStep2 = () => (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold mb-2">Notification Preferences</h2>
        <p className="text-gray-600">Choose when you want to be notified</p>
      </div>

      <div className="space-y-3">
        <label className="flex items-center">
          <input type="checkbox" defaultChecked className="mr-2" />
          Notify me when someone comments on my works
        </label>

        <label className="flex items-center">
          <input type="checkbox" defaultChecked className="mr-2" />
          Notify me when someone gives kudos
        </label>

        <label className="flex items-center">
          <input type="checkbox" className="mr-2" />
          Notify me when someone bookmarks my works
        </label>

        <label className="flex items-center">
          <input type="checkbox" defaultChecked className="mr-2" />
          Notify me when someone follows me
        </label>

        <label className="flex items-center">
          <input type="checkbox" defaultChecked className="mr-2" />
          Send notifications via email
        </label>
      </div>
    </div>
  )

  const renderStep3 = () => (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold mb-2">Personalization</h2>
        <p className="text-gray-600">Customize your experience</p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-2">Language</label>
          <select className="w-full border border-gray-300 rounded-md px-3 py-2">
            <option value="en">English</option>
            <option value="es">Español</option>
            <option value="fr">Français</option>
            <option value="de">Deutsch</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium mb-2">Theme</label>
          <select className="w-full border border-gray-300 rounded-md px-3 py-2">
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium mb-2">Timezone</label>
          <select className="w-full border border-gray-300 rounded-md px-3 py-2">
            <option value="America/New_York">Eastern Time (US)</option>
            <option value="America/Chicago">Central Time (US)</option>
            <option value="America/Denver">Mountain Time (US)</option>
            <option value="America/Los_Angeles">Pacific Time (US)</option>
            <option value="Europe/London">London</option>
            <option value="Europe/Paris">Paris</option>
            <option value="Asia/Tokyo">Tokyo</option>
          </select>
        </div>
      </div>
    </div>
  )

  return (
    <div className="min-h-screen bg-gray-50 py-12">
      <div className="max-w-2xl mx-auto px-4">
        <div className="bg-white rounded-lg shadow-lg p-8">
          <div className="mb-8">
            <div className="flex items-center justify-between mb-4">
              <h1 className="text-3xl font-bold">Welcome! Let's set up your account</h1>
              <span className="text-sm text-gray-500">Step {currentStep} of 3</span>
            </div>
            
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div 
                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                style={{ width: `${(currentStep / 3) * 100}%` }}
              />
            </div>
          </div>

          {error && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-md">
              <p className="text-red-600">{error}</p>
            </div>
          )}

          {currentStep === 1 && renderStep1()}
          {currentStep === 2 && renderStep2()}
          {currentStep === 3 && renderStep3()}

          <div className="flex justify-between mt-8 pt-6 border-t">
            <div className="flex space-x-2">
              <Button
                variant="outline"
                onClick={handleBack}
                disabled={currentStep === 1}
              >
                Back
              </Button>
              
              <Button
                variant="outline"
                onClick={handleSkip}
              >
                Skip Setup
              </Button>
            </div>

            <Button
              onClick={handleNext}
              disabled={isLoading}
            >
              {isLoading ? 'Saving...' : currentStep === 3 ? 'Complete Setup' : 'Next'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}