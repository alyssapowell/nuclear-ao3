// The OAuth pages against a mocked fetch: the authorize/consent page,
// connected apps, your apps and the admin review.
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

const h = vi.hoisted(() => ({ search: '', navigate: vi.fn() }))
vi.mock('react-router-dom', async () =>
  (await import('@/test/router-mock')).routerMock({
    useSearchParams: () => [new URLSearchParams(h.search), vi.fn()],
    useNavigate: () => h.navigate,
  }))

import OAuthAuthorize from '@/pages/OAuthAuthorize'
import ConnectedApps from '@/pages/ConnectedApps'
import MyApps from '@/pages/MyApps'
import AdminApps from '@/pages/AdminApps'
import Login from '@/pages/Login'
import { nav } from '@/lib/oauth'
import { logout, safeRedirect } from '@/utils/auth'

type Reply = { status: number; body?: unknown }
const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>

/** Answer fetch by "METHOD /path" (query string left out); anything else fails the test. */
function mockApi(routes: Record<string, Reply | ((init?: RequestInit) => Reply)>) {
  fetchMock.mockImplementation((url: string, init?: RequestInit) => {
    const key = `${init?.method ?? 'GET'} ${String(url).split('?')[0]}`
    const route = routes[key]
    if (!route) return Promise.reject(new Error(`unexpected request ${key}`))
    const r = typeof route === 'function' ? route(init) : route
    return Promise.resolve({
      ok: r.status >= 200 && r.status < 300,
      status: r.status,
      json: () => (r.body === undefined ? Promise.reject(new Error('no body')) : Promise.resolve(r.body)),
    } as Response)
  })
}
const calls = () => fetchMock.mock.calls as [string, RequestInit][]
const callTo = (key: string) => calls().find(([u, i]) => `${i?.method ?? 'GET'} ${u.split('?')[0]}` === key)
const bodyOf = (key: string) => JSON.parse(callTo(key)![1].body as string)

const logIn = () => vi.mocked(localStorage.getItem).mockImplementation((k: string) => (k === 'auth_token' ? 'tok' : null))

// Fixture values that must never appear on a page.
const ACCOUNT_ID = '7d1f0c52-aaaa-4bbb-8ccc-000000000001'
const USERNAME = 'secret_account_name'
const EMAIL = 'private.person@example.org'
const PSEUD_A = { id: 'pppppppp-0000-4000-8000-00000000000a', name: 'Quill Shown' }
const PSEUD_B = { id: 'pppppppp-0000-4000-8000-00000000000b', name: 'Ink Second' }

const QUERY = 'response_type=code&client_id=c1&redirect_uri=https%3A%2F%2Fapp.example.org%2Fcb&scope=openid+profile+email&code_challenge=abc&code_challenge_method=S256&state=xyz'

const consent = (scopes: string[]) => ({
  consent_id: 'consent-1',
  app: { name: 'Reader Plus', website: 'https://reader.example.org', logo_url: 'https://reader.example.org/logo.png' },
  scopes: scopes.map((s) => ({ name: s, description: `can use ${s}` })),
  pseuds: scopes.includes('profile') ? [PSEUD_A, PSEUD_B] : undefined,
  default_pseud_id: scopes.includes('profile') ? PSEUD_A.id : undefined,
})

let assign: ReturnType<typeof vi.spyOn>
beforeEach(() => {
  fetchMock.mockReset()
  h.search = ''
  h.navigate.mockReset()
  vi.mocked(localStorage.getItem).mockReset()
  assign = vi.spyOn(nav, 'to').mockImplementation(() => {})
})
afterEach(() => {
  assign.mockRestore()
  vi.restoreAllMocks()
})

function expectNoAccountDetails() {
  const text = document.body.textContent ?? ''
  for (const secret of [ACCOUNT_ID, USERNAME, EMAIL]) expect(text).not.toContain(secret)
}

describe('safeRedirect for the authorize page', () => {
  it('allows the same-site authorize path with its query and nothing else', () => {
    const path = `/oauth/authorize?${QUERY}`
    expect(safeRedirect(path)).toBe(path)
    expect(safeRedirect('https://evil.example.org/oauth/authorize')).toBe('/')
    expect(safeRedirect('//evil.example.org')).toBe('/')
  })
})

