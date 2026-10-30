import { safeRedirect } from '../auth';

describe('safeRedirect', () => {
  test.each([
    ['/works/123?view=full#c1', '/works/123?view=full#c1'],
    ['/dashboard', '/dashboard'],
    [null, '/'],
    ['', '/'],
    ['https://evil.example/phish', '/'],
    ['//evil.example', '/'],
    ['/\\evil.example', '/'],
    ['javascript:alert(document.cookie)', '/'],
    ['JaVaScRiPt:alert(1)', '/'],
    ['data:text/html,<script>alert(1)</script>', '/'],
  ])('%s -> %s', (raw, expected) => {
    expect(safeRedirect(raw)).toBe(expected);
  });
});
