// The Contact and Feedback forms and the admin support page, against a mocked fetch.
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import Contact from '@/pages/Contact'
import Feedback from '@/pages/Feedback'
import AdminSupport from '@/pages/AdminSupport'

type Reply = { status: number; body?: unknown }
const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>

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

beforeEach(() => {
  fetchMock.mockReset()
  vi.mocked(localStorage.getItem).mockReset()
})

describe('Contact form', () => {
  it('sends a support request from a guest with an email address', async () => {
    mockApi({ 'POST /api/v1/support': { status: 201, body: { received: true } } })
    render(<Contact />)
    await userEvent.type(screen.getByLabelText(/Your email address/), 'guest@example.org')
    await userEvent.type(screen.getByLabelText('Subject'), 'Cannot log in')
    await userEvent.type(screen.getByLabelText('Message'), 'It says no.')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))
    expect(await screen.findByText('Message sent')).toBeInTheDocument()
    expect(bodyOf('POST /api/v1/support')).toEqual({
      kind: 'support', subject: 'Cannot log in', body: 'It says no.', email: 'guest@example.org',
    })
    expect(callTo('POST /api/v1/support')![1].headers).not.toHaveProperty('Authorization')
  })

  it('sends a policy and abuse report with the page address', async () => {
    mockApi({ 'POST /api/v1/support': { status: 201, body: { received: true } } })
    render(<Contact />)
    expect(screen.queryByLabelText(/Address of the page/)).not.toBeInTheDocument()
    await userEvent.click(screen.getByLabelText(/Policy and abuse/))
    await userEvent.type(screen.getByLabelText(/Your email address/), 'g@example.org')
    await userEvent.type(screen.getByLabelText(/Address of the page/), 'https://archive.example/works/1')
    await userEvent.type(screen.getByLabelText('Subject'), 'Spam')
    await userEvent.type(screen.getByLabelText('Message'), 'This is spam.')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))
    await screen.findByText('Message sent')
    expect(bodyOf('POST /api/v1/support')).toMatchObject({ kind: 'abuse', url: 'https://archive.example/works/1' })
  })

  it('asks no email address of a logged-in person and does not send one', async () => {
    logIn()
    mockApi({ 'POST /api/v1/support': { status: 201, body: { received: true } } })
    render(<Contact />)
    expect(screen.queryByLabelText(/Your email address/)).not.toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Subject'), 'Hi')
    await userEvent.type(screen.getByLabelText('Message'), 'Hello')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))
    await screen.findByText('Message sent')
    expect(bodyOf('POST /api/v1/support')).toEqual({ kind: 'support', subject: 'Hi', body: 'Hello' })
    expect((callTo('POST /api/v1/support')![1].headers as Record<string, string>).Authorization).toBe('Bearer tok')
  })

  it('shows the server message when it refuses, and keeps what was typed', async () => {
    mockApi({ 'POST /api/v1/support': { status: 429, body: { error: 'rate_limited', message: 'You have sent several messages recently. Please try again later.' } } })
    render(<Contact />)
    await userEvent.type(screen.getByLabelText(/Your email address/), 'g@example.org')
    await userEvent.type(screen.getByLabelText('Subject'), 'Hi')
    await userEvent.type(screen.getByLabelText('Message'), 'Hello')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('several messages recently')
    expect(screen.getByLabelText('Message')).toHaveValue('Hello')
  })

  it('shows an error when the server cannot be reached', async () => {
    fetchMock.mockRejectedValue(new Error('offline'))
    render(<Contact />)
    await userEvent.type(screen.getByLabelText(/Your email address/), 'g@example.org')
    await userEvent.type(screen.getByLabelText('Subject'), 'Hi')
    await userEvent.type(screen.getByLabelText('Message'), 'Hello')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not reach the server')
  })
})

