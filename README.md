# auth-v2

Go/Fiber replacement for the GDG Babcock platform authentication backend. It removes Firebase and issues platform JWTs with URI audiences.

Copy `.env.example` to `.env`, set a real `JWT_HMAC_SECRET`, then run `docker compose up --build`. The API listens on `http://localhost:8080`; Mailpit is at `http://localhost:8025`.

Endpoints currently provided: `POST /auth/register`, `POST /auth/login`, `GET /auth/google`, `GET /auth/google/callback`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me`, `GET /auth/verify`, and `GET /health`.

Tokens use `iss=JWT_ISSUER` and an audience chosen from `JWT_AUDIENCES`. The receiving service must validate its own URI. Firebase tokens are intentionally not accepted.

To enable Google sign-in, set `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and `GOOGLE_REDIRECT_URL` (registered in Google Cloud as `<your auth origin>/auth/google/callback`). Visiting `GET /auth/google` starts an OAuth authorization-code flow. The callback validates Google's ID token, requires a verified email, links an existing matching email or creates a user, and returns the usual session response. OAuth state is protected by a short-lived, HTTP-only cookie.
