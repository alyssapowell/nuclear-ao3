// The admin screens against a mocked fetch: role gating, confirm dialogs,
// what is sent, focus, and how errors show. The server is the real check;
// these tests are about what the screens do and show.
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import AdminHome from '@/pages/AdminHome'
import AdminAccounts from '@/pages/AdminAccounts'
import AdminContent from '@/pages/AdminContent'
import AdminReports from '@/pages/AdminReports'
import AdminAudit from '@/pages/AdminAudit'
import Navigation from '@/components/Navigation'
import { parseContentRef } from '@/lib/admin'

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
const keyOf = ([u, i]: [string, RequestInit]) => `${i?.method ?? 'GET'} ${u.split('?')[0]}`
const callTo = (key: string) => calls().find((c) => keyOf(c) === key)
const callsTo = (key: string) => calls().filter((c) => keyOf(c) === key)
const bodyOf = (key: string) => JSON.parse(callTo(key)![1].body as string)
const adminCalls = () => calls().filter(([u]) => u.startsWith('/api/v1/admin') || u.startsWith('/api/v1/auth/admin'))

const logIn = (user?: object) =>
  vi.mocked(localStorage.getItem).mockImplementation((k: string) =>
    k === 'auth_token' ? 'tok' : k === 'user' && user ? JSON.stringify(user) : null)

const ME = 'GET /api/v1/auth/me'
const as = (...roles: string[]) => ({ status: 200, body: { id: 'me', roles } })

const TARGET = '7d1f0c52-aaaa-4bbb-8ccc-000000000001'
const WORK = '11111111-2222-4333-8444-555555555555'
const CHAPTER = '66666666-7777-4888-8999-000000000000'
const account = (over: object = {}) => ({
  id: TARGET, email: 'target@example.org', created_at: '2026-01-02T03:04:05Z', is_verified: true,
  banned: false, suspended: false, roles: [], ...over,
})

beforeEach(() => {
  fetchMock.mockReset()
  vi.mocked(localStorage.getItem).mockReset()
  logIn()
})

describe('parseContentRef', () => {
  it('reads ids and page addresses', () => {
    expect(parseContentRef(WORK)).toEqual({ id: WORK })
    expect(parseContentRef(`https://archive.example.org/works/${WORK}`)).toEqual({ kind: 'work', id: WORK })
    expect(parseContentRef(`/works/${WORK}/chapters/${CHAPTER}`)).toEqual({ kind: 'chapter', id: CHAPTER })
    expect(parseContentRef(`/series/${WORK}`)).toEqual({ kind: 'series', id: WORK })
    expect(parseContentRef(`/bookmarks/${WORK}`)).toEqual({ kind: 'bookmark', id: WORK })
    expect(parseContentRef(`/works/${WORK}#comment-${CHAPTER}`)).toEqual({ kind: 'comment', id: CHAPTER })
    expect(parseContentRef('not an id')).toBeNull()
  })
})

describe('Navigation admin link', () => {
  const pen = { id: 'u', email: 'me@example.org', default_pseud: { id: 'p', name: 'Pen' } }

  it('is shown to a moderator and an admin', async () => {
    for (const roles of [['moderator'], ['admin']]) {
      logIn(pen)
      mockApi({ [ME]: as(...roles) })
      const { unmount } = render(<Navigation />)
      await userEvent.click(await screen.findByRole('button', { name: 'User menu' }))
      expect(await screen.findByRole('link', { name: 'Admin' })).toHaveAttribute('href', '/admin')
      unmount()
    }
  })

  it('is not shown to a wrangler, an ordinary person or when the roles cannot be read', async () => {
    for (const reply of [as('tag_wrangler'), as(), { status: 401, body: { error: 'x' } }]) {
      logIn(pen)
      mockApi({ [ME]: reply })
      const { unmount } = render(<Navigation />)
      await waitFor(() => expect(callsTo(ME).length).toBeGreaterThan(0))
      await userEvent.click(await screen.findByRole('button', { name: 'User menu' }))
      expect(screen.getByRole('link', { name: 'Profile Settings' })).toBeInTheDocument()
      expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument()
      unmount()
      fetchMock.mockReset()
    }
  })

  it('asks the server nothing for a logged-out visitor', async () => {
    vi.mocked(localStorage.getItem).mockReset()
    mockApi({})
    render(<Navigation />)
    expect(await screen.findByRole('link', { name: 'Log In' })).toBeInTheDocument()
    expect(calls()).toHaveLength(0)
  })
})

