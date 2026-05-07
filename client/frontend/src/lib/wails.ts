// Wails injects a global `window.go.main.App` with the methods we
// declared on *App in app.go. This module gives us typed access to
// them and degrades gracefully when running in a regular browser
// (npm run dev without `wails dev` wrapping it) so the UI is still
// previewable without a backend.

import type {
  PortalView,
  PeerView,
  ServiceView,
  ChatMessage,
  NATResult,
  HistoryEntry,
  TurnConfig,
  LocalListener,
  TurnTestResult,
  CloudflareTurnConfig,
  BandwidthResult,
  LANDiscovery,
  RiskAssessment,
  ActivityEntry,
} from "../types";

type Bridge = {
  SignalingURL: () => Promise<string>;
  SetSignalingURL: (url: string) => Promise<void>;
  CreatePortal: (nickname: string, publicNick: boolean) => Promise<PortalView>;
  JoinPortal: (nickname: string, portalId: string, code: string) => Promise<PortalView>;
  Leave: () => Promise<void>;
  CurrentPortal: () => Promise<PortalView>;
  Peers: () => Promise<PeerView[]>;
  SendChat: (text: string) => Promise<number>;
  LocalServices: () => Promise<ServiceView[]>;
  ExposeService: (name: string, protocol: "tcp" | "udp", port: number, target: string) => Promise<void>;
  SetExposeEnabled: (port: number, protocol: "tcp" | "udp", enabled: boolean) => Promise<void>;
  UnexposeService: (port: number) => Promise<void>;
  DialService: (peerId: string, protocol: "tcp" | "udp", remotePort: number, localPort: number) => Promise<string>;
  NATInfo: () => Promise<NATResult>;
  SaveDir: () => Promise<string>;
  SendFile: (peerId: string) => Promise<string>;
  SendFilePath: (peerId: string, path: string) => Promise<string>;
  OpenSaveDir: () => Promise<void>;
  RecentPortals: (n: number) => Promise<HistoryEntry[]>;
  ClearHistory: () => Promise<void>;
  GetTurnConfig: () => Promise<TurnConfig>;
  SetTurnConfig: (c: TurnConfig) => Promise<void>;
  LocalListeners: () => Promise<LocalListener[]>;
  ScanLAN: () => Promise<LANDiscovery[]>;
  AssessExposeRisk: (target: string, protocol: "tcp" | "udp", port: number) => Promise<RiskAssessment>;
  ProxyActivity: () => Promise<ActivityEntry[]>;
  LogLines: (n: number) => Promise<string[]>;
  LogFilePath: () => Promise<string>;
  OpenLogFolder: () => Promise<void>;
  ClearLogs: () => Promise<void>;
  TestTurn: () => Promise<TurnTestResult>;
  GetCloudflareTurn: () => Promise<CloudflareTurnConfig>;
  SetCloudflareTurn: (c: CloudflareTurnConfig) => Promise<void>;
  TestCloudflareTurn: () => Promise<TurnTestResult>;
  MeasureBandwidth: (peerId: string) => Promise<BandwidthResult>;

  // Updater + crash reporting (added v0.4.0)
  AppVersion: () => Promise<string>;
  CheckForUpdate: (refresh: boolean) => Promise<UpdateResult>;
  OpenReleasePage: (url: string) => Promise<void>;
  CrashReports: () => Promise<CrashReport[]>;
  OpenCrashFolder: () => Promise<void>;
  ClearCrashReports: () => Promise<void>;

  // Local vault unlock (added v0.5.0). HasPassword tells us whether to
  // show the setup screen vs. the unlock screen on launch. SetPassword
  // throws on too-short input ("password_too_short"). VerifyPassword
  // returns false on mismatch — never throws so the UI can show its
  // own localised error. ResetVault wipes hash + history.
  HasPassword: () => Promise<boolean>;
  SetPassword: (password: string) => Promise<void>;
  VerifyPassword: (password: string) => Promise<boolean>;
  ResetVault: () => Promise<void>;
};

export type UpdateResult = {
  available: boolean;
  currentVersion: string;
  latestVersion: string;
  releaseUrl: string;
  releaseNotes: string;
  publishedAt: string;
  assetForOs: string;
  checkedAt: string;
  error?: string;
};

export type CrashReport = {
  id: string;
  capturedAt: string;
  portalVersion: string;
  goVersion: string;
  os: string;
  arch: string;
  panicMessage: string;
  stack: string;
  goroutineCount: number;
};

