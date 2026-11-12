#!/usr/bin/env node
// Lists every server address the frontend calls, so a Go test can check that
// each one is a route the app really mounts.
//
//   node scripts/frontend-api-paths.mjs          rewrite the checked-in list
//   node scripts/frontend-api-paths.mjs --check  fail if the list is out of date
//
// Output: backend/cmd/nuclear/frontend_api_paths.txt, one "path<TAB>file" per
// line. Dynamic parts (${...}) become :p, and query strings are dropped.
// Test files are skipped. Four call shapes are understood:
//   1. a string holding /api/...            (fetch, call, links)
//   2. `${API_BASE_URL}/x` where the file sets API_BASE_URL = '/api/v1'
//   3. pseudRequest('GET', '/x')            -> /api/v1/x
//   4. accountCall('x', 'GET')              -> /api/v1/auth/x
// plus the fixed '/health' and '/ws' addresses. If you add a new way of
// calling the server, teach this script about it, or the guard will not see it.
import { readdirSync, readFileSync, writeFileSync, statSync } from 'node:fs';
import { join, relative, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const src = join(root, 'frontend', 'src');
const out = join(root, 'backend', 'cmd', 'nuclear', 'frontend_api_paths.txt');

function* walk(dir) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) {
      if (name === '__tests__' || name === 'node_modules') continue;
      yield* walk(p);
    } else if (/\.(ts|tsx)$/.test(name) && !/\.(test|spec)\.(ts|tsx)$/.test(name)) {
      yield p;
    }
  }
}

function normalise(p) {
  p = p.replace(/\$\{[^}]*(\?|query)[^}]*\}/gi, ''); // ${qs ? '?' + qs : ''}, ${pageQuery(x)}
  p = p.replace(/\$\{[^}]*\}/g, ':p');
  p = p.split('?')[0];
  p = p.replace(/\/+$/, '');
  return p;
}

const found = new Map(); // "path\tfile"
function add(path, file) {
  path = normalise(path);
  if (!path.startsWith('/')) return;
  // The helpers' own templates (/api/v1, /api/v1${path}, /api/v1/auth/${path});
  // their callers are picked up by the pseudRequest/accountCall patterns.
  if (['/api/v1', '/api/v1:p', '/api/v1/auth/:p'].includes(path)) return;
  found.set(`${path}\t${file}`, true);
}

for (const f of walk(src)) {
  const file = relative(root, f);
  const text = readFileSync(f, 'utf8');
  const base = /const API_BASE_URL = '(\/api\/v1)'/.exec(text);

  // Strings and templates only; the pattern stops at the closing quote.
  for (const m of text.matchAll(/(['"`])((?:\\.|(?!\1)[^\\\n])*)\1/g)) {
    let s = m[2];
    if (s.includes('\n')) continue;
    s = s.replace(/^\$\{API_GATEWAY_URL\}|^\$\{API_BASE_URL\}(?=\/api)/, '');
    if (base && s.startsWith('${API_BASE_URL}/')) {
      add(base[1] + s.slice('${API_BASE_URL}'.length), file);
    } else if (/^\/api\/[A-Za-z0-9_\-.${}:/?=&+, ]*$/.test(s) && !/localhost|curl /.test(s)) {
      add(s, file);
    } else if (/^\$\{[A-Za-z_]+\}\/api\/v1\//.test(s)) {
      add(s.replace(/^\$\{[A-Za-z_]+\}/, ''), file);
    } else if (/^\/(health|ws)$/.test(s) && /fetch|WebSocket|GATEWAY_HEALTH/.test(text)) {
      add(s, file);
    }
  }
  if (text.includes('}/ws`')) add('/ws', file); // new WebSocket(`${scheme}://${host}/ws`)
  for (const m of text.matchAll(/pseudRequest(?:<[^(]*>)?\(\s*'[A-Z]+',\s*([`'])((?:\\.|(?!\1)[^\\])*)\1/g)) {
    add('/api/v1' + m[2], file);
  }
  for (const m of text.matchAll(/accountCall(?:<[^(]*>)?\(\s*([`'])((?:\\.|(?!\1)[^\\])*)\1/g)) {
    add('/api/v1/auth/' + m[2], file);
  }
}

const lines = [...found.keys()].sort();
const body = lines.join('\n') + '\n';
if (process.argv.includes('--check')) {
  let cur = '';
  try { cur = readFileSync(out, 'utf8'); } catch {}
  if (cur !== body) {
    console.error('backend/cmd/nuclear/frontend_api_paths.txt is out of date; run: node scripts/frontend-api-paths.mjs');
    process.exit(1);
  }
} else {
  writeFileSync(out, body);
  console.log(`${lines.length} calls written to ${relative(root, out)}`);
}
