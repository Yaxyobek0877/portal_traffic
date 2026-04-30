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
  services: ServiceView[];
};

export type ChatMessage = {
  from: string;
  nickname: string;
  text: string;
  at: string; // ISO date string from Go's time.Time
  isLocal: boolean;
};
