// Single Zustand store for the desktop app's runtime state. Mirrors
// what Wails events stream over from the Go side so the UI never
// has to poll.
//
// Multi-portal model (rewrite):
//
//   - sessions[sessionId] holds the per-portal state: PortalSummary,
//     peers map, messages, transfers. One entry per live session.
//   - activeSessionId is the localID of the foreground session — the
//     one whose peers / chat / services the Portal screen renders.
//     "" while the user is on Welcome / no session foreground.
//   - portal / peers / messages / transfers at the top level are
//     **derived** projections of the active session, kept in sync by
//     the action helpers below. Older view code (Portal.tsx, ChatPanel,
//     etc.) that read these without knowing about sessions keeps
//     working unchanged.
//
// When a peer event arrives, the helper updates both the session's
// own peers map and (if it's the active session) the top-level one.
// When portal:switched fires, setActiveSession swaps both the id and
// the projections in one set() so readers never see a half-updated
// store.

import { create } from "zustand";
import type {
  ChatMessage,
  HistoryEntry,
  NATResult,
  PeerView,
  PortalSummary,
  PortalView,
  ServiceView,
  TransferProgress,
} from "../types";

type Screen = "welcome" | "portal" | "settings";

const REMEMBER_KEY = "portal:remembered";

const initialRemembered =
  typeof localStorage !== "undefined" &&
  localStorage.getItem(REMEMBER_KEY) === "1";

// SessionState is everything we track per active portal connection.
// Mirrors the Go-side portalSession but only the bits the UI needs.
export type SessionState = {
  sessionId: string;
  summary: PortalSummary;
  // The full PortalView (with code for owners) — fed in from
  // CreatePortal/JoinPortal return values and the portal:ready event.
  // Stays available across switches so the Portal screen can show
  // ID/code immediately.
  portal: PortalView | null;
  peers: Record<string, PeerView>;
  messages: ChatMessage[];
  transfers: Record<string, TransferProgress>;
  // Optional user-given label, hydrated from RecentPortals on
  // session bring-up. Lets the active-portals strip render
  // "Oilaviy portal" instead of just "616530".
  label: string;
};

type Store = {
  unlocked: boolean;
  setUnlocked: (b: boolean) => void;

  remembered: boolean;
  setRemembered: (b: boolean) => void;

  screen: Screen;
  setScreen: (s: Screen) => void;

  nickname: string;
  setNickname: (n: string) => void;

  signalingUrl: string;
  setSignalingUrl: (u: string) => void;

  // Multi-portal state. Sessions are keyed by their Go-side localID.
  sessions: Record<string, SessionState>;
  activeSessionId: string;

  // Derived/active projections. Kept top-level so view components
  // that don't know about sessions yet (PeerTable, ChatPanel,
  // ServicesPanel, TransferRow) continue to work without changes.
  portal: PortalView | null;
  peers: Record<string, PeerView>;
  messages: ChatMessage[];
  transfers: Record<string, TransferProgress>;
  localServices: ServiceView[];

  setLocalServices: (s: ServiceView[]) => void;

  // Multi-portal action helpers. Always go through these — never
  // mutate sessions / peers / messages directly from outside.
  upsertSession: (init: Partial<SessionState> & { sessionId: string }) => void;
  removeSession: (sessionId: string) => void;
  setActiveSession: (sessionId: string) => void;
  setSessionSummaries: (list: PortalSummary[]) => void;

  upsertPeer: (p: PeerView) => void;
  removePeer: (peerId: string, sessionId?: string) => void;
  clearPeers: () => void;

  setPortal: (p: PortalView | null) => void;

  addMessage: (m: ChatMessage) => void;
  clearMessages: () => void;

  banner: string;
  setBanner: (b: string) => void;

  connecting: boolean;
  setConnecting: (b: boolean) => void;

  nat: NATResult | null;
  setNat: (n: NATResult | null) => void;

  upsertTransfer: (t: TransferProgress) => void;
  clearFinishedTransfers: () => void;

  history: HistoryEntry[];
  setHistory: (h: HistoryEntry[]) => void;

  saveDir: string;
  setSaveDir: (d: string) => void;
};

const MAX_MESSAGES = 500;
const transferKey = (t: TransferProgress) =>
  `${t.peerId}:${t.xferId}:${t.direction}`;

// emptySummary fills in the PortalSummary skeleton we use when an
// event arrives before ActivePortals() has caught up. The strip will
// still render — peerCount=0, state="connecting" — and the next
// poll fills in the rest.
function emptySummary(sessionId: string): PortalSummary {
  return {
    sessionId,
    portalId: "",
    nickname: "",
    isOwner: false,
    state: "connecting",
    peerCount: 0,
    isActive: false,
  };
}

