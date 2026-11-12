// The public pseud page and the "My pseuds" screen, against a mocked API.
// The pseud page must show one pseud and nothing that ties it to another.
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

const h = vi.hoisted(() => ({
  id: 'pseud-1',
  auth: { user: null as null | { id: string; email: string }, token: null as null | string },
  api: {} as Record<string, ReturnType<typeof vi.fn>>,
}))

vi.mock('react-router-dom', async () =>
  (await import('@/test/router-mock')).routerMock({ useParams: () => ({ id: h.id }) }))

vi.mock('@/utils/auth', () => ({ useAuth: () => h.auth }))

// The subscribe button has its own tests; here we check what the page hands it.
vi.mock('@/components/SubscriptionButton', () => ({
  SubscriptionButton: (p: { type: string; targetId: string; targetName: string }) => (
    <button type="button" data-testid="subscribe" data-type={p.type} data-target={p.targetId}>
      Subscribe to {p.targetName}
    </button>
  ),
}))

vi.mock('@/lib/api', () => {
  for (const name of [
    'getPseud', 'getPseudWorks', 'getPseudSeries', 'getPseudBookmarks', 'getPseudGifts',
    'listMyPseuds', 'createPseud', 'updatePseud', 'deletePseud',
  ]) h.api[name] = vi.fn()
  return h.api
})

import PseudPage from '@/pages/PseudPage'
import MyPseuds from '@/pages/MyPseuds'
import PseudPicker from '@/components/PseudPicker'
import { PseudList } from '@/components/PseudLink'
import { getPseudUrl } from '@/lib/urls'

const pagination = { page: 1, limit: 20, total: 1, total_pages: 1 }
const head = {
  id: 'pseud-1', name: 'Quill Pen', description: '<p>I write <b>things</b>.</p>',
  icon_url: '', icon_alt_text: '', icon_comment_text: '', owned: false,
}
const work = {
  id: 'w1', title: 'A Tale', summary: '<p>Short</p>', rating: 'general', fandoms: ['Some Fandom'],
  relationships: [], word_count: 1200, chapter_count: 2, is_complete: true,
  authors: [{ id: 'pseud-1', name: 'Quill Pen' }],
}

function loadPseudPage(over: Record<string, unknown> = {}) {
  h.api.getPseud.mockResolvedValue({ ...head, ...over })
  h.api.getPseudWorks.mockResolvedValue({ pseud: { id: 'pseud-1', name: 'Quill Pen' }, works: [work], pagination })
  h.api.getPseudSeries.mockResolvedValue({ series: [], pagination: { ...pagination, total: 0, total_pages: 0 } })
  h.api.getPseudBookmarks.mockResolvedValue({ bookmarks: [], pagination: { ...pagination, total: 0, total_pages: 0 } })
  h.api.getPseudGifts.mockResolvedValue({ gifts: [], pagination: { ...pagination, total: 0, total_pages: 0 } })
}

beforeEach(() => {
  for (const f of Object.values(h.api)) f.mockReset()
  h.id = 'pseud-1'
  h.auth = { user: null, token: null }
})

describe('urls', () => {
  it('links to a pseud by its own id only', () => {
    expect(getPseudUrl('abc-123')).toBe('/pseuds/abc-123')
  })
})

