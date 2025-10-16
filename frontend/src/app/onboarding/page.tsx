'use client';

import { useState, useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import PrivacyWizard from '@/components/privacy/PrivacyWizard';
import AuthGuard from '@/components/auth/AuthGuard';

function OnboardingContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [isNewUser, setIsNewUser] = useState(false);

  useEffect(() => {
    // Check if this is a new user from registration
    const newUser = searchParams.get('new') === 'true';
    setIsNewUser(newUser);
  }, [searchParams]);

  const handlePrivacyComplete = (settings: any) => {
    // Privacy wizard completed successfully
    console.log('Privacy settings saved:', settings);
    
    // Redirect to dashboard or home page
    router.push('/dashboard');
  };

  const handleSkip = () => {
    // User chose to skip the privacy wizard
    router.push('/dashboard');
  };

  return (
    <AuthGuard>
      <div className="min-h-screen bg-slate-50 py-8">
        <PrivacyWizard
          onComplete={handlePrivacyComplete}
          onSkip={handleSkip}
          showSkipOption={true}
          title={isNewUser ? "Welcome to Nuclear AO3!" : "Privacy Setup"}
          subtitle={
            isNewUser 
              ? "Let's set up your privacy preferences to create a safe and comfortable experience."
              : "Configure your privacy and safety preferences"
          }
        />
      </div>
    </AuthGuard>
  );
}

export default function OnboardingPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-slate-50 py-8 flex items-center justify-center">Loading...</div>}>
      <OnboardingContent />
    </Suspense>
  );
}