export const usePortalStore = create<Store>((set) => ({
  unlocked: initialRemembered,
  setUnlocked: (b) => set({ unlocked: b }),

  remembered: initialRemembered,
  setRemembered: (b) => {
    try {
      if (b) localStorage.setItem(REMEMBER_KEY, "1");
      else localStorage.removeItem(REMEMBER_KEY);
    } catch {}
    set({ remembered: b });
  },

  screen: "welcome",
  setScreen: (s) => set({ screen: s }),

  nickname: localStorage.getItem("portal:nick") ?? "",
  setNickname: (n) => {
    localStorage.setItem("portal:nick", n);
    set({ nickname: n });
  },

  signalingUrl: "",
  setSignalingUrl: (u) => set({ signalingUrl: u }),

  sessions: {},
  activeSessionId: "",

  portal: null,
  peers: {},
  messages: [],
  transfers: {},
  localServices: [],

  setLocalServices: (s) => set({ localServices: Array.isArray(s) ? s : [] }),

  upsertSession: (init) =>
    set((state) => {
      const existing = state.sessions[init.sessionId];
      const merged: SessionState = {
        sessionId: init.sessionId,
        summary: init.summary ?? existing?.summary ?? emptySummary(init.sessionId),
        portal: init.portal !== undefined ? init.portal : existing?.portal ?? null,
        peers: init.peers ?? existing?.peers ?? {},
        messages: init.messages ?? existing?.messages ?? [],
        transfers: init.transfers ?? existing?.transfers ?? {},
        label: init.label ?? existing?.label ?? "",
      };
      const sessions = { ...state.sessions, [init.sessionId]: merged };
      // Keep top-level projection in sync if this is the active one.
      const isActive = init.sessionId === state.activeSessionId;
      const patch: Partial<Store> = { sessions };
      if (isActive) {
        if (init.portal !== undefined) patch.portal = merged.portal;
        if (init.peers !== undefined) patch.peers = merged.peers;
        if (init.messages !== undefined) patch.messages = merged.messages;
        if (init.transfers !== undefined) patch.transfers = merged.transfers;
      }
      return patch;
    }),

  removeSession: (sessionId) =>
    set((state) => {
      const { [sessionId]: gone, ...rest } = state.sessions;
      void gone;
      const wasActive = state.activeSessionId === sessionId;
      return {
        sessions: rest,
        activeSessionId: wasActive ? "" : state.activeSessionId,
        ...(wasActive
          ? { portal: null, peers: {}, messages: [], transfers: {} }
          : {}),
      };
    }),

  setActiveSession: (sessionId) =>
    set((state) => {
      const next = state.sessions[sessionId];
      // Update the isActive flag on every summary so the dashboard
      // strip re-renders the highlight.
      const sessions: Record<string, SessionState> = {};
      for (const [id, s] of Object.entries(state.sessions)) {
        sessions[id] = {
          ...s,
          summary: { ...s.summary, isActive: id === sessionId },
        };
      }
      // Don't blank the existing top-level projection if we're
      // racing setPortal: a freshly created portal can hit this
      // reducer before the session entry has its `portal` populated
      // (the create response returns first; portal:ready / event-
      // bus updates land microseconds later). Prefer the session's
      // own data when present, else keep whatever the previous
      // foreground had — the next event will reconcile.
      return {
        sessions,
        activeSessionId: sessionId,
        portal: next?.portal ?? state.portal,
        peers: next?.peers ?? state.peers,
        messages: next?.messages ?? state.messages,
        transfers: next?.transfers ?? state.transfers,
      };
    }),

  setSessionSummaries: (list) =>
    set((state) => {
      const next: Record<string, SessionState> = {};
      for (const sum of list) {
        const existing = state.sessions[sum.sessionId];
        next[sum.sessionId] = existing
          ? { ...existing, summary: sum }
          : {
              sessionId: sum.sessionId,
              summary: sum,
              portal: null,
              peers: {},
              messages: [],
              transfers: {},
              label: "",
            };
      }
      // Drop sessions that aren't in the list anymore (server says
      // they're gone). Carry over messages/peers from existing where
      // present so we don't lose UI history mid-render race.
      return { sessions: next };
    }),

  upsertPeer: (p) =>
    set((state) => {
      const sid = p.sessionId || state.activeSessionId;
      const sessions = { ...state.sessions };
      const session = sessions[sid];
      if (session) {
        const peer = { ...session.peers[p.peerId], ...p };
        sessions[sid] = {
          ...session,
          peers: { ...session.peers, [p.peerId]: peer },
          summary: {
            ...session.summary,
            peerCount: Object.keys({ ...session.peers, [p.peerId]: peer }).length,
          },
        };
      }
      const patch: Partial<Store> = { sessions };
      if (sid === state.activeSessionId) {
        patch.peers = sessions[sid]?.peers ?? state.peers;
      }
      return patch;
    }),

  removePeer: (peerId, sessionId) =>
    set((state) => {
      const sid = sessionId || state.activeSessionId;
      const sessions = { ...state.sessions };
      const session = sessions[sid];
      if (session) {
        const { [peerId]: _gone, ...rest } = session.peers;
        sessions[sid] = {
          ...session,
          peers: rest,
          summary: { ...session.summary, peerCount: Object.keys(rest).length },
        };
      }
      const patch: Partial<Store> = { sessions };
      if (sid === state.activeSessionId) {
        patch.peers = sessions[sid]?.peers ?? state.peers;
      }
      return patch;
    }),

  clearPeers: () =>
    set((state) => {
      const sid = state.activeSessionId;
      if (!sid) return { peers: {} };
      const sessions = { ...state.sessions };
      if (sessions[sid]) {
        sessions[sid] = { ...sessions[sid], peers: {} };
      }
      return { sessions, peers: {} };
    }),

  setPortal: (p) =>
    set((state) => {
      // setPortal is the legacy hook used by Welcome's create/join
      // returns. It updates BOTH the active session's portal and the
      // top-level projection.
      if (!p) return { portal: null };
      const sid = (p as any).sessionId || state.activeSessionId;
      if (!sid) return { portal: p };
      const sessions = { ...state.sessions };
      const existing = sessions[sid];
      sessions[sid] = {
        sessionId: sid,
        summary: existing?.summary ?? emptySummary(sid),
        portal: p,
        peers: existing?.peers ?? {},
        messages: existing?.messages ?? [],
        transfers: existing?.transfers ?? {},
        label: existing?.label ?? "",
      };
      return {
        sessions,
        portal: sid === state.activeSessionId ? p : state.portal,
      };
    }),

  addMessage: (m) =>
    set((state) => {
      const sid = m.sessionId || state.activeSessionId;
      const sessions = { ...state.sessions };
      const session = sessions[sid];
      if (session) {
        const messages = [...session.messages.slice(-MAX_MESSAGES + 1), m];
        sessions[sid] = { ...session, messages };
      }
      const patch: Partial<Store> = { sessions };
      if (sid === state.activeSessionId) {
        patch.messages = sessions[sid]?.messages ?? state.messages;
      }
      return patch;
    }),

  clearMessages: () =>
    set((state) => {
      const sid = state.activeSessionId;
      if (!sid) return { messages: [] };
      const sessions = { ...state.sessions };
      if (sessions[sid]) {
        sessions[sid] = { ...sessions[sid], messages: [] };
      }
      return { sessions, messages: [] };
    }),

  banner: "",
  setBanner: (b) => set({ banner: b }),

  connecting: false,
  setConnecting: (b) => set({ connecting: b }),

  nat: null,
  setNat: (n) => set({ nat: n }),

  upsertTransfer: (t) =>
    set((state) => {
      const sid = t.sessionId || state.activeSessionId;
      const key = transferKey(t);
      const sessions = { ...state.sessions };
      const session = sessions[sid];
      if (session) {
        sessions[sid] = {
          ...session,
          transfers: { ...session.transfers, [key]: t },
        };
      }
      const patch: Partial<Store> = { sessions };
      if (sid === state.activeSessionId) {
        patch.transfers = sessions[sid]?.transfers ?? state.transfers;
      }
      return patch;
    }),

  clearFinishedTransfers: () =>
    set((state) => {
      const sid = state.activeSessionId;
      const filtered = (m: Record<string, TransferProgress>) =>
        Object.fromEntries(Object.entries(m).filter(([, t]) => !t.done));
      const sessions = { ...state.sessions };
      if (sid && sessions[sid]) {
        sessions[sid] = { ...sessions[sid], transfers: filtered(sessions[sid].transfers) };
      }
      return {
        sessions,
        transfers: sid ? sessions[sid]?.transfers ?? state.transfers : filtered(state.transfers),
      };
    }),

  history: [],
  setHistory: (h) => set({ history: Array.isArray(h) ? h : [] }),

  saveDir: "",
  setSaveDir: (d) => set({ saveDir: d }),
}));
