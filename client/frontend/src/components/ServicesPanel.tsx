import React, { useEffect, useState } from "react";
import { Plus, Server, Link2, Trash2, Globe, Search, Zap } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import type { LocalListener, PeerView, ServiceView } from "../types";
import { app } from "../lib/wails";
import { shortId } from "../lib/format";

type Props = {
  localServices: ServiceView[];
  peers: PeerView[];
  refreshLocalServices: () => Promise<void>;
};

export function ServicesPanel({ localServices, peers, refreshLocalServices }: Props) {
  const [name, setName] = useState("");
  const [port, setPort] = useState<number | "">("");
  const [error, setError] = useState("");
  const [dialing, setDialing] = useState<{ peerId: string; port: number } | null>(null);
  const [dialedAddrs, setDialedAddrs] = useState<Record<string, string>>({});
  const [detected, setDetected] = useState<LocalListener[]>([]);
  const [scanning, setScanning] = useState(false);

  const refreshDetected = async () => {
    setScanning(true);
    try {
      const list = await app.LocalListeners();
      setDetected(list);
    } finally {
      setScanning(false);
    }
  };

  useEffect(() => {
    refreshDetected();
  }, []);

  const exposeDetected = async (l: LocalListener) => {
    try {
      await app.ExposeService(l.process || `tcp:${l.port}`, l.port);
      await refreshLocalServices();
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  const submitExpose = async () => {
    setError("");
    if (typeof port !== "number" || port < 1 || port > 65535) {
      setError("Port 1–65535 oralig'ida bo'lishi kerak");
      return;
    }
    try {
      await app.ExposeService(name || `tcp:${port}`, port);
      setName("");
      setPort("");
      await refreshLocalServices();
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  const removeExposed = async (p: number) => {
    try {
      await app.UnexposeService(p);
      await refreshLocalServices();
    } catch {}
  };

  const dialPeerService = async (peer: PeerView, svc: ServiceView) => {
    setDialing({ peerId: peer.peerId, port: svc.port });
    try {
      const addr = await app.DialService(peer.peerId, svc.port, 0);
      setDialedAddrs((s) => ({ ...s, [`${peer.peerId}:${svc.port}`]: addr }));
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setDialing(null);
    }
  };

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b border-white/5">
        <div className="flex items-center gap-2 mb-3">
          <Server className="w-4 h-4 text-violet-400" strokeWidth={2} />
          <h3 className="font-semibold text-sm">Mening servislarim</h3>
        </div>
        <p className="text-xs text-zinc-500 mb-3">
          Lokal portni mesh ga oching — boshqa peerlar to'g'ridan-to'g'ri ulana oladi.
        </p>
        <div className="flex gap-2 items-stretch">
          <input
            type="text"
            placeholder="Nom (masalan minecraft)"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="input-base flex-1 text-sm"
          />
          <input
            type="number"
            placeholder="Port"
            value={port}
            onChange={(e) => setPort(e.target.value === "" ? "" : Number(e.target.value))}
            className="input-base w-28 text-sm font-mono"
          />
          <button
            onClick={submitExpose}
            className="btn-primary rounded-btn px-3 flex items-center gap-1 text-sm"
          >
            <Plus className="w-4 h-4" strokeWidth={2} />
            Och
          </button>
        </div>
        {error && <div className="text-xs text-rose-400 mt-2">{error}</div>}

        <AnimatePresence>
          {localServices.length > 0 && (
            <motion.div layout className="mt-3 space-y-1.5">
              {localServices.map((s) => (
                <motion.div
                  key={s.port}
                  layout
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  className="panel rounded-input px-3 py-2 flex items-center justify-between text-sm"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Globe className="w-3.5 h-3.5 text-emerald-400" strokeWidth={2} />
                    <span className="font-medium truncate">{s.name}</span>
                    <span className="font-mono text-xs text-zinc-500">{s.protocol}:{s.port}</span>
                  </div>
                  <button
                    onClick={() => removeExposed(s.port)}
                    className="p-1 rounded hover:bg-white/5 text-zinc-400 hover:text-rose-400"
                    title="Olib tashlash"
                  >
                    <Trash2 className="w-3.5 h-3.5" strokeWidth={2} />
                  </button>
                </motion.div>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-5">
        <div>
          <div className="flex items-center justify-between mb-2.5">
            <h3 className="font-semibold text-sm flex items-center gap-2">
              <Search className="w-4 h-4 text-amber-400" strokeWidth={2} />
              Lokalda topilgan portlar
            </h3>
            <button
              onClick={refreshDetected}
              disabled={scanning}
              className="text-[11px] text-zinc-400 hover:text-white disabled:opacity-50"
            >
              {scanning ? "Skanerlanmoqda..." : "Yangilash"}
            </button>
          </div>
          {detected.length === 0 && !scanning && (
            <div className="text-xs text-zinc-500 panel rounded-input px-3 py-3 text-center">
              Lokalda port topilmadi (yoki barchasi tizim portlari).
            </div>
          )}
          <div className="space-y-1.5">
            {detected.map((d) => {
              const exposed = localServices.some((s) => s.port === d.port);
              return (
                <div
                  key={d.port}
                  className="panel rounded-input px-3 py-2 flex items-center justify-between text-sm"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Zap className="w-3.5 h-3.5 text-amber-400 shrink-0" strokeWidth={2} />
                    <span className="truncate">{d.process || "?"}</span>
                    <span className="font-mono text-xs text-zinc-500">tcp:{d.port}</span>
                  </div>
                  {exposed ? (
                    <span className="text-[11px] text-emerald-400 font-medium">ochilgan ✓</span>
                  ) : (
                    <button
                      onClick={() => exposeDetected(d)}
                      className="px-2 py-1 rounded text-[11px] font-medium bg-violet-500/15 text-violet-300 hover:bg-violet-500/25"
                    >
                      Bir click bilan och
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        <h3 className="font-semibold text-sm mb-3 flex items-center gap-2">
          <Link2 className="w-4 h-4 text-cyan-400" strokeWidth={2} />
          Boshqa peerlardagi servislar
        </h3>
        {peers.every((p) => p.services.length === 0) && (
          <div className="text-xs text-zinc-500 text-center py-8">
            Hozircha hech kim servis e'lon qilmagan.
          </div>
        )}
        <div className="space-y-3">
          {peers.map((p) =>
            p.services.length === 0 ? null : (
              <div key={p.peerId} className="panel rounded-card p-3">
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-sm font-medium truncate">{p.nickname || shortId(p.peerId)}</span>
                  <span className="text-xs text-zinc-500 font-mono">{p.virtualIp}</span>
                </div>
                <div className="space-y-1.5">
                  {p.services.map((s) => {
                    const key = `${p.peerId}:${s.port}`;
                    const local = dialedAddrs[key];
                    return (
                      <div
                        key={s.port}
                        className="flex items-center justify-between gap-2 text-xs bg-black/20 rounded p-2"
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          <Globe className="w-3 h-3 text-cyan-400 shrink-0" strokeWidth={2} />
                          <span className="truncate">{s.name}</span>
                          <span className="font-mono text-zinc-500">{s.protocol}:{s.port}</span>
                        </div>
                        {local ? (
                          <span className="font-mono text-emerald-400 text-[10px]">→ {local}</span>
                        ) : (
                          <button
                            disabled={dialing?.peerId === p.peerId && dialing?.port === s.port}
                            onClick={() => dialPeerService(p, s)}
                            className="px-2 py-1 rounded bg-white/[0.06] hover:bg-white/[0.12] text-[11px] font-medium disabled:opacity-50"
                          >
                            Ulash
                          </button>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )
          )}
        </div>
      </div>
    </div>
  );
}
