# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

"Can I Do It?" (我能做这个吗？) — analyzes entrepreneurs' uploaded business plans with an AI provider and returns feasibility feedback. Go API + Vue 3 SPA + PostgreSQL (or MySQL) + Redis. Built from the gloscai Go/Vue3/Docker starter; the Go module is still `github.com/gloscai/template-go-vue3-docker/server`.

In-repo docs (Chinese): `dev.md` is the full developer guide, `docs/architecture.md` covers design boundaries, `docs/api.md` mirrors the actually-registered HTTP routes — update it when adding or changing endpoints.

## Commands

```bash
make dev        # full dev env in Docker: Vite HMR :5173, Go API :8080, Postgres, Redis
make test       # cd server && go test -race ./...  &&  cd web && npm run build (vue-tsc + vite)
make check      # go vet ./...  &&  npm run typecheck
make fmt        # gofmt -w server/

# Host-based dev (DB/Redis still in Docker)
make init       # cp .env.example .env, go mod download, npm ci
make db         # start Postgres + Redis only
make api        # go run . (needs env vars; Go does NOT auto-read .env — inject via shell/IDE,
                # and point DATABASE_URL at localhost instead of the compose service names)
make web        # vite dev server

# Single Go test
cd server && go test -race ./tasks/ -run TestName

# Add a shadcn-vue component (managed source under web/src/components/ui)
cd web && npx shadcn-vue@latest add <name>
```

`API_PORT=18080 make dev` if 8080 is taken. CI (`.github/workflows/ci.yml`) runs exactly `make test`/`make check` equivalents.

## Backend architecture (server/)

- **stdlib `net/http` only**, no framework. Each domain package registers its own routes with Go 1.22 method patterns (`mux.HandleFunc("GET /api/v1/plans/{id}", ...)`) via a `Register(mux)` method. All wiring is in `server.go:run()`: config → DB open/migrate → Redis → route registration → middleware chain (recovery → CORS → request log → request ID → session middleware).
- **Flat domain packages** (`tasks`, `users`, `plans`, `settings`, `analysis`, `auth`, `cache`, `health`) — no controller/service/repository layering. Each package owns its model, SQL, HTTP handlers, and tests. `tasks/` is the reference implementation; copy its shape for new domains.
- **Auth flow**: OAuth2/OIDC SSO against Gloscai SSO (`users` package). On callback the server sets an HttpOnly session cookie (`can_i_session`); `users.Service.Middleware` resolves it into a `User` in request context for every `/api/v1/**` path except `/api/v1/auth/**`. Handlers read it via `users.UserFromContext(ctx)`; owner-only endpoints check `u.Role == "owner"` themselves. The first user row ever created becomes `owner`.
- **Analysis pipeline is DB-polled, not a queue**: `plans.analyze` inserts an `analysis_jobs` row with `status='queued'`; `analysis.Worker` (a goroutine started in `server.go`) polls every 2s, claims one job (`FOR UPDATE SKIP LOCKED` on Postgres), calls the OpenAI-compatible `/chat/completions` endpoint, and writes the result back.
- **AI provider config lives in the `app_settings` table** (managed by the owner via `/api/v1/admin/settings/ai`), not env vars. The API key is AES-GCM encrypted with `APP_ENCRYPTION_KEY`, which must be exactly 32 bytes or saving a key fails with `503 encryption_not_configured`.
- **Dual database drivers**: handlers write SQL with `?` placeholders and pass it through a per-package `q()` helper that rewrites them to `$1, $2, …` when the driver is `postgres`. Both `database/migrations/postgres/` and `database/migrations/mysql/` must get every new numbered migration; separate statements within one file with `-- statement-breakpoint`; never edit an already-applied migration. Migrations run at startup when `AUTO_MIGRATE=true`, in filename order.

## Frontend architecture (web/)

- Vue 3 + TypeScript + Vite, Pinia, Vue Router, shadcn-vue (`nova` style, reka-ui) + Tailwind CSS v4.
- **Relative URLs everywhere**: the browser never sees a backend host. Vite dev proxies `/api` and `/health` to `VITE_API_PROXY_TARGET` (default `http://localhost:8080`); in production the Go binary embeds the built SPA (`server/webui`) and serves it itself, with an `index.html` fallback for Vue Router history mode — no separate web server.
- Session is the HttpOnly cookie, so API calls use `fetch(..., { credentials: 'include' })` — see `src/api/client.ts`.
- Layering: `src/api/` typed HTTP clients → `src/features/<domain>/` Pinia store + domain components → `src/views/` page composition only (no direct fetching in views). `src/components/ui/` is shadcn-vue CLI-managed source — use the CLI to add/upgrade, don't hand-write new primitives there.
- Router guards (`src/router/index.ts`): `meta.requiresAuth` redirects unauthenticated users to `/api/v1/auth/login` (full-page SSO redirect); `meta.requiresOwner` checks `auth.user?.role`.

## Conventions and gotchas

- API responses: success `{"data": ...}`, error `{"error": {"code": "snake_case", "message": "..."}}`.
- All runtime config comes from env vars (`server/config/config.go` is the single source). `production` rejects empty/short `JWT_SECRET` (< 32 chars); `development` gets a built-in fallback secret.
- `.env` is consumed by docker compose only; the Go process itself does not load it.
