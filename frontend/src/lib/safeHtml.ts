// The one place user-written HTML is turned into something safe to put on the
// page. Every component that shows HTML goes through safeHtml() or <SafeHtml>;
// a test fails if dangerouslySetInnerHTML is used anywhere else.
//
// The rules are the same as the server's (backend/internal/html), which are
// AO3's: config/initializers/gem-plugin_config/sanitizer_config.rb. The server
// cleans text when it is saved. This is a second layer, for text that was
// stored before that, text that is built in the browser (the markdown
// preview), and any mistake in the first layer.

import DOMPurify from 'dompurify';
import { createElement, type HTMLAttributes, type ReactElement } from 'react';

const ALLOWED_TAGS = [
  'a', 'abbr', 'acronym', 'address', 'b', 'big', 'blockquote', 'br', 'caption', 'center', 'cite', 'code', 'col',
  'colgroup', 'details', 'figcaption', 'figure', 'dd', 'del', 'dfn', 'div', 'dl', 'dt', 'em', 'h1', 'h2', 'h3', 'h4',
  'h5', 'h6', 'hr', 'i', 'img', 'ins', 'kbd', 'li', 'ol', 'p', 'pre', 'q', 'rp', 'rt', 'ruby', 's', 'samp', 'small',
  'span', 'strike', 'strong', 'sub', 'summary', 'sup', 'table', 'tbody', 'td', 'tfoot', 'th', 'thead', 'tr', 'tt', 'u',
  'ul', 'var',
];

// Attributes allowed on every element, and on particular elements.
const GLOBAL_ATTRIBUTES = ['align', 'title', 'dir'];
const ELEMENT_ATTRIBUTES: Record<string, string[]> = {
  a: ['href', 'name'],
  blockquote: ['cite'],
  col: ['span', 'width'],
  colgroup: ['span', 'width'],
  details: ['open'],
  hr: ['align', 'width'],
  img: ['align', 'alt', 'border', 'height', 'src', 'width'],
  ol: ['start', 'type'],
  q: ['cite'],
  table: ['border', 'summary', 'width'],
  td: ['abbr', 'axis', 'colspan', 'height', 'rowspan', 'width'],
  th: ['abbr', 'axis', 'colspan', 'height', 'rowspan', 'scope', 'width'],
  ul: ['type'],
};

// http, https and mailto, or no protocol at all (a relative link or #anchor).
const ALLOWED_URI = /^(?:(?:https?|mailto):|[^a-z]|[a-z+.-]+(?:[^a-z+.\-:]|$))/i;
// Images need a full address.
const IMAGE_SRC = /^https?:\/\//i;

const purify = DOMPurify(window);

purify.addHook('afterSanitizeAttributes', (node) => {
  const tag = node.nodeName.toLowerCase();
  const allowed = new Set([...GLOBAL_ATTRIBUTES, ...(ELEMENT_ATTRIBUTES[tag] ?? [])]);
  // DOMPurify allows the union of all attributes on all elements; narrow that
  // to the attributes of this element.
  for (const attr of Array.from(node.attributes)) {
    if (!allowed.has(attr.name)) node.removeAttribute(attr.name);
  }
  // An image without a full http(s) address shows nothing, so drop it.
  if (tag === 'img' && !IMAGE_SRC.test(node.getAttribute('src') ?? '')) {
    node.parentNode?.removeChild(node);
    return;
  }
  if (tag === 'a' && node.hasAttribute('href')) {
    node.setAttribute('rel', 'nofollow noopener');
  }
});

const CONFIG = {
  ALLOWED_TAGS,
  ALLOWED_ATTR: Array.from(new Set([...GLOBAL_ATTRIBUTES, ...Object.values(ELEMENT_ATTRIBUTES).flat()])),
  ALLOWED_URI_REGEXP: ALLOWED_URI,
  ALLOW_DATA_ATTR: false,
  ALLOW_ARIA_ATTR: false,
  ALLOW_UNKNOWN_PROTOCOLS: false,
  RETURN_TRUSTED_TYPE: false,
} as const;

/** Returns `dirty` with everything the archive does not allow removed. */
export function safeHtml(dirty: string | null | undefined): string {
  if (!dirty) return '';
  if (!purify.isSupported) return '';
  return String(purify.sanitize(dirty, CONFIG));
}

interface SafeHtmlProps extends Omit<HTMLAttributes<HTMLDivElement>, 'children' | 'dangerouslySetInnerHTML'> {
  html: string | null | undefined;
}

/** A div that shows user-written HTML after cleaning it. */
export function SafeHtml({ html, ...rest }: SafeHtmlProps): ReactElement {
  return createElement('div', { ...rest, dangerouslySetInnerHTML: { __html: safeHtml(html) } });
}

export default SafeHtml;
