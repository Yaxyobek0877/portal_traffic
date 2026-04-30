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
  ExposeService: (name: string, port: number) => Promise<void>;
  UnexposeService: (port: number) => Promise<void>;
  DialService: (peerId: string, remotePort: number, localPort: number) => Promise<string>;
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
