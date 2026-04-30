# Contributing

Thanks for wanting to help. This is currently a small project — the
fastest way to be useful is to file a clear issue or send a focused
patch.

## Filing issues

Helpful issues include:

- **What you tried** (commands, screenshots, exact wording of buttons)
- **What happened** (error message, log line, broken behaviour)
- **What you expected** (one sentence is enough)
- **Environment** (OS, version of Portal, signaling URL if non-default)

If your issue includes a log file, redact the `peer_id` UUIDs and your
own `virtual_ip` before posting — they're harmless on their own but
correlatable across other people's logs.

---

## Sending patches

1. Fork and create a branch off `main`.
2. Make the change, with a focused commit history. We squash-merge, so
   the final commit message is what matters; intermediate commits can
   be messy.
3. Run the tests for whichever module you touched:
   ```sh
   cd shared && go test ./...
   ```
4. Run `go fmt ./...` and `go vet ./...`. CI will reject anything that
   doesn't pass.
5. Open a pull request. The description should explain *why* the
   change exists, not just what it does.

---

## Style notes

### Go

- Standard `gofmt` style. We don't use any custom linters beyond `go vet`.
- Prefer accepting `context.Context` as the first parameter on any
  function that touches I/O.
- Errors flow up. Don't `log.Fatal` from a library function.
- Avoid panics outside `init()` and `main()`. Return errors instead.
- Comments explain *why*, not *what*. The code already shows what.

### TypeScript / React (when `client/` lands)

- Tailwind for styling. No CSS-in-JS, no styled-components.
- Zustand for state, not Redux.
- Functional components with hooks. No class components.
- Avoid `any`. If you really need it, leave a one-line comment why.

---

## Backwards compatibility

The wire protocol in [`shared/protocol/messages.go`](shared/protocol/messages.go)
is a public commitment. **Adding** optional fields and new message
types is always safe; **renaming** or **removing** fields breaks every
deployed client and server. If you think you need to do that, open an
issue first to discuss the migration plan.

---

## What's currently the most useful

See [ROADMAP.md](ROADMAP.md). Phase 2 is the active work area —
specifically the `client/peer/` package (WebRTC mesh wiring) and the
`client/signaling/` client (WebSocket → handshake orchestration).
Phase 3+ will need design help on the React side.

---

## What's out of scope

- Cryptocurrency or token-based discovery
- Mandatory account systems / centralised identity
- Integrations with proprietary chat platforms
- Anything that requires the signaling server to see message contents

The whole point of Portal is that the operator can't read your traffic.
Features that conflict with that aren't going to land.

---

## Communication

For now: GitHub issues only. If the project grows, we'll set up a
matrix room or similar.
