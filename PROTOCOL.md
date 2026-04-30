# Wire Protocol

This is the canonical reference for the messages Portal exchanges. The
authoritative form lives in [`shared/protocol/messages.go`](shared/protocol/messages.go);
this document is the human-readable companion.

---

## Transport

* **Signaling layer:** WebSocket carrying JSON text frames. Each frame
  is one independent message. Endpoint: `wss://<signaling-host>/ws`.
* **P2P layer:** WebRTC data channels carrying JSON text frames (for
  control / chat / service announcements) or binary frames (for file
  chunks and tunneled traffic).

Every JSON message has a top-level `"type"` field. To decode, parse
just that field first, then re-parse into the type-specific struct.

---

## Identifiers

| Field | Format | Notes |
| --- | --- | --- |
| `portal_id` | 6 ASCII digits | shown to users, drawn from `crypto/rand` |
| `code`      | 6 ASCII digits | secret, shown alongside the ID |
| `peer_id`   | UUID v4 string | server-assigned at WebSocket connect |
| `virtual_ip`| `10.42.0.X`    | per-portal allocation, X ∈ [1, 254] |
| `request_id`| free-form string | client-chosen, echoed back for correlation |

---

## Signaling messages — Client → Server

### `portal.create`

Allocate a fresh portal and admit the sender as owner.

```json
{
  "type": "portal.create",
  "nickname": "alice",
  "public_nick": true,
  "capacity": 16
}
```

| Field | Type | Notes |
| --- | --- | --- |
| `nickname` | string | 2–24 chars, `[A-Za-z0-9_.-]` |
| `public_nick` | bool | if true, others can `portal.join_by_nick` you |
| `capacity` | int (optional) | 2–254; omit for server default (16) |

### `portal.join`

Join an existing portal by ID and code.

```json
{ "type": "portal.join", "portal_id": "428591", "code": "739204", "nickname": "bob" }
```

### `portal.join_by_nick`

Ask the server to forward a join request to the user with the given
public nickname. The target receives `portal.join_request` and replies
with `portal.join_response`.

```json
{
  "type": "portal.join_by_nick",
  "target_nickname": "alice",
  "my_nickname": "bob",
  "request_id": "client-generated-id"
}
```

### `portal.join_response` (owner only)

Reply to an inbound join request.

```json
{ "type": "portal.join_response", "request_id": "...", "accept": true }
```

### `portal.leave`

Leave the current portal. The connection stays open.

```json
{ "type": "portal.leave" }
```

### `portal.kick` (owner only)

```json
{ "type": "portal.kick", "peer_id": "..." }
```

### `portal.lock` (owner only)

When locked, no new peers may join even with a correct code.

```json
{ "type": "portal.lock", "locked": true }
```

### `nick.set_visibility`

Update your nickname directory entry. Values: `"hidden"`, `"friends_only"`, `"public"`.

```json
{ "type": "nick.set_visibility", "nickname": "alice", "visibility": "public" }
```

---

## Signaling messages — Server → Client

### `portal.created`

Confirms a successful `portal.create`.

```json
{
  "type": "portal.created",
  "portal_id": "428591",
  "code": "739204",
  "peer_id": "...",
  "virtual_ip": "10.42.0.1",
  "capacity": 16
}
```

### `portal.joined`

Confirms a successful join (whether by ID/code or accepted nick request)
and gives you the existing peer roster so you can start handshakes.

```json
{
  "type": "portal.joined",
  "portal_id": "428591",
  "peer_id": "your-peer-id",
  "virtual_ip": "10.42.0.3",
  "peers": [
    { "peer_id": "...", "nickname": "alice", "virtual_ip": "10.42.0.1", "is_owner": true },
    { "peer_id": "...", "nickname": "bob",   "virtual_ip": "10.42.0.2", "is_owner": false }
  ]
}
```

### `portal.peer_joined`

Pushed to existing members when somebody new joins.

```json
{ "type": "portal.peer_joined", "peer_id": "...", "nickname": "charlie", "virtual_ip": "10.42.0.3" }
```

### `portal.peer_left`

Pushed to remaining members when a peer leaves or is kicked.

```json
{ "type": "portal.peer_left", "peer_id": "...", "reason": "left" }
```

