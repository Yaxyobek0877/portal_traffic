// Single Zustand store for the desktop app's runtime state. Mirrors
// what Wails events stream over from the Go side so the UI never
// has to poll.

import { create } from "zustand";
import type {
  ChatMessage,
  HistoryEntry,
  NATResult,
  PeerView,
  PortalView,
  ServiceView,
  TransferProgress,
} from "../types";

type Screen = "welcome" | "portal" | "settings";

type Store = {
  screen: Screen;
  setScreen: (s: Screen) => void;

  nickname: string;
  setNickname: (n: string) => void;

  signalingUrl: string;
  setSignalingUrl: (u: string) => void;

  portal: PortalView | null;
  setPortal: (p: PortalView | null) => void;

  peers: Record<string, PeerView>;
  upsertPeer: (p: PeerView) => void;
  removePeer: (peerId: string) => void;
  clearPeers: () => void;

  localServices: ServiceView[];
  setLocalServices: (s: ServiceView[]) => void;

  messages: ChatMessage[];
  addMessage: (m: ChatMessage) => void;
  clearMessages: () => void;

  banner: string;
  setBanner: (b: string) => void;

  connecting: boolean;
  setConnecting: (b: boolean) => void;

  // Phase 4 additions
  nat: NATResult | null;
  setNat: (n: NATResult | null) => void;

  transfers: Record<string, TransferProgress>;
  upsertTransfer: (t: TransferProgress) => void;
  clearFinishedTransfers: () => void;

  history: HistoryEntry[];
  setHistory: (h: HistoryEntry[]) => void;

  saveDir: string;
  setSaveDir: (d: string) => void;
};

const MAX_MESSAGES = 500;
const transferKey = (t: TransferProgress) => `${t.peerId}:${t.xferId}:${t.direction}`;

export const usePortalStore = create<Store>((set) => ({
  screen: "welcome",
  setScreen: (s) => set({ screen: s }),

  nickname: localStorage.getItem("portal:nick") ?? "",
  setNickname: (n) => {
    localStorage.setItem("portal:nick", n);
    set({ nickname: n });
  },

  signalingUrl: "",
  setSignalingUrl: (u) => set({ signalingUrl: u }),

  portal: null,
  setPortal: (p) => set({ portal: p }),

  peers: {},
  upsertPeer: (p) =>
    set((s) => ({ peers: { ...s.peers, [p.peerId]: { ...s.peers[p.peerId], ...p } } })),
  removePeer: (id) =>
    set((s) => {
      const { [id]: _gone, ...rest } = s.peers;
      return { peers: rest };
    }),
  clearPeers: () => set({ peers: {} }),

  localServices: [],
  setLocalServices: (s) => set({ localServices: Array.isArray(s) ? s : [] }),

  messages: [],
  addMessage: (m) =>
    set((s) => ({ messages: [...s.messages.slice(-MAX_MESSAGES + 1), m] })),
  clearMessages: () => set({ messages: [] }),

  banner: "",
  setBanner: (b) => set({ banner: b }),

  connecting: false,
  setConnecting: (b) => set({ connecting: b }),

  nat: null,
  setNat: (n) => set({ nat: n }),

  transfers: {},
  upsertTransfer: (t) =>
    set((s) => ({ transfers: { ...s.transfers, [transferKey(t)]: t } })),
  clearFinishedTransfers: () =>
    set((s) => ({
      transfers: Object.fromEntries(
        Object.entries(s.transfers).filter(([, t]) => !t.done)
      ),
    })),

  history: [],
  setHistory: (h) => set({ history: Array.isArray(h) ? h : [] }),

  saveDir: "",
  setSaveDir: (d) => set({ saveDir: d }),
}));