declare global {
  interface Window {
    go?: { main?: { App?: Bridge } };
    runtime?: {
      EventsOn: (event: string, cb: (...args: any[]) => void) => () => void;
      EventsOff: (event: string) => void;
      EventsEmit: (event: string, ...args: any[]) => void;
    };
  }
}

const stub: Bridge = {
  SignalingURL: async () => "wss://signaling.1pro.uz/ws (preview mode)",
  SetSignalingURL: async () => {},
  CreatePortal: async (nickname) => ({
    portalId: "424242",
    code: "131313",
    ownerId: "preview-owner",
    ownPeerId: "preview-owner",
    ownVip: "10.42.0.1",
    isOwner: true,
  }),
  JoinPortal: async (nickname, portalId) => ({
    portalId,
    code: "",
    ownerId: "preview-owner",
    ownPeerId: "preview-self",
    ownVip: "10.42.0.2",
    isOwner: false,
  }),
  Leave: async () => {},
  CurrentPortal: async () => ({
    portalId: "",
    code: "",
    ownerId: "",
    ownPeerId: "",
    ownVip: "",
    isOwner: false,
  }),
  Peers: async () => [],
  SendChat: async () => 0,
  LocalServices: async () => [],
  ExposeService: async () => {},
  SetExposeEnabled: async () => {},
  UnexposeService: async () => {},
  DialService: async () => "127.0.0.1:0 (preview)",
  NATInfo: async () => ({
    type: 0,
    label: "preview",
    localAddr: "192.168.0.1:0",
    reflexiveAddrs: [],
    needsTurn: false,
    detectedAt: new Date().toISOString(),
    servers: [],
  }),
  SaveDir: async () => "~/Downloads/Portal",
  SendFile: async () => "0",
  SendFilePath: async () => "0",
  OpenSaveDir: async () => {},
  RecentPortals: async () => [],
  ClearHistory: async () => {},
  GetTurnConfig: async () => ({ url: "", username: "", credential: "" }),
  SetTurnConfig: async () => {},
  LocalListeners: async () => [],
  ScanLAN: async () => [],
  AssessExposeRisk: async () => ({ level: "safe" as const, reason: "", hint: "" }),
  ProxyActivity: async () => [],
  LogLines: async () => ["[preview] no logs"],
  LogFilePath: async () => "~/.portal/logs/portal.log",
  OpenLogFolder: async () => {},
  ClearLogs: async () => {},
  TestTurn: async () => ({
    ok: false,
    message: "preview mode",
    types: [],
    hadRelay: false,
    gatherMs: 0,
    urls: [],
  }),
  GetCloudflareTurn: async () => ({ tokenId: "", apiToken: "" }),
  SetCloudflareTurn: async () => {},
  TestCloudflareTurn: async () => ({
    ok: false,
    message: "preview",
    types: [],
    hadRelay: false,
    gatherMs: 0,
    urls: [],
  }),
  MeasureBandwidth: async (peerId: string) => ({
    peerId,
    mbps: 0,
    bytesSent: 0,
    durationMs: 0,
  }),
  AppVersion: async () => "0.4.0-preview",
  CheckForUpdate: async () => ({
    available: false,
    currentVersion: "0.4.0-preview",
    latestVersion: "",
    releaseUrl: "",
    releaseNotes: "",
    publishedAt: "",
    assetForOs: "",
    checkedAt: new Date().toISOString(),
  }),
  OpenReleasePage: async () => {},
  CrashReports: async () => [],
  OpenCrashFolder: async () => {},
  ClearCrashReports: async () => {},

  // In preview mode treat the vault as unlocked + no password set.
  // SetPassword silently succeeds; VerifyPassword accepts any input
  // long enough to be valid. The lock screen mostly stays out of the
  // dev-loop developer's way.
  HasPassword: async () => false,
  SetPassword: async () => {},
  VerifyPassword: async (pwd) => pwd.length >= 4,
  ResetVault: async () => {},
};

export const app: Bridge =
  (typeof window !== "undefined" && window.go?.main?.App) || stub;

export type ChatHandler = (msg: ChatMessage) => void;
export type PeerHandler = (peer: PeerView) => void;
export type PortalHandler = (portal: PortalView) => void;

// Subscribe to a Wails event. Returns an unsubscribe function. In
// preview mode (no Wails runtime), returns a no-op.
export function subscribe<T = unknown>(
  event: string,
  cb: (payload: T) => void
): () => void {
  if (typeof window === "undefined" || !window.runtime) {
    return () => {};
  }
  return window.runtime.EventsOn(event, cb as any);
}
