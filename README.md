# GDG Babcock Go auth service

This Fiber service is the authentication portion of the legacy `../auth` Express application. Firebase Authentication handles sign-in. The client sends a Firebase ID token to `POST /auth/login`; this service verifies it with Firebase Admin, preserves the user's existing PostgreSQL UUID, and issues platform JWTs. GORM accesses only `users`, `refresh_tokens`, `user_sessions`, and `auth_audit_log`. No schema changes run at startup.

## Run locally

1. Configure PostgreSQL with the four auth tables in [`database/auth.sql`](database/auth.sql), and set `DATABASE_URL` to its connection string.
2. Copy `.env.example` to `.env` if you do not already have a local `.env`. Set the real `FIREBASE_PROJECT_ID`, a random `JWT_SECRET` of at least 32 characters, and Firebase Admin credentials. Use the same `JWT_SECRET` as the legacy Express service while both must accept each other's platform tokens.
3. Export those variables and run `go run ./cmd/auth`, or build with `go build ./cmd/auth`. The service listens on `PORT`, then `HTTP_PORT`, then default `8080`.

For Docker, `docker compose up --build api` starts the disposable auth database and API on `localhost:8080`. Compose reads `.env`. Provide Firebase Admin credentials in one of these ways:

- Set `FIREBASE_SERVICE_ACCOUNT` to the service-account JSON on one line in `.env`.
- Put the JSON file at `.secrets/firebase-service-account.json`. Compose mounts that directory read-only and sets `FIREBASE_SERVICE_ACCOUNT_PATH` inside the container. The file must be readable by the container's nonroot user.

Set `FIREBASE_PROJECT_ID` to the `project_id` from that service account. A host-side path in `FIREBASE_SERVICE_ACCOUNT_PATH` is not automatically visible inside Docker. The `.env` and `.secrets` paths are ignored by Git and excluded from the image build context. For a database-only start, run `docker compose up -d db`.

## Deploy to Vercel

Vercel builds `Dockerfile.vercel` and supplies `PORT` at runtime; the service gives that variable precedence over `HTTP_PORT`. Set `DATABASE_URL`, `FIREBASE_PROJECT_ID`, `FIREBASE_SERVICE_ACCOUNT` (the service-account JSON), `JWT_SECRET`, and `CORS_ORIGINS` in the Vercel project environment for the deployment environment you use. Use a reachable hosted PostgreSQL database with the auth schema already applied. The Compose database hostname `db` and local `.secrets` mount are only available when running Compose locally.

Set `SWAGGER_HOST` to the deployment's public host without a scheme (dev: `auth-v2.pxxlspace.cv`) and `SWAGGER_BASE_PATH` to the path the API is mounted under (`/`). They default to `localhost:8080` and `/`, and are applied to the served OpenAPI document at startup, so `/swagger` issues its "Try it out" requests against the right origin.

After redeploying, request `/health`, which returns `{"status":"healthy"}`. If startup fails, check Vercel runtime logs for `configuration error`, `database unavailable`, or `firebase initialization failed`.

## Endpoints

Email verification is not required to sign up or sign in. `POST /auth/login` accepts a valid Firebase ID token and creates a user on first login, storing the token's actual email verification status. An email already associated with another Firebase UID still requires identity linking.

`POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me`, `PUT /auth/profile`, `GET /auth/verify`, `GET /health`.

Admin member management: `GET /admin/users` (add `?archived=true` for the archive), `POST /admin/users`, `GET /admin/users/:id`, `PUT /admin/users/:id`, `DELETE /admin/users/:id`, and `POST /admin/users/:id/restore`.

## Dashboard

`GET /dashboard` serves a built-in web dashboard from `internal/httpapi/web`, embedded in the binary. Members sign in with Google, then edit their own profile; admins also get a searchable member roster with role, team and account-status editing. It calls the same API on the same origin, so it needs no `CORS_ORIGINS` entry.

Sign-in needs the Firebase Web SDK values, which are public client identifiers rather than credentials:

- `FIREBASE_WEB_API_KEY` — Web API key from Firebase console, Project settings, General.
- `FIREBASE_AUTH_DOMAIN` — usually `<project-id>.firebaseapp.com`; defaults from `FIREBASE_PROJECT_ID`.

The dashboard reads them from `GET /dashboard/config.json` at load. If the API key is missing it shows setup instructions instead of a sign-in button. Add the deployment's host to Firebase, Authentication, Settings, Authorized domains, or the Google popup is rejected.

### Member management

Admins get full management of the register from the dashboard:

- **View** — `GET /admin/users/:id` returns the whole record. The roster list stays a summary.
- **Add** — `POST /admin/users` pre-registers someone by email with roles and teams already assigned. Because `firebase_uid` is `UNIQUE NOT NULL`, the waiting row holds a `pending:<uuid>` sentinel; when that person first signs in with Google on the same address, `/auth/login` claims the row instead of returning 409, so the UUID, roles and teams they were given survive. No email is sent from the service — tell the person to sign in.
- **Edit** — `PUT /admin/users/:id` takes roles, teams, `is_active` and the profile columns (department, matric number, tracks, WhatsApp, and so on). `email`, `firebase_uid`, `deleted_at` and `tos_agreed` stay rejected: the first three are system-managed, and terms of service is a consent record an admin must not tick for someone else.
- **Archive** — `DELETE /admin/users/:id` is a soft delete. It sets `deleted_at`, deactivates the account and revokes every refresh token, so open sessions on Radar and the GDG site stop working. Archived users drop out of the default roster and appear under the "Archived" filter.
- **Restore** — `POST /admin/users/:id/restore` clears `deleted_at` and reactivates.

Every one of these writes an entry to `auth_audit_log` with the acting admin's ID.

The dashboard prevents an admin from removing their own admin role or deactivating themselves; the API does not enforce that yet, so that guard is only in the UI. Archiving yourself *is* blocked server-side.

### Granting the first admin

New accounts are created with `roles = ['user']`, and `PUT /admin/users/{id}` requires the `admin` role, so the first admin has to be promoted directly in the database. Sign in to the dashboard once so the user row exists, then:

```sql
UPDATE users SET roles = ARRAY['user','admin'] WHERE email = 'you@babcock.edu.ng';
```

Roles are baked into the access token when it is issued, so sign out and back in before the Members section appears. Every admin after the first can be promoted from the dashboard.

Note that `database/auth.sql` uses `user` as the default role and documents the vocabulary as `user`, `admin`, `moderator`, `lead`. The PRD says `member`; the code is the authority here, and the dashboard follows the code.

The Go API intentionally does not serve `/events`, `/radar`, `/team`, or `/analytics`. Existing website profile enrichment from those tables needs another owner before `/auth/me` can be switched without changing the website UI.

## Code layout

`internal/domain` owns the auth models and repository interfaces. `internal/auth` implements login, token, profile, and admin rules against those interfaces. `internal/infrastructure/firebase` verifies Firebase ID tokens. `internal/database` opens the PostgreSQL connection using `DATABASE_URL`. `internal/infrastructure/postgres` keeps user, refresh-token, and audit repository methods in separate files. `internal/httpapi/api.go` registers Fiber routes, `handler.go` handles requests, and `internal/httpapi/dto/request.go` and `response.go` define the JSON contract.
