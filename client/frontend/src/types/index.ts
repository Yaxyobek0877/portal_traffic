// Mirrors Go side (client/app.go). Keep in sync.

export type PortalView = {
  portalId: string;
  code: string;
  ownerId: string;
  ownPeerId: string;
  ownVip: string;
  isOwner: boolean;
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