describe('Admin home', () => {
  it('links a moderator to accounts, content and reports only', async () => {
    mockApi({ [ME]: as('moderator') })
    render(<AdminHome />)
    expect(await screen.findByRole('link', { name: 'Accounts' })).toHaveAttribute('href', '/admin/accounts')
    expect(screen.getByRole('link', { name: 'Content' })).toHaveAttribute('href', '/admin/content')
    expect(screen.getByRole('link', { name: 'Reports' })).toHaveAttribute('href', '/admin/reports')
    expect(screen.queryByRole('link', { name: 'Audit log' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'App review' })).not.toBeInTheDocument()
  })

  it('links an admin to everything', async () => {
    mockApi({ [ME]: as('admin') })
    render(<AdminHome />)
    expect(await screen.findByRole('link', { name: 'Audit log' })).toHaveAttribute('href', '/admin/audit')
    expect(screen.getByRole('link', { name: 'App review' })).toHaveAttribute('href', '/admin/apps')
  })

  it('refuses an ordinary person', async () => {
    mockApi({ [ME]: as() })
    render(<AdminHome />)
    expect(await screen.findByRole('alert')).toHaveTextContent(/for the archive's admins and moderators/)
    expect(screen.queryByRole('link', { name: 'Accounts' })).not.toBeInTheDocument()
  })
})

describe('Every screen refuses people without the role and asks for nothing', () => {
  it.each([
    ['accounts', AdminAccounts, as('tag_wrangler')],
    ['content', AdminContent, as()],
    ['reports', AdminReports, as('tag_wrangler')],
    ['audit (moderator)', AdminAudit, as('moderator')],
  ])('%s', async (_name, Page, me) => {
    mockApi({ [ME]: me })
    render(<Page />)
    expect(await screen.findByRole('alert')).toHaveTextContent(/This page is for/)
    expect(adminCalls()).toHaveLength(0)
  })
})

describe('Accounts', () => {
  const find = 'GET /api/v1/admin/accounts'

  it('finds by email, shows status and roles, and offers no role controls to a moderator', async () => {
    mockApi({
      [ME]: as('moderator'),
      [find]: { status: 200, body: { account: account({ suspended: true, suspended_until: '2030-05-06T00:00:00Z', roles: ['tag_wrangler'] }) } },
    })
    render(<AdminAccounts />)
    await userEvent.type(await screen.findByLabelText('Email address or account id'), ' target@example.org ')
    await userEvent.click(screen.getByRole('button', { name: 'Find' }))
    expect(await screen.findByRole('heading', { name: 'Account' })).toBeInTheDocument()
    expect(callTo(find)![0]).toBe('/api/v1/admin/accounts?q=target%40example.org')
    expect(screen.getByText('target@example.org')).toBeInTheDocument()
    expect(screen.getByText(TARGET)).toBeInTheDocument()
    expect(screen.getByText(/Suspended until/)).toBeInTheDocument()
    expect(screen.getByText('tag wrangler')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lift suspension' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Suspend…' })).not.toBeInTheDocument()
    expect(screen.queryByText(/Roles \(admins only\)/)).not.toBeInTheDocument()
  })

  it('says so when there is no such account, and shows other errors', async () => {
    mockApi({ [ME]: as('moderator'), [find]: { status: 404, body: { error: 'not_found' } } })
    render(<AdminAccounts />)
    await userEvent.type(await screen.findByLabelText('Email address or account id'), 'nobody@example.org')
    await userEvent.click(screen.getByRole('button', { name: 'Find' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('No account has that email address or id.')

    mockApi({ [ME]: as('moderator'), [find]: { status: 500, body: { error: 'server_error' } } })
    await userEvent.click(screen.getByRole('button', { name: 'Find' }))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('Something went wrong'))
  })

  async function showAccount(roles: string[], acc = account()) {
    mockApi({
      [ME]: as(...roles),
      [find]: { status: 200, body: { account: acc } },
      [`POST /api/v1/admin/users/${TARGET}/ban`]: { status: 200, body: { ok: true } },
      [`POST /api/v1/admin/users/${TARGET}/suspend`]: { status: 200, body: { ok: true } },
      [`POST /api/v1/admin/users/${TARGET}/logout-everywhere`]: { status: 200, body: { ok: true } },
      [`POST /api/v1/auth/admin/users/${TARGET}/roles`]: { status: 200, body: { granted: true } },
      [`DELETE /api/v1/auth/admin/users/${TARGET}/roles/moderator`]: { status: 200, body: { revoked: true } },
    })
    render(<AdminAccounts />)
    await userEvent.type(await screen.findByLabelText('Email address or account id'), TARGET)
    await userEvent.click(screen.getByRole('button', { name: 'Find' }))
    await screen.findByRole('heading', { name: 'Account' })
  }

  it('asks before banning, needs a reason, and sends nothing on Cancel', async () => {
    await showAccount(['moderator'])
    const banButton = screen.getByRole('button', { name: 'Ban…' })
    await userEvent.click(banButton)
    const dialog = screen.getByRole('alertdialog', { name: 'Ban this account?' })
    expect(dialog).toBeInTheDocument()
    expect(screen.getByLabelText(/Reason \(required\)/)).toHaveFocus()
    expect(within(dialog).getByRole('button', { name: 'Ban' })).toBeDisabled()

    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(banButton).toHaveFocus()
    expect(callsTo(`POST /api/v1/admin/users/${TARGET}/ban`)).toHaveLength(0)

    await userEvent.click(banButton)
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(callsTo(`POST /api/v1/admin/users/${TARGET}/ban`)).toHaveLength(0)
  })

  it('bans with the reason, then reloads the account and puts focus on its heading', async () => {
    await showAccount(['moderator'])
    await userEvent.click(screen.getByRole('button', { name: 'Ban…' }))
    await userEvent.type(screen.getByLabelText(/Reason \(required\)/), 'harassment ring')
    await userEvent.click(screen.getByRole('button', { name: 'Ban' }))
    await screen.findByText('Done: banned.')
    expect(bodyOf(`POST /api/v1/admin/users/${TARGET}/ban`)).toEqual({ reason: 'harassment ring' })
    expect(callsTo(find).length).toBe(2)
    await waitFor(() => expect(screen.getByRole('heading', { name: 'Account' })).toHaveFocus())
  })

  it('suspends with an end date and a reason', async () => {
    await showAccount(['moderator'])
    await userEvent.click(screen.getByRole('button', { name: 'Suspend…' }))
    const confirm = screen.getByRole('button', { name: 'Suspend' })
    const date = screen.getByLabelText(/Suspended until/)
    expect(date).toHaveFocus()
    await userEvent.type(screen.getByLabelText(/Reason \(required\)/), 'cool off')
    expect(confirm).toBeDisabled() // no date yet
    await userEvent.type(date, '2099-12-31')
    expect(confirm).toBeEnabled()
    await userEvent.click(confirm)
    await screen.findByText('Done: suspended.')
    expect(bodyOf(`POST /api/v1/admin/users/${TARGET}/suspend`)).toEqual({ reason: 'cool off', until: '2099-12-31' })
  })

  it('does not accept a suspension that ends in the past', async () => {
    await showAccount(['moderator'])
    await userEvent.click(screen.getByRole('button', { name: 'Suspend…' }))
    await userEvent.type(screen.getByLabelText(/Reason \(required\)/), 'x')
    await userEvent.type(screen.getByLabelText(/Suspended until/), '2001-01-01')
    expect(screen.getByRole('button', { name: 'Suspend' })).toBeDisabled()
  })

  it('shows the lift buttons for a banned account', async () => {
    await showAccount(['moderator'], account({ banned: true }) as ReturnType<typeof account>)
    expect(screen.getByText('Banned')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lift ban' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Ban…' })).not.toBeInTheDocument()
  })

  it('logs out everywhere after asking, and shows the server error inside the box', async () => {
    await showAccount(['moderator'])
    mockApi({
      [ME]: as('moderator'),
      [`POST /api/v1/admin/users/${TARGET}/logout-everywhere`]: { status: 403, body: { error: 'forbidden', message: 'only an admin can do that to an admin or moderator' } },
    })
    await userEvent.click(screen.getByRole('button', { name: 'Log out everywhere' }))
    const dialog = screen.getByRole('alertdialog', { name: 'Log this account out everywhere?' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Log out everywhere' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent('only an admin can do that to an admin or moderator')
    expect(screen.getByRole('alertdialog')).toBeInTheDocument() // stays open so nothing is lost
  })

  it('keeps Tab inside the box', async () => {
    await showAccount(['moderator'])
    await userEvent.click(screen.getByRole('button', { name: 'Ban…' }))
    const dialog = screen.getByRole('alertdialog')
    const reason = screen.getByLabelText(/Reason/)
    await userEvent.type(reason, 'x')
    await userEvent.tab() // Cancel
    await userEvent.tab() // Ban
    await userEvent.tab() // wraps to the reason field
    expect(dialog).toContainElement(document.activeElement as HTMLElement)
    expect(reason).toHaveFocus()
  })

  it('lets an admin grant and revoke roles, after asking', async () => {
    await showAccount(['admin'], account({ roles: ['moderator'] }) as ReturnType<typeof account>)
    await userEvent.click(screen.getByRole('button', { name: 'Grant the tag wrangler role' }))
    await userEvent.click(within(screen.getByRole('alertdialog', { name: 'Give the tag wrangler role?' })).getByRole('button', { name: 'Give role' }))
    await screen.findByText('Done: tag wrangler role granted.')
    expect(bodyOf(`POST /api/v1/auth/admin/users/${TARGET}/roles`)).toEqual({ role: 'tag_wrangler' })

    await userEvent.click(screen.getByRole('button', { name: 'Revoke the moderator role' }))
    expect(callsTo(`DELETE /api/v1/auth/admin/users/${TARGET}/roles/moderator`)).toHaveLength(0)
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Take away role' }))
    await screen.findByText('Done: moderator role revoked.')
  })
})

describe('Content', () => {
  const hide = (kind: string, id: string, action = 'hide') => `POST /api/v1/admin/${kind}/${id}/${action}`

  it('hides a chapter found from its address, after asking for a reason', async () => {
    mockApi({ [ME]: as('moderator'), [hide('chapters', CHAPTER)]: { status: 200, body: { hidden: true } } })
    render(<AdminContent />)
    await userEvent.type(await screen.findByLabelText('Address or id'), `https://x.example.org/works/${WORK}/chapters/${CHAPTER}`)
    await userEvent.click(screen.getByRole('button', { name: 'Continue' }))
    expect(await screen.findByRole('heading', { name: /Chapter/ })).toBeInTheDocument()
    expect(screen.getByLabelText(/Type/)).toHaveValue('chapter')
    // A chapter has no spam mark.
    expect(screen.queryByRole('button', { name: /spam/i })).not.toBeInTheDocument()

    const opener = screen.getByRole('button', { name: `Hide: chapter ${CHAPTER}` })
    await userEvent.click(opener)
    const dialog = screen.getByRole('alertdialog', { name: 'Hide this chapter?' })
    expect(within(dialog).getByRole('button', { name: 'Hide' })).toBeDisabled()
    await userEvent.type(within(dialog).getByLabelText(/Reason/), 'doxxing')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Hide' }))
    await screen.findByText(`Done: chapter ${CHAPTER} hidden.`)
    expect(bodyOf(hide('chapters', CHAPTER))).toEqual({ reason: 'doxxing' })
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
  })

  it('unhides at once, with no box', async () => {
    mockApi({ [ME]: as('moderator'), [hide('series', WORK, 'unhide')]: { status: 200, body: { hidden: false } } })
    render(<AdminContent />)
    await userEvent.type(await screen.findByLabelText('Address or id'), WORK)
    await userEvent.selectOptions(screen.getByLabelText(/Type/), 'series')
    await userEvent.click(screen.getByRole('button', { name: 'Continue' }))
    await userEvent.click(await screen.findByRole('button', { name: `Unhide: series ${WORK}` }))
    await screen.findByText(`Done: series ${WORK} unhidden.`)
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('marks a comment as spam and shows the server refusal', async () => {
    mockApi({ [ME]: as('moderator'), [hide('comments', CHAPTER, 'spam')]: { status: 409, body: { error: 'no_change', message: 'It is already in that state.' } } })
    render(<AdminContent />)
    await userEvent.type(await screen.findByLabelText('Address or id'), `/works/${WORK}#comment-${CHAPTER}`)
    await userEvent.click(screen.getByRole('button', { name: 'Continue' }))
    await userEvent.click(await screen.findByRole('button', { name: `Mark as spam: comment ${CHAPTER}` }))
    await userEvent.type(screen.getByLabelText(/Reason/), 'ad')
    await userEvent.click(screen.getByRole('button', { name: 'Mark as spam' }))
    expect(await within(screen.getByRole('alertdialog')).findByRole('alert')).toHaveTextContent('It is already in that state.')
  })

  it('rejects something that is not an id or address', async () => {
    mockApi({ [ME]: as('admin') })
    render(<AdminContent />)
    await userEvent.type(await screen.findByLabelText('Address or id'), 'hello')
    await userEvent.click(screen.getByRole('button', { name: 'Continue' }))
    expect(screen.getByRole('alert')).toHaveTextContent(/not an id/)
    expect(adminCalls()).toHaveLength(0)
  })
})

describe('Reports', () => {
  const list = 'GET /api/v1/admin/abuse-reports'
  const report = (over: object = {}) => ({
    kind: 'work', id: 'r1', target_id: WORK, work_id: WORK, target_title: 'A reported work', target_hidden: false,
    reason: 'harassment', description: 'it targets me', status: 'pending', created_at: '2026-10-01T00:00:00Z', ...over,
  })

  it('lists reports with a link to the content and sends the filters', async () => {
    mockApi({
      [ME]: as('moderator'),
      [list]: { status: 200, body: { reports: [report(), report({ kind: 'comment', id: 'r2', target_id: CHAPTER, target_title: 'rude words', description: 'rude' })], has_more: false } },
    })
    render(<AdminReports />)
    expect(await screen.findByText('A reported work')).toBeInTheDocument()
    expect(callTo(list)![0]).toContain('status=pending')
    expect(screen.getByText('it targets me')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open the work' })).toHaveAttribute('href', `/works/${WORK}`)
    expect(screen.getByRole('link', { name: /Open the comment/ })).toHaveAttribute('href', `/works/${WORK}#comment-${CHAPTER}`)
    expect(screen.getByRole('button', { name: `Hide: work ${WORK}` })).toBeInTheDocument()
    expect(screen.getByText(/Reports about tags are not collected yet/)).toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('About'), 'comment')
    await waitFor(() => expect(calls().some(([u]) => u.includes('kind=comment'))).toBe(true))
  })

  it('resolves a report after asking, and reloads', async () => {
    let n = 0
    mockApi({
      [ME]: as('moderator'),
      [list]: () => ({ status: 200, body: { reports: n++ === 0 ? [report()] : [], has_more: false } }),
      'POST /api/v1/admin/abuse-reports/work/r1/resolve': { status: 200, body: { status: 'resolved' } },
    })
    render(<AdminReports />)
    await userEvent.click(await screen.findByRole('button', { name: `Resolve report about work ${WORK}` }))
    expect(callsTo('POST /api/v1/admin/abuse-reports/work/r1/resolve')).toHaveLength(0)
    await userEvent.type(screen.getByLabelText('Note (optional)'), 'hid it')
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Resolve' }))
    await screen.findByText('Nothing here.')
    expect(bodyOf('POST /api/v1/admin/abuse-reports/work/r1/resolve')).toEqual({ reason: 'hid it' })
  })

  it('pages with Next and Previous', async () => {
    mockApi({ [ME]: as('admin'), [list]: { status: 200, body: { reports: [report()], has_more: true } } })
    render(<AdminReports />)
    await screen.findByText('A reported work')
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() => expect(calls().some(([u]) => u.includes('offset=25'))).toBe(true))
    expect(await screen.findByRole('button', { name: 'Previous' })).toBeEnabled()
  })

  it('shows an error when the list cannot load', async () => {
    mockApi({ [ME]: as('moderator'), [list]: { status: 500, body: { error: 'server_error' } } })
    render(<AdminReports />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong')
  })
})

describe('Audit log', () => {
  const list = 'GET /api/v1/admin/audit'
  const entry = (id: number) => ({
    id, at: '2026-10-02T10:00:00Z', actor: 'boss@example.org', action: 'grant_role',
    target_type: 'user', target_id: TARGET, reason: `because ${id}`,
  })

  it('shows rows, filters and pages for an admin', async () => {
    mockApi({ [ME]: as('admin'), [list]: { status: 200, body: { actions: [entry(2), entry(1)], has_more: true } } })
    render(<AdminAudit />)
    const table = await screen.findByRole('table')
    expect(within(table).getAllByRole('row')).toHaveLength(3)
    expect(within(table).getByText('because 2')).toBeInTheDocument()
    expect(within(table).getAllByText('grant role')).toHaveLength(2)

    await userEvent.type(screen.getByLabelText('From date'), '2026-10-01')
    await userEvent.type(screen.getByLabelText('Up to and including date'), '2026-10-31')
    await userEvent.type(screen.getByLabelText('Action'), 'ban')
    await userEvent.type(screen.getByLabelText(/Done by/), 'boss@example.org')
    await userEvent.click(screen.getByRole('button', { name: 'Apply filters' }))
    await waitFor(() => expect(calls().some(([u]) => u.includes('action=ban'))).toBe(true))
    const url = new URL(calls().filter(([u]) => u.includes('action=ban')).pop()![0], 'http://x')
    expect(Object.fromEntries(url.searchParams)).toEqual({
      limit: '50', offset: '0', since: '2026-10-01', until: '2026-10-31', action: 'ban', actor: 'boss@example.org',
    })

    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() => expect(calls().some(([u]) => u.includes('offset=50'))).toBe(true))
    expect(await screen.findByText('Page 2')).toBeInTheDocument()
  })

  it('says when nothing matches and shows errors', async () => {
    mockApi({ [ME]: as('admin'), [list]: { status: 200, body: { actions: [], has_more: false } } })
    const { unmount } = render(<AdminAudit />)
    expect(await screen.findByText('Nothing matches.')).toBeInTheDocument()
    unmount()

    mockApi({ [ME]: as('admin'), [list]: { status: 400, body: { error: 'bad_request', message: 'until must be a date like 2026-10-01 or a time like 2026-10-01T00:00:00Z' } } })
    render(<AdminAudit />)
    expect(await screen.findByRole('alert')).toHaveTextContent(/until must be a date/)
  })
})
