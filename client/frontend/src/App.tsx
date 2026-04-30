import React, { useEffect } from "react";
import { Welcome } from "./views/Welcome";
import { PortalView } from "./views/Portal";
import { Settings } from "./views/Settings";
import { usePortalStore } from "./stores/portalStore";
import { app, subscribe } from "./lib/wails";
import type {
  ChatMessage,
  NATResult,
  PeerView,
  PortalView as PortalT,
  TransferProgress,
} from "./types";

export default function App() {
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
  const setNat = usePortalStore((s) => s.setNat);
  const upsertTransfer = usePortalStore((s) => s.upsertTransfer);
  const setSaveDir = usePortalStore((s) => s.setSaveDir);

  useEffect(() => {
    // One-shot bootstrap calls.
    app.NATInfo().then((r) => r && setNat(r));
    app.SaveDir().then(setSaveDir);

    const offs: Array<() => void> = [];

    offs.push(
      subscribe<PortalT>("portal:ready", (p) => {
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
    offs.push(subscribe<PeerView>("peer:left", (p) => removePeer(p.peerId)));
    offs.push(
      subscribe("portal:closed", () => {
        setBanner("Portal yopildi.");
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
      subscribe("reconnecting", () =>
        setBanner("Signal serveri uzildi — qayta ulanmoqda...")
      )
    );
    offs.push(
      subscribe("reconnected", () => setBanner("Qayta ulandi ✓"))
    );
    offs.push(
      subscribe("reconnect_give_up", () =>
        setBanner("Qayta ulanish muvaffaqiyatsiz. Qaytadan portal yarating.")
      )
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
  ]);

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
      {screen === "welcome" && <Welcome />}
      {screen === "portal" && <PortalView />}
      {screen === "settings" && <Settings />}
    </div>
  );
}
