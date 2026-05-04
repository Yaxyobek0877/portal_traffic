import React, { useEffect, useState } from "react";
import { Welcome } from "./views/Welcome";
import { PortalView } from "./views/Portal";
import { Settings } from "./views/Settings";
import { usePortalStore } from "./stores/portalStore";
import { app, subscribe } from "./lib/wails";
import type { UpdateResult } from "./lib/wails";
import { useT } from "./i18n";
import type {
  ChatMessage,
  NATResult,
  PeerView,
  PortalView as PortalT,
  TransferProgress,
} from "./types";

export default function App() {
  const { t } = useT();
  const screen = usePortalStore((s) => s.screen);
  const setScreen = usePortalStore((s) => s.setScreen);
  const setPortal = usePortalStore((s) => s.setPortal);
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

  useEffect(() => {
    // One-shot bootstrap calls.
    app.NATInfo().then((r) => r && setNat(r));
    app.SaveDir().then(setSaveDir);

    const offs: Array<() => void> = [];

    offs.push(
      subscribe<PortalT>("portal:ready", (p) => {
        // Defensive: an older backend (or a backend with the legacy
        // bug where *ev.Portal was emitted directly) sends payloads
        // shaped like {PortalID:…, Code:…} — capitalised Go field
        // names instead of our lowerCamel JSON tags. Ignore those
        // rather than letting them overwrite the good data Welcome
        // already put in the store from app.CreatePortal()'s return.
        if (!p || !p.portalId) {
          return;
        }
        setPortal(p);
        setScreen("portal");
      })
    );
    offs.push(
      subscribe<PeerView>("peer:joining", (p) => upsertPeer({ ...p, state: "connecting" }))
    );
    offs.push(
      subscribe<PeerView>("peer:ready", (p) => upsertPeer({ ...p, state: "connected" }))
    );
    offs.push(subscribe<PeerView>("peer:rtt", (p) => upsertPeer(p)));
    offs.push(subscribe<PeerView>("peer:transport", (p) => upsertPeer(p)));
    offs.push(subscribe<PeerView>("peer:left", (p) => removePeer(p.peerId)));
    offs.push(
      subscribe("portal:closed", () => {
        setBanner(t("banner.portal_closed"));
        setPortal(null);
        clearPeers();
        clearMessages();
        setScreen("welcome");
      })
    );
    offs.push(subscribe<ChatMessage>("chat", (m) => addMessage(m)));
    offs.push(subscribe<PeerView>("peer:services", (p) => upsertPeer(p)));
    offs.push(subscribe<string>("error", (msg) => msg && setBanner(msg)));
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
    setPortal,
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
      {screen === "welcome" && <Welcome />}
      {screen === "portal" && <PortalView />}
      {screen === "settings" && <Settings />}
    </div>
  );
}