describe('Authorize page', () => {
  it('sends a logged-out person to login with a safe way back', async () => {
    h.search = QUERY
    render(<OAuthAuthorize />)
    await waitFor(() => expect(h.navigate).toHaveBeenCalled())
    const [target, opts] = h.navigate.mock.calls[0]
    expect(target.startsWith('/auth/login?redirect=')).toBe(true)
    const back = decodeURIComponent(target.slice('/auth/login?redirect='.length))
    expect(back.startsWith('/oauth/authorize?')).toBe(true)
    expect(new URLSearchParams(back.split('?')[1]).get('client_id')).toBe('c1')
    expect(safeRedirect(back)).toBe(back)
    expect(opts).toEqual({ replace: true })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('goes to login if the server says the login is no longer good', async () => {
    logIn()
    h.search = QUERY
    mockApi({ 'GET /api/v1/oauth/authorize': { status: 401, body: { error: 'invalid_token' } } })
    render(<OAuthAuthorize />)
    await waitFor(() => expect(h.navigate).toHaveBeenCalled())
    expect(h.navigate.mock.calls[0][0]).toMatch(/^\/auth\/login\?redirect=/)
  })

  it('shows scopes, a pseud picker and an unchecked email box when profile and email are asked for', async () => {
    logIn()
    h.search = QUERY
    mockApi({ 'GET /api/v1/oauth/authorize': { status: 200, body: { consent: consent(['openid', 'profile', 'email']) } } })
    render(<OAuthAuthorize />)

    expect(await screen.findByRole('heading', { name: /Reader Plus/ })).toBeInTheDocument()
    const request = callTo('GET /api/v1/oauth/authorize')!
    expect(request[0]).toContain('client_id=c1')
    expect((request[1].headers as Record<string, string>).Authorization).toBe('Bearer tok')
    expect(screen.getByText('can use profile')).toBeInTheDocument()
    expect(screen.getByText('can use email')).toBeInTheDocument()
    expect(screen.getByText(/reviewed by the archive's admins/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'https://reader.example.org' })).toHaveAttribute('href', 'https://reader.example.org')
    expect(screen.getByAltText('Reader Plus logo')).toBeInTheDocument()

    const picker = screen.getByLabelText(/Which pseud/)
    expect(picker).toHaveValue(PSEUD_A.id)
    expect(within(picker).getAllByRole('option').map((o) => o.textContent)).toEqual([PSEUD_A.name, PSEUD_B.name])
    expect(screen.getByLabelText(/share the email address/i)).not.toBeChecked()
    expect(screen.getByRole('button', { name: 'Allow' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Deny' })).toBeInTheDocument()
    expectNoAccountDetails()
  })

  it('offers no pseud picker or email box when they are not asked for', async () => {
    logIn()
    h.search = QUERY
    mockApi({ 'GET /api/v1/oauth/authorize': { status: 200, body: { consent: consent(['openid', 'read']) } } })
    render(<OAuthAuthorize />)
    await screen.findByRole('heading', { name: /Reader Plus/ })
    expect(screen.queryByLabelText(/Which pseud/)).toBeNull()
    expect(screen.queryByLabelText(/share the email address/i)).toBeNull()
  })

  it('Allow posts the chosen pseud and the email choice, then goes to the app', async () => {
    logIn()
    h.search = QUERY
    mockApi({
      'GET /api/v1/oauth/authorize': { status: 200, body: { consent: consent(['openid', 'profile', 'email']) } },
      'POST /api/v1/oauth/consent/consent-1': { status: 200, body: { redirect: 'https://app.example.org/cb?code=thecode&state=xyz' } },
    })
    render(<OAuthAuthorize />)
    await userEvent.selectOptions(await screen.findByLabelText(/Which pseud/), PSEUD_B.id)
    await userEvent.click(screen.getByLabelText(/share the email address/i))
    await userEvent.click(screen.getByRole('button', { name: 'Allow' }))

    await waitFor(() => expect(assign).toHaveBeenCalledWith('https://app.example.org/cb?code=thecode&state=xyz'))
    expect(bodyOf('POST /api/v1/oauth/consent/consent-1')).toEqual({ approved: true, pseud_id: PSEUD_B.id, share_email: true })
  })

  it('Allow with the defaults sends the default pseud and no email', async () => {
    logIn()
    h.search = QUERY
    mockApi({
      'GET /api/v1/oauth/authorize': { status: 200, body: { consent: consent(['openid', 'profile', 'email']) } },
      'POST /api/v1/oauth/consent/consent-1': { status: 200, body: { redirect: 'https://app.example.org/cb?code=c' } },
    })
    render(<OAuthAuthorize />)
    await userEvent.click(await screen.findByRole('button', { name: 'Allow' }))
    await waitFor(() => expect(assign).toHaveBeenCalled())
    expect(bodyOf('POST /api/v1/oauth/consent/consent-1')).toEqual({ approved: true, pseud_id: PSEUD_A.id, share_email: false })
  })

  it('Deny posts a denial and follows the redirect', async () => {
    logIn()
    h.search = QUERY
    mockApi({
      'GET /api/v1/oauth/authorize': { status: 200, body: { consent: consent(['openid', 'profile']) } },
      'POST /api/v1/oauth/consent/consent-1': { status: 200, body: { redirect: 'https://app.example.org/cb?error=access_denied' } },
    })
    render(<OAuthAuthorize />)
    await userEvent.click(await screen.findByRole('button', { name: 'Deny' }))
    await waitFor(() => expect(assign).toHaveBeenCalledWith('https://app.example.org/cb?error=access_denied'))
    expect(bodyOf('POST /api/v1/oauth/consent/consent-1')).toEqual({ approved: false })
  })

  it('follows a redirect the server gives straight away (consent already on file)', async () => {
    logIn()
    h.search = QUERY
    mockApi({ 'GET /api/v1/oauth/authorize': { status: 200, body: { redirect: 'https://app.example.org/cb?code=now' } } })
    render(<OAuthAuthorize />)
    await waitFor(() => expect(assign).toHaveBeenCalledWith('https://app.example.org/cb?code=now'))
  })

  it('shows an error that must not go to the app, and goes nowhere', async () => {
    logIn()
    h.search = QUERY
    mockApi({
      'GET /api/v1/oauth/authorize': { status: 400, body: { error: 'invalid_request', error_description: 'Invalid redirect_uri' } },
    })
    render(<OAuthAuthorize />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Invalid redirect_uri')
    expect(assign).not.toHaveBeenCalled()
    expect(h.navigate).not.toHaveBeenCalled()
    expect(screen.queryByRole('button', { name: 'Allow' })).toBeNull()
  })

  it('shows an error if the answer cannot be sent, and stays on the page', async () => {
    logIn()
    h.search = QUERY
    mockApi({
      'GET /api/v1/oauth/authorize': { status: 200, body: { consent: consent(['openid']) } },
      'POST /api/v1/oauth/consent/consent-1': { status: 400, body: { error: 'Invalid consent request' } },
    })
    render(<OAuthAuthorize />)
    await userEvent.click(await screen.findByRole('button', { name: 'Allow' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Invalid consent request')
    expect(assign).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Allow' })).toBeEnabled()
  })
})

describe('Connected apps', () => {
  const app = {
    client_id: 'c1', client_name: 'Reader Plus', website: 'https://reader.example.org',
    scopes: ['openid', 'profile', 'email'], pseud: PSEUD_A, share_email: true,
    granted_at: '2026-10-01T10:00:00Z', last_used: null,
  }

  it('lists apps and disconnects one after a confirm', async () => {
    logIn()
    let listed = [app]
    mockApi({
      'GET /api/v1/oauth/authorized-applications': () => ({ status: 200, body: { applications: listed } }),
      'DELETE /api/v1/oauth/authorized-applications/c1': () => {
        listed = []
        return { status: 200, body: { message: 'ok' } }
      },
    })
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<ConnectedApps />)

    expect(await screen.findByText('Reader Plus')).toBeInTheDocument()
    expect(screen.getByText(/Sees the pseud: Quill Shown/)).toBeInTheDocument()
    expect(screen.getByText(/Sees your email address: yes/)).toBeInTheDocument()
    expect(screen.getByText(/signs it out everywhere/i)).toBeInTheDocument()
    expect(screen.getByText(/has to ask you again/i)).toBeInTheDocument()
    expectNoAccountDetails()

    await userEvent.click(screen.getByRole('button', { name: 'Disconnect Reader Plus' }))
    expect(confirm).toHaveBeenCalled()
    expect(await screen.findByText(/No apps are connected/)).toBeInTheDocument()
    expect(callTo('DELETE /api/v1/oauth/authorized-applications/c1')).toBeTruthy()
  })

  it('does nothing when the confirm is cancelled', async () => {
    logIn()
    mockApi({ 'GET /api/v1/oauth/authorized-applications': { status: 200, body: { applications: [app] } } })
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    render(<ConnectedApps />)
    await userEvent.click(await screen.findByRole('button', { name: 'Disconnect Reader Plus' }))
    expect(callTo('DELETE /api/v1/oauth/authorized-applications/c1')).toBeUndefined()
  })
})

describe('Your apps', () => {
  const mine = {
    client_id: 'mine-1', client_name: 'My Reader', description: 'reads', website: '', logo_url: '',
    status: 'pending', review_note: '', redirect_uris: ['https://me.example.org/cb'], scopes: ['openid', 'profile'],
    is_public: false, created_at: '2026-11-01T00:00:00Z', reviewed_at: null, display_review_pending: false, proposed: null,
  }

  it('registers an app, shows the secret once with a warning, and the pending status', async () => {
    logIn()
    let listed: unknown[] = []
    mockApi({
      'GET /api/v1/apps': () => ({ status: 200, body: { apps: listed } }),
      'POST /api/v1/apps': () => {
        listed = [mine]
        return { status: 201, body: { ...mine, client_secret: 'sekrit-value-123' } }
      },
    })
    render(<MyApps />)
    expect(await screen.findByText(/You have not registered any apps/)).toBeInTheDocument()
    expect(screen.getByText(/starts as/i)).toHaveTextContent(/pending/)
    expect(screen.getByText(/sends it back to review and signs everyone out/i)).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('App name'), 'My Reader')
    await userEvent.type(screen.getAllByLabelText(/Redirect addresses/)[0], 'https://me.example.org/cb')
    await userEvent.click(screen.getByRole('button', { name: 'Register app' }))

    expect(await screen.findByTestId('client-secret')).toHaveTextContent('sekrit-value-123')
    expect(screen.getByText(/Copy it now, it won't be shown again/)).toBeInTheDocument()
    const sent = bodyOf('POST /api/v1/apps')
    expect(sent).toMatchObject({ client_name: 'My Reader', redirect_uris: ['https://me.example.org/cb'], is_public: false })
    expect(sent.scopes).toEqual(['openid', 'profile'])
    expect(await screen.findByText(/Waiting for review/)).toBeInTheDocument()
    expect(screen.getAllByText('pending')).toHaveLength(2) // the explanation and the new app's status

    await userEvent.click(screen.getByRole('button', { name: 'I have copied it' }))
    expect(screen.queryByTestId('client-secret')).toBeNull()
  })

  it('shows a rejection note, resets a secret once, and deletes after a confirm', async () => {
    logIn()
    let listed: unknown[] = [{ ...mine, status: 'rejected', review_note: 'Use https redirects' }]
    mockApi({
      'GET /api/v1/apps': () => ({ status: 200, body: { apps: listed } }),
      'POST /api/v1/apps/mine-1/reset-secret': { status: 200, body: { client_id: 'mine-1', client_secret: 'fresh-secret-456' } },
      'DELETE /api/v1/apps/mine-1': () => {
        listed = []
        return { status: 200, body: { message: 'App deleted' } }
      },
    })
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<MyApps />)
    expect(await screen.findByText(/Use https redirects/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Reset secret for My Reader' }))
    expect(await screen.findByTestId('client-secret')).toHaveTextContent('fresh-secret-456')

    await userEvent.click(screen.getByRole('button', { name: 'Delete My Reader' }))
    expect(await screen.findByText(/You have not registered any apps/)).toBeInTheDocument()
  })

  it('says a display change is waiting for review and still shows the old name', async () => {
    logIn()
    mockApi({
      'GET /api/v1/apps': {
        status: 200,
        body: { apps: [{ ...mine, status: 'approved', display_review_pending: true, proposed: { client_name: 'New Name', description: '', website: '', logo_url: '' } }] },
      },
    })
    render(<MyApps />)
    expect(await screen.findByText(/Waiting for review: your changes to the name/)).toHaveTextContent('New Name')
    expect(screen.getByText('My Reader', { exact: false })).toBeInTheDocument()
  })

  it('edits an app and explains when it went back to review', async () => {
    logIn()
    mockApi({
      'GET /api/v1/apps': { status: 200, body: { apps: [{ ...mine, status: 'approved' }] } },
      'PATCH /api/v1/apps/mine-1': { status: 200, body: { client_id: 'mine-1', status: 'pending', display_review_pending: false } },
    })
    render(<MyApps />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit My Reader' }))
    const redirects = screen.getByLabelText(/Redirect addresses/, { selector: '#edit-mine-1-redirects' })
    await userEvent.clear(redirects)
    await userEvent.type(redirects, 'https://new.example.org/cb')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText(/sent back to review and everyone was signed out/)).toBeInTheDocument()
    expect(bodyOf('PATCH /api/v1/apps/mine-1').redirect_uris).toEqual(['https://new.example.org/cb'])
  })
})

describe('Admin app review', () => {
  const queued = {
    client_id: 'q1', client_name: 'Queued App', description: 'a reader', website: 'https://q.example.org', logo_url: '',
    status: 'pending', review_note: '', redirect_uris: ['https://q.example.org/cb'], scopes: ['openid'], grant_types: ['authorization_code'],
    is_public: false, is_trusted: false, created_at: '2026-11-01T00:00:00Z', reviewed_at: null,
    revision: 4, owner: PSEUD_A, display_review_pending: false, proposed: null,
  }

  it('approves an app, sending the revision that was reviewed', async () => {
    logIn()
    let pending = [queued]
    mockApi({
      'GET /api/v1/admin/apps': (init) => ({ status: 200, body: { apps: pending } }),
      'POST /api/v1/admin/apps/q1/approve': () => {
        pending = []
        return { status: 200, body: { client_id: 'q1', status: 'approved' } }
      },
    })
    render(<AdminApps />)
    expect(await screen.findByText('Queued App', { exact: false })).toBeInTheDocument()
    expect(screen.getByText(PSEUD_A.name)).toBeInTheDocument()
    expectNoAccountDetails()
    await userEvent.click(screen.getByRole('button', { name: 'Approve Queued App' }))
    await waitFor(() => expect(screen.getByText('Nothing here.')).toBeInTheDocument())
    expect(bodyOf('POST /api/v1/admin/apps/q1/approve')).toEqual({ revision: 4 })
  })

  it('rejects an app with a note', async () => {
    logIn()
    mockApi({
      'GET /api/v1/admin/apps': { status: 200, body: { apps: [queued] } },
      'POST /api/v1/admin/apps/q1/reject': { status: 200, body: { client_id: 'q1', status: 'rejected' } },
    })
    render(<AdminApps />)
    await screen.findByText('Queued App', { exact: false })
    await userEvent.type(screen.getByLabelText(/Note for the owner/), 'Redirect looks wrong')
    await userEvent.click(screen.getByRole('button', { name: 'Reject Queued App' }))
    await waitFor(() => expect(callTo('POST /api/v1/admin/apps/q1/reject')).toBeTruthy())
    expect(bodyOf('POST /api/v1/admin/apps/q1/reject')).toEqual({ revision: 4, note: 'Redirect looks wrong' })
  })

  it('suspends an app with a note', async () => {
    logIn()
    mockApi({
      'GET /api/v1/admin/apps': { status: 200, body: { apps: [{ ...queued, status: 'approved' }] } },
      'POST /api/v1/admin/apps/q1/suspend': { status: 200, body: { status: 'suspended' } },
    })
    render(<AdminApps />)
    await screen.findByText('Queued App', { exact: false })
    await userEvent.type(screen.getByLabelText(/Note for the owner/), 'scraping')
    await userEvent.click(screen.getByRole('button', { name: 'Suspend Queued App' }))
    await waitFor(() => expect(callTo('POST /api/v1/admin/apps/q1/suspend')).toBeTruthy())
    expect(bodyOf('POST /api/v1/admin/apps/q1/suspend')).toEqual({ revision: 4, note: 'scraping' })
  })

  it('shows old and new display details side by side and decides on them', async () => {
    logIn()
    mockApi({
      'GET /api/v1/admin/apps': {
        status: 200,
        body: { apps: [{ ...queued, status: 'approved', display_review_pending: true,
          proposed: { client_name: 'Official Support', description: 'a reader', website: 'https://look.example.org', logo_url: '' } }] },
      },
      'POST /api/v1/admin/apps/q1/approve-display': { status: 200, body: { status: 'approved' } },
    })
    render(<AdminApps />)
    const table = await screen.findByRole('table')
    expect(within(table).getByText('Official Support')).toBeInTheDocument()
    expect(within(table).getByText('Queued App')).toBeInTheDocument()
    expect(within(table).getByText('https://look.example.org')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Approve display changes for Queued App' }))
    await waitFor(() => expect(callTo('POST /api/v1/admin/apps/q1/approve-display')).toBeTruthy())
    expect(bodyOf('POST /api/v1/admin/apps/q1/approve-display')).toEqual({ revision: 4 })
    expect(screen.getByRole('button', { name: 'Reject display changes for Queued App' })).toBeInTheDocument()
  })

  it('reloads and says so when the app changed since the admin looked', async () => {
    logIn()
    mockApi({
      'GET /api/v1/admin/apps': { status: 200, body: { apps: [queued] } },
      'POST /api/v1/admin/apps/q1/approve': { status: 409, body: { error: 'app_changed', error_description: 'changed' } },
    })
    render(<AdminApps />)
    await userEvent.click(await screen.findByRole('button', { name: 'Approve Queued App' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/changed while you were looking/)
    expect(calls().filter(([u]) => u.startsWith('/api/v1/admin/apps?')).length).toBeGreaterThan(2)
  })

  it('tells non-admins the page is not for them', async () => {
    logIn()
    mockApi({ 'GET /api/v1/admin/apps': { status: 403, body: { error: "you don't have permission to do this" } } })
    render(<AdminApps />)
    expect(await screen.findByRole('alert')).toHaveTextContent(/for the archive's admins/)
  })
})

describe('Login and logout cookie', () => {
  it('does not write the token to a cookie on login, and logout clears a leftover one', async () => {
    const cookieSet = vi.spyOn(document, 'cookie', 'set')
    mockApi({ 'POST /api/v1/auth/login': { status: 200, body: { access_token: 't', user: { id: '1', username: 'u', email: 'e@x.org' } } } })
    const original = window.location
    Object.defineProperty(window, 'location', { value: { ...original, href: '' }, writable: true, configurable: true })
    render(<Login />)
    await userEvent.type(screen.getByLabelText('Username or email'), 'u')
    await userEvent.type(screen.getByLabelText('Password'), 'secret1')
    await userEvent.click(screen.getByRole('button', { name: 'Sign In' }))
    await waitFor(() => expect(localStorage.setItem).toHaveBeenCalledWith('auth_token', 't'))
    expect(cookieSet.mock.calls.filter(([v]) => String(v).includes('auth_token='))).toHaveLength(0)
    Object.defineProperty(window, 'location', { value: original, writable: true, configurable: true })

    await logout()
    expect(cookieSet.mock.calls.some(([v]) => String(v).startsWith('auth_token=;') && String(v).includes('1970'))).toBe(true)
  })
})
