import { useEffect } from 'react'
import { loadSite, useSite } from '@/lib/site'

// Shown at the top of every page while the server is in trial mode
// (TRIAL_NO_EMAIL). It is part of the page flow, so it pushes the page down
// instead of covering anything, and it can't be dismissed.
export default function TrialBanner() {
  const { trialNoEmail } = useSite()
  useEffect(() => loadSite(), [])
  if (!trialNoEmail) return null
  return (
    <div
      role="region"
      aria-label="Trial mode"
      className="bg-yellow-100 border-b border-yellow-400 text-yellow-900 text-sm text-center px-4 py-1"
    >
      Trial: email is off. Accounts are confirmed at once, and password resets and notifications aren't sent.
    </div>
  )
}
