# ADR-0006 Same origin behind Caddy with server side cookie sessions

Status: Accepted, 2026-10-17

## Context

`web` and `api` are separate processes. Sessions must be revocable at once (suspension,
password reset, role change), officers need short idle timeouts, and no token should be
readable by JavaScript.

## Decision

Serve both under one origin: Caddy routes `/api/*` to `api` and everything else to `web`.
Authenticate with an opaque random session token in an `HttpOnly`, `Secure`,
`SameSite=Lax` cookie, stored hashed in the `sessions` table and checked on every request.
Next.js server components forward the cookie to `api` over the internal network.

## Consequences

- No CORS configuration, no tokens in local storage, no refresh token dance.
- Revocation is a row update and takes effect on the next request.
- One indexed database lookup per request, deliberately not cached, so a revocation is
  effective on every instance at once. At our volume it costs well under a millisecond.
- CSRF is handled by `SameSite=Lax` plus an origin check on unsafe methods.

## Alternatives

- **JWT access tokens with refresh tokens.** Stateless verification, but revocation needs a
  deny list anyway, and the token lifetime becomes the security window.
- **Separate origins with CORS.** More configuration and third party cookie behaviour to
  manage in browsers, for no gain.
