import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { SafeHtml, safeHtml } from '../safeHtml';

describe('safeHtml removes what the archive does not allow', () => {
  const stripped: [string, string, string[]][] = [
    ['onerror', '<img src=x onerror=alert(1)>', ['onerror']],
    ['onerror on a good image', '<img src="https://example.org/a.png" onerror="alert(1)">', ['onerror']],
    ['onclick', '<p onclick="alert(1)">hi</p>', ['onclick']],
    ['javascript: href', '<a href="javascript:alert(1)">x</a>', ['javascript']],
    ['javascript: href, mixed case', '<a href="JaVaScRiPt:alert(1)">x</a>', ['javascript']],
    ['javascript: href with a tab entity', '<a href="jav&#x09;ascript:alert(1)">x</a>', ['javascript', 'ascript:']],
    ['javascript: href with a newline entity', '<a href="jav&#x0A;ascript:alert(1)">x</a>', ['ascript:']],
    ['javascript: href, all entities', '<a href="&#106;&#97;&#118;&#97;&#115;&#99;&#114;&#105;&#112;&#116;&#58;alert(1)">x</a>', ['alert(1)"', 'script:']],
    ['data: src', '<img src="data:image/svg+xml;base64,PHN2Zz4=">', ['data:']],
    ['data: href', '<a href="data:text/html,<b>x</b>">x</a>', ['data:']],
    ['script', 'a<script>alert(1)</script>b', ['script', 'alert']],
    ['svg onload', '<svg onload=alert(1)><circle/></svg>', ['svg', 'onload']],
    ['style element', '<style>body{display:none}</style>ok', ['style', 'display']],
    ['style attribute', '<p style="position:fixed">x</p>', ['style']],
    ['iframe', '<iframe src="https://example.org"></iframe>', ['iframe']],
    ['object', '<object data="x.swf"></object>', ['object']],
    ['embed', '<embed src="x.swf">', ['embed']],
    ['form', '<form action="https://evil.example"><input name=a></form>', ['form', 'input']],
    ['target', '<a href="https://example.org" target="_blank">x</a>', ['target']],
    ['class and id', '<p class="a" id="b">x</p>', ['class', 'id=']],
    ['relative image', '<img src="/a.png">', ['src']],
  ];
  it.each(stripped)('%s', (_name, dirty, gone) => {
    const out = safeHtml(dirty);
    for (const bad of gone) expect(out.toLowerCase()).not.toContain(bad.toLowerCase());
  });

  it('puts rel on every link with an address', () => {
    expect(safeHtml('<a href="https://example.org" rel="opener" target="_blank">x</a>')).toBe(
      '<a href="https://example.org" rel="nofollow noopener">x</a>',
    );
  });

  it('drops the contents of script and style, not just the tags', () => {
    expect(safeHtml('a<script>alert(1)</script><style>p{}</style>b')).toBe('ab');
  });
});

describe('safeHtml keeps what the archive allows', () => {
  const same = [
    '<p>Hello</p>',
    '<em>a</em> <strong>b</strong>',
    '<blockquote><p>quoted</p></blockquote>',
    '<img src="https://example.org/a.png" alt="a picture" width="10">',
    '<table border="1"><thead><tr><th>h</th></tr></thead><tbody><tr><td colspan="2">c</td></tr></tbody></table>',
    '<ruby>漢<rp>(</rp><rt>kan</rt><rp>)</rp></ruby>',
    '<p align="center" dir="rtl" title="t">x</p>',
    '<p>1 &lt; 2 &amp; 3</p>',
  ];
  it.each(same)('%s', (html) => {
    expect(safeHtml(html)).toBe(html);
  });

  it('keeps links, adding rel', () => {
    expect(safeHtml('<a href="https://example.org/page">x</a>')).toBe(
      '<a href="https://example.org/page" rel="nofollow noopener">x</a>',
    );
    expect(safeHtml('<a href="mailto:a@example.org">x</a>')).toContain('href="mailto:a@example.org"');
    expect(safeHtml('<a href="#note-1">x</a>')).toContain('href="#note-1"');
  });
});

describe('safeHtml with nothing to show', () => {
  it('returns an empty string', () => {
    expect(safeHtml('')).toBe('');
    expect(safeHtml(null)).toBe('');
    expect(safeHtml(undefined)).toBe('');
  });
});

describe('<SafeHtml>', () => {
  it('renders cleaned HTML inside a div with the given props', () => {
    const out = renderToStaticMarkup(
      <SafeHtml id="s" className="x" html={'<p>hi</p><img src=x onerror=alert(1)>'} />,
    );
    expect(out).toBe('<div id="s" class="x"><p>hi</p></div>');
  });
});
