# GDG Babcock Go auth service

This Fiber service is the authentication portion of the legacy `../auth` Express application. Firebase Authentication handles sign-in. The client sends a Firebase ID token to `POST /auth/login`; this service verifies it with Firebase Admin, preserves the user's existing PostgreSQL UUID, and issues platform JWTs. GORM accesses only `users`, `refresh_tokens`, `user_sessions`, and `auth_audit_log`. No schema changes run at startup.

## Run locally

1. Configure PostgreSQL with the four auth tables in [`database/auth.sql`](database/auth.sql), and set `DATABASE_URL` to its connection string.
2. Copy `.env.example` to `.env` if you do not already have a local `.env`. Set the real `FIREBASE_PROJECT_ID`, a random `JWT_SECRET` of at least 32 characters, and Firebase Admin credentials. Use the same `JWT_SECRET` as the legacy Express service while both must accept each other's platform tokens.
3. Export those variables and run `go run ./cmd/auth`, or build with `go build ./cmd/auth`. The service listens on `HTTP_PORT` (default `8080`).

For Docker, `docker compose up --build api` starts the disposable auth database and API on `localhost:8080`. Compose reads `.env`. Provide Firebase Admin credentials in one of these ways:

- Set `FIREBASE_SERVICE_ACCOUNT` to the service-account JSON on one line in `.env`.
- Put the JSON file at `.secrets/firebase-service-account.json`. Compose mounts that directory read-only and sets `FIREBASE_SERVICE_ACCOUNT_PATH` inside the container. The file must be readable by the container's nonroot user.

Set `FIREBASE_PROJECT_ID` to the `project_id` from that service account. A host-side path in `FIREBASE_SERVICE_ACCOUNT_PATH` is not automatically visible inside Docker. The `.env` and `.secrets` paths are ignored by Git and excluded from the image build context. For a database-only start, run `docker compose up -d db`.

## Endpoints

`POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me`, `PUT /auth/profile`, `GET /auth/verify`, `GET /admin/users`, `PUT /admin/users/:id`, and `GET /health`. See [the API contract](docs/go-auth-contract.md) for the legacy response shape and [the deployment guide](docs/go-auth-operations.md) for cutover constraints.

The Go API intentionally does not serve `/events`, `/radar`, `/team`, or `/analytics`. Existing website profile enrichment from those tables needs another owner before `/auth/me` can be switched without changing the website UI.

## Code layout

`internal/domain` owns the auth models and repository interfaces. `internal/auth` implements login, token, profile, and admin rules against those interfaces. `internal/infrastructure/firebase` verifies Firebase ID tokens. `internal/infrastructure/postgres` opens the database and keeps user, refresh-token, and audit repository methods in separate files. `internal/httpapi/api.go` registers Fiber routes, `handler.go` handles requests, and `internal/httpapi/dto/request.go` and `response.go` define the JSON contract.
