import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
  ArrowLeft,
  Copy,
  FileText,
  Folder,
  Globe,
  History,
  Info,
  RefreshCcw,
  Shield,
  Trash2,
  Zap,
} from "lucide-react";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";
import type { HistoryEntry, TurnConfig } from "../types";

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

  const [turn, setTurn] = useState<TurnConfig>({ url: "", username: "", credential: "" });
  const [turnSavedAt, setTurnSavedAt] = useState<number | null>(null);
  const [turnError, setTurnError] = useState("");

  const [logs, setLogs] = useState<string[]>([]);
  const [logPath, setLogPath] = useState("");
  const [logsOpen, setLogsOpen] = useState(false);

  useEffect(() => {
    setDraftUrl(signalingUrl);
  }, [signalingUrl]);

  useEffect(() => {
    app.SaveDir().then(setSaveDir);
    app.RecentPortals(20).then(setHistory);
    app.GetTurnConfig().then(setTurn);
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
    const t = await app.GetTurnConfig();
    setTurn(t);
  };

  const saveTurn = async () => {
    setTurnError("");
    try {
      await app.SetTurnConfig(turn);
      setTurnSavedAt(Date.now());
    } catch (e: any) {
      setTurnError(e?.message || String(e));
    }
  };

  const clearTurn = async () => {
    const empty = { url: "", username: "", credential: "" };
    setTurn(empty);
    try {
      await app.SetTurnConfig(empty);
      setTurnSavedAt(Date.now());
    } catch (e: any) {
      setTurnError(e?.message || String(e));
    }
  };

  // Open Relay Project public TURN. We list every variant they
  // expose: UDP/80, TCP/80, TLS/443. Pion picks whichever the
  // network actually permits, which matters a lot on mobile carriers
  // that block UDP outbound or specific ports.
  const fillFreePublicTurn = () => {
    setTurn({
      url: [
        "turn:openrelay.metered.ca:80",
        "turn:openrelay.metered.ca:80?transport=tcp",
        "turns:openrelay.metered.ca:443?transport=tcp",
      ].join("\n"),
      username: "openrelayproject",
      credential: "openrelayproject",
    });
  };

  const showLogs = async () => {
    const [lines, path] = await Promise.all([app.LogLines(200), app.LogFilePath()]);
    setLogs(lines);
    setLogPath(path);
    setLogsOpen(true);
  };
  const copyLogs = async () => {
    try {
      await navigator.clipboard.writeText(logs.join("\n"));
    } catch {}
  };
  const clearLogsLocal = async () => {
    await app.ClearLogs();
    setLogs([]);
  };

  return (
    <div className="h-full flex flex-col">
      <div className="draggable titlebar-pad flex items-center gap-3 px-4 border-b border-white/5" style={{ height: 88 }}>
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

        {/* TURN */}
        <Section icon={<Shield className="w-4 h-4" />} title="TURN serveri (Simmetrik NAT uchun)">
          <p className="text-xs text-zinc-500 -mt-2">
            Agar siz va do'stingiz har xil tarmoqlarda Simmetrik NAT ortida
            bo'lsangiz, to'g'ridan-to'g'ri ulanish ishlamaydi — TURN serveri
            ma'lumotni o'tkazib beradi. Cloudflare / Twilio / Metered.ca
            yoki o'zingizning coturn instance dan kredensiallarni shu yerga
            kiriting.
          </p>
          <Field label="TURN URL (har qatorga bittadan ham yozish mumkin)">
            <textarea
              rows={3}
              placeholder={"turn:turn.example.com:3478\nturn:turn.example.com:3478?transport=tcp\nturns:turn.example.com:5349"}
              value={turn.url}
              onChange={(e) => setTurn({ ...turn, url: e.target.value })}
              className="input-base w-full font-mono text-xs resize-none leading-relaxed"
              spellCheck={false}
            />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Foydalanuvchi nomi">
              <input
                type="text"
                value={turn.username}
                onChange={(e) => setTurn({ ...turn, username: e.target.value })}
                className="input-base w-full font-mono text-sm"
              />
            </Field>
            <Field label="Parol / kredensial">
              <input
                type="password"
                value={turn.credential}
                onChange={(e) => setTurn({ ...turn, credential: e.target.value })}
                className="input-base w-full font-mono text-sm"
              />
            </Field>
          </div>
          <div className="flex items-center gap-2 flex-wrap">
            <button
              onClick={saveTurn}
              className="btn-primary rounded-btn px-4 py-2 text-sm font-medium"
            >
              Saqlash
            </button>
            <button
              onClick={fillFreePublicTurn}
              className="panel rounded-btn px-3 py-2 text-xs flex items-center gap-1.5 hover:bg-white/[0.07]"
              title="Open Relay Project — bepul, sekinroq, lekin tezda sinash uchun yetadi"
            >
              <Zap className="w-3.5 h-3.5 text-amber-400" />
              Bepul TURN ni yoqish
            </button>
            {turn.url && (
              <button
                onClick={clearTurn}
                className="px-3 py-2 text-xs text-zinc-400 hover:text-rose-300 hover:bg-rose-500/10 rounded-btn"
              >
                Tozalash
              </button>
            )}
            {turnError && <span className="text-xs text-rose-400">{turnError}</span>}
            {turnSavedAt && !turnError && (
              <span className="text-xs text-emerald-400">Saqlandi ✓</span>
            )}
          </div>
          <p className="text-xs text-zinc-600 mt-1">
            Sozlangandan keyin keyingi portal yaratish/qo'shilishda kuchga kiradi.
            "Bepul TURN" — tezda sinash uchun: Open Relay Project ning ommaviy
            relay, doim ishlamasligi mumkin.
          </p>
        </Section>

        {/* Logs */}
        <Section icon={<FileText className="w-4 h-4" />} title="Loglar">
          <p className="text-xs text-zinc-500 -mt-2">
            Dastur ichidagi voqealar oxirgi 500 qatorda saqlanadi va bir
            kunlik fayl sifatida ham diskka yoziladi.
          </p>
          <div className="flex gap-2 flex-wrap">
            <button
              onClick={showLogs}
              className="btn-primary rounded-btn px-4 py-2 text-sm font-medium"
            >
              Oxirgi 200 qatorni ko'rish
            </button>
            <button
              onClick={() => app.OpenLogFolder()}
              className="panel rounded-btn px-3 py-2 text-xs flex items-center gap-1.5 hover:bg-white/[0.07]"
            >
              <Folder className="w-3.5 h-3.5" />
              Fayl papkasini ochish
            </button>
          </div>
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

      {logsOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="fixed inset-0 z-[60] bg-black/70 backdrop-blur-md flex items-center justify-center p-6 titlebar-pad"
          onClick={() => setLogsOpen(false)}
        >
          <motion.div
            initial={{ scale: 0.95, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ type: "spring", stiffness: 320, damping: 26 }}
            className="bg-[#0d1322] panel rounded-card flex flex-col w-full max-w-3xl max-h-[80vh] overflow-hidden"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="px-5 py-3 border-b border-white/5 flex items-center justify-between">
              <div className="flex items-center gap-2 min-w-0">
                <FileText className="w-4 h-4 text-violet-300 shrink-0" />
                <span className="font-mono text-xs text-zinc-400 truncate">{logPath}</span>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <button
                  onClick={copyLogs}
                  className="text-xs text-zinc-400 hover:text-white px-2 py-1 rounded hover:bg-white/[0.05] flex items-center gap-1"
                >
                  <Copy className="w-3 h-3" /> Nusxa
                </button>
                <button
                  onClick={clearLogsLocal}
                  className="text-xs text-zinc-400 hover:text-rose-300 px-2 py-1 rounded hover:bg-rose-500/10"
                >
                  Tozalash
                </button>
                <button
                  onClick={() => setLogsOpen(false)}
                  className="text-xs text-zinc-400 hover:text-white px-2 py-1 rounded hover:bg-white/[0.05]"
                >
                  Yopish
                </button>
              </div>
            </div>
            <pre className="flex-1 overflow-auto px-5 py-4 font-mono text-[11px] text-zinc-300 leading-relaxed whitespace-pre-wrap">
              {logs.length === 0 ? "(bo'sh)" : logs.join("\n")}
            </pre>
          </motion.div>
        </motion.div>
      )}
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
