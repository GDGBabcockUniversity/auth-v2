# Go/Fiber/Firebase auth migration tasks

The scope and design choices are in [plan.md](plan.md). Each task is independently reviewable; complete checkpoints in order.

## Implementation status (2026-10-11)

The Go service, auth-only schema for local testing, Docker package, route example, and focused service, HTTP, and repository tests are implemented. The repository test uses `AUTH_TEST_DATABASE_URL` and a disposable PostgreSQL database. Production migration is pending the read-only data inventory, representative legacy token and UUID comparisons, Firebase staging login, website profile enrichment decision, and route switch. The checkboxes below remain open where these external checks are required; code presence alone does not prove a staging checkpoint.

## Task 1: Freeze the auth contract and data inventory

**Description:** Record real request/response shapes, JWT claims, auth-only table columns, representative account UUIDs, and all callers of the current auth endpoints. Identify which `/auth/me` fields come from other tables.

**Acceptance criteria:**
- [ ] Auth routes, status codes, response envelopes, JWT claims, and frontend callers are catalogued.
- [ ] A read-only audit reports duplicate emails/Firebase UIDs, inactive users, and existing refresh-token counts without personal data in committed files.
- [ ] The four in-scope tables and excluded enrichment fields are named explicitly.

**Verification:** Review `src/routes/authRoutes.js`, `src/routes/adminRoutes.js`, `src/services/authService.js`, `src/models/userModel.js`, `database/schema.sql`, and website `lib/auth-service.ts`; validate aggregate queries on a backup.

**Dependencies:** None.

**Files likely touched:** `tasks/plan.md`, `tasks/todo.md`, `docs/go-auth-contract.md`.

**Estimated scope:** Small.

## Task 2: Bootstrap the Go service

**Description:** Create a Go module and Fiber server with strict configuration, a health endpoint, GORM PostgreSQL connection, Firebase Admin initialization, and test seams. This does not replace the Express entrypoint yet.

**Acceptance criteria:**
- [ ] Go starts only with valid database, Firebase project/credentials, JWT key, and allowed origins.
- [ ] `/health` works and startup does not modify the database schema.
- [ ] A disposable database can be used in tests with no production credentials.

**Verification:** `go test ./...`, `go build ./...`, and a manual startup against a disposable database.

**Dependencies:** Task 1.

**Files likely touched:** `go.mod`, `cmd/auth/main.go`, `internal/config/config.go`, `internal/database/database.go`, `internal/httpapi/api.go`.

**Estimated scope:** Medium.

## Task 3: Add Firebase identity resolution and login

**Description:** Verify Firebase ID tokens server-side, then find or create a user by Firebase UID within the existing `users` table. Preserve UUIDs, default roles, and response shape. Reject unproven email-only links and handle concurrent inserts.

**Acceptance criteria:**
- [ ] Returning UID preserves the exact `users.id`; a new UID creates one user; inactive users are denied.
- [ ] Expired/wrong-project/invalid Firebase tokens and same-email different-UID conflicts are rejected without modifying an existing identity.
- [ ] `POST /auth/login` returns the documented user and token envelope without joining non-auth tables.

**Verification:** Focused service and Fiber handler tests with a Firebase verifier seam and disposable Postgres; `go test ./...` and `go build ./...`.

**Dependencies:** Tasks 1–2.

**Files likely touched:** `internal/domain/user.go`, `internal/repository/user.go`, `internal/auth/login.go`, `internal/httpapi/auth.go`, focused tests.

**Estimated scope:** Medium.

## Checkpoint: login

- [ ] Representative existing UUIDs and login response fields match the legacy auth API.
- [ ] No test or runtime startup touched non-auth tables.
- [ ] The email collision path is rejected and documented for account recovery.

## Task 4: Implement platform JWT verification and refresh

**Description:** Issue and verify compatible platform JWTs, store only refresh-token hashes in `refresh_tokens`, and implement `/auth/refresh`. Bind refresh to an active user and current roles; distinguish access and refresh token use.

