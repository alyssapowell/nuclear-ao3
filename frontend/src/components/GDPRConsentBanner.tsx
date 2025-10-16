'use client';

import { useState, useEffect } from 'react';
import { Cookie, Shield, Info, X, Settings } from 'lucide-react';

interface ConsentState {
  essential: boolean;
  analytics: boolean;
  marketing: boolean;
  exports: boolean;
}

interface GDPRConsentBannerProps {
  onConsentUpdate?: (consent: ConsentState) => void;
  className?: string;
}

export default function GDPRConsentBanner({ onConsentUpdate, className = '' }: GDPRConsentBannerProps) {
  const [showBanner, setShowBanner] = useState(false);
  const [showDetails, setShowDetails] = useState(false);
  const [consent, setConsent] = useState<ConsentState>({
    essential: true, // Always required
    analytics: false,
    marketing: false,
    exports: false,
  });

  useEffect(() => {
    // Check if user has already given consent
    if (typeof window !== 'undefined') {
      const existingConsent = localStorage.getItem('nuclear-ao3-gdpr-consent');
      if (!existingConsent) {
        setShowBanner(true);
      } else {
        try {
          const parsed = JSON.parse(existingConsent);
          setConsent(parsed);
        } catch (error) {
          // Invalid stored consent, show banner again
          setShowBanner(true);
        }
      }
    }
  }, []);

  const handleConsentChange = (type: keyof ConsentState, value: boolean) => {
    if (type === 'essential') return; // Essential cannot be disabled
    
    setConsent(prev => {
      const updated = { ...prev, [type]: value };
      return updated;
    });
  };

  const saveConsent = async (consentData: ConsentState) => {
    try {
      // Save to localStorage
      if (typeof window !== 'undefined') {
        localStorage.setItem('nuclear-ao3-gdpr-consent', JSON.stringify(consentData));
        localStorage.setItem('nuclear-ao3-gdpr-consent-date', new Date().toISOString());
        localStorage.setItem('nuclear-ao3-privacy-policy-version', '1.0');
      }

      // Send to backend with timeout and error handling
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 3000);
        
        const response = await fetch('/api/v1/gdpr/consent', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            consent: consentData,
            privacy_policy_version: '1.0',
            timestamp: new Date().toISOString(),
          }),
          signal: controller.signal
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          console.warn('Failed to save consent to backend, but proceeding with local storage');
        }
      } catch (fetchError) {
        console.warn('Network error saving consent to backend, proceeding with local storage:', fetchError);
      }

      // Notify parent component
      onConsentUpdate?.(consentData);
      
    } catch (error) {
      console.warn('Error saving consent:', error);
    }
  };

  const acceptAll = () => {
    const allConsent = {
      essential: true,
      analytics: true,
      marketing: true,
      exports: true,
    };
    setConsent(allConsent);
    saveConsent(allConsent);
    setShowBanner(false);
  };

  const acceptEssentialOnly = () => {
    const essentialOnly = {
      essential: true,
      analytics: false,
      marketing: false,
      exports: false,
    };
    setConsent(essentialOnly);
    saveConsent(essentialOnly);
    setShowBanner(false);
  };

  const acceptCustom = () => {
    saveConsent(consent);
    setShowBanner(false);
  };

  if (!showBanner) return null;

  return (
    <div className={`fixed bottom-0 left-0 right-0 z-50 bg-white border-t-2 border-orange-500 shadow-xl ${className}`}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        {!showDetails ? (
          // Simplified banner
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
            <div className="flex items-start gap-3 flex-1">
              <Cookie className="w-6 h-6 text-orange-500 mt-1 flex-shrink-0" />
              <div>
                <h3 className="font-semibold text-slate-900 mb-1">We Collect Almost Nothing</h3>
                <p className="text-sm text-slate-600">
                  <strong>Required:</strong> Only your email (for login) and content you choose to publish. 
                  <strong>Optional:</strong> Anonymous usage stats to improve the platform. 
                  <strong>Never:</strong> Real names, browsing history, or personal details.
                </p>
              </div>
            </div>
            
            <div className="flex items-center gap-3 flex-shrink-0">
              <button
                onClick={() => setShowDetails(true)}
                className="text-sm text-orange-600 hover:text-orange-700 font-medium flex items-center gap-1"
              >
                <Settings className="w-4 h-4" />
                Customize
              </button>
              <button
                onClick={acceptEssentialOnly}
                className="px-4 py-2 text-sm border border-slate-300 text-slate-700 hover:bg-slate-50 rounded-md transition-colors"
              >
                Essential Only
              </button>
              <button
                onClick={acceptAll}
                className="px-4 py-2 text-sm bg-orange-500 text-white hover:bg-orange-600 rounded-md transition-colors"
              >
                Accept All
              </button>
            </div>
          </div>
        ) : (
          // Detailed consent form
          <div className="max-w-4xl mx-auto">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <Shield className="w-6 h-6 text-orange-500" />
                <h3 className="text-lg font-semibold text-slate-900">Privacy Preferences</h3>
              </div>
              <button
                onClick={() => setShowDetails(false)}
                className="text-slate-400 hover:text-slate-600"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="grid gap-4 mb-6">
              {/* Essential Cookies */}
              <div className="bg-slate-50 p-4 rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-medium text-slate-900">Essential Functionality</h4>
                  <div className="flex items-center">
                    <span className="text-sm text-slate-500 mr-2">Always Active</span>
                    <div className="w-10 h-6 bg-orange-500 rounded-full relative">
                      <div className="w-4 h-4 bg-white rounded-full absolute top-1 right-1"></div>
                    </div>
                  </div>
                </div>
                <p className="text-sm text-slate-600">
                  <strong>What we collect:</strong> Your email address (for login only), your chosen username/pseuds, 
                  works and comments you publish, and basic account settings.
                </p>
                <p className="text-sm text-slate-600 mt-1">
                  <strong>What we DON'T collect:</strong> Real names, addresses, phone numbers, social media profiles, 
                  or any personal information beyond your email.
                </p>
                <p className="text-xs text-slate-500 mt-2">
                  <strong>Technical data:</strong> Session cookies (expires when you log out), security tokens (auto-expire)
                </p>
              </div>

              {/* Analytics */}
              <div className="bg-white border border-slate-200 p-4 rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-medium text-slate-900">Analytics & Improvements</h4>
                  <label className="flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={consent.analytics}
                      onChange={(e) => handleConsentChange('analytics', e.target.checked)}
                      className="sr-only"
                    />
                    <div className={`w-10 h-6 rounded-full relative transition-colors ${
                      consent.analytics ? 'bg-orange-500' : 'bg-slate-300'
                    }`}>
                      <div className={`w-4 h-4 bg-white rounded-full absolute top-1 transition-transform ${
                        consent.analytics ? 'translate-x-4' : 'translate-x-1'
                      }`}></div>
                    </div>
                  </label>
                </div>
                <p className="text-sm text-slate-600">
                  Anonymized usage statistics to help us improve the platform and understand 
                  how features are being used by the community.
                </p>
                <p className="text-xs text-slate-500 mt-1">
                  Data: Page views, feature usage, performance metrics (no personal identification)
                </p>
              </div>

              {/* Marketing */}
              <div className="bg-white border border-slate-200 p-4 rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-medium text-slate-900">Community Updates</h4>
                  <label className="flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={consent.marketing}
                      onChange={(e) => handleConsentChange('marketing', e.target.checked)}
                      className="sr-only"
                    />
                    <div className={`w-10 h-6 rounded-full relative transition-colors ${
                      consent.marketing ? 'bg-orange-500' : 'bg-slate-300'
                    }`}>
                      <div className={`w-4 h-4 bg-white rounded-full absolute top-1 transition-transform ${
                        consent.marketing ? 'translate-x-4' : 'translate-x-1'
                      }`}></div>
                    </div>
                  </label>
                </div>
                <p className="text-sm text-slate-600">
                  Occasional emails about important platform updates, new features, 
                  and community events. We promise not to spam you.
                </p>
                <p className="text-xs text-slate-500 mt-1">
                  Data: Email address, communication preferences
                </p>
              </div>

              {/* Exports */}
              <div className="bg-white border border-slate-200 p-4 rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-medium text-slate-900">Export Features</h4>
                  <label className="flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={consent.exports}
                      onChange={(e) => handleConsentChange('exports', e.target.checked)}
                      className="sr-only"
                    />
                    <div className={`w-10 h-6 rounded-full relative transition-colors ${
                      consent.exports ? 'bg-orange-500' : 'bg-slate-300'
                    }`}>
                      <div className={`w-4 h-4 bg-white rounded-full absolute top-1 transition-transform ${
                        consent.exports ? 'translate-x-4' : 'translate-x-1'
                      }`}></div>
                    </div>
                  </label>
                </div>
                <p className="text-sm text-slate-600">
                  Enable EPUB/MOBI export features and track download statistics. 
                  Helps authors see how their works are being shared.
                </p>
                <p className="text-xs text-slate-500 mt-1">
                  Data: Export history, download statistics, file generation logs
                </p>
              </div>
            </div>

            <div className="flex items-center justify-between pt-4 border-t border-slate-200">
              <div className="flex items-center gap-2 text-sm text-slate-600">
                <Info className="w-4 h-4" />
                <span>You can change these preferences anytime in your account settings.</span>
              </div>
              
              <div className="flex items-center gap-3">
                <button
                  onClick={acceptEssentialOnly}
                  className="px-4 py-2 text-sm border border-slate-300 text-slate-700 hover:bg-slate-50 rounded-md transition-colors"
                >
                  Essential Only
                </button>
                <button
                  onClick={acceptCustom}
                  className="px-6 py-2 text-sm bg-orange-500 text-white hover:bg-orange-600 rounded-md transition-colors"
                >
                  Save Preferences
                </button>
              </div>
            </div>

            <div className="mt-4 text-xs text-slate-500 text-center">
              By using Nuclear AO3, you agree to our{' '}
              <a href="/privacy-policy" className="text-orange-600 hover:text-orange-700 underline">
                Privacy Policy
              </a>{' '}
              and{' '}
              <a href="/terms-of-service" className="text-orange-600 hover:text-orange-700 underline">
                Terms of Service
              </a>
              . We are committed to protecting your privacy and the fan community.
            </div>
          </div>
        )}
      </div>
    </div>
  );
}