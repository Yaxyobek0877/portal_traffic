# Security policy

## Reporting a vulnerability

If you've found a security issue in Portal, please report it privately
before disclosing publicly.

- **Email:** *(maintainer email — TBD; use the address in `git log` for
  now)*
- **Subject line:** start with `[security]` so it's easy to triage.

Please include:

1. A description of the issue and its impact.
2. Reproduction steps or a proof-of-concept.
3. Whether you've already shared the details elsewhere.

Reports get acknowledged within 72 hours. We aim to ship fixes within
14 days of acknowledgement and credit reporters in the release notes
unless they ask not to be named.

---

## Scope

In scope:

- The desktop client (`client/`)
- The shared wire protocol (`shared/protocol/`)
- The signaling server design (the deployed instance is the operator's
  responsibility, but design-level issues are in scope)

Out of scope:

- Vulnerabilities that require the user to install a malicious
  third-party signaling server. Self-hosting is supported, but the
  threat model assumes that endpoint is trusted.
- Theoretical brute-force attacks on the 6-digit portal code that don't
  account for per-IP rate limits (see [ARCHITECTURE.md](ARCHITECTURE.md)).
- Bugs in dependencies (Go stdlib, Wails, pion/webrtc, gorilla/websocket).
  Please report those upstream.

---

## What we promise

- The signaling server cannot read application traffic. If you find
  a way to make it, that's a critical issue.
- A peer in portal A cannot reach a peer in portal B through Portal.
  If you find a way, that's a critical issue.
- The `from` field on relayed `webrtc.*` messages is server-stamped
  and cannot be forged. If you find a way to forge it, that's a high
  issue.

---

## Hardening tips for self-hosters

If you run your own signaling server:

- Use TLS. Cloudflare Origin Certificates are free; see the deployment
  guide.
- Run behind Cloudflare (or another DDoS-mitigating CDN) with WebSocket
  support enabled.
- Set non-default rate limits if you expect heavy use; the defaults are
  conservative.
- Run as a non-root user with the supplied systemd hardening profile.
- Rotate Origin Certificates before they expire (15-year default from
  Cloudflare, but still — set a calendar reminder).
