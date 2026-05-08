// Portal screen layout (post-rework v2).
//
// User clarified the previous swap: the CENTER column should host
// services that OTHER peers have opened (the consumption / dial
// view), not the user's own services + form. The user's own
// service management was moved back next to the chat — they're
// both "outgoing" actions (talking to peers / publishing services
// to peers) and pair sensibly in the right column.
//
// Final layout:
//
//   ┌─────────────┬──────────────────────────┬──────────────────┐
//   │ A'zolar     │ Boshqalar ulashgan       │ Mening servislarim│
//   │ (members)   │ servislar                │  + Forma          │
//   │  (offline   │  (PeerServicesGrid)      │ ─────             │
//   │   visible)  │  prominent Ulash         │ Chat              │
//   │             │  buttons per service     │                   │
//   └─────────────┴──────────────────────────┴──────────────────┘
//
// Both side columns are 320 wide; centre is 1fr. The right column
// is split vertically — top half has the user's own services (form
// + exposed list, scrollable), bottom half has the chat.

import React, { useEffect } from "react";
import { AnimatePresence } from "framer-motion";
import { PortalHeader } from "../components/PortalHeader";
import { PeerCard } from "../components/PeerCard";
import { ChatPanel } from "../components/ChatPanel";
import { ServicesPanel } from "../components/ServicesPanel";
import { PeerServicesGrid } from "../components/PeerServicesGrid";
import { StatusBar } from "../components/StatusBar";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";

export function PortalView() {
  const portal = usePortalStore((s) => s.portal);
  const peers = Object.values(usePortalStore((s) => s.peers));
  const messages = usePortalStore((s) => s.messages);
  const localServices = usePortalStore((s) => s.localServices);
  const setLocalServices = usePortalStore((s) => s.setLocalServices);
  const signalingUrl = usePortalStore((s) => s.signalingUrl);
  const setScreen = usePortalStore((s) => s.setScreen);
  const setPortal = usePortalStore((s) => s.setPortal);
  const removePeer = usePortalStore((s) => s.removePeer);
  const clearPeers = usePortalStore((s) => s.clearPeers);
  const clearMessages = usePortalStore((s) => s.clearMessages);
  const nickname = usePortalStore((s) => s.nickname);
  const nat = usePortalStore((s) => s.nat);
  const transfers = Object.values(usePortalStore((s) => s.transfers));

  useEffect(() => {
    if (!portal?.portalId) return;
    refreshLocalServices();
    const t = window.setInterval(refreshLocalServices, 30000);
    return () => window.clearInterval(t);
  }, [portal?.portalId]);

  const refreshLocalServices = async () => {
    try {
      const s = await app.LocalServices();
      setLocalServices(s);
    } catch {}
  };

  const onBack = () => {
    setScreen("welcome");
  };

  const onClose = async () => {
    await app.Leave();
    setPortal(null);
    clearPeers();
    clearMessages();
    setLocalServices([]);
    setScreen("welcome");
  };

  if (!portal) return null;

  const sortedPeers = [...peers].sort((a, b) => {
    const liveScore = (p: typeof a) => (p.state === "closed" ? 1 : 0);
    const ds = liveScore(a) - liveScore(b);
    if (ds !== 0) return ds;
    const w = (p: typeof a) =>
      (p.isOwner ? 0 : 4) +
      (p.state === "connected" ? 1 : p.state === "connecting" ? 2 : 3);
    const dw = w(a) - w(b);
    if (dw !== 0) return dw;
    return (a.nickname || a.peerId).localeCompare(b.nickname || b.peerId);
  });

  const livePeers = sortedPeers.filter((p) => p.state !== "closed");

  return (
    <div className="h-full flex flex-col">
      <PortalHeader portal={portal} onBack={onBack} onClose={onClose} />

      {/* Three columns: members | peer-services | my-services + chat. */}
      <div className="flex-1 grid grid-cols-[280px_1fr_360px] min-h-0">
        {/* Left: members. Each peer card stays compact — no inline
            services here anymore, since the consumption view is the
            centre. Offline peers render at low opacity with a hover-
            reveal X to forget. */}
        <aside className="border-r border-white/5 flex flex-col min-h-0">
          <div className="px-4 py-3 border-b border-white/5">
            <div className="text-xs uppercase tracking-widest text-zinc-500">
              A'zolar
            </div>
            <div className="text-sm text-zinc-300 mt-0.5">
              {peers.filter((p) => p.state !== "closed").length + 1}
              <span className="text-zinc-500"> / 16</span>
              {peers.some((p) => p.state === "closed") && (
                <span className="ml-2 text-[11px] text-zinc-500">
                  · {peers.filter((p) => p.state === "closed").length} offline
                </span>
              )}
            </div>
          </div>

          <div className="flex-1 overflow-y-auto p-3 space-y-2">
            <div className="panel rounded-card p-3 flex items-center gap-3 border-violet-500/30">
              <div className="w-9 h-9 rounded-full bg-gradient-to-br from-violet-500 to-cyan-400 flex items-center justify-center font-semibold text-sm shrink-0">
                {(nickname || "Y")[0]?.toUpperCase()}
              </div>
              <div className="min-w-0 flex-1">
                <div className="font-medium text-sm">
                  {nickname}{" "}
                  <span className="text-zinc-500 text-xs">(siz)</span>
                </div>
                <div className="text-xs text-zinc-400 font-mono mt-0.5">
                  {portal.ownVip}
                </div>
              </div>
            </div>

            <AnimatePresence mode="popLayout">
              {sortedPeers.map((p) => (
                <PeerCard
                  key={p.peerId}
                  peer={p}
                  onForget={
                    p.state === "closed"
                      ? () => removePeer(p.peerId)
                      : undefined
                  }
                />
              ))}
            </AnimatePresence>

            {peers.length === 0 && (
              <div className="text-xs text-zinc-500 text-center mt-6">
                Boshqa peer hali ulanmagan. ID + kodni do'stga ulashing.
              </div>
            )}
          </div>
        </aside>

        {/* Centre: PEER SERVICES — what the user can dial into. */}
        <main className="flex flex-col min-h-0 overflow-hidden">
          <PeerServicesGrid peers={livePeers} />
        </main>

        {/* Right: top half "Mening servislarim" (form + exposed list
            + LAN scan), bottom half Chat. Resizable via flex so the
            user can lean on either when they need more room. */}
        <aside className="border-l border-white/5 flex flex-col min-h-0">
          {/* Top: my services. Capped at 50% of the column so chat
              stays visible. Internal scroll. */}
          <div className="basis-1/2 min-h-0 flex flex-col border-b border-white/5">
            <ServicesPanel
              localServices={localServices}
              peers={livePeers}
              refreshLocalServices={refreshLocalServices}
            />
          </div>
          {/* Bottom: chat. */}
          <div className="basis-1/2 min-h-0 flex flex-col">
            <ChatPanel
              messages={messages}
              myPeerId={portal.ownPeerId}
              peers={livePeers}
              transfers={transfers.filter((t) =>
                livePeers.some((p) => p.peerId === t.peerId),
              )}
            />
          </div>
        </aside>
      </div>

      <StatusBar
        ownVip={portal.ownVip}
        peers={peers}
        signalingUrl={signalingUrl}
        nat={nat}
      />
    </div>
  );
}
