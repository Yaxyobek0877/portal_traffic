import React, { useEffect, useState } from "react";
import { MessageCircle, Server } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { PortalHeader } from "../components/PortalHeader";
import { PeerCard } from "../components/PeerCard";
import { MeshDiagram } from "../components/MeshDiagram";
import { ChatPanel } from "../components/ChatPanel";
import { ServicesPanel } from "../components/ServicesPanel";
import { StatusBar } from "../components/StatusBar";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";

type RightTab = "chat" | "services";

export function PortalView() {
  const portal = usePortalStore((s) => s.portal);
  const peers = Object.values(usePortalStore((s) => s.peers));
  const messages = usePortalStore((s) => s.messages);
  const localServices = usePortalStore((s) => s.localServices);
  const setLocalServices = usePortalStore((s) => s.setLocalServices);
  const signalingUrl = usePortalStore((s) => s.signalingUrl);
  const setScreen = usePortalStore((s) => s.setScreen);
  const setPortal = usePortalStore((s) => s.setPortal);
  const clearPeers = usePortalStore((s) => s.clearPeers);
  const clearMessages = usePortalStore((s) => s.clearMessages);
  const nickname = usePortalStore((s) => s.nickname);
  const nat = usePortalStore((s) => s.nat);
  const transfers = Object.values(usePortalStore((s) => s.transfers));

  const [rightTab, setRightTab] = useState<RightTab>("chat");
  const [hovered, setHovered] = useState<string | null>(null);

  useEffect(() => {
    refreshLocalServices();
  }, [portal?.portalId]);

  const refreshLocalServices = async () => {
    try {
      const s = await app.LocalServices();
      setLocalServices(s);
    } catch {}
  };

  const onLeave = async () => {
    await app.Leave();
    setPortal(null);
    clearPeers();
    clearMessages();
    setLocalServices([]);
    setScreen("welcome");
  };

  if (!portal) return null;

  const sortedPeers = [...peers].sort((a, b) => {
    // owner → connected → connecting → others; alphabetical within
    const w = (p: typeof a) =>
      (p.isOwner ? 0 : 4) + (p.state === "connected" ? 1 : p.state === "connecting" ? 2 : 3);
    const dw = w(a) - w(b);
    if (dw !== 0) return dw;
    return (a.nickname || a.peerId).localeCompare(b.nickname || b.peerId);
  });

  return (
    <div className="h-full flex flex-col">
      <PortalHeader portal={portal} onLeave={onLeave} />

      <div className="flex-1 grid grid-cols-[300px_1fr_360px] min-h-0">
        {/* Left: peer list */}
        <aside className="border-r border-white/5 flex flex-col min-h-0">
          <div className="px-4 py-3 border-b border-white/5">
            <div className="text-xs uppercase tracking-widest text-zinc-500">A'zolar</div>
            <div className="text-sm text-zinc-300 mt-0.5">
              {peers.length + 1} <span className="text-zinc-500">/ 16</span>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto p-3 space-y-2">
            {/* Self pseudo-card */}
            <div className="panel rounded-card p-3 flex items-center gap-3 border-violet-500/30">
              <div className="w-9 h-9 rounded-full bg-gradient-to-br from-violet-500 to-cyan-400 flex items-center justify-center font-semibold text-sm shrink-0">
                {(nickname || "Y")[0]?.toUpperCase()}
              </div>
              <div className="min-w-0 flex-1">
                <div className="font-medium text-sm">
                  {nickname} <span className="text-zinc-500 text-xs">(siz)</span>
                </div>
                <div className="text-xs text-zinc-400 font-mono mt-0.5">{portal.ownVip}</div>
              </div>
            </div>

            <AnimatePresence mode="popLayout">
              {sortedPeers.map((p) => (
                <PeerCard
                  key={p.peerId}
                  peer={p}
                  highlighted={hovered === p.peerId}
                  onClick={() => {}}
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

        {/* Centre: mesh diagram */}
        <main className="flex flex-col min-h-0 items-center justify-center p-4">
          <MeshDiagram
            selfNickname={nickname}
            selfVip={portal.ownVip}
            peers={sortedPeers}
            hovered={hovered}
            onHover={setHovered}
          />
        </main>

        {/* Right: chat / services */}
        <aside className="border-l border-white/5 flex flex-col min-h-0">
          <div className="flex border-b border-white/5">
            <TabButton active={rightTab === "chat"} onClick={() => setRightTab("chat")}>
              <MessageCircle className="w-3.5 h-3.5" />
              Chat
              {messages.length > 0 && (
                <span className="text-[10px] text-zinc-500 ml-1">({messages.length})</span>
              )}
            </TabButton>
            <TabButton
              active={rightTab === "services"}
              onClick={() => setRightTab("services")}
            >
              <Server className="w-3.5 h-3.5" />
              Servislar
              {localServices.length > 0 && (
                <span className="text-[10px] text-emerald-400 ml-1">({localServices.length})</span>
              )}
            </TabButton>
          </div>
          <div className="flex-1 min-h-0">
            {rightTab === "chat" ? (
              <ChatPanel
                messages={messages}
                myPeerId={portal.ownPeerId}
                peers={sortedPeers}
                transfers={transfers.filter((t) => sortedPeers.some((p) => p.peerId === t.peerId))}
              />
            ) : (
              <ServicesPanel
                localServices={localServices}
                peers={sortedPeers}
                refreshLocalServices={refreshLocalServices}
              />
            )}
          </div>
        </aside>
      </div>

      <StatusBar ownVip={portal.ownVip} peers={peers} signalingUrl={signalingUrl} nat={nat} />
    </div>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex-1 h-10 text-xs font-medium flex items-center justify-center gap-1.5 transition-colors ${
        active
          ? "text-white border-b-2 border-violet-500"
          : "text-zinc-500 hover:text-zinc-300 border-b-2 border-transparent"
      }`}
    >
      {children}
    </button>
  );
}
