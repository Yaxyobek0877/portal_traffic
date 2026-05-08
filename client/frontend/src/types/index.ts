// Mirrors Go side (client/app.go). Keep in sync.

export type PortalView = {
  portalId: string;
  code: string;
  ownerId: string;
  ownPeerId: string;
  ownVip: string;
  isOwner: boolean;
  // Local session id assigned by the Go side. Stable for the
  // lifetime of one portal connection; used to route per-session
  // events into the right slot in the frontend's session map.
  // Empty in legacy callers (CurrentPortal pre-multi-portal).
  sessionId?: string;
};

// PortalSummary is the projection ActivePortals returns. The
// Welcome dashboard's "Faol ulanishlar" strip iterates this.
export type PortalSummary = {
  sessionId: string;
  portalId: string;
  nickname: string;
  isOwner: boolean;
  state: "connecting" | "connected" | "failed" | "closed";
  error?: string;
  peerCount: number;
  isActive: boolean;
};

// Payload for the "portal:ready" event after the multi-portal
// rewrite. Carries everything PortalView used to plus the
// sessionId / nickname / background flag.
export type PortalReadyEvent = PortalView & {
  sessionId: string;
  nickname: string;
  background: boolean;
};

// Payload shape for peer:* events after the multi-portal rewrite.
// Wraps the PeerView with the routing fields the frontend store
// uses to find the right session.
export type PeerEvent = {
  sessionId: string;
  portalId: string;
  peer: PeerView;
};

export type ServiceView = {
  name: string;
  protocol: string;
  port: number;
  target?: string;
  health?: "ok" | "down" | "unknown";
  healthError?: string;
  paused?: boolean;
};

export type PeerView = {
  // sessionId is set on every peer:* event after the multi-portal
  // rewrite. Empty on snapshot calls that don't carry the field.
  sessionId?: string;
  peerId: string;
  nickname: string;
  virtualIp: string;
  isOwner: boolean;
  state: "connecting" | "connected" | "failed" | "closed";
  rttMs: number;
  bytesSent: number;
  bytesRecv: number;
  services: ServiceView[];
  // Empty before ICE nominates a path; "direct" = host/srflx P2P,
  // "relay" = via TURN.
  transport: "" | "direct" | "relay";
  transportLocal: string;
  transportRemote: string;
  // Selected ICE pair addresses (e.g. "192.168.1.53:54538"). Lets
  // the UI tell LAN apart from internet apart from TURN.
  pathLocalAddr: string;
  pathRemoteAddr: string;
};

export type BandwidthResult = {
  peerId: string;
  mbps: number;
  bytesSent: number;
  durationMs: number;
};

export type ChatMessage = {
  sessionId?: string;
  from: string;
  nickname: string;
  text: string;
  at: string;
  isLocal: boolean;
};

export type NATResult = {
  type: number;
  label: string;
  localAddr: string;
  reflexiveAddrs: string[];
  needsTurn: boolean;
  detectedAt: string;
  servers: string[];
};

export type Manifest = {
  name: string;
  size: number;
  mime?: string;
};

export type TransferProgress = {
  sessionId?: string;
  xferId: number;
  peerId: string;
  direction: "send" | "recv";
  manifest: Manifest;
  bytes: number;
  total: number;
  done: boolean;
  error?: string;
  startedAt: string;
  updatedAt: string;
  savePath?: string;
};

export type HistoryEntry = {
  id: number;
  portalId: string;
  code: string;
  nickname: string;
  isOwner: boolean;
  // Optional user-given name. When empty the UI falls back to the
  // portal id. Set via app.RenamePortal(historyId, label).
  label: string;
  joinedAt: string;
  lastSeen: string;
};

export type TurnConfig = {
  url: string;
  username: string;
  credential: string;
};

export type LocalListener = {
  port: number;
  protocol: "tcp" | "udp";
  process: string;
  pid: number;
  local: string;
};

export type LANDiscovery = {
  ip: string;
  port: number;
  protocol: "tcp";
  service: string; // "rtsp" | "http" | "ipp" | …
  hostname: string;
};

export type RiskAssessment = {
  level: "safe" | "warn" | "danger";
  reason: string;
  hint: string;
};

export type ActivityEntry = {
  time: string; // RFC3339
  peerId: string;
  nickname?: string;
  protocol: "tcp" | "udp";
  port: number;
  target: string;
  result: string; // "ok" | "error: ..."
};

export type TurnTestResult = {
  ok: boolean;
  message: string;
  types: string[];
  hadRelay: boolean;
  gatherMs: number;
  urls: string[];
};

export type CloudflareTurnConfig = {
  tokenId: string;
  apiToken: string;
};
