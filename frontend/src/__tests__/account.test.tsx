// Account pages against a mocked fetch (the backend contract is in the auth API docs).
import { StrictMode } from 'react'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

const h = vi.hoisted(() => ({ search: '', navigate: vi.fn() }))
vi.mock('react-router-dom', async () =>
  (await import('@/test/router-mock')).routerMock({
    useSearchParams: () => [new URLSearchParams(h.search), vi.fn()],
    useNavigate: () => h.navigate,
  }))

import Login from '@/pages/Login'
import Register from '@/pages/Register'
import Activate from '@/pages/Activate'
import ForgotPassword from '@/pages/ForgotPassword'
import ResetPassword from '@/pages/ResetPassword'
import ChangePassword from '@/pages/ChangePassword'
import { logout } from '@/utils/auth'

type Reply = { status: number; body?: unknown }
const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>

/** Make fetch answer by "METHOD /path"; unknown routes fail the test. */
function mockApi(routes: Record<string, Reply | (() => Reply)>) {
  fetchMock.mockImplementation((url: string, init?: RequestInit) => {
    const key = `${init?.method ?? 'GET'} ${String(url).replace('/api/v1/auth', '')}`
    const route = routes[key]
    if (!route) return Promise.reject(new Error(`unexpected request ${key}`))
    const r = typeof route === 'function' ? route() : route
    return Promise.resolve({
      ok: r.status >= 200 && r.status < 300,
      status: r.status,
      json: () => (r.body === undefined ? Promise.reject(new Error('no body')) : Promise.resolve(r.body)),
    } as Response)
  })
}
const calls = () => fetchMock.mock.calls as [string, RequestInit][]
const bodyOf = (i: number) => JSON.parse(calls()[i][1].body as string)

beforeEach(() => {
  fetchMock.mockReset()
  h.search = ''
  h.navigate.mockReset()
  vi.mocked(localStorage.getItem).mockReset()
  vi.mocked(localStorage.setItem).mockReset()
  vi.mocked(localStorage.removeItem).mockReset()
})

describe('Login', () => {
  it('sends the username-or-email field as `login`', async () => {
    mockApi({ 'POST /login': { status: 401, body: { error: 'invalid_credentials' } } })
    render(<Login />)
    await userEvent.type(screen.getByLabelText('Username or email'), 'alice')
    await userEvent.type(screen.getByLabelText('Password'), 'secret1')
    await userEvent.click(screen.getByRole('button', { name: 'Sign In' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/incorrect/i)
    expect(bodyOf(0)).toEqual({ login: 'alice', password: 'secret1' })
    expect(screen.getByRole('link', { name: 'Forgot password?' })).toHaveAttribute('href', '/auth/forgot-password')
  })

  it('offers to resend the activation email when not activated (username asks for an email)', async () => {
    mockApi({
      'POST /login': { status: 403, body: { error: 'not_activated' } },
      'POST /resend-verification': { status: 202, body: {} },
    })
    render(<Login />)
    await userEvent.type(screen.getByLabelText('Username or email'), 'alice')
    await userEvent.type(screen.getByLabelText('Password'), 'secret1')
    await userEvent.click(screen.getByRole('button', { name: 'Sign In' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/hasn't been activated/i)
    await userEvent.type(screen.getByLabelText('Email address'), 'a@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Resend activation email' }))
    expect(await screen.findByRole('status')).toHaveTextContent(/sent a new activation email/i)
    expect(bodyOf(1)).toEqual({ email: 'a@example.com' })
  })

  it('resends straight to the entered email when the login is an email', async () => {
    mockApi({
      'POST /login': { status: 403, body: { error: 'not_activated' } },
      'POST /resend-verification': { status: 202, body: {} },
    })
    render(<Login />)
    await userEvent.type(screen.getByLabelText('Username or email'), 'a@example.com')
    await userEvent.type(screen.getByLabelText('Password'), 'secret1')
    await userEvent.click(screen.getByRole('button', { name: 'Sign In' }))
    await screen.findByRole('button', { name: 'Resend activation email' })
    expect(screen.queryByLabelText('Email address')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Resend activation email' }))
    await waitFor(() => expect(calls()).toHaveLength(2))
    expect(bodyOf(1)).toEqual({ email: 'a@example.com' })
  })
})

describe('Register', () => {
  async function fill(password = 'secret1') {
    await userEvent.type(screen.getByLabelText('Username'), 'alice')
    await userEvent.type(screen.getByLabelText('Email'), 'a@example.com')
    await userEvent.type(screen.getByLabelText('Password'), password)
    await userEvent.type(screen.getByLabelText('Confirm Password'), password)
    fireEvent.submit(screen.getByRole('button', { name: 'Create Account' }).closest('form')!)
  }

  it('shows check-your-email on 201 and stores no token', async () => {
    mockApi({ 'POST /register': { status: 201, body: { message: 'ok' } } })
    render(<Register />)
    await fill()
    expect(await screen.findByText('Check your email to activate your account')).toBeInTheDocument()
    expect(screen.getByText('a@example.com')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create Account' })).not.toBeInTheDocument()
    expect(localStorage.setItem).not.toHaveBeenCalled()
  })

  it('rejects passwords over 40 characters without calling the server', async () => {
    mockApi({})
    render(<Register />)
    // The input's maxLength stops typing past 40, so set the values directly.
    const long = 'x'.repeat(41)
    fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'alice' } })
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'a@example.com' } })
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: long } })
    fireEvent.change(screen.getByLabelText('Confirm Password'), { target: { value: long } })
    fireEvent.submit(screen.getByRole('button', { name: 'Create Account' }).closest('form')!)
    expect(await screen.findByRole('alert')).toHaveTextContent(/6 to 40/)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('shows the server user_exists message', async () => {
    mockApi({ 'POST /register': { status: 409, body: { error: 'user_exists' } } })
    render(<Register />)
    await fill()
    expect(await screen.findByRole('alert')).toHaveTextContent(/already in use/i)
  })
})

