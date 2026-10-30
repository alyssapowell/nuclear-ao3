import { useNavigate, useSearchParams } from 'react-router-dom';
import { useState, useEffect, Suspense } from 'react';
import PrivacyWizard from '@/components/privacy/PrivacyWizard';

function OnboardingContent() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
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
    navigate('/dashboard');
  };

  const handleSkip = () => {
    // User chose to skip the privacy wizard
    navigate('/dashboard');
  };

  return (
    <>
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
    </>
  );
}

export default function OnboardingPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-slate-50 py-8 flex items-center justify-center">Loading...</div>}>
      <OnboardingContent />
    </Suspense>
  );
}