describe('Feedback page', () => {
  it('is the support kind and offers no abuse choice', async () => {
    mockApi({ 'POST /api/v1/support': { status: 201, body: { received: true } } })
    render(<Feedback />)
    expect(screen.queryByLabelText(/Policy and abuse/)).not.toBeInTheDocument()
    await userEvent.type(screen.getByLabelText(/Your email address/), 'g@example.org')
    await userEvent.type(screen.getByLabelText('Subject'), 'Idea')
    await userEvent.type(screen.getByLabelText('Message'), 'More colours')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))
    await screen.findByText('Message sent')
    expect(bodyOf('POST /api/v1/support').kind).toBe('support')
  })
})

describe('Admin support page', () => {
  const ticket = {
    id: 't1', kind: 'abuse', subject: 'Bad <b>thing</b>', status: 'open', created_at: '2026-11-01T00:00:00Z',
    reporter_email: 'rep@example.org',
  }
  const detail = { ...ticket, body: '<img src=x onerror=alert(1)>', url: 'https://archive.example/works/1', notes: [] }

  it('lists tickets and shows text as text', async () => {
    logIn()
    mockApi({
      'GET /api/v1/admin/support': { status: 200, body: { tickets: [ticket], total: 1 } },
      'GET /api/v1/admin/support/t1': { status: 200, body: detail },
    })
    const { container } = render(<AdminSupport />)
    expect(await screen.findByText('Bad <b>thing</b>')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /View Bad/ }))
    expect(await screen.findByText('<img src=x onerror=alert(1)>')).toBeInTheDocument()
    expect(screen.getByText('rep@example.org')).toBeInTheDocument()
    expect(container.querySelector('img')).toBeNull()
  })

  it('filters by status and kind', async () => {
    logIn()
    mockApi({ 'GET /api/v1/admin/support': { status: 200, body: { tickets: [], total: 0 } } })
    render(<AdminSupport />)
    await screen.findByText('Nothing here.')
    expect(callTo('GET /api/v1/admin/support')![0]).toContain('status=open')
    await userEvent.selectOptions(screen.getByLabelText('Kind'), 'abuse')
    await waitFor(() => expect(calls().some(([u]) => u.includes('kind=abuse') && u.includes('status=open'))).toBe(true))
  })

  it('adds a note, sends a reply and closes the ticket', async () => {
    logIn()
    mockApi({
      'GET /api/v1/admin/support': { status: 200, body: { tickets: [ticket], total: 1 } },
      'GET /api/v1/admin/support/t1': { status: 200, body: detail },
      'POST /api/v1/admin/support/t1/notes': { status: 201, body: { added: true } },
      'POST /api/v1/admin/support/t1/close': { status: 200, body: { status: 'closed' } },
    })
    render(<AdminSupport />)
    await userEvent.click(await screen.findByRole('button', { name: /View Bad/ }))
    const box = await screen.findByLabelText('Add a note')
    await userEvent.type(box, 'checked it')
    await userEvent.click(screen.getByRole('button', { name: 'Add note' }))
    await waitFor(() => expect(callTo('POST /api/v1/admin/support/t1/notes')).toBeTruthy())
    expect(bodyOf('POST /api/v1/admin/support/t1/notes')).toEqual({ body: 'checked it', reply: false })

    await userEvent.type(screen.getByLabelText('Add a note'), 'we fixed it')
    await userEvent.click(screen.getByLabelText('Also email this to the sender'))
    await userEvent.click(screen.getByRole('button', { name: 'Send reply' }))
    await waitFor(() => expect(calls().filter(([u, i]) => u.endsWith('/t1/notes') && i.method === 'POST')).toHaveLength(2))

    await userEvent.click(screen.getByRole('button', { name: 'Close request' }))
    await waitFor(() => expect(callTo('POST /api/v1/admin/support/t1/close')).toBeTruthy())
  })

  it('tells someone without the role that the page is not for them', async () => {
    logIn()
    mockApi({ 'GET /api/v1/admin/support': { status: 403, body: { error: 'you don\'t have permission to do this' } } })
    render(<AdminSupport />)
    expect(await screen.findByRole('alert')).toHaveTextContent('admins and moderators')
  })
})
