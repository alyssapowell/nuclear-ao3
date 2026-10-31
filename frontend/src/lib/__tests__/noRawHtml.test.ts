import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// User-written HTML must go through safeHtml() (src/lib/safeHtml.ts), which is
// the only file allowed to hand a string to the DOM as HTML.
const src = resolve(__dirname, '../..');
const allowed = new Set(['lib/safeHtml.ts']);
const forbidden = [/dangerouslySetInnerHTML/, /\.innerHTML\s*=/, /\.outerHTML\s*=/, /insertAdjacentHTML/, /document\.write\(/];

function files(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) return name === '__tests__' ? [] : files(path);
    return /\.(ts|tsx|js|jsx)$/.test(name) && !/\.test\./.test(name) ? [path] : [];
  });
}

describe('raw HTML in the DOM', () => {
  it('only happens in safeHtml.ts', () => {
    const offenders: string[] = [];
    for (const file of files(src)) {
      const name = relative(src, file);
      if (allowed.has(name)) continue;
      const text = readFileSync(file, 'utf8');
      for (const pattern of forbidden) {
        if (pattern.test(text)) offenders.push(`${name}: ${pattern}`);
      }
    }
    expect(offenders).toEqual([]);
  });
});
