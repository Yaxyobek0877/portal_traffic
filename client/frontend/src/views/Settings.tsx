import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
  ArrowLeft,
  Folder,
  Globe,
  History,
  Info,
  RefreshCcw,
  Trash2,
} from "lucide-react";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";
import type { HistoryEntry } from "../types";

export function Settings() {
  const setScreen = usePortalStore((s) => s.setScreen);
  const signalingUrl = usePortalStore((s) => s.signalingUrl);
  const setSignalingUrl = usePortalStore((s) => s.setSignalingUrl);
  const nat = usePortalStore((s) => s.nat);
  const saveDir = usePortalStore((s) => s.saveDir);
  const setSaveDir = usePortalStore((s) => s.setSaveDir);
  const history = usePortalStore((s) => s.history);
  const setHistory = usePortalStore((s) => s.setHistory);

  const [draftUrl, setDraftUrl] = useState(signalingUrl);
  const [savedAt, setSavedAt] = useState<number | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    setDraftUrl(signalingUrl);
  }, [signalingUrl]);

  useEffect(() => {
    app.SaveDir().then(setSaveDir);
    app.RecentPortals(20).then(setHistory);
  }, [setSaveDir, setHistory]);

  const saveUrl = async () => {
    setError("");
    try {
      await app.SetSignalingURL(draftUrl);
      setSignalingUrl(draftUrl);
      setSavedAt(Date.now());
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  const refresh = async () => {
    const r = await app.NATInfo();
    usePortalStore.getState().setNat(r);
    const h = await app.RecentPortals(20);
    setHistory(h);
  };

  return (
    <div className="h-full flex flex-col">
      <div className="draggable h-[60px] flex items-center gap-3 px-4 border-b border-white/5">
        <button
          onClick={() => setScreen("welcome")}
          className="no-drag p-2 rounded-md hover:bg-white/5 text-zinc-400"
        >
          <ArrowLeft className="w-4 h-4" />
        </button>
        <h2 className="text-base font-semibold">Sozlamalar</h2>
        <button
          onClick={refresh}
          className="no-drag ml-auto p-2 rounded-md hover:bg-white/5 text-zinc-400"
          title="Yangilash"
        >
          <RefreshCcw className="w-4 h-4" />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-6 space-y-6 max-w-3xl mx-auto w-full">
        {/* Network */}
        <Section icon={<Globe className="w-4 h-4" />} title="Tarmoq">
          <Field label="Signal serveri URL">
            <div className="flex gap-2">
              <input
                type="text"
                value={draftUrl}
                onChange={(e) => setDraftUrl(e.target.value)}
                className="input-base flex-1 font-mono text-sm"
              />
              <button
                onClick={saveUrl}
                className="btn-primary rounded-btn px-4 text-sm font-medium"
              >
                Saqlash
              </button>
            </div>
            {error && <div className="text-xs text-rose-400 mt-1.5">{error}</div>}
            {savedAt && !error && (
              <div className="text-xs text-emerald-400 mt-1.5">Saqlandi.</div>
            )}
            <p className="text-xs text-zinc-500 mt-2">
              Standart: <span className="font-mono">wss://signaling.1pro.uz/ws</span>
            </p>
          </Field>
        </Section>

        {/* Diagnostics */}
        <Section icon={<Info className="w-4 h-4" />} title="Diagnostika">
          <Field label="NAT turi">
            {nat ? (
              <div className="space-y-1.5">
                <div
                  className={`text-base font-semibold ${
                    nat.type === 1
                      ? "text-emerald-400"
                      : nat.type === 2
                      ? "text-amber-400"
                      : nat.type === 3
                      ? "text-rose-400"
                      : "text-zinc-300"
                  }`}
                >
                  {nat.label}
                </div>
                {nat.needsTurn && (
                  <div className="text-xs text-amber-300 panel rounded-input px-3 py-2 border-amber-500/30 bg-amber-500/5">
                    ⚠ Simmetrik NAT/CGNAT aniqlandi. To'g'ridan-to'g'ri ulanish
                    ishlamasligi mumkin — TURN serveri sozlash tavsiya etiladi.
                  </div>
                )}
                <div className="text-xs text-zinc-500 font-mono space-y-0.5">
                  <div>lokal: {nat.localAddr}</div>
                  {nat.reflexiveAddrs.map((a, i) => (
                    <div key={i}>tashqi: {a}</div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="text-sm text-zinc-500">Aniqlanmoqda...</div>
            )}
          </Field>
        </Section>

        {/* Files */}
        <Section icon={<Folder className="w-4 h-4" />} title="Fayllar">
          <Field label="Qabul qilingan fayllar joylashuvi">
            <div className="flex gap-2 items-center">
              <code className="font-mono text-sm flex-1 panel rounded-input px-3 py-2 truncate">
                {saveDir || "—"}
              </code>
              <button
                onClick={() => app.OpenSaveDir()}
                className="panel rounded-btn px-3 py-2 text-sm hover:bg-white/[0.07]"
              >
                Ochish
              </button>
            </div>
          </Field>
        </Section>

        {/* History */}
        <Section icon={<History className="w-4 h-4" />} title="Yaqindagi portallar">
          <div>
            {history.length === 0 ? (
              <div className="text-sm text-zinc-500 py-4 text-center">
                Hozircha tarix bo'sh.
              </div>
            ) : (
              <div className="space-y-1.5">
                {history.map((h) => (
                  <HistoryRow key={h.id} h={h} />
                ))}
              </div>
            )}
            {history.length > 0 && (
              <button
                onClick={async () => {
                  await app.ClearHistory();
                  setHistory([]);
                }}
                className="mt-3 text-xs text-rose-400 hover:text-rose-300 flex items-center gap-1"
              >
                <Trash2 className="w-3 h-3" />
                Tarixni tozalash
              </button>
            )}
          </div>
        </Section>
      </div>
    </div>
  );
}

function Section({
  title,
  icon,
  children,
}: {
  title: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      className="panel rounded-card p-5"
    >
      <h3 className="text-sm font-semibold flex items-center gap-2 mb-4">
        <span className="text-violet-300">{icon}</span>
        {title}
      </h3>
      <div className="space-y-4">{children}</div>
    </motion.div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="text-xs uppercase tracking-widest text-zinc-500 mb-1.5 block">
        {label}
      </label>
      {children}
    </div>
  );
}

function HistoryRow({ h }: { h: HistoryEntry }) {
  const dt = new Date(h.lastSeen).toLocaleString();
  return (
    <div className="panel rounded-input px-3 py-2 flex items-center justify-between text-sm">
      <div className="flex items-center gap-3 min-w-0">
        <span className="font-mono text-violet-300">{h.portalId}</span>
        <span className="text-zinc-500">·</span>
        <span className="truncate">{h.nickname}</span>
        {h.isOwner && (
          <span className="text-[10px] uppercase tracking-wider text-amber-400">owner</span>
        )}
      </div>
      <span className="text-xs text-zinc-500 shrink-0 ml-3">{dt}</span>
    </div>
  );
}
