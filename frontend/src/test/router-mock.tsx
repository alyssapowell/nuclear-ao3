// A stand-in for react-router-dom, so components can be tested on their own
// without wrapping each one in a router. Every test gets it (see setup.ts).
// To check where a component navigates, override useNavigate in that test:
//
//   const mockNavigate = vi.fn()
//   vi.mock('react-router-dom', async () =>
//     (await import('@/test/router-mock')).routerMock({ useNavigate: () => mockNavigate }))
import type { ReactNode } from 'react'

type LinkProps = { to: string; children?: ReactNode; [prop: string]: unknown }

export function routerMock(overrides: Record<string, unknown> = {}) {
  return {
    Link: ({ to, children, ...rest }: LinkProps) => <a href={to} {...rest}>{children}</a>,
    NavLink: ({ to, children, ...rest }: LinkProps) => <a href={to} {...rest}>{children}</a>,
    useNavigate: () => vi.fn(),
    useParams: () => ({ id: 'test-id' }),
    useLocation: () => ({ pathname: '/test-path', search: '', hash: '', state: null, key: 'test' }),
    useSearchParams: () => [new URLSearchParams(), vi.fn()],
    ...overrides,
  }
}