`reason` is one of `"left"`, `"kicked"`, `"disconnected"`.

### `portal.join_request`

Delivered to a portal owner whose public nickname was looked up.

```json
{
  "type": "portal.join_request",
  "request_id": "server-generated",
  "from_peer_id": "...",
  "from_nickname": "bob",
  "from_ip_hash_short": "a3f1c2"
}
```

### `portal.kicked`

Delivered to the peer who was kicked.

```json
{ "type": "portal.kicked", "portal_id": "..." }
```

### `portal.locked`

Broadcast to all members on lock state change.

```json
{ "type": "portal.locked", "locked": true }
```

### `portal.closed`

Broadcast when the portal goes away (most commonly because the owner
disconnected).

```json
{ "type": "portal.closed", "portal_id": "...", "reason": "owner_left" }
```

### `error`

Any server-side rejection.

```json
{ "type": "error", "code": "PORTAL_CODE_WRONG", "message": "code does not match", "request_id": "..." }
```

| Code | Meaning |
| --- | --- |
| `INVALID_MESSAGE` | malformed JSON or unknown type |
| `PORTAL_NOT_FOUND` | no portal with that ID |
| `PORTAL_CODE_WRONG` | wrong access code |
| `PORTAL_FULL` | capacity reached |
| `PORTAL_LOCKED` | owner has locked the portal |
| `NICKNAME_INVALID` | malformed nickname |
| `NICKNAME_TAKEN` | another live connection is using that nickname |
| `NICKNAME_NOT_FOUND` | no such online user |
| `NICKNAME_NOT_PUBLIC` | user is not accepting invites |
| `NOT_IN_PORTAL` | action requires being in a portal |
| `ALREADY_IN_PORTAL` | leave first before creating/joining |
| `PEER_NOT_FOUND` | target peer is not in your portal |
| `NOT_OWNER` | action is owner-only |
| `RATE_LIMITED` | per-IP rate limit hit |
| `INTERNAL` | server-side error; retry safely |

---

## WebRTC handshake relay (bidirectional)

The server forwards these verbatim from sender to `to`, **after**
overwriting the `from` field with the sender's authenticated peer ID
(so a malicious peer cannot impersonate another).

### `webrtc.offer`

```json
{ "type": "webrtc.offer", "from": "set-by-server", "to": "...", "sdp": "..." }
```

### `webrtc.answer`

```json
{ "type": "webrtc.answer", "from": "set-by-server", "to": "...", "sdp": "..." }
```

### `webrtc.ice`

```json
{
  "type": "webrtc.ice",
  "from": "set-by-server",
  "to": "...",
  "candidate": "candidate:...",
  "sdp_mid": "0",
  "sdp_mline_index": 0
}
```

An empty `candidate` string signals end-of-candidates per the ICE spec.

---

## P2P data channel: `control`

After the WebRTC connection is up, peers exchange these JSON frames
over the `control` data channel. The server is not involved.

### `ping` / `pong`

```json
{ "type": "ping", "ts": 1730476800000 }
{ "type": "pong", "ts": 1730476800000, "echo_ts": 1730476800000 }
```

`ts` is a unix-millisecond timestamp from the sender's local clock.
`pong.echo_ts` echoes the originating ping's `ts` so RTT is computable
without needing synchronised clocks. Default cadence is 5 s.

### `presence`

```json
{ "type": "presence", "status": "active" }
```

`status` ∈ `"active"`, `"idle"`, `"away"`.

### `service.expose` / `service.unexpose`

Announce or retract a locally exposed service for the proxy layer.

```json
{ "type": "service.expose", "name": "Minecraft", "protocol": "tcp", "port": 25565 }
{ "type": "service.unexpose", "port": 25565 }
```

### `service.list_request` / `service.list_response`

```json
{ "type": "service.list_request" }
{ "type": "service.list_response", "services": [ ... ] }
```

---

## Extension policy

* Adding a new optional field to an existing message type is always
  safe. Older clients ignore unknown fields.
* Adding a new top-level message type is safe. Older clients respond
  with `INVALID_MESSAGE`; senders should be prepared for that.
* Renaming or removing fields is a breaking change. Don't.
* Changing the format of `portal_id` or `code` (e.g. expanding to 8
  digits) is a major version bump.
