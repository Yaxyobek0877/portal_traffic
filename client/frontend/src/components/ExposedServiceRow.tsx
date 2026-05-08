// ExposedServiceRow — one row in the user's "I have opened these
// ports for the room" list. Originally lived inside ServicesPanel
// as a private component; pulled out so the unified services grid
// in the centre can render it too without duplicating the inline
// target-edit / pause / remove behaviour.
//
// Self-contained per-row state:
//   - editing toggles the LAN-target-rewriter inline (pencil icon)
//   - the rest of the controls just call back into the parent
//
// The parent supplies togglePause / remove / retarget callbacks so
// every consumer (ServicesPanel, UnifiedServicesGrid, anything new
// that wants to show the same row) can wire them to its own
// app.* handlers without ExposedServiceRow knowing about Wails.

import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";
import { Globe, Pause, Play, Trash2, Pencil, Check, X } from "lucide-react";
import type { ServiceView } from "../types";

export type ExposedServiceRowProps = {
  s: ServiceView;
  onTogglePause: () => void;
  onRemove: () => void;
  onRetarget: (target: string) => Promise<boolean>;
};

export function ExposedServiceRow({
  s,
  onTogglePause,
  onRemove,
  onRetarget,
}: ExposedServiceRowProps) {
  const paused = !!s.paused;
  const health = s.health || "unknown";
  const isLocalhost = !s.target || s.target === `127.0.0.1:${s.port}`;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(isLocalhost ? "" : s.target || "");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!editing) {
      setDraft(isLocalhost ? "" : s.target || "");
    }
  }, [s.target, isLocalhost, editing]);

  const submit = async () => {
    setSaving(true);
    try {
      const ok = await onRetarget(draft);
      if (ok) setEditing(false);
    } finally {
      setSaving(false);
    }
  };

  const dotClass = paused
    ? "bg-amber-400 shadow-[0_0_4px_#fbbf24]"
    : health === "ok"
    ? "bg-emerald-400 shadow-[0_0_4px_#34d399]"
    : health === "down"
    ? "bg-rose-400 shadow-[0_0_4px_#fb7185]"
    : "bg-zinc-500";
  const dotLabel = paused
    ? "Pauza qilingan — peer'lar ulanolmaydi"
    : health === "ok"
    ? `Target ${s.target || "localhost"} javob bermoqda`
    : health === "down"
    ? `Ulanish muvaffaqiyatsiz: ${s.healthError || s.target || "?"}\n\nNimalarni tekshirish kerak:\n• Qurilma yoqilganmi?\n• IP manzili to'g'rimi?\n• Shu portda haqiqatan ham servis ishlayaptimi?`
    : s.protocol === "udp"
    ? "UDP holatini avtomatik tekshirib bo'lmaydi — peer ulanganda aniqlanadi"
    : "Holati hali tekshirilmagan";

  if (editing) {
    return (
      <motion.div
        layout
        className="panel rounded-input px-3 py-2 space-y-2 text-sm border-violet-500/30"
      >
        <div className="flex items-center gap-2">
          <Pencil className="w-3.5 h-3.5 text-violet-300 shrink-0" strokeWidth={2} />
          <span className="font-medium truncate flex-1">{s.name}</span>
          <span
            className={`text-[10px] font-mono px-1.5 py-0.5 rounded shrink-0 ${
              s.protocol === "udp"
                ? "bg-cyan-400/10 text-cyan-300"
                : "bg-violet-400/10 text-violet-300"
            }`}
          >
            {s.protocol.toUpperCase()}
          </span>
          <span className="font-mono text-xs text-zinc-500 shrink-0">:{s.port}</span>
        </div>
        <div className="flex gap-2 items-stretch">
          <input
            type="text"
            autoFocus
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") submit();
              if (e.key === "Escape") setEditing(false);
            }}
            placeholder={`LAN target (192.168.1.100:${s.port}) — bo'sh = localhost`}
            className="input-base text-sm flex-1 font-mono"
            disabled={saving}
          />
          <button
            onClick={submit}
            disabled={saving}
            title="Saqlash"
            className="px-2 rounded text-emerald-300 hover:bg-emerald-500/15 disabled:opacity-50"
          >
            <Check className="w-4 h-4" strokeWidth={2} />
          </button>
          <button
            onClick={() => setEditing(false)}
            disabled={saving}
            title="Bekor qilish"
            className="px-2 rounded text-zinc-400 hover:bg-white/[0.07] disabled:opacity-50"
          >
            <X className="w-4 h-4" strokeWidth={2} />
          </button>
        </div>
      </motion.div>
    );
  }

  return (
    <motion.div
      layout
      initial={{ opacity: 0, height: 0 }}
      animate={{ opacity: 1, height: "auto" }}
      exit={{ opacity: 0, height: 0 }}
      className={`panel rounded-input px-3 py-2 flex items-center gap-2 text-sm flex-wrap ${
        paused ? "opacity-60" : ""
      }`}
      title={s.target ? `→ ${s.target}` : undefined}
    >
      <span title={dotLabel} className={`w-2 h-2 rounded-full shrink-0 ${dotClass}`} />
      <Globe
        className={`w-3.5 h-3.5 shrink-0 ${paused ? "text-amber-400" : "text-emerald-400"}`}
        strokeWidth={2}
      />
      <div className="min-w-0 flex-1 leading-tight">
        <div className="truncate font-medium flex items-center gap-1.5">
          {s.name}
          {!isLocalhost && (
            <span
              className="text-[9px] uppercase tracking-wider font-mono px-1 py-0.5 rounded bg-cyan-500/15 text-cyan-300 shrink-0"
              title={`LAN qurilma — ${s.target}`}
            >
              LAN
            </span>
          )}
        </div>
        {!isLocalhost && (
          <div className="font-mono text-[10px] text-cyan-300/70 truncate" title={s.target}>
            → {s.target}
          </div>
        )}
        {isLocalhost && (
          <div className="font-mono text-[10px] text-zinc-500 truncate">
            → 127.0.0.1:{s.port} <span className="text-zinc-600">(lokal)</span>
          </div>
        )}
      </div>
      <span
        className={`text-[10px] font-mono px-1.5 py-0.5 rounded shrink-0 ${
          s.protocol === "udp"
            ? "bg-cyan-400/10 text-cyan-300"
            : "bg-violet-400/10 text-violet-300"
        }`}
      >
        {s.protocol.toUpperCase()}
      </span>
      <span className="font-mono text-xs text-zinc-500 shrink-0">:{s.port}</span>
      <button
        onClick={() => setEditing(true)}
        className="p-1 rounded hover:bg-white/5 text-zinc-400 hover:text-violet-300 shrink-0"
        title="Target o'zgartirish (LAN qurilmaga forward qilish)"
      >
        <Pencil className="w-3.5 h-3.5" strokeWidth={2} />
      </button>
      <button
        onClick={onTogglePause}
        className="p-1 rounded hover:bg-white/5 text-zinc-400 hover:text-amber-300 shrink-0"
        title={paused ? "Davom ettirish" : "Vaqtincha to'xtatish"}
      >
        {paused ? <Play className="w-3.5 h-3.5" strokeWidth={2} /> : <Pause className="w-3.5 h-3.5" strokeWidth={2} />}
      </button>
      <button
        onClick={onRemove}
        className="p-1 rounded hover:bg-white/5 text-zinc-400 hover:text-rose-400 shrink-0"
        title="Olib tashlash"
      >
        <Trash2 className="w-3.5 h-3.5" strokeWidth={2} />
      </button>
    </motion.div>
  );
}
