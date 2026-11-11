import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

const h = vi.hoisted(() => ({ navigate: vi.fn() }))
vi.mock('react-router-dom', async () =>
  (await import('@/test/router-mock')).routerMock({ useNavigate: () => h.navigate }))

import TrialBanner from '@/components/TrialBanner'
import Register from '@/pages/Register'
import { resetSite } from '@/lib/site'

const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>

function answer(site: unknown) {
  fetchMock.mockImplementation((url: string) =>
    Promise.resolve({
      ok: true,
      status: String(url).endsWith('/site') ? 200 : 201,
      json: () => Promise.resolve(String(url).endsWith('/site') ? site : { message: 'Your account is ready. Log in with your email and password.' }),
    } as Response))
}

beforeEach(() => {
  fetchMock.mockReset()
  h.navigate.mockReset()
  resetSite()
})

describe('TrialBanner', () => {
  it('shows a labelled notice when the server is in trial mode, asking once', async () => {
    answer({ trial_no_email: true })
    const { rerender } = render(<TrialBanner />)
    const banner = await screen.findByRole('region', { name: 'Trial mode' })
    expect(banner).toHaveTextContent("Trial: email is off. Accounts are confirmed at once, and password resets and notifications aren't sent.")
    expect(banner).not.toHaveClass('fixed')
    rerender(<TrialBanner />)
    expect(fetchMock.mock.calls.filter(([u]) => String(u) === '/api/v1/site')).toHaveLength(1)
  })

  it('shows nothing otherwise, or when the server cannot be asked', async () => {
    answer({ trial_no_email: false })
    const { container, unmount } = render(<TrialBanner />)
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(container).toBeEmptyDOMElement()
    unmount()

    resetSite()
    fetchMock.mockReset()
    fetchMock.mockRejectedValue(new Error('down'))
    const second = render(<TrialBanner />)
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(second.container).toBeEmptyDOMElement()
  })

  it('sends a new account to the login page with the message in trial mode', async () => {
    answer({ trial_no_email: true })
    render(<><TrialBanner /><Register /></>)
    await screen.findByRole('region', { name: 'Trial mode' })
    await userEvent.type(screen.getByLabelText('Pen name'), 'Alice Pen')
    await userEvent.type(screen.getByLabelText('Email'), 'a@example.com')
    await userEvent.type(screen.getByLabelText('Password'), 'secret1')
    await userEvent.type(screen.getByLabelText('Confirm Password'), 'secret1')
    fireEvent.submit(screen.getByRole('button', { name: 'Create Account' }).closest('form')!)
    await waitFor(() => expect(h.navigate).toHaveBeenCalledWith('/auth/login', {
      state: { message: 'Your account is ready. Log in with your email and password.' },
    }))
    expect(screen.queryByText('Check your email to activate your account')).not.toBeInTheDocument()
  })
})