describe('Activate', () => {
  it('verifies once under StrictMode and links to log in', async () => {
    h.search = 'token=abc'
    mockApi({ 'POST /verify-email': { status: 204 } })
    render(<StrictMode><Activate /></StrictMode>)
    expect(await screen.findByText(/account is activated/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Log in' })).toHaveAttribute('href', '/auth/login')
    expect(calls()).toHaveLength(1)
    expect(bodyOf(0)).toEqual({ token: 'abc' })
  })

  it('shows an invalid message with a resend form', async () => {
    h.search = 'token=bad'
    mockApi({ 'POST /verify-email': { status: 400, body: { error: 'invalid_token' } } })
    render(<Activate />)
    expect(await screen.findByRole('alert')).toHaveTextContent(/invalid or has expired/i)
    expect(screen.getByLabelText('Email address')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Resend activation email' })).toBeInTheDocument()
  })
})

describe('ForgotPassword', () => {
  const message = "If an account uses that address, we've sent a reset link."
  it.each([
    ['202', { status: 202, body: {} }],
    ['an error', { status: 500, body: { error: 'internal' } }],
  ])('always shows the same message (%s)', async (_n, reply) => {
    mockApi({ 'POST /reset-password': reply })
    render(<ForgotPassword />)
    await userEvent.type(screen.getByLabelText('Email'), 'nobody@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Send reset link' }))
    expect(await screen.findByRole('status')).toHaveTextContent(message)
    expect(bodyOf(0)).toEqual({ email: 'nobody@example.com' })
  })
})

describe('ResetPassword', () => {
  async function submit(a: string, b = a) {
    await userEvent.type(screen.getByLabelText('New password'), a)
    await userEvent.type(screen.getByLabelText('Confirm new password'), b)
    fireEvent.submit(screen.getByRole('button', { name: 'Change password' }).closest('form')!)
  }

  it('links to request a new one on invalid_token', async () => {
    h.search = 'token=old'
    mockApi({ 'POST /reset-password/confirm': { status: 400, body: { error: 'invalid_token' } } })
    render(<ResetPassword />)
    await submit('secret1')
    expect(await screen.findByRole('alert')).toHaveTextContent(/invalid or has expired/i)
    expect(screen.getByRole('link', { name: 'Request a new reset link' })).toHaveAttribute('href', '/auth/forgot-password')
    expect(bodyOf(0)).toEqual({ token: 'old', password: 'secret1' })
  })

  it('succeeds with a log in link, and checks the confirmation matches', async () => {
    h.search = 'token=good'
    mockApi({ 'POST /reset-password/confirm': { status: 204 } })
    render(<ResetPassword />)
    await submit('secret1', 'secret2')
    expect(await screen.findByRole('alert')).toHaveTextContent(/do not match/i)
    expect(fetchMock).not.toHaveBeenCalled()
    await userEvent.clear(screen.getByLabelText('Confirm new password'))
    await userEvent.type(screen.getByLabelText('Confirm new password'), 'secret1')
    fireEvent.submit(screen.getByRole('button', { name: 'Change password' }).closest('form')!)
    expect(await screen.findByRole('link', { name: 'Log in' })).toHaveAttribute('href', '/auth/login')
  })
})

describe('ChangePassword and sessions', () => {
  const sessions = [
    { id: 's1', created_at: '2026-01-01T00:00:00Z', last_seen: '2026-01-02T00:00:00Z', ip_address: '1.2.3.4', user_agent: 'Firefox', current: true },
    { id: 's2', created_at: '2026-01-01T00:00:00Z', last_seen: '2026-01-02T00:00:00Z', ip_address: '5.6.7.8', user_agent: 'Safari', current: false },
  ]
  beforeEach(() => {
    vi.mocked(localStorage.getItem).mockImplementation((k: string) => (k === 'auth_token' ? 'tok' : null))
  })

  it('shows the wrong_password message', async () => {
    mockApi({
      'GET /sessions': { status: 200, body: sessions },
      'POST /change-password': { status: 401, body: { error: 'wrong_password' } },
    })
    render(<ChangePassword />)
    await userEvent.type(screen.getByLabelText('Current password'), 'nope')
    await userEvent.type(screen.getByLabelText('New password'), 'secret1')
    await userEvent.type(screen.getByLabelText('Confirm new password'), 'secret1')
    await userEvent.click(screen.getByRole('button', { name: 'Change password' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Your current password is not correct.')
    const post = calls().find(([, i]) => i.method === 'POST')!
    expect(JSON.parse(post[1].body as string)).toEqual({ current_password: 'nope', new_password: 'secret1' })
    expect((post[1].headers as Record<string, string>).Authorization).toBe('Bearer tok')
  })

  it('lists sessions, marks the current one, and logs another out', async () => {
    mockApi({
      'GET /sessions': { status: 200, body: sessions },
      'DELETE /sessions/s2': { status: 204 },
    })
    render(<ChangePassword />)
    expect(await screen.findByText('(this session)')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Log out Safari' }))
    await waitFor(() => expect(calls().some(([u, i]) => i.method === 'DELETE' && u.endsWith('/sessions/s2'))).toBe(true))
  })
})

describe('logout', () => {
  it('calls POST /logout with the bearer token, then clears local state', async () => {
    vi.mocked(localStorage.getItem).mockImplementation((k: string) => (k === 'auth_token' ? 'tok' : null))
    const order: string[] = []
    fetchMock.mockImplementation(() => {
      order.push('fetch')
      return Promise.resolve({ ok: true, status: 204 } as Response)
    })
    vi.mocked(localStorage.removeItem).mockImplementation(() => { order.push('clear') })
    await logout()
    expect(calls()[0][0]).toBe('/api/v1/auth/logout')
    expect(calls()[0][1].method).toBe('POST')
    expect((calls()[0][1].headers as Record<string, string>).Authorization).toBe('Bearer tok')
    expect(order[0]).toBe('fetch')
    expect(localStorage.removeItem).toHaveBeenCalledWith('auth_token')
    expect(localStorage.removeItem).toHaveBeenCalledWith('user')
  })

  it('still clears local state when the network fails', async () => {
    vi.mocked(localStorage.getItem).mockImplementation((k: string) => (k === 'auth_token' ? 'tok' : null))
    fetchMock.mockRejectedValue(new Error('offline'))
    await logout()
    expect(localStorage.removeItem).toHaveBeenCalledWith('auth_token')
  })
})