**Acceptance criteria:**
- [ ] Existing valid access-token claims and issuer/audience are preserved during rollout.
- [ ] Refresh succeeds only for an unexpired, active stored token and active user; reuse/expired/wrong-type tokens are rejected.
- [ ] Secrets and raw refresh tokens are absent from logs and storage.

**Verification:** Focused token and repository tests, compatibility checks against fixture tokens, `go test ./...`, `go build ./...`.

**Dependencies:** Task 3.

**Files likely touched:** `internal/auth/tokens.go`, `internal/repository/refresh.go`, `internal/httpapi/auth.go`, focused tests.

**Estimated scope:** Medium.

## Task 5: Add logout and protected profile routes

**Description:** Implement `/auth/logout`, `/auth/me`, `/auth/profile`, and `/auth/verify` using Fiber middleware and the `users` table only. Apply a strict profile update allowlist and preserve the auth-owned response fields.

**Acceptance criteria:**
- [ ] Logout revokes one refresh token or all of the user's refresh tokens as the request specifies.
- [ ] Protected routes reject missing, malformed, expired, wrong-type, or disabled-user tokens.
- [ ] Profile updates cannot change ID, Firebase UID, email, roles, active flag, or timestamps; no excluded table is queried.

**Verification:** Handler tests for success and denial paths, `go test ./...`, `go build ./...`, and an HTTP contract comparison for auth-owned fields.

**Dependencies:** Task 4.

**Files likely touched:** `internal/httpapi/middleware.go`, `internal/httpapi/profile.go`, `internal/auth/profile.go`, focused tests.

**Estimated scope:** Medium.

## Checkpoint: core auth

- [ ] Login, refresh, logout, profile, and verification work against disposable Postgres and Firebase test identities.
- [ ] Existing website login flow works against the Go service in a test environment.
- [ ] Missing non-auth `/auth/me` enrichment has an explicit owner before switching production `/auth/me`.

## Task 6: Add admin user endpoints and audit events

**Description:** Port `/admin/users` list/search and update for roles, teams, and active status. Write meaningful login/logout/role-change records to `auth_audit_log`; do not port event/team/Radar administration.

**Acceptance criteria:**
- [ ] Only current active admins can list or update users; paging and search are bounded.
- [ ] Role and active-status changes are audited without logging tokens or private credentials.
- [ ] Admin methods touch only `users` and `auth_audit_log`.

**Verification:** Role-denial and update tests with disposable Postgres; `go test ./...`, `go build ./...`.

**Dependencies:** Task 5.

**Files likely touched:** `internal/httpapi/admin.go`, `internal/auth/admin.go`, `internal/repository/user.go`, focused tests.

**Estimated scope:** Medium.

## Task 7: Prepare deployment and path routing

**Description:** Package the Go service and route only auth-owned endpoints to it. Keep all other Express paths on the old service. Document environment values, database backup, monitoring, and rollback.

**Acceptance criteria:**
- [ ] Go and Express coexist without any non-auth route moving to Go.
- [ ] Both services can resolve the same UUID and token contract in staging.
- [ ] A route-level rollback is documented and rehearsed without changing database identities.

**Verification:** Staging smoke checks for each auth route and one excluded route, `go test ./...`, `go build ./...`, and observed route/identity logs.

**Dependencies:** Tasks 5–6 and the `/auth/me` enrichment decision.

**Files likely touched:** `Dockerfile`, deployment routing config, `.env.example`, `docs/go-auth-operations.md`.

**Estimated scope:** Medium.

## Checkpoint: cutover readiness

- [ ] Existing user IDs, roles, Firebase login, and website session behavior remain correct in staging.
- [ ] The old service still serves all excluded routes.
- [ ] Backup, route switch, and rollback are verified; unresolved `/auth/me` enrichment blocks cutover.
