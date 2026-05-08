// Portal screen layout (post-rework).
//
// Three columns instead of three columns + tabs. The user reported
// that the previous layout was right-heavy: the services form was
// crammed into a 360px panel where TCP/UDP/Both + the Och button
// clipped, while the wide centre column was occupied by a peer table
// that mostly duplicated information already on the left peer list.
//
// New layout:
//
//   ┌─────────────┬──────────────────────────┬─────────────┐
//   │ A'zolar     │ Mening servislarim        │ Chat        │
//   │ (members)   │  + form                   │             │
//   │  + their    │  + lokal portlar          │             │
//   │    services │  + LAN scan               │             │
//   │    (dial)   │                           │             │
//   └─────────────┴──────────────────────────┴─────────────┘
//
// Members column shows each peer's announced services inline with a
// 'Ulash' (dial) button — the old "Boshqa peerlardagi servislar"
// section that was at the bottom of the right tab moves here.
// Offline peers (state==="closed") stay in the list so the user can
// see who was in the room earlier; the per-peer Forget button is
// the only path to actually drop a row.
//
// Chat moves to the right column on its own (no tabs needed). Width
// trims from 360 → 320 since chat doesn't need as much horizontal
// room as the services form did.

import React, { useEffect, useState } from "react";
import { AnimatePresence } from "framer-motion";
import { PortalHeader } from "../components/PortalHeader";
import { PeerCard } from "../components/PeerCard";
import { ChatPanel } from "../components/ChatPanel";
import { ServicesPanel } from "../components/ServicesPanel";
import { StatusBar } from "../components/StatusBar";
import { PeerServicesList } from "../components/PeerServicesList";
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
    // Poll while in a portal so the per-service health indicator
    // (green/red dot) stays current — Go side re-probes targets
    // every 30s, we pull the latest snapshot at the same cadence.
    const t = window.setInterval(refreshLocalServices, 30000);
    return () => window.clearInterval(t);
  }, [portal?.portalId]);

  const refreshLocalServices = async () => {
    try {
      const s = await app.LocalServices();
      setLocalServices(s);
    } catch {}
  };

  // onBack just navigates back to the dashboard. The portal stays
  // CONNECTED in the background — exposed services keep working,
  // peers stay alive, the active-portals strip on Welcome shows it.
  const onBack = () => {
    setScreen("welcome");
  };

  // onClose tears down THIS specific session and goes back. Distinct
  // from onBack so the user can leave the screen without forcing
  // a disconnect. The header surfaces both paths via different
  // buttons.
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
    // online (any non-closed) → offline; within each, owner first,
    // then by state (connected → connecting → others), then alpha.
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

  return (
    <div className="h-full flex flex-col">
      <PortalHeader portal={portal} onBack={onBack} onClose={onClose} />

      {/* Three columns. The members column is wider (340) than before
          (300) so each peer card has room for the inline services
          list with dial buttons. The chat column is narrower (320 vs
          previous 360) since it no longer carries the services tab.
          The center stays as 1fr — that's where the form lives, and
          it has all the breathing room it needs. */}
      <div className="flex-1 grid grid-cols-[340px_1fr_320px] min-h-0">
        {/* Left: members + each peer's services */}
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

          <div className="flex-1 overflow-y-auto p-3 space-y-2.5">
            {/* Self pseudo-card */}
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
                <div key={p.peerId} className="space-y-1.5">
                  <PeerCard
                    peer={p}
                    onForget={
                      p.state === "closed"
                        ? () => removePeer(p.peerId)
                        : undefined
                    }
                  />
                  {/* Inline services for this peer. Renders a small
                      indented list under the card so the dial button
                      sits next to the peer that owns the service —
                      no more cross-referencing the bottom of the
                      right panel. Hidden for offline peers (their
                      services are unreachable until they rejoin). */}
                  {p.state !== "closed" && p.services.length > 0 && (
                    <PeerServicesList peer={p} />
                  )}
                </div>
              ))}
            </AnimatePresence>

            {peers.length === 0 && (
              <div className="text-xs text-zinc-500 text-center mt-6">
                Boshqa peer hali ulanmagan. ID + kodni do'stga ulashing.
              </div>
            )}
          </div>
        </aside>

        {/* Centre: my services panel — form, exposed list, LAN scan,
            local-listener detection. Inherits the wide centre column
            so TCP/UDP/Both/Och never clip. */}
        <main className="flex flex-col min-h-0 overflow-hidden">
          <ServicesPanel
            localServices={localServices}
            peers={sortedPeers.filter((p) => p.state !== "closed")}
            refreshLocalServices={refreshLocalServices}
          />
        </main>

        {/* Right: chat. No more tabs — the services panel that used
            to share this column moved to the centre. The chat lives
            here permanently with full vertical room. */}
        <aside className="border-l border-white/5 flex flex-col min-h-0">
          <ChatPanel
            messages={messages}
            myPeerId={portal.ownPeerId}
            peers={sortedPeers.filter((p) => p.state !== "closed")}
            transfers={transfers.filter((t) =>
              sortedPeers.some((p) => p.peerId === t.peerId)
            )}
          />
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
