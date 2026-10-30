# The website

The site people use in their browser. It's a single-page app (React, React
Router, Tailwind) that is built into plain files and served by Caddy. There's no
Node.js server in production: Node is only used to build the files.

The site gets everything from the Go app through `/api`. It never talks to
the database.

## Working on it

You need Node 20 or newer, and the app running (`docker-compose up -d app`).

```bash
npm install
npm run dev        # http://localhost:3001, reloads as you edit
```

`npm run dev` forwards `/api` to the app at http://localhost:8088. To use a
different address: `API_URL=http://localhost:8080 npm run dev`.

## Where things are

| What | Where |
| --- | --- |
| Every page and its address | `src/App.tsx` |
| The pages | `src/pages/` |
| Pieces used on several pages | `src/components/` |
| Calls to the API | `src/lib/api.ts` |
| Who's logged in | `src/utils/auth.ts` (`useAuth`) |

To add a page, create it in `src/pages/` and add a `<Route>` for it in
`src/App.tsx`. If it's only for logged-in users, put it inside the `LoggedIn`
group there. A test checks that every page has a route.

## Checking your work

```bash
npm run build      # type-checks, then builds into dist/
npm test           # unit tests (Vitest)
npm run test:e2e   # browser tests (Playwright); needs the app running
```

## How it gets to production

`frontend/Dockerfile` builds the files and copies them into a Caddy image. The
`caddy` service in `docker-compose.yml` uses that image together with the
`Caddyfile` at the top of the repo. You don't deploy the website separately:
rebuilding `caddy` ships it.
