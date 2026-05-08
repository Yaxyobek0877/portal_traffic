import React, { useEffect, useState } from "react";
import { Welcome } from "./views/Welcome";
import { PortalView } from "./views/Portal";
import { Settings } from "./views/Settings";
import { Lock } from "./views/Lock";
import { usePortalStore } from "./stores/portalStore";
import { app, subscribe } from "./lib/wails";
import type { UpdateResult } from "./lib/wails";
import { useT } from "./i18n";
import type {
  ChatMessage,
  NATResult,
  PeerEvent,
  PortalReadyEvent,
  PortalSummary,
  TransferProgress,
} from "./types";

export default function App() {
  const { t } = useT();
  const unlocked = usePortalStore((s) => s.unlocked);
  const screen = usePortalStore((s) => s.screen);
  const setScreen = usePortalStore((s) => s.setScreen);
  const upsertSession = usePortalStore((s) => s.upsertSession);
  const removeSession = usePortalStore((s) => s.removeSession);
  const setActiveSession = usePortalStore((s) => s.setActiveSession);
  const setSessionSummaries = usePortalStore((s) => s.setSessionSummaries);
  const upsertPeer = usePortalStore((s) => s.upsertPeer);
  const removePeer = usePortalStore((s) => s.removePeer);
  const clearPeers = usePortalStore((s) => s.clearPeers);
  const addMessage = usePortalStore((s) => s.addMessage);
  const clearMessages = usePortalStore((s) => s.clearMessages);
  const setBanner = usePortalStore((s) => s.setBanner);
  const banner = usePortalStore((s) => s.banner);

  // Auto-dismiss the banner after a few seconds so transient blips
  // don't linger on screen.
  useEffect(() => {
    if (!banner) return;
    const timer = setTimeout(() => setBanner(""), 7000);
    return () => clearTimeout(timer);
  }, [banner, setBanner]);
  const setNat = usePortalStore((s) => s.setNat);
  const upsertTransfer = usePortalStore((s) => s.upsertTransfer);
  const setSaveDir = usePortalStore((s) => s.setSaveDir);

  // Update banner state — null until the first check completes; even
  // if a newer version is available, a dismissed banner stays hidden
  // for the rest of this session.
  const [update, setUpdate] = useState<UpdateResult | null>(null);
  const [updateDismissed, setUpdateDismissed] = useState<boolean>(() => {
    try {
      return sessionStorage.getItem("portal:update-dismissed") === "1";
    } catch {
      return false;
    }
  });

  // Check for an update on startup. The cache means this is essentially
  // free after the first call (24h TTL on the backend side). We don't
  // block the UI — `then` fires asynchronously.
  useEffect(() => {
    app.CheckForUpdate(false)
      .then((res) => {
        if (res?.available) setUpdate(res);
      })
      .catch(() => {});
  }, []);

  // Auto-resume sessions the user had open last time, once they've
  // unlocked the vault. The Go side walks active_sessions and dials
  // each in the background — owner rows recreate (fresh portal_id),
  // joiner rows attempt the saved id+code and silently drop if the
  // portal is gone. Active-portals strip on Welcome shows them as
  // they come up.
  //
  // Guarded so the call only fires when unlock flips from false →
  // true; otherwise a sign-out → sign-back-in cycle would re-dial,
  // which is what we want, but a re-render loop wouldn't.
  useEffect(() => {
    if (!unlocked) return;
    app.ResumeActiveSessions().catch(() => {});
  }, [unlocked]);

  useEffect(() => {
    // One-shot bootstrap calls.
    app.NATInfo().then((r) => r && setNat(r));
    app.SaveDir().then(setSaveDir);
    // Hydrate the sessions map up front so a hot reload (or a
    // launch with sessions already running) shows the active-portals
    // strip immediately instead of waiting for the next event.
    app.ActivePortals().then((list) => {
      if (Array.isArray(list)) setSessionSummaries(list);
    });
    app.ActiveSessionID().then((id) => {
      if (id) setActiveSession(id);
    });

    // refreshSummaries pulls a fresh ActivePortals list. Cheap (one
    // map walk on the Go side) and ensures any race between event
    // arrival and an out-of-band session change is reconciled.
    const refreshSummaries = () => {
      app.ActivePortals().then((list) => {
        if (Array.isArray(list)) setSessionSummaries(list);
      });
    };

    const offs: Array<() => void> = [];

    offs.push(
      subscribe<PortalReadyEvent>("portal:ready", (p) => {
        if (!p || !p.sessionId) return;
        upsertSession({
          sessionId: p.sessionId,
          portal: {
            portalId: p.portalId,
            code: p.code,
            ownerId: p.ownerId,
            ownPeerId: p.ownPeerId,
            ownVip: p.ownVip,
            isOwner: p.isOwner,
            sessionId: p.sessionId,
          },
        });
        // Background sessions don't take the screen — the UI stays
        // wherever it was. Foreground sessions land on the portal
        // screen. portal:switched fires alongside for foreground
        // sessions; we use either signal.
        if (!p.background) {
          setActiveSession(p.sessionId);
          setScreen("portal");
        }
        refreshSummaries();
      })
    );
    offs.push(
      subscribe<{ sessionId: string; portalId: string }>("portal:switched", (p) => {
        if (!p || !p.sessionId) return;
        setActiveSession(p.sessionId);
      })
    );
    offs.push(
      subscribe<PeerEvent>("peer:joining", (e) => {
        if (!e || !e.peer) return;
        upsertPeer({ ...e.peer, sessionId: e.sessionId, state: "connecting" });
        refreshSummaries();
      })
    );
    offs.push(
      subscribe<PeerEvent>("peer:ready", (e) => {
        if (!e || !e.peer) return;
        upsertPeer({ ...e.peer, sessionId: e.sessionId, state: "connected" });
      })
    );
    offs.push(
      subscribe<PeerEvent>("peer:rtt", (e) => {
        if (!e || !e.peer) return;
        upsertPeer({ ...e.peer, sessionId: e.sessionId });
      })
    );
    offs.push(
      subscribe<PeerEvent>("peer:transport", (e) => {
        if (!e || !e.peer) return;
        upsertPeer({ ...e.peer, sessionId: e.sessionId });
      })
    );
    offs.push(
      subscribe<PeerEvent>("peer:left", (e) => {
        if (!e || !e.peer) return;
        // Mark as closed instead of yanking the row out of the
        // session's peer map — the user asked to keep ever-connected
        // peers visible (with an offline indicator) so they can see
        // who was in the room earlier. Explicit removal lives behind
        // the per-peer Forget button now.
        upsertPeer({ ...e.peer, sessionId: e.sessionId, state: "closed" });
        refreshSummaries();
      })
    );
    offs.push(
      subscribe<{ sessionId?: string }>("portal:closed", (p) => {
        const sid = p?.sessionId;
        if (sid) {
          // Specific session ended — drop just that one.
          removeSession(sid);
          // If it was the active one, the store already cleared the
          // top-level peers/messages/portal as part of removeSession.
          // Send the user back to Welcome only when no sessions are
          // left, so a single-portal close doesn't yank them from a
          // background-connected session they're actively using.
          const remaining = usePortalStore.getState().sessions;
          if (Object.keys(remaining).length === 0) {
            clearPeers();
            clearMessages();
            setScreen("welcome");
          }
          setBanner(t("banner.portal_closed"));
        } else {
          // Older event shape (no sessionId) — treat as "everything
          // is gone" for backwards compatibility.
          setBanner(t("banner.portal_closed"));
          clearPeers();
          clearMessages();
          setScreen("welcome");
        }
        refreshSummaries();
      })
    );
    offs.push(subscribe<ChatMessage>("chat", (m) => addMessage(m)));
    offs.push(
      subscribe<PeerEvent>("peer:services", (e) => {
        if (!e || !e.peer) return;
        upsertPeer({ ...e.peer, sessionId: e.sessionId });
      })
    );
    offs.push(
      subscribe<string | { sessionId?: string; message?: string }>("error", (msg) => {
        if (!msg) return;
        if (typeof msg === "string") setBanner(msg);
        else if (msg.message) setBanner(msg.message);
      })
    );
    offs.push(subscribe<NATResult>("nat:result", (r) => setNat(r)));
    offs.push(subscribe<TransferProgress>("transfer:progress", (t) => upsertTransfer(t)));
    offs.push(
      subscribe("reconnecting", () => setBanner(t("banner.signaling_disconnected")))
    );
    offs.push(
      subscribe("reconnected", () => setBanner(t("banner.signaling_reconnected")))
    );
    offs.push(
      subscribe("reconnect_give_up", () => setBanner(t("banner.reconnect_failed")))
    );

    return () => offs.forEach((off) => off());
  }, [
    upsertSession,
    removeSession,
    setActiveSession,
    setSessionSummaries,
    upsertPeer,
    removePeer,
    clearPeers,
    addMessage,
    clearMessages,
    setBanner,
    setScreen,
    setNat,
    upsertTransfer,
    setSaveDir,
    t,
  ]);

  const dismissUpdate = () => {
    setUpdateDismissed(true);
    try {
      sessionStorage.setItem("portal:update-dismissed", "1");
    } catch {}
  };

  return (
    <div className="h-full w-full overflow-hidden">
      {banner && (
        <div className="absolute top-3 left-1/2 -translate-x-1/2 z-50 bg-rose-500/15 border border-rose-500/30 text-rose-200 text-xs px-3 py-1.5 rounded-full backdrop-blur-md shadow-lg">
          {banner}
          <button
            onClick={() => setBanner("")}
            className="ml-3 text-rose-300/70 hover:text-rose-200"
          >
            ×
          </button>
        </div>
      )}
      {update?.available && !updateDismissed && (
        <div className="absolute top-3 right-3 z-40 max-w-[280px] bg-violet-500/15 border border-violet-500/30 text-violet-100 text-xs px-3 py-2 rounded-md backdrop-blur-md shadow-lg flex items-center gap-3">
          <div className="flex-1 min-w-0">
            <div className="opacity-80">{t("update.available")}</div>
            <div className="font-mono font-semibold truncate">v{update.latestVersion}</div>
          </div>
          <button
            onClick={() => app.OpenReleasePage(update.releaseUrl)}
            className="px-2 py-1 rounded-md bg-violet-500/30 hover:bg-violet-500/50 text-violet-50 text-[11px] font-semibold whitespace-nowrap"
          >
            {t("update.download")}
          </button>
          <button
            onClick={dismissUpdate}
            className="text-violet-300/70 hover:text-violet-100"
            title={t("update.dismiss")}
          >
            ×
          </button>
        </div>
      )}
      {/* Vault gate. Until the user creates or enters their master
          password the rest of the UI stays unmounted, so e.g. a peek
          at the laptop can't see portal history or trigger a re-join.
          Banner and update toast intentionally render above this so
          a "Reconnecting..." blip fired by background re-tries is
          still visible — they don't leak any post-unlock state. */}
      {!unlocked && <Lock />}
      {unlocked && screen === "welcome" && <Welcome />}
      {unlocked && screen === "portal" && <PortalView />}
      {unlocked && screen === "settings" && <Settings />}
    </div>
  );
}
