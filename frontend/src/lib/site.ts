// What the server says about itself: today, only whether it is in trial mode
// (TRIAL_NO_EMAIL), where no email is sent. Asked once; if the server can't
// be asked the answer is "not a trial".
import { useSyncExternalStore } from 'react'

export interface SiteInfo {
  trialNoEmail: boolean
}

let info: SiteInfo = { trialNoEmail: false }
let started = false
const listeners = new Set<() => void>()

/** Fetches /api/v1/site the first time it is called; later calls do nothing. */
export function loadSite(): void {
  if (started) return
  started = true
  fetch('/api/v1/site', { headers: { Accept: 'application/json' } })
    .then((r) => (r.ok ? r.json() : null))
    .then((data) => {
      if (data?.trial_no_email === true) {
        info = { trialNoEmail: true }
        listeners.forEach((l) => l())
      }
    })
    .catch(() => {})
}

/** For tests: forget what was learned. */
export function resetSite(): void {
  info = { trialNoEmail: false }
  started = false
  listeners.forEach((l) => l())
}

export function useSite(): SiteInfo {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l)
      return () => listeners.delete(l)
    },
    () => info,
  )
}