describe('author names', () => {
  it('link to the pseud page, and a work with no authors says Anonymous with no link', () => {
    const { rerender } = render(<PseudList pseuds={[{ id: 'a', name: 'One' }, { id: 'b', name: 'Two' }]} />)
    expect(screen.getByRole('link', { name: 'One' })).toHaveAttribute('href', '/pseuds/a')
    expect(screen.getByRole('link', { name: 'Two' })).toHaveAttribute('href', '/pseuds/b')
    rerender(<PseudList pseuds={[]} />)
    expect(screen.getByText('Anonymous')).toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})

describe('PseudPage', () => {
  it('shows the name, description and the works, and asks the API for this pseud only', async () => {
    loadPseudPage()
    render(<PseudPage />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Quill Pen' })).toBeInTheDocument()
    expect(screen.getByText('things')).toBeInTheDocument()
    expect(await screen.findByRole('link', { name: 'A Tale' })).toHaveAttribute('href', '/works/w1')
    expect(screen.getByRole('link', { name: 'Quill Pen' })).toHaveAttribute('href', '/pseuds/pseud-1')
    expect(h.api.getPseud.mock.calls[0][0]).toBe('pseud-1')
    expect(h.api.getPseudWorks.mock.calls[0][0]).toBe('pseud-1')
    // Nothing offers other pseuds of the person.
    expect(screen.queryByText(/other pseuds|also writes|same (user|person|account)/i)).not.toBeInTheDocument()
  })

  it('shows the icon with its alt text when there is one', async () => {
    loadPseudPage({ icon_url: 'https://example.org/i.png', icon_alt_text: 'A quill', icon_comment_text: 'Hello' })
    render(<PseudPage />)
    const img = await screen.findByRole('img', { name: 'A quill' })
    expect(img).toHaveAttribute('src', 'https://example.org/i.png')
  })

  it('shows an empty state for each tab', async () => {
    loadPseudPage()
    h.api.getPseudWorks.mockResolvedValue({ works: [], pagination: { ...pagination, total: 0, total_pages: 0 } })
    const user = userEvent.setup()
    render(<PseudPage />)
    expect(await screen.findByText('No works yet.')).toBeInTheDocument()
    await user.click(screen.getByRole('tab', { name: 'Series' }))
    expect(await screen.findByText('No series yet.')).toBeInTheDocument()
    await user.click(screen.getByRole('tab', { name: 'Bookmarks' }))
    expect(await screen.findByText('No public bookmarks yet.')).toBeInTheDocument()
    await user.click(screen.getByRole('tab', { name: 'Gifts' }))
    expect(await screen.findByText('No gifts yet.')).toBeInTheDocument()
  })

  it('lists series, bookmarks and gifts under their tabs', async () => {
    loadPseudPage()
    h.api.getPseudSeries.mockResolvedValue({
      series: [{ id: 's1', title: 'A Saga', authors: [{ id: 'pseud-1', name: 'Quill Pen' }], work_count: 3, is_complete: false }],
      pagination,
    })
    h.api.getPseudBookmarks.mockResolvedValue({
      bookmarks: [{ id: 'b1', notes: 'Loved it', tags: ['comfort'], work: { ...work, id: 'w2', title: 'Bookmarked One', authors: [{ id: 'p9', name: 'Other Author' }] } }],
      pagination,
    })
    h.api.getPseudGifts.mockResolvedValue({
      gifts: [{ work: { ...work, id: 'w3', title: 'A Gift Work', authors: [] }, created_at: '2024-01-01T00:00:00Z' }],
      pagination,
    })
    const user = userEvent.setup()
    render(<PseudPage />)
    await user.click(await screen.findByRole('tab', { name: 'Series' }))
    expect(await screen.findByRole('link', { name: 'A Saga' })).toHaveAttribute('href', '/series/s1')
    await user.click(screen.getByRole('tab', { name: 'Bookmarks' }))
    expect(await screen.findByRole('link', { name: 'Bookmarked One' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Other Author' })).toHaveAttribute('href', '/pseuds/p9')
    expect(screen.getByText('Loved it')).toBeInTheDocument()
    await user.click(screen.getByRole('tab', { name: 'Gifts' }))
    expect(await screen.findByRole('link', { name: 'A Gift Work' })).toBeInTheDocument()
    expect(screen.getByText('Anonymous')).toBeInTheDocument()
  })

  it('tabs work from the keyboard', async () => {
    loadPseudPage()
    const user = userEvent.setup()
    render(<PseudPage />)
    const works = await screen.findByRole('tab', { name: 'Works' })
    expect(works).toHaveAttribute('aria-selected', 'true')
    works.focus()
    await user.keyboard('{ArrowRight}')
    expect(screen.getByRole('tab', { name: 'Series' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: 'Series' })).toHaveFocus()
    await user.keyboard('{End}')
    expect(screen.getByRole('tab', { name: 'Gifts' })).toHaveFocus()
    await user.keyboard('{ArrowRight}')
    expect(screen.getByRole('tab', { name: 'Works' })).toHaveFocus()
    expect(screen.getByRole('tabpanel')).toBeInTheDocument()
  })

  it('offers subscribe to a signed-in reader, aimed at the pseud', async () => {
    loadPseudPage()
    h.auth = { user: { id: 'u', email: 'r@example.org' }, token: 't' }
    render(<PseudPage />)
    const btn = await screen.findByTestId('subscribe')
    expect(btn).toHaveAttribute('data-type', 'author')
    expect(btn).toHaveAttribute('data-target', 'pseud-1')
  })

  it('sends a signed-out reader to sign in instead', async () => {
    loadPseudPage()
    render(<PseudPage />)
    const link = await screen.findByRole('link', { name: 'Sign in to subscribe' })
    expect(link.getAttribute('href')).toContain('/auth/login?redirect=')
    expect(screen.queryByTestId('subscribe')).not.toBeInTheDocument()
  })

  it('offers its owner management instead of subscribing', async () => {
    loadPseudPage({ owned: true })
    h.auth = { user: { id: 'u', email: 'o@example.org' }, token: 't' }
    render(<PseudPage />)
    expect(await screen.findByRole('link', { name: 'Manage my pseuds' })).toHaveAttribute('href', '/my/pseuds')
    expect(screen.queryByTestId('subscribe')).not.toBeInTheDocument()
  })

  it('says so when the pseud is not found', async () => {
    h.api.getPseud.mockRejectedValue(new Error('Pseud not found'))
    render(<PseudPage />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Pseud not found')
    expect(h.api.getPseudWorks).not.toHaveBeenCalled()
  })

  it('pages through a long list', async () => {
    loadPseudPage()
    h.api.getPseudWorks.mockResolvedValue({ works: [work], pagination: { page: 1, limit: 20, total: 40, total_pages: 2 } })
    const user = userEvent.setup()
    render(<PseudPage />)
    await screen.findByRole('link', { name: 'A Tale' })
    await user.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() => expect(h.api.getPseudWorks.mock.calls.at(-1)![1]).toEqual({ page: 2, limit: 20 }))
  })
})

describe('MyPseuds', () => {
  const mine = [
    { id: 'p1', name: 'Main Pen', description: 'Home base', is_default: true, icon_url: '', icon_alt_text: '', icon_comment_text: '' },
    { id: 'p2', name: 'Side Pen', description: '', is_default: false, icon_url: '', icon_alt_text: '', icon_comment_text: '' },
  ]
  beforeEach(() => {
    h.auth = { user: { id: 'u', email: 'me@example.org' }, token: 'tok' }
    h.api.listMyPseuds.mockResolvedValue(mine)
  })

  it('lists my pseuds, marks the default, and cannot delete it', async () => {
    render(<MyPseuds />)
    expect(await screen.findByRole('heading', { level: 2, name: 'Main Pen' })).toBeInTheDocument()
    const main = screen.getByRole('heading', { name: 'Main Pen' }).closest('li')!
    expect(within(main).getByText('Default')).toBeInTheDocument()
    expect(within(main).queryByRole('button', { name: /Delete/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete Side Pen' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'View the page of Side Pen' })).toHaveAttribute('href', '/pseuds/p2')
  })

  it('creates a pseud with a name and description', async () => {
    h.api.createPseud.mockResolvedValue({ ...mine[1], id: 'p3', name: 'Third Pen' })
    const user = userEvent.setup()
    render(<MyPseuds />)
    await screen.findByText('Side Pen')
    await user.type(screen.getByLabelText('Name'), 'Third Pen')
    await user.type(screen.getAllByLabelText('Description').at(-1)!, 'for fluff')
    await user.click(screen.getByRole('button', { name: 'Create pseud' }))
    await waitFor(() => expect(h.api.createPseud).toHaveBeenCalledWith({ name: 'Third Pen', description: 'for fluff' }, 'tok'))
    expect(await screen.findByRole('status')).toHaveTextContent('Created Third Pen.')
    expect(h.api.listMyPseuds).toHaveBeenCalledTimes(2)
  })

  it('shows the server error when a name is taken', async () => {
    h.api.createPseud.mockRejectedValue(new Error('You already have a pseud with that name'))
    const user = userEvent.setup()
    render(<MyPseuds />)
    await screen.findByText('Side Pen')
    await user.type(screen.getByLabelText('Name'), 'Main Pen')
    await user.click(screen.getByRole('button', { name: 'Create pseud' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('You already have a pseud with that name')
  })

  it('renames a pseud and edits its description', async () => {
    h.api.updatePseud.mockResolvedValue({ ...mine[1], name: 'Renamed' })
    const user = userEvent.setup()
    render(<MyPseuds />)
    await user.click(await screen.findByRole('button', { name: 'Edit Side Pen' }))
    const name = screen.getByDisplayValue('Side Pen')
    await user.clear(name)
    await user.type(name, 'Renamed')
    const li = name.closest('li')!
    await user.type(within(li).getByLabelText('Description'), 'new words')
    await user.click(within(li).getByRole('button', { name: 'Save changes' }))
    await waitFor(() =>
      expect(h.api.updatePseud).toHaveBeenCalledWith('p2', { name: 'Renamed', description: 'new words' }, 'tok'))
    expect(await screen.findByRole('status')).toHaveTextContent('Saved Renamed.')
  })

  it('makes another pseud the default', async () => {
    h.api.updatePseud.mockResolvedValue({ ...mine[1], is_default: true })
    const user = userEvent.setup()
    render(<MyPseuds />)
    await user.click(await screen.findByRole('button', { name: 'Make Side Pen my default' }))
    await waitFor(() => expect(h.api.updatePseud).toHaveBeenCalledWith('p2', { is_default: true }, 'tok'))
  })

  it('explains that works move to the default pseud before deleting, and deletes on confirm', async () => {
    h.api.deletePseud.mockResolvedValue({ message: 'ok', moved_to: 'p1' })
    const user = userEvent.setup()
    render(<MyPseuds />)
    await user.click(await screen.findByRole('button', { name: 'Delete Side Pen' }))
    const dialog = screen.getByRole('alertdialog')
    expect(dialog).toHaveTextContent(/move to your default pseud \(Main Pen\)/)
    expect(h.api.deletePseud).not.toHaveBeenCalled()
    await user.click(within(dialog).getByRole('button', { name: /Yes, delete Side Pen/ }))
    await waitFor(() => expect(h.api.deletePseud).toHaveBeenCalledWith('p2', 'tok'))
  })

  it('does not delete when the person keeps it', async () => {
    const user = userEvent.setup()
    render(<MyPseuds />)
    await user.click(await screen.findByRole('button', { name: 'Delete Side Pen' }))
    await user.click(screen.getByRole('button', { name: 'Keep it' }))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(h.api.deletePseud).not.toHaveBeenCalled()
  })

  it('every field has a label', async () => {
    render(<MyPseuds />)
    await screen.findByText('Side Pen')
    expect(screen.getByLabelText('Name')).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: 'My pseuds' })).toBeInTheDocument()
  })
})

describe('PseudPicker', () => {
  it('shows nothing with only one pseud, so nothing is learnt from the form', async () => {
    h.api.listMyPseuds.mockResolvedValue([{ id: 'p1', name: 'Only', is_default: true }])
    const { container } = render(<PseudPicker token="t" value="" onChange={() => {}} />)
    await waitFor(() => expect(h.api.listMyPseuds).toHaveBeenCalled())
    expect(container).toBeEmptyDOMElement()
  })

  it('lets a person choose among their own pseuds; the default sends nothing', async () => {
    h.api.listMyPseuds.mockResolvedValue([
      { id: 'p1', name: 'Main', is_default: true },
      { id: 'p2', name: 'Side', is_default: false },
    ])
    const onChange = vi.fn()
    const user = userEvent.setup()
    render(<PseudPicker token="t" value="" onChange={onChange} label="Post as" />)
    const select = await screen.findByLabelText('Post as')
    expect(within(select).getByRole('option', { name: 'Main (default)' })).toHaveValue('')
    await user.selectOptions(select, 'Side')
    expect(onChange).toHaveBeenCalledWith('p2')
  })

  it('is absent when signed out', () => {
    const { container } = render(<PseudPicker token={null} value="" onChange={() => {}} />)
    expect(container).toBeEmptyDOMElement()
    expect(h.api.listMyPseuds).not.toHaveBeenCalled()
  })
})
