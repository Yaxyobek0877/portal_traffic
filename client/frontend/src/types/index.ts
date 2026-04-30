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
  process: string;
  pid: number;
  local: string;
};

export type TurnTestResult = {
  ok: boolean;
  message: string;
  types: string[];
  hadRelay: boolean;
  gatherMs: number;
  urls: string[];
};
