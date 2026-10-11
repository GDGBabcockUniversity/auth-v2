# Plan: migrate the legacy auth application to Go

## Objective and scope

Replace the authentication portion of `../auth` (Express) with a Go service using Fiber, GORM, PostgreSQL, and the Firebase Admin SDK. Firebase Authentication remains the source of sign-in identity: clients continue to authenticate with Firebase and send a Firebase ID token to `POST /auth/login`. The Go service verifies that token, resolves a local user, and issues the platform access and refresh tokens expected by existing clients. Keep existing `users.id` UUIDs, Firebase UIDs, and auth data. Work only with `users`, `refresh_tokens`, `user_sessions`, and `auth_audit_log` from `database/schema.sql`.

This plan does not migrate or query the event, certificate, Radar, team, wrapped, analytics, or seed-data tables. It does not migrate their routes or controllers. The old Express deployment must continue to serve those routes until a separate ownership plan exists. No production cutover is authorized by this document.

## Existing behavior and compatibility seams

The endpoint, token, and client details are recorded in [the legacy auth contract](../docs/go-auth-contract.md).

- `POST /auth/login` verifies a Firebase ID token and returns `{ success, user, tokens }`; platform JWTs have issuer `auth-service`, audience `platform-services`, and `user_id`/`roles` claims. Firebase ID tokens are distinct from platform JWTs.
- `POST /auth/refresh` currently returns a new access token; `POST /auth/logout` revokes refresh tokens; `GET /auth/me`, `PUT /auth/profile`, and `GET /auth/verify` use a platform access token. `GET /admin/users` and `PUT /admin/users/:id` operate on `users` and are inside the auth-table scope. `/health` is needed for deployment.
- The website's `lib/auth-service.ts` calls every `/auth/*` endpoint and merges partial profile responses. Its profile screen also expects `activity`, `certificates`, and `team_membership` computed from non-auth tables. The Go auth service cannot provide those fields within this scope. Before switching `/auth/me` traffic, move those lookups behind another service or explicitly accept the UI regression; do not copy their queries into Go.
- The legacy `loginWithFirebase` reassigns `firebase_uid` when a new Firebase identity has the same email. This can attach an unproven identity to an existing account. The Go implementation must reject this conflict until Firebase account linking or another proof-of-ownership procedure resolves it. The same rule applies to concurrent first logins.
- The legacy schema has default UUIDs, text arrays for roles/teams, nullable timestamps, and a database update trigger. GORM models must map these precisely. GORM `AutoMigrate` must not run against the existing database.
- The legacy Express app also owns non-auth routes, so switching the entire hostname to Go would break them. Route only `/auth/*`, `/admin/users*`, and the Go health route to the new service during a staged cutover. Preserve the existing service for all other paths.

## Architecture choices

1. Use Firebase Admin SDK for Go only to verify Firebase ID tokens. Do not accept unverified email claims and do not put Firebase service-account credentials in the repo. Reuse the existing Firebase project and support Application Default Credentials or a private service-account file/JSON through environment configuration.
2. Use Fiber for routing, request limits, CORS allowlisting, response envelopes, and error handling. Keep route handlers thin; an auth service owns identity and token rules; a GORM repository owns only the four auth tables.
3. Preserve the existing PostgreSQL schema and data. Use explicit GORM table/column mappings and SQL migrations only where a proven schema change is necessary. Do not migrate unrelated tables or rerun the legacy all-table migration scripts.
4. Preserve the current platform token contract during rollout, including issuer, audience, and required claims. Require a strong configured signing key, reject wrong token type/issuer/audience/expiry, hash refresh tokens in storage, and check active-user status and current roles on protected operations. Add refresh-token rotation as a separate compatible change only after client behavior is confirmed.
5. Resolve users by verified Firebase UID. For a new UID with an existing email, return a conflict that requires account linking or owner verification. Never reassign `firebase_uid` based only on email. Provision under a transaction and unique constraints so concurrent requests yield one account.
6. Use an HTTP contract and a small set of representative existing UUIDs to compare the Go service with Express before any traffic switch. The auth-owned fields of `/auth/me` must match; non-auth enrichment needs a separate provider before full UI parity.

## Dependency and delivery order

```text
contract/inventory → Go bootstrap → user identity/login → JWT and refresh/logout
                                           ↓
                                profile/verify → admin users → cutover
```

Implement one endpoint path at a time. Tasks and acceptance checks are in [todo.md](todo.md). Run Go tests and a build at each checkpoint. Use a test Firebase project or Firebase Auth emulator and a disposable PostgreSQL database; never point tests at the production database.

## Risks and decisions

| Risk | Handling |
|---|---|
| Same-email Firebase identities are silently merged today | Record conflicts; require proof of ownership; do not reproduce unsafe relinking. |
| Website loses activity/certificate/team fields from `/auth/me` | Keep legacy enrichment or move it to a separate service before switching that route. This is an external dependency, not part of the Go auth implementation. |
| Other services validate old JWT signatures/claims | Keep signing contract during transition and inventory validators before changing key or claims. |
| Old and new services issue refresh tokens concurrently | Share the auth tables and signing configuration, test both directions, and make one route owner at a time. |
| Go deployment replaces non-auth Express routes | Route by path; retain the Express service until those routes get separate migrations. |
| Firebase and Postgres identities drift | Preserve `firebase_uid` and `users.id`; audit duplicates/conflicts from a read-only export before cutover. |

## Open decisions before cutover

- Which service will supply `activity`, `certificates`, and `team_membership` to the website after `/auth/me` moves? The Go auth service will not read their tables.
- Which deployment/router owns path routing on `auth.gdgbabcock.com`? The current repo has a Vercel Express entrypoint; a Go runtime or container target is needed.
- Should the new service retain the current 24-hour access JWT and non-rotating seven-day refresh token initially, or should session hardening be a separate client-coordinated release? The plan assumes contract preservation first.
