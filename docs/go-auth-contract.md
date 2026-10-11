# Legacy auth contract for the Go migration

This is a code-level inventory of `../auth` as inspected on 2026-10-11. It records what the Express application currently does; it is not a claim that every behavior is safe or should be copied unchanged. The migration plan is in [../tasks/plan.md](../tasks/plan.md).

## Ownership boundary

Firebase Authentication verifies the person's credentials. The browser sends a Firebase ID token to the auth API. The API verifies it with Firebase Admin, resolves a PostgreSQL `users` row, then issues its own platform JWTs. The migration covers only `users`, `refresh_tokens`, `user_sessions`, and `auth_audit_log`. `user_sessions` and `auth_audit_log` exist in the schema but are not used by the current login service.

The Express process also serves `/events`, `/radar`, `/team`, `/analytics`, and other routes. Their tables, handlers, and migrations are outside this Go migration. `/admin/users` and `/admin/users/:id` are included because they operate on the `users` auth table.

## HTTP endpoints

All responses below are JSON. Protected endpoints expect `Authorization: Bearer <platform access token>`. The common middleware currently returns `401` when the token is missing and `403` when JWT verification fails. A replacement should document any deliberate error-code changes before clients switch.

| Method and path | Request | Success response | Current error behavior |
|---|---|---|---|
| `POST /auth/login` | `{ "firebase_token": "..." }` | `{ "success": true, "user": { "id", "email", "full_name", "avatar_url", "roles", "teams" }, "tokens": { "access_token", "refresh_token", "token_type": "Bearer", "expires_in": 86400 } }` | `400` missing token; `401` verification or login failure |
| `POST /auth/refresh` | `{ "refresh_token": "..." }` | `{ "success": true, "tokens": { "access_token", "token_type": "Bearer", "expires_in": 86400 } }` | `400` missing token; `401` invalid, expired, or inactive stored token |
| `POST /auth/logout` | Optional `{ "refresh_token": "..." }` plus access bearer token | `{ "success": true, "message": "Logged out successfully" }` | `500` on handler failure; middleware errors as above |
| `GET /auth/me` | Access bearer token | `{ "success": true, "user": { ... } }` | `404` absent user; `500` query failure |
| `PUT /auth/profile` | Access bearer token plus allowed profile fields | `{ "success": true, "user": { ... } }` | `500` update/decorate failure |
| `GET /auth/verify` | Access bearer token | `{ "success": true, "valid": true, "user": <decoded JWT claims> }` | Middleware errors as above |
| `GET /admin/users` | Admin bearer token; `search`, `role`, `page`, `limit` | `{ "success": true, "users", "total", "page", "limit" }` | `401`/`403` auth/role failure; `500` query failure |
| `PUT /admin/users/:id` | Admin bearer token; subset of `roles`, `teams`, `is_active` | `{ "success": true, "user": { ... } }` | `404` absent user; `401`/`403` auth/role failure; `500` update failure |
| `GET /health` | None | `{ "status": "healthy" }` | Always returns `200` in Express; it does not check dependencies per request |

`GET /auth/me` selects `id`, `email`, `full_name`, `avatar_url`, `student_status`, `primary_track`, `teams`, `roles`, `tos_agreed`, and `created_at` from `users`. It then adds `activity`, `certificates`, and `team_membership` by querying non-auth tables. `PUT /auth/profile` returns an updated `users` row plus the same enrichment. Those three derived fields need another service owner before `/auth/me` and `/auth/profile` can be switched without changing the website experience.

The profile update allowlist is `full_name`, `whatsapp_number`, `avatar_url`, `gender`, `birthday`, `student_status`, `matric_no`, `department`, `faculty`, `primary_track`, `secondary_track`, `primary_skill_level`, `secondary_skill_level`, `tos_agreed`, `tos_agreed_at`, and `tos_version`. Admin updates are limited to `roles`, `teams`, and `is_active`. The Go API must use an allowlist as well.

## Identity and token rules

- Firebase ID tokens are verified by `firebase-admin` in `src/services/authService.js`. Existing users are found by `firebase_uid`. A new user is created in the local `users` table; the database assigns a UUID. The Go version must preserve existing UUIDs and Firebase UIDs.
- The current code may **replace an existing `firebase_uid` solely because a new Firebase token has the same email**. This is an account-linking risk. The Go version should return a conflict and require verified account ownership instead of copying that behavior.
- Platform access JWTs carry `user_id`, `email`, `full_name`, `roles`, and `teams`; refresh JWTs carry `user_id` and `email`. Both use issuer `auth-service`, audience `platform-services`, and the configured HMAC secret. Defaults are 24 hours for access and seven days for refresh. The legacy code has no explicit token-type claim.
- Refresh JWTs are stored as SHA-256 hashes in `refresh_tokens`; the plaintext token is returned to the client. Refresh checks signature, database hash, `is_active`, and expiry, then issues a new access JWT. It does not rotate the refresh token. Logout revokes the specified refresh-token hash or all active refresh tokens for the user if none was supplied.
- The current middleware only verifies JWT signature, issuer, audience, and expiry. It does not check the database user's current active status or current roles on every protected request. The Go version should add those checks and distinguish access from refresh tokens. It must not use the legacy hardcoded fallback JWT secret.

## Client compatibility findings

`../GDGWebsite/lib/auth-service.ts` calls all `/auth/*` endpoints. It stores both tokens and merges partial profile responses in local storage. Its refresh helper calls `storeTokens(data.tokens)`, but the current `/auth/refresh` response contains **no `refresh_token`**. That mismatch can overwrite the stored refresh token with an invalid value. Coordinate a compatible response or client correction before switching the route.

The website also uses `activity` and `certificates` from `/auth/me`, while the Go auth scope excludes their tables. Full profile parity therefore depends on a separate enrichment service. Radar's current Keycloak session path is separate; do not assume it consumes this legacy Firebase API without a fresh caller audit.

## Sources inspected

- `../auth/src/routes/authRoutes.js`, `../auth/src/routes/adminRoutes.js`
- `../auth/src/controllers/authController.js`, `../auth/src/controllers/adminController.js`
- `../auth/src/services/authService.js`, `../auth/src/models/userModel.js`
- `../auth/src/middleware/authMiddleware.js`, `../auth/src/utils/jwt.js`
- `../auth/database/schema.sql`, `../GDGWebsite/lib/auth-service.ts`
