# GDG Babcock Go auth service

This Fiber service is the authentication portion of the legacy `../auth` Express application. Firebase Authentication handles sign-in. The client sends a Firebase ID token to `POST /auth/login`; this service verifies it with Firebase Admin, preserves the user's existing PostgreSQL UUID, and issues platform JWTs. GORM accesses only `users`, `refresh_tokens`, `user_sessions`, and `auth_audit_log`. No schema changes run at startup.

## Run locally

1. Configure PostgreSQL with the four auth tables in [`database/auth.sql`](database/auth.sql), and set `DATABASE_URL` to its connection string.
2. Copy `.env.go-auth.example` to `.env.go-auth`. Configure `FIREBASE_PROJECT_ID`, one Firebase Admin credential source (service-account JSON, service-account file, or Application Default Credentials), and a random `JWT_SECRET` of at least 32 characters. Use the same `JWT_SECRET` as the legacy Express service while both must accept each other's platform tokens.
3. Export those variables and run `go run ./cmd/auth`, or build with `go build ./cmd/auth`. The service listens on `HTTP_PORT` (default `8080`).

For a disposable local database, `docker compose -f docker-compose.go-auth.yml up -d db` starts PostgreSQL and initializes only the auth tables. The Compose API container requires working Firebase credentials in `.env.go-auth`; `FIREBASE_SERVICE_ACCOUNT_PATH` must refer to a path inside the container if that method is used. The local API host port is `18080` to avoid the Keycloak pilot on `8080`.

## Endpoints

`POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me`, `PUT /auth/profile`, `GET /auth/verify`, `GET /admin/users`, `PUT /admin/users/:id`, and `GET /health`. See [the API contract](docs/go-auth-contract.md) for the legacy response shape and [the deployment guide](docs/go-auth-operations.md) for cutover constraints.

The Go API intentionally does not serve `/events`, `/radar`, `/team`, or `/analytics`. Existing website profile enrichment from those tables needs another owner before `/auth/me` can be switched without changing the website UI.

## Code layout

`internal/domain` owns the auth models and repository interfaces. `internal/auth` implements login, token, profile, and admin rules against those interfaces. `internal/repository` is the GORM/PostgreSQL adapter. `internal/httpapi/api.go` registers Fiber routes, `handler.go` handles requests, and `internal/httpapi/dto/request.go` and `response.go` define the JSON contract.
