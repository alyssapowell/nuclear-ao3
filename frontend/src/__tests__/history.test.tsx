// The history page and the Mark for Later button, against a mocked API.
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

const h = vi.hoisted(() => ({
  auth: { user: { id: 'u1', email: 'a@example.org' }, token: 'tok' as string | null },
  search: '',
  api: {} as Record<string, ReturnType<typeof vi.fn>>,
}))

vi.mock('react-router-dom', async () =>
  (await import('@/test/router-mock')).routerMock({
    useSearchParams: () => [new URLSearchParams(h.search), vi.fn()],
  }))
vi.mock('@/utils/auth', () => ({ useAuth: () => h.auth }))
vi.mock('@/lib/api', () => {
  for (const name of [
    'getMyHistory', 'getHistoryEntry', 'setMarkedForLater', 'deleteHistoryEntry', 'clearMyHistory', 'setHistoryEnabled',
  ]) h.api[name] = vi.fn()
  return h.api
})

import History from '@/pages/History'
import { MarkForLaterButton } from '@/components/MarkForLaterButton'

const pagination = { page: 1, limit: 20, total: 2, total_pages: 1 }
const reading = {
  work_id: 'w1', available: true, last_visited: '2026-11-01T12:00:00Z', visit_count: 3, marked_for_later: true,
  work: {
    id: 'w1', title: 'A Tale', summary: '<p>Short</p>', rating: 'general', fandoms: ['Some Fandom'],
    word_count: 1200, chapter_count: 2, authors: [{ id: 'p1', name: 'Quill Pen' }],
  },
}
const hiddenCreator = {
  work_id: 'w2', available: true, last_visited: '2026-10-01T12:00:00Z', visit_count: 1, marked_for_later: false,
  work: { id: 'w2', title: 'Anon Tale', rating: 'general', word_count: 10, chapter_count: 1, authors: [] },
}
const gone = { work_id: 'w3', available: false, last_visited: '2026-09-01T12:00:00Z', visit_count: 2, marked_for_later: false }

beforeEach(() => {
  for (const f of Object.values(h.api)) f.mockReset()
  h.search = ''
  h.auth = { user: { id: 'u1', email: 'a@example.org' }, token: 'tok' }
})

describe('History page', () => {
  it('lists works with visit info, the Marked for Later flag, and a placeholder for an unavailable work', async () => {
    h.api.getMyHistory.mockResolvedValue({ history_enabled: true, readings: [reading, hiddenCreator, gone], pagination })
    render(<History />)

    expect(await screen.findByRole('link', { name: 'A Tale' })).toBeInTheDocument()
    expect(screen.getByText(/Visited 3 times/)).toBeInTheDocument()
    expect(screen.getByText(/\(Marked for Later\)/)).toBeInTheDocument()
    expect(screen.getByText(/Visited 1 time(?!s)/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Quill Pen' })).toBeInTheDocument()
    // A work that hides its creator shows no "by".
    const anon = screen.getByText('Anon Tale').closest('article')!
    expect(anon.textContent).not.toMatch(/\bby\b/)
    expect(screen.getByText('This work is no longer available')).toBeInTheDocument()
    expect(h.api.getMyHistory).toHaveBeenCalledWith({ show: undefined, page: 1 }, 'tok')
  })

  it('asks for the to-read list with ?show=to-read', async () => {
    h.search = 'show=to-read'
    h.api.getMyHistory.mockResolvedValue({ history_enabled: true, readings: [reading], pagination })
    render(<History />)
    expect(await screen.findByRole('heading', { name: 'Marked for Later', level: 1 })).toBeInTheDocument()
    expect(h.api.getMyHistory).toHaveBeenCalledWith({ show: 'to-read', page: 1 }, 'tok')
  })

  it('marks as read, deletes one entry and clears all, reloading each time', async () => {
    const user = userEvent.setup()
    h.api.getMyHistory.mockResolvedValue({ history_enabled: true, readings: [reading], pagination })
    h.api.setMarkedForLater.mockResolvedValue({ marked_for_later: false })
    h.api.deleteHistoryEntry.mockResolvedValue({})
    h.api.clearMyHistory.mockResolvedValue({})
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<History />)

    await user.click(await screen.findByRole('button', { name: 'Mark as Read' }))
    expect(h.api.setMarkedForLater).toHaveBeenCalledWith('w1', false, 'tok')
    await user.click(await screen.findByRole('button', { name: 'Delete from History' }))
    expect(h.api.deleteHistoryEntry).toHaveBeenCalledWith('w1', 'tok')
    await user.click(screen.getByRole('button', { name: 'Clear History' }))
    expect(h.api.clearMyHistory).toHaveBeenCalledWith('tok')
    await waitFor(() => expect(h.api.getMyHistory.mock.calls.length).toBeGreaterThanOrEqual(4))
  })

  it('turns history off from the page and hides Mark for Later while off', async () => {
    const user = userEvent.setup()
    h.api.getMyHistory.mockResolvedValue({ history_enabled: true, readings: [reading], pagination })
    h.api.setHistoryEnabled.mockResolvedValue({ history_enabled: false })
    render(<History />)
    await user.click(await screen.findByRole('checkbox', { name: /Keep my reading history/ }))
    expect(h.api.setHistoryEnabled).toHaveBeenCalledWith(false, 'tok')

    h.api.getMyHistory.mockResolvedValue({ history_enabled: false, readings: [reading], pagination })
    render(<History />)
    await waitFor(() => expect(screen.getAllByText(/Nothing you read is being recorded/).length).toBeGreaterThan(0))
  })
})

describe('Mark for Later button', () => {
  it('offers Mark for Later, then Mark as Read', async () => {
    const user = userEvent.setup()
    h.api.getHistoryEntry.mockResolvedValue({ history_enabled: true, marked_for_later: false })
    h.api.setMarkedForLater.mockResolvedValue({ marked_for_later: true })
    render(<MarkForLaterButton workId="w1" authToken="tok" />)

    await user.click(await screen.findByRole('button', { name: 'Mark for Later' }))
    expect(h.api.setMarkedForLater).toHaveBeenCalledWith('w1', true, 'tok')
    expect(await screen.findByRole('button', { name: 'Mark as Read' })).toBeInTheDocument()
  })

  it('shows Mark as Read for a work already marked', async () => {
    h.api.getHistoryEntry.mockResolvedValue({ history_enabled: true, marked_for_later: true })
    render(<MarkForLaterButton workId="w1" authToken="tok" />)
    expect(await screen.findByRole('button', { name: 'Mark as Read' })).toBeInTheDocument()
  })

  it('is not shown with history off, or to a guest', async () => {
    h.api.getHistoryEntry.mockResolvedValue({ history_enabled: false, marked_for_later: false })
    const { container, rerender } = render(<MarkForLaterButton workId="w1" authToken="tok" />)
    await waitFor(() => expect(h.api.getHistoryEntry).toHaveBeenCalled())
    expect(container).toBeEmptyDOMElement()

    h.api.getHistoryEntry.mockClear()
    rerender(<MarkForLaterButton workId="w1" />)
    expect(container).toBeEmptyDOMElement()
    expect(h.api.getHistoryEntry).not.toHaveBeenCalled()
  })
